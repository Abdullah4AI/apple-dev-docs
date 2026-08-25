package cmdtest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func writePreviewFile(t *testing.T, path string) int64 {
	t.Helper()

	if err := os.WriteFile(path, []byte("preview-bytes-"+filepath.Base(path)), 0o600); err != nil {
		t.Fatalf("write preview %s: %v", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat preview %s: %v", path, err)
	}
	return info.Size()
}

func previewRelationshipIDs(t *testing.T, req *http.Request) []string {
	t.Helper()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read relationship body: %v", err)
	}
	var payload asc.RelationshipRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode relationship body: %v\nbody=%s", err, body)
	}
	ids := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		if item.Type != "appPreviews" {
			t.Fatalf("relationship type = %q, want appPreviews", item.Type)
		}
		ids = append(ids, item.ID)
	}
	return ids
}

// previewUploadTransport serves a two-file preview upload where App Store
// Connect reports the previews in reverse filename order.
func previewUploadTransport(t *testing.T, sizes map[string]int64, remoteOrder []string, onPatch func([]string)) roundTripFunc {
	t.Helper()

	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/LOC_123/appPreviewSets":
			return statusJSONResponse(`{"data":[{"type":"appPreviewSets","id":"set-1","attributes":{"previewType":"IPHONE_65"}}],"links":{}}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/appPreviews":
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read create preview body: %v", err)
			}
			id := "preview-first"
			fileName := "01-first.mov"
			if strings.Contains(string(body), "02-second.mov") {
				id = "preview-second"
				fileName = "02-second.mov"
			}
			return statusJSONResponse(fmt.Sprintf(
				`{"data":{"type":"appPreviews","id":"%s","attributes":{"fileName":"%s","uploadOperations":[{"method":"PUT","url":"https://upload.example/%s","length":%d,"offset":0}]}}}`,
				id, fileName, id, sizes[fileName],
			)), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appPreviewSets/set-1/appPreviews":
			return statusJSONResponse(`{"data":[],"links":{}}`), nil
		case req.Method == http.MethodPut && req.URL.Host == "upload.example":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		case req.Method == http.MethodPatch && strings.HasPrefix(req.URL.Path, "/v1/appPreviews/"):
			id := strings.TrimPrefix(req.URL.Path, "/v1/appPreviews/")
			return statusJSONResponse(fmt.Sprintf(`{"data":{"type":"appPreviews","id":"%s","attributes":{"uploaded":true}}}`, id)), nil
		case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/v1/appPreviews/"):
			id := strings.TrimPrefix(req.URL.Path, "/v1/appPreviews/")
			return statusJSONResponse(fmt.Sprintf(`{"data":{"type":"appPreviews","id":"%s","attributes":{"assetDeliveryState":{"state":"COMPLETE"}}}}`, id)), nil
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appPreviewSets/set-1/relationships/appPreviews":
			linkages := make([]string, 0, len(remoteOrder))
			for _, id := range remoteOrder {
				linkages = append(linkages, fmt.Sprintf(`{"type":"appPreviews","id":"%s"}`, id))
			}
			return statusJSONResponse(fmt.Sprintf(`{"data":[%s],"links":{}}`, strings.Join(linkages, ","))), nil
		case req.Method == http.MethodPatch && req.URL.Path == "/v1/appPreviewSets/set-1/relationships/appPreviews":
			onPatch(previewRelationshipIDs(t, req))
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
}

func TestVideoPreviewsUploadAppliesSortedFileNameOrder(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")

	pathDir := t.TempDir()
	sizes := map[string]int64{
		"01-first.mov":  writePreviewFile(t, filepath.Join(pathDir, "01-first.mov")),
		"02-second.mov": writePreviewFile(t, filepath.Join(pathDir, "02-second.mov")),
	}

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	patched := make([][]string, 0, 1)
	http.DefaultTransport = previewUploadTransport(t, sizes, []string{"preview-second", "preview-first"}, func(ids []string) {
		patched = append(patched, ids)
	})

	stdout, stderr, runErr := runRootCommand(t, []string{
		"video-previews", "upload",
		"--version-localization", "LOC_123",
		"--path", pathDir,
		"--device-type", "IPHONE_65",
		"--output", "json",
	})

	if runErr != nil {
		t.Fatalf("run error: %v (stderr=%q)", runErr, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if len(patched) != 1 {
		t.Fatalf("expected exactly one preview order PATCH, got %d (%v)", len(patched), patched)
	}
	want := []string{"preview-first", "preview-second"}
	if !reflect.DeepEqual(patched[0], want) {
		t.Fatalf("preview order PATCH = %v, want %v", patched[0], want)
	}
	if !strings.Contains(stdout, `"setId":"set-1"`) {
		t.Fatalf("expected receipt for set-1, got %s", stdout)
	}
}

func TestVideoPreviewsUploadSkipsOrderPatchWhenAlreadyOrdered(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	t.Setenv("ASC_APP_ID", "")

	pathDir := t.TempDir()
	sizes := map[string]int64{
		"01-first.mov":  writePreviewFile(t, filepath.Join(pathDir, "01-first.mov")),
		"02-second.mov": writePreviewFile(t, filepath.Join(pathDir, "02-second.mov")),
	}

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = previewUploadTransport(t, sizes, []string{"preview-first", "preview-second"}, func(ids []string) {
		t.Fatalf("expected no preview order PATCH when the set is already ordered, got %v", ids)
	})

	_, stderr, runErr := runRootCommand(t, []string{
		"video-previews", "upload",
		"--version-localization", "LOC_123",
		"--path", pathDir,
		"--device-type", "IPHONE_65",
		"--output", "json",
	})

	if runErr != nil {
		t.Fatalf("run error: %v (stderr=%q)", runErr, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
}
