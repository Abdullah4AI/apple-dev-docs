package asc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"strings"
	"testing"
)

// newRejectingClient fails the test if any HTTP request escapes the client.
func newRejectingClient(t *testing.T) *Client {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected request escaped validation: %s %s", req.Method, req.URL.String())
		return nil, nil
	})

	return &Client{
		httpClient: &http.Client{Transport: transport},
		keyID:      "KEY123",
		issuerID:   "ISS456",
		privateKey: key,
	}
}

func TestResourcePathRejectsIdentifiersThatEscapeTheirSegment(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{name: "query marker", id: "act-1?"},
		{name: "query marker with parameters", id: "act-1?fields=x"},
		{name: "fragment marker", id: "act-1#frag"},
		{name: "path separator", id: "act-1/relationships"},
		{name: "percent sequence", id: "act-1%2F"},
		{name: "backslash", id: `act-1\x`},
		{name: "traversal", id: ".."},
		{name: "control character", id: "act-1\nx"},
		{name: "empty", id: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := resourcePath("/v1/gameCenterActivities/%s/relationships/achievements", tt.id)
			if err == nil {
				t.Fatalf("resourcePath(%q) = %q, want error", tt.id, path)
			}
		})
	}
}

func TestResourcePathBuildsExactSegmentForNormalIdentifiers(t *testing.T) {
	path, err := resourcePath("/v1/gameCenterActivities/%s/relationships/achievements", " act-1 ")
	if err != nil {
		t.Fatalf("resourcePath() error: %v", err)
	}
	if path != "/v1/gameCenterActivities/act-1/relationships/achievements" {
		t.Fatalf("resourcePath() = %q, want /v1/gameCenterActivities/act-1/relationships/achievements", path)
	}
}

func TestResourcePathRejectsTemplateArityMismatch(t *testing.T) {
	if _, err := resourcePath("/v1/apps/%s/builds/%s", "app-1"); err == nil {
		t.Fatal("resourcePath() = nil error, want arity mismatch error")
	}
}

// TestGameCenterActivityRelationshipMutationsRejectQueryInjection proves that an
// activity ID containing a reserved delimiter cannot turn a relationship
// mutation into a mutation of a different endpoint (for example
// DELETE /v1/gameCenterActivities/{id}).
func TestGameCenterActivityRelationshipMutationsRejectQueryInjection(t *testing.T) {
	const injected = "act-1?"

	tests := []struct {
		name string
		call func(*Client) error
	}{
		{
			name: "RemoveGameCenterActivityAchievements",
			call: func(c *Client) error {
				return c.RemoveGameCenterActivityAchievements(context.Background(), injected, []string{"ach-1"})
			},
		},
		{
			name: "AddGameCenterActivityAchievements",
			call: func(c *Client) error {
				return c.AddGameCenterActivityAchievements(context.Background(), injected, []string{"ach-1"})
			},
		},
		{
			name: "RemoveGameCenterActivityLeaderboards",
			call: func(c *Client) error {
				return c.RemoveGameCenterActivityLeaderboards(context.Background(), injected, []string{"lb-1"})
			},
		},
		{
			name: "AddGameCenterActivityLeaderboards",
			call: func(c *Client) error {
				return c.AddGameCenterActivityLeaderboards(context.Background(), injected, []string{"lb-1"})
			},
		},
		{
			name: "RemoveGameCenterActivityAchievementsV2",
			call: func(c *Client) error {
				return c.RemoveGameCenterActivityAchievementsV2(context.Background(), injected, []string{"ach-1"})
			},
		},
		{
			name: "AddGameCenterActivityAchievementsV2",
			call: func(c *Client) error {
				return c.AddGameCenterActivityAchievementsV2(context.Background(), injected, []string{"ach-1"})
			},
		},
		{
			name: "RemoveGameCenterActivityLeaderboardsV2",
			call: func(c *Client) error {
				return c.RemoveGameCenterActivityLeaderboardsV2(context.Background(), injected, []string{"lb-1"})
			},
		},
		{
			name: "AddGameCenterActivityLeaderboardsV2",
			call: func(c *Client) error {
				return c.AddGameCenterActivityLeaderboardsV2(context.Background(), injected, []string{"lb-1"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newRejectingClient(t)
			err := tt.call(client)
			if err == nil {
				t.Fatal("expected error for activity ID containing a query marker, got nil")
			}
			if !strings.Contains(err.Error(), "single path segment") {
				t.Fatalf("error = %v, want it to explain the path-segment requirement", err)
			}
		})
	}
}

// TestMutatingRequestRejectsQueryString is the defense-in-depth guard: App Store
// Connect defines no query parameters for POST, PATCH, PUT, or DELETE, so a
// query string on a mutating request means an identifier reinterpreted the
// endpoint.
func TestMutatingRequestRejectsQueryString(t *testing.T) {
	client := newRejectingClient(t)

	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			_, err := client.do(context.Background(), method, "/v1/gameCenterActivities/act-1?relationships/achievements", nil)
			if err == nil {
				t.Fatalf("do(%s) = nil error, want rejection of query string on mutating request", method)
			}
			if !strings.Contains(err.Error(), "query string") {
				t.Fatalf("do(%s) error = %v, want query-string rejection", method, err)
			}
		})
	}
}

func TestMutatingRequestAllowsPathsWithoutQueryString(t *testing.T) {
	client := newTestClient(t, func(req *http.Request) {
		if req.URL.RawQuery != "" {
			t.Fatalf("unexpected query: %q", req.URL.RawQuery)
		}
	}, jsonResponse(http.StatusNoContent, ""))

	if _, err := client.do(context.Background(), http.MethodDelete, "/v1/gameCenterActivities/act-1", nil); err != nil {
		t.Fatalf("do() error: %v", err)
	}
}
