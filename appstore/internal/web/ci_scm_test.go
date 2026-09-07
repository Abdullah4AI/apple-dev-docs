package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetCIScmProvidersUsesExactGETPathAndPreservesUnknownFields(t *testing.T) {
	const response = `[
		{
			"id":"provider-1",
			"provider":"github",
			"provider_display_name":"GitHub",
			"is_on_premise":false,
			"is_registered":true,
			"is_user_connected":false,
			"supports_registration_flow":true,
			"register_type":"oauth",
			"install_type":"app",
			"connect_type":"oauth",
			"host":"github.com",
			"username":"octocat",
			"oauth_callback_base_uri":"https://example.test/callback?opaque=1",
			"webhook_uri":"https://example.test/webhook",
			"future_field":{"keep":true}
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/teams/team-uuid/scm-providers-v2" {
			t.Fatalf("path = %q, want /teams/team-uuid/scm-providers-v2", r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Fatalf("query = %q, want empty", r.URL.RawQuery)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if len(body) != 0 {
			t.Fatalf("request body = %q, want empty", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, response)
	}))
	defer server.Close()

	result, err := testWebClient(server).GetCIScmProviders(context.Background(), " team-uuid ")
	if err != nil {
		t.Fatalf("GetCIScmProviders() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("provider count = %d, want 1", len(result))
	}
	provider := result[0]
	if provider.ID != "provider-1" || provider.Provider != "github" || provider.ProviderDisplayName != "GitHub" {
		t.Fatalf("provider identity = %+v", provider)
	}
	if provider.IsRegistered == nil || !*provider.IsRegistered {
		t.Fatalf("is_registered = %v, want pointer true", provider.IsRegistered)
	}
	if provider.IsUserConnected == nil || *provider.IsUserConnected {
		t.Fatalf("is_user_connected = %v, want pointer false", provider.IsUserConnected)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal(provider list) error = %v", err)
	}
	for _, want := range []string{
		`"provider_display_name":"GitHub"`,
		`"is_on_premise":false`,
		`"supports_registration_flow":true`,
		`"register_type":"oauth"`,
		`"install_type":"app"`,
		`"connect_type":"oauth"`,
		`"host":"github.com"`,
		`"username":"octocat"`,
		`"oauth_callback_base_uri"`,
		`"webhook_uri"`,
		`"future_field":{"keep":true}`,
	} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("marshaled provider list missing %s: %s", want, encoded)
		}
	}
}

func TestGetCIScmProvidersDistinguishesEmptyArrayFromMalformedResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantErr  bool
	}{
		{name: "empty array", response: `[]`},
		{name: "null", response: `null`, wantErr: true},
		{name: "object", response: `{}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, tt.response)
			}))
			defer server.Close()

			result, err := testWebClient(server).GetCIScmProviders(context.Background(), "team-uuid")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected malformed response error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetCIScmProviders() error = %v", err)
			}
			if result == nil {
				t.Fatal("empty JSON array decoded to nil result")
			}
			if len(result) != 0 {
				t.Fatalf("provider count = %d, want 0", len(result))
			}
		})
	}
}

func TestGetCIScmConnectionStatusUsesExactGETPathAndPreservesOpaqueError(t *testing.T) {
	const response = `{"status":"connection_issue","error":{"code":"provider_error","detail":"keep"},"future_status_field":"new"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/teams/team-uuid/scm-providers/provider-1/connection-v2" {
			t.Fatalf("path = %q, want /teams/team-uuid/scm-providers/provider-1/connection-v2", r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Fatalf("query = %q, want empty", r.URL.RawQuery)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if len(body) != 0 {
			t.Fatalf("request body = %q, want empty", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, response)
	}))
	defer server.Close()

	result, err := testWebClient(server).GetCIScmConnectionStatus(context.Background(), " team-uuid ", " provider-1 ")
	if err != nil {
		t.Fatalf("GetCIScmConnectionStatus() error = %v", err)
	}
	if result.Status != "connection_issue" {
		t.Fatalf("status = %q, want connection_issue", result.Status)
	}
	if got := string(result.Error); got != `{"code":"provider_error","detail":"keep"}` {
		t.Fatalf("error = %s, want opaque object", got)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal(connection status) error = %v", err)
	}
	for _, want := range []string{`"error":{"code":"provider_error","detail":"keep"}`, `"future_status_field":"new"`} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("marshaled status missing %s: %s", want, encoded)
		}
	}
}

func TestCIScmReadsRejectEmptyIDsBeforeHTTP(t *testing.T) {
	client := &Client{httpClient: http.DefaultClient, baseURL: "http://127.0.0.1:1"}

	if _, err := client.GetCIScmProviders(context.Background(), " "); err == nil || !strings.Contains(err.Error(), "team id is required") {
		t.Fatalf("GetCIScmProviders(empty team) error = %v", err)
	}
	if _, err := client.GetCIScmConnectionStatus(context.Background(), " ", "provider-1"); err == nil || !strings.Contains(err.Error(), "team id is required") {
		t.Fatalf("GetCIScmConnectionStatus(empty team) error = %v", err)
	}
	if _, err := client.GetCIScmConnectionStatus(context.Background(), "team-1", " "); err == nil || !strings.Contains(err.Error(), "scm provider id is required") {
		t.Fatalf("GetCIScmConnectionStatus(empty provider) error = %v", err)
	}
}

func TestCIScmReadsEscapePathSegments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.EscapedPath(), "/teams/team%2Fuuid/scm-providers/provider%2Fone/connection-v2"; got != want {
			t.Fatalf("escaped path = %q, want %q (decoded path %q)", got, want, r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"status":"success"}`)
	}))
	defer server.Close()

	if _, err := testWebClient(server).GetCIScmConnectionStatus(context.Background(), "team/uuid", "provider/one"); err != nil {
		t.Fatalf("GetCIScmConnectionStatus() error = %v", err)
	}
}

func TestGetCIScmConnectionStatusRequiresNonemptyStatus(t *testing.T) {
	for _, response := range []string{`null`, `{}`, `{"status":""}`, `{"status":null}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, response)
		}))
		result, err := testWebClient(server).GetCIScmConnectionStatus(context.Background(), "team-uuid", "provider-1")
		server.Close()
		if err == nil {
			t.Fatalf("response %s unexpectedly succeeded with %#v", response, result)
		}
		if !strings.Contains(err.Error(), "connection status") {
			t.Fatalf("response %s error = %v, want connection status context", response, err)
		}
	}
}

func TestGetCIScmConnectionStatusAcceptsUnknownNonemptyStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":"provider_paused"}`)
	}))
	defer server.Close()

	result, err := testWebClient(server).GetCIScmConnectionStatus(context.Background(), "team-uuid", "provider-1")
	if err != nil {
		t.Fatalf("GetCIScmConnectionStatus() error = %v", err)
	}
	if result.Status != "provider_paused" {
		t.Fatalf("status = %q, want provider_paused", result.Status)
	}
}
