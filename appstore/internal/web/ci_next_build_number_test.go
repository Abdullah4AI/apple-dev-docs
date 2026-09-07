package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetCINextBuildNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/teams/team-uuid/products/product-uuid/next-build-number" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"next_build_number":102,"testflight_url":"https://example.invalid/sensitive?token=secret"}`)
	}))
	defer server.Close()

	result, err := testWebClient(server).GetCINextBuildNumber(context.Background(), "team-uuid", "product-uuid")
	if err != nil {
		t.Fatalf("GetCINextBuildNumber() error = %v", err)
	}
	if result.NextBuildNumber != 102 {
		t.Fatalf("next build number = %d, want 102", result.NextBuildNumber)
	}
	if !strings.Contains(result.TestFlightURL, "token=secret") {
		t.Fatalf("testflight URL was not decoded: %q", result.TestFlightURL)
	}
}

func TestSetCINextBuildNumber(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/teams/team-uuid/products/product-uuid/next-build-number" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("next_build_number"); got != "102" {
			t.Fatalf("next_build_number = %q, want 102", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if len(body) != 0 {
			t.Fatalf("request body = %q, want empty", body)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := testWebClient(server).SetCINextBuildNumber(context.Background(), "team-uuid", "product-uuid", 102); err != nil {
		t.Fatalf("SetCINextBuildNumber() error = %v", err)
	}
}

func TestCINextBuildNumberRejectsInvalidInputs(t *testing.T) {
	client := &Client{httpClient: http.DefaultClient, baseURL: "http://localhost"}
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "get empty team", run: func() error { _, err := client.GetCINextBuildNumber(context.Background(), "", "product"); return err }, want: "team id and product id are required"},
		{name: "get empty product", run: func() error { _, err := client.GetCINextBuildNumber(context.Background(), "team", " "); return err }, want: "team id and product id are required"},
		{name: "set empty team", run: func() error { return client.SetCINextBuildNumber(context.Background(), "", "product", 1) }, want: "team id and product id are required"},
		{name: "set empty product", run: func() error { return client.SetCINextBuildNumber(context.Background(), "team", "", 1) }, want: "team id and product id are required"},
		{name: "set nonpositive", run: func() error { return client.SetCINextBuildNumber(context.Background(), "team", "product", 0) }, want: "greater than 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestCINextBuildNumberEscapesPathSegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.EscapedPath(); got != "/teams/team%2Fone/products/product%2Fone/next-build-number" {
			t.Fatalf("escaped path = %q", got)
		}
		_, _ = io.WriteString(w, `{"next_build_number":1}`)
	}))
	defer server.Close()

	client := testWebClient(server)
	if _, err := client.GetCINextBuildNumber(context.Background(), "team/one", "product/one"); err != nil {
		t.Fatalf("GetCINextBuildNumber() error = %v", err)
	}
}
