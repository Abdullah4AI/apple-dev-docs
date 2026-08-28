package cmdtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleIDsListSplitRequiresPaginate(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	identifiers := makeLongBundleIDIdentifierFilter()
	requestCount := 0
	setBundleIDPlatformTestServer(t, func(w http.ResponseWriter, req *http.Request) {
		requestCount++
		if req.URL.Query().Get("filter[identifier]") == "" {
			t.Fatalf("request %d unexpectedly followed a continuation URL: %s", requestCount, req.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = io.WriteString(w, `{"data":[{"type":"bundleIds","id":"bundle-first"}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/bundleIds?cursor=chunk-one-next"}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":[{"type":"bundleIds","id":"bundle-second"}]}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{"bundle-ids", "list", "--identifier", identifiers, "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if runErr == nil || !strings.Contains(runErr.Error(), "split identifier filter requires --paginate") {
		t.Fatalf("run error = %v, want pagination requirement", runErr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	wantStderr := "Error: split identifier filter requires --paginate because multiple continuation URLs cannot be represented\n"
	if stderr != wantStderr {
		t.Fatalf("stderr = %q, want %q", stderr, wantStderr)
	}
	if requestCount != 0 {
		t.Fatalf("request count = %d, want no HTTP request", requestCount)
	}
}

func TestBundleIDsListPaginatePreservesIncludedAcrossPages(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	requestCount := 0
	setBundleIDPlatformTestServer(t, func(w http.ResponseWriter, req *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		switch requestCount {
		case 1:
			_, _ = io.WriteString(w, `{"data":[{"type":"bundleIds","id":"bundle-1","attributes":{"name":"One"}}],"included":[{"type":"profiles","id":"profile-1","attributes":{"name":"First"}}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/bundleIds?cursor=second"}}`)
		case 2:
			if req.URL.Query().Get("cursor") != "second" {
				t.Fatalf("continuation query = %q, want cursor=second", req.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"data":[{"type":"bundleIds","id":"bundle-2","attributes":{"name":"Two"}}],"included":[{"type":"profiles","id":"profile-1","attributes":{"name":"First"}},{"type":"profiles","id":"profile-2","attributes":{"name":"Second"}}]}`)
		default:
			t.Fatalf("unexpected request %d", requestCount)
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{"bundle-ids", "list", "--include", "profiles", "--paginate", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	var payload struct {
		Included json.RawMessage `json:"included"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	wantIncluded := `[{"type":"profiles","id":"profile-1","attributes":{"name":"First"}},{"type":"profiles","id":"profile-2","attributes":{"name":"Second"}}]`
	if string(payload.Included) != wantIncluded {
		t.Fatalf("included = %s, want %s", payload.Included, wantIncluded)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want 2", requestCount)
	}
}

func TestBundleIDsListPaginatePreservesEmptyIncluded(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	setBundleIDPlatformTestServer(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[],"included":[],"links":{}}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{"bundle-ids", "list", "--include", "profiles", "--paginate", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	want := `{"data":[],"links":{},"included":[]}`
	if strings.TrimSpace(stdout) != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestBundleIDsListSparseRelationshipResponseOmitsAttributes(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	setBundleIDPlatformTestServer(t, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[{"type":"bundleIds","id":"bundle-1","relationships":{"profiles":{"data":{"type":"profiles","id":"profile-1"}}}}],"links":{}}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{"bundle-ids", "list", "--fields", "profiles", "--include", "profiles", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	want := `{"data":[{"type":"bundleIds","id":"bundle-1","relationships":{"profiles":{"data":{"type":"profiles","id":"profile-1"}}}}],"links":{}}`
	if strings.TrimSpace(stdout) != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestBundleIDsListSplitPaginatesEveryChunk(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	identifiers := makeLongBundleIDIdentifierFilter()
	requestCount := 0
	setBundleIDPlatformTestServer(t, func(w http.ResponseWriter, req *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		switch requestCount {
		case 1:
			if req.URL.Query().Get("filter[identifier]") == "" {
				t.Fatalf("first request missing split identifier filter")
			}
			_, _ = io.WriteString(w, `{"data":[{"type":"bundleIds","id":"bundle-first"}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/bundleIds?cursor=chunk-one-next"}}`)
		case 2:
			if req.URL.Query().Get("cursor") != "chunk-one-next" {
				t.Fatalf("second request query = %q, want first chunk continuation", req.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"data":[{"type":"bundleIds","id":"bundle-continuation"}]}`)
		case 3:
			if req.URL.Query().Get("filter[identifier]") == "" {
				t.Fatalf("third request missing second split identifier filter")
			}
			_, _ = io.WriteString(w, `{"data":[{"type":"bundleIds","id":"bundle-second"}]}`)
		default:
			t.Fatalf("unexpected extra request %d: %s", requestCount, req.URL.String())
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{"bundle-ids", "list", "--identifier", identifiers, "--paginate", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if requestCount != 3 {
		t.Fatalf("request count = %d, want every split page, got %q", requestCount, stdout)
	}
	for _, id := range []string{"bundle-first", "bundle-continuation", "bundle-second"} {
		if !strings.Contains(stdout, `"id":"`+id+`"`) {
			t.Fatalf("stdout = %q, missing %s", stdout, id)
		}
	}
}

func makeLongBundleIDIdentifierFilter() string {
	identifiers := make([]string, 0, 250)
	for i := 0; i < 250; i++ {
		identifiers = append(identifiers, fmt.Sprintf("com.example.%012d", i))
	}
	return strings.Join(identifiers, ",")
}
