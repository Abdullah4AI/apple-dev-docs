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

func TestGetCIVersionAliases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/teams/team-uuid/products/product-uuid/configuration-options/version-aliases-v3" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "100" {
			t.Fatalf("limit = %q, want 100", got)
		}
		_, _ = io.WriteString(w, `{"items":[{"id":"alias-1","name":"Release","type":"CUSTOM","locked":true,"build":{"signed_url":"https://example.invalid/?token=secret"},"build_name":"42","related_workflow_summaries":[{"id":"wf-1","name":"Deploy","disabled":false,"locked":false}],"build_supported":true}]}`)
	}))
	defer server.Close()

	result, err := testWebClient(server).GetCIVersionAliases(context.Background(), "team-uuid", "product-uuid")
	if err != nil {
		t.Fatalf("GetCIVersionAliases() error = %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.ID != "alias-1" || item.Name != "Release" || item.Type != "CUSTOM" || !item.Locked || item.BuildName != "42" || !item.BuildSupported {
		t.Fatalf("unexpected item: %+v", item)
	}
	if len(item.RelatedWorkflowSummaries) != 1 || item.RelatedWorkflowSummaries[0].ID != "wf-1" {
		t.Fatalf("unexpected workflow summaries: %+v", item.RelatedWorkflowSummaries)
	}
}

func TestCIVersionAliasesRejectInvalidInputs(t *testing.T) {
	client := &Client{httpClient: http.DefaultClient, baseURL: "http://localhost"}
	tests := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "list empty team", run: func() error { _, err := client.GetCIVersionAliases(context.Background(), "", "product"); return err }, want: "team id and product id are required"},
		{name: "list empty product", run: func() error { _, err := client.GetCIVersionAliases(context.Background(), "team", " "); return err }, want: "team id and product id are required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestGetCIVersionAliasUsesDetailPathAndDecodesRawResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/teams/team-uuid/products/product-uuid/configuration-options/version-aliases-v3/alias-1" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"id":"alias-1","name":"Release","type":"xcode_version","locked":true,"build":"build-1","build_name":"42","related_workflow_summaries":[],"build_supported":true}`)
	}))
	defer server.Close()

	result, err := testWebClient(server).GetCIVersionAlias(context.Background(), "team-uuid", "product-uuid", "alias-1")
	if err != nil {
		t.Fatalf("GetCIVersionAlias() error = %v", err)
	}
	if result.ID != "alias-1" || result.Name != "Release" || result.Type != "xcode_version" || !result.Locked || result.BuildName != "42" || !result.BuildSupported {
		t.Fatalf("unexpected alias: %+v", result)
	}
	if string(result.Build) != `"build-1"` {
		t.Fatalf("build = %s, want JSON string", result.Build)
	}
}

func TestPutCIVersionAliasSendsCapturedBodyAndIgnoresResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/teams/team-uuid/products/product-uuid/configuration-options/version-aliases-v3/alias-1" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(body, &fields); err != nil {
			t.Fatalf("decode body %q: %v", body, err)
		}
		if len(fields) != 4 {
			t.Fatalf("body fields = %d, want exactly 4: %s", len(fields), body)
		}
		for key, want := range map[string]string{
			"name":   `"Release"`,
			"type":   `"xcode_version"`,
			"build":  `"build-1"`,
			"locked": `false`,
		} {
			if got := string(fields[key]); got != want {
				t.Fatalf("%s = %s, want %s", key, got, want)
			}
		}
		_, _ = io.WriteString(w, `{"id":"alias-1","name":"Release","type":"xcode_version","locked":false,"build":"build-1","build_name":"42","related_workflow_summaries":[],"build_supported":true}`)
	}))
	defer server.Close()

	if err := testWebClient(server).PutCIVersionAlias(context.Background(), "team-uuid", "product-uuid", "alias-1", CIVersionAliasRequest{
		Name:   "Release",
		Type:   "xcode_version",
		Build:  json.RawMessage(`"build-1"`),
		Locked: false,
	}); err != nil {
		t.Fatalf("PutCIVersionAlias() error = %v", err)
	}
}

func TestPutCIVersionAliasAcceptsEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %q, want PUT", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := testWebClient(server).PutCIVersionAlias(context.Background(), "team-uuid", "product-uuid", "alias-1", CIVersionAliasRequest{
		Name:  "Release",
		Type:  "xcode_version",
		Build: json.RawMessage(`"build-1"`),
	}); err != nil {
		t.Fatalf("PutCIVersionAlias() error = %v", err)
	}
}

func TestDeleteCIVersionAliasUsesDetailPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/teams/team-uuid/products/product-uuid/configuration-options/version-aliases-v3/alias-1" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := testWebClient(server).DeleteCIVersionAlias(context.Background(), "team-uuid", "product-uuid", "alias-1"); err != nil {
		t.Fatalf("DeleteCIVersionAlias() error = %v", err)
	}
}

func TestCIVersionAliasMutationsRejectInvalidInputs(t *testing.T) {
	client := &Client{httpClient: http.DefaultClient, baseURL: "http://localhost"}
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "get empty alias", run: func() error {
			_, err := client.GetCIVersionAlias(context.Background(), "team", "product", " ")
			return err
		}},
		{name: "put empty product", run: func() error {
			return client.PutCIVersionAlias(context.Background(), "team", " ", "alias", CIVersionAliasRequest{})
		}},
		{name: "delete empty team", run: func() error { return client.DeleteCIVersionAlias(context.Background(), "", "product", "alias") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil || !strings.Contains(err.Error(), "team id, product id, and version alias id are required") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
