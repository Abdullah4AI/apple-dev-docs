package cmdtest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestAppsListSortsBySKU covers the sku and -sku members of the documented
// GET /v1/apps sort enum reaching the API unchanged.
func TestAppsListSortsBySKU(t *testing.T) {
	setupAuth(t)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "list ascending", args: []string{"apps", "list", "--sort", "sku", "--output", "json"}, want: "sku"},
		{name: "list descending", args: []string{"apps", "list", "--sort", "-sku", "--output", "json"}, want: "-sku"},
		{name: "group ascending", args: []string{"apps", "--sort", "sku", "--output", "json"}, want: "sku"},
	}

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.Path != "/v1/apps" {
					t.Fatalf("request = %s %s, want GET /v1/apps", req.Method, req.URL.Path)
				}
				if got := req.URL.Query().Get("sort"); got != test.want {
					t.Errorf("sort = %q, want %q", got, test.want)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
					Header:     http.Header{"Content-Type": []string{"application/json"}},
				}, nil
			})

			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)
			_, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				if err := root.Run(context.Background()); err != nil {
					t.Fatalf("run error: %v", err)
				}
			})
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			if calls != 1 {
				t.Fatalf("calls = %d, want 1", calls)
			}
		})
	}
}

// TestAppsListRejectsUnsupportedSort keeps the allow-list aligned with the
// documented sort enum for GET /v1/apps.
func TestAppsListRejectsUnsupportedSort(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{"apps", "list", "--sort", "createdDate"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	err := root.Run(context.Background())
	if err == nil {
		t.Fatal("expected an error for an unsupported --sort value")
	}
	want := "--sort must be one of: name, -name, bundleId, -bundleId, sku, -sku"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want it to contain %q", err, want)
	}
}
