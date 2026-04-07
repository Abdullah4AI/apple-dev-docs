package cmdtest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestReviewsListKeepsAlpha2Territory(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", req.Method)
		}
		if req.URL.Path != "/v1/apps/app-1/customerReviews" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		if got := req.URL.Query().Get("filter[territory]"); got != "US" {
			t.Fatalf("expected alpha-2 review territory filter US, got %q", got)
		}
		return jsonResponse(http.StatusOK, `{
			"data":[
				{"type":"customerReviews","id":"review-1","attributes":{"rating":5,"title":"Great","body":"Nice","reviewerNickname":"Tester","createdDate":"2026-01-20T00:00:00Z","territory":"US"}}
			]
		}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"reviews", "list", "--app", "app-1", "--territory", "US"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"review-1"`) {
		t.Fatalf("expected review output, got %q", stdout)
	}
}
