package cmdtest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSandboxListNormalizesTerritoryFilter(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", req.Method)
		}
		if req.URL.Path != "/v2/sandboxTesters" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{
			"data":[
				{"type":"sandboxTesters","id":"tester-usa","attributes":{"email":"usa@example.com","territory":"USA"}},
				{"type":"sandboxTesters","id":"tester-fra","attributes":{"email":"fra@example.com","territory":"FRA"}}
			]
		}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"sandbox", "list", "--territory", "United States"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	assertOnlyDeprecatedCommandWarnings(t, stderr)
	if !strings.Contains(stdout, `"id":"tester-usa"`) {
		t.Fatalf("expected USA tester in output, got %q", stdout)
	}
	if strings.Contains(stdout, `"id":"tester-fra"`) {
		t.Fatalf("did not expect FRA tester in output, got %q", stdout)
	}
}

func TestSandboxListAllowsEmptyTerritoryFilter(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", req.Method)
		}
		if req.URL.Path != "/v2/sandboxTesters" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{
			"data":[
				{"type":"sandboxTesters","id":"tester-usa","attributes":{"territory":"USA"}},
				{"type":"sandboxTesters","id":"tester-fra","attributes":{"territory":"FRA"}}
			]
		}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"sandbox", "list", "--territory="}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	assertOnlyDeprecatedCommandWarnings(t, stderr)
	if !strings.Contains(stdout, `"id":"tester-usa"`) || !strings.Contains(stdout, `"id":"tester-fra"`) {
		t.Fatalf("expected unfiltered tester output, got %q", stdout)
	}
}
