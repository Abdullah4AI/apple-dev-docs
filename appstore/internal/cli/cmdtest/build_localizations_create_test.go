package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBuildLocalizationsCreateWarnsForIncompleteCreate(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	var seenPayload struct {
		Data struct {
			Attributes struct {
				Locale   string `json:"locale"`
				WhatsNew string `json:"whatsNew"`
			} `json:"attributes"`
			Relationships struct {
				AppStoreVersion struct {
					Data struct {
						ID string `json:"id"`
					} `json:"data"`
				} `json:"appStoreVersion"`
			} `json:"relationships"`
		} `json:"data"`
	}

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/builds/build-1/appStoreVersion":
			body := `{"data":{"type":"appStoreVersions","id":"version-1"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/appStoreVersionLocalizations":
			if err := json.NewDecoder(req.Body).Decode(&seenPayload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			body := `{"data":{"type":"appStoreVersionLocalizations","id":"loc-build-1","attributes":{"locale":"en-US","whatsNew":"Bug fixes"}}}`
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"build-localizations", "create",
			"--build", "build-1",
			"--locale", "en-US",
			"--whats-new", "Bug fixes",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if !strings.Contains(stderr, "created locale en-US now participates in submission validation") {
		t.Fatalf("expected create warning on stderr, got %q", stderr)
	}
	if !strings.Contains(stderr, "description, keywords, supportUrl") {
		t.Fatalf("expected missing submit fields in warning, got %q", stderr)
	}
	if seenPayload.Data.Attributes.Locale != "en-US" {
		t.Fatalf("expected locale en-US, got %q", seenPayload.Data.Attributes.Locale)
	}
	if seenPayload.Data.Attributes.WhatsNew != "Bug fixes" {
		t.Fatalf("expected whatsNew Bug fixes, got %q", seenPayload.Data.Attributes.WhatsNew)
	}
	if seenPayload.Data.Relationships.AppStoreVersion.Data.ID != "version-1" {
		t.Fatalf("expected version relationship version-1, got %q", seenPayload.Data.Relationships.AppStoreVersion.Data.ID)
	}

	var out struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("stdout should be valid json: %v\nstdout=%q", err, stdout)
	}
	if out.Data.ID != "loc-build-1" {
		t.Fatalf("expected localization id loc-build-1, got %q", out.Data.ID)
	}
}
