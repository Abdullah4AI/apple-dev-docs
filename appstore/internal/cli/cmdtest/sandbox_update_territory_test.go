package cmdtest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSandboxUpdateNormalizesTerritory(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch {
			t.Fatalf("expected PATCH request, got %s", req.Method)
		}
		if req.URL.Path != "/v2/sandboxTesters/tester-1" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		body := string(bodyBytes)
		if !strings.Contains(body, `"territory":"USA"`) {
			t.Fatalf("expected normalized sandbox territory in payload, got %s", body)
		}
		return jsonResponse(http.StatusOK, `{
			"data":{
				"type":"sandboxTesters",
				"id":"tester-1",
				"attributes":{"territory":"USA"}
			}
		}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"sandbox", "update", "--id", "tester-1", "--territory", "US"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"tester-1"`) {
		t.Fatalf("expected sandbox tester output, got %q", stdout)
	}
}
