package reviews

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/itunes"
)

func TestNormalizeRatingsOutput(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		pretty     bool
		wantFormat string
		wantErr    string
	}{
		{name: "default json", input: "", pretty: false, wantFormat: "json"},
		{name: "markdown alias md", input: "md", pretty: false, wantFormat: "markdown"},
		{name: "trim and lowercase", input: "  TABLE  ", pretty: false, wantFormat: "table"},
		{name: "pretty table rejected", input: "table", pretty: true, wantErr: "--pretty is only valid with JSON output"},
		{name: "unsupported format rejected", input: "yaml", pretty: false, wantErr: "unsupported format: yaml"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeRatingsOutput(tc.input, tc.pretty)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantFormat {
				t.Fatalf("expected format %q, got %q", tc.wantFormat, got)
			}
		})
	}
}

func TestUniqueExactRatingsAppMatch(t *testing.T) {
	results := []itunes.SearchResult{
		{AppID: 123, Name: "Alpha", BundleID: "com.example.alpha"},
		{AppID: 456, Name: "Alpha", BundleID: "com.example.alpha.pro"},
	}

	if got, ok := uniqueExactRatingsAppMatch(results, "com.example.alpha", func(result itunes.SearchResult) string {
		return result.BundleID
	}); !ok || got != "123" {
		t.Fatalf("bundle match = %q, %v; want 123, true", got, ok)
	}

	if got, ok := uniqueExactRatingsAppMatch(results, "Alpha", func(result itunes.SearchResult) string {
		return result.Name
	}); ok || got != "" {
		t.Fatalf("ambiguous name match = %q, %v; want empty, false", got, ok)
	}
}

func TestRatingsAppLookupCountryUsesDefaultForAllStorefronts(t *testing.T) {
	if got := ratingsAppLookupCountry("kz", true); got != "us" {
		t.Fatalf("ratingsAppLookupCountry(all) = %q, want us", got)
	}
	if got := ratingsAppLookupCountry("kz", false); got != "kz" {
		t.Fatalf("ratingsAppLookupCountry(single) = %q, want kz", got)
	}
}

func TestResolveRatingsAppIDLooksUpTrimmedBundleWithHttptest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lookup" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("bundleId"); got != "com.example.alpha" {
			t.Fatalf("expected trimmed lookup bundleId com.example.alpha, got %q", got)
		}
		if got := r.URL.Query().Get("country"); got != "us" {
			t.Fatalf("expected lookup country=us, got %q", got)
		}
		if got := r.URL.Query().Get("entity"); got != "software" {
			t.Fatalf("expected entity=software, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"resultCount":1,"results":[{"trackId":123,"trackName":"Alpha","bundleId":"com.example.alpha"}]}`)
	}))
	defer server.Close()

	client := &itunes.Client{BaseURL: server.URL, HTTPClient: server.Client()}
	got, err := resolveRatingsAppID(context.Background(), client, "  com.example.alpha  ", "us")
	if err != nil {
		t.Fatalf("resolveRatingsAppID() error: %v", err)
	}
	if got != "123" {
		t.Fatalf("resolveRatingsAppID() = %q, want 123", got)
	}
}

func TestResolveRatingsAppIDHidesBundleLookupFailureDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lookup" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		http.Error(w, "backend leaked com.example.secret", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &itunes.Client{BaseURL: server.URL, HTTPClient: server.Client()}
	_, err := resolveRatingsAppID(context.Background(), client, "com.example.secret", "us")
	if err == nil {
		t.Fatal("expected lookup failure")
	}
	if strings.Contains(err.Error(), "com.example.secret") || strings.Contains(err.Error(), "backend leaked") {
		t.Fatalf("expected generic lookup error, got %q", err.Error())
	}
}

func TestResolveRatingsAppIDSearchesBeyondFirstTenForExactName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("term"); got != "Alpha" {
			t.Fatalf("expected search term Alpha, got %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != strconv.Itoa(ratingsAppSearchLimit) {
			t.Fatalf("expected search limit=%d, got %q", ratingsAppSearchLimit, got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"resultCount":11,"results":[
			{"trackId":1,"trackName":"Alpha Notes 1","bundleId":"com.example.alpha1"},
			{"trackId":2,"trackName":"Alpha Notes 2","bundleId":"com.example.alpha2"},
			{"trackId":3,"trackName":"Alpha Notes 3","bundleId":"com.example.alpha3"},
			{"trackId":4,"trackName":"Alpha Notes 4","bundleId":"com.example.alpha4"},
			{"trackId":5,"trackName":"Alpha Notes 5","bundleId":"com.example.alpha5"},
			{"trackId":6,"trackName":"Alpha Notes 6","bundleId":"com.example.alpha6"},
			{"trackId":7,"trackName":"Alpha Notes 7","bundleId":"com.example.alpha7"},
			{"trackId":8,"trackName":"Alpha Notes 8","bundleId":"com.example.alpha8"},
			{"trackId":9,"trackName":"Alpha Notes 9","bundleId":"com.example.alpha9"},
			{"trackId":10,"trackName":"Alpha Notes 10","bundleId":"com.example.alpha10"},
			{"trackId":123,"trackName":"Alpha","bundleId":"com.example.alpha"}
		]}`)
	}))
	defer server.Close()

	client := &itunes.Client{BaseURL: server.URL, HTTPClient: server.Client()}
	got, err := resolveRatingsAppID(context.Background(), client, "Alpha", "us")
	if err != nil {
		t.Fatalf("resolveRatingsAppID() error: %v", err)
	}
	if got != "123" {
		t.Fatalf("resolveRatingsAppID() = %q, want 123", got)
	}
}
