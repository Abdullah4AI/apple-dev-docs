package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestGetAppTransferStatus(t *testing.T) {
	for _, tc := range []struct{ name, relationship, included, presence, id, state string }{
		{"no request", `"appTransferRequest":{"data":null}`, `[]`, "none", "", ""},
		{"omitted", ``, `[]`, "unknown", "", ""},
		{"link only", `"appTransferRequest":{"links":{"related":"/example"}}`, `[]`, "unknown", "", ""},
		{"future state", `"appTransferRequest":{"data":{"type":"appTransferRequests","id":"transfer-1"}}`, `[{"type":"unrelated","id":"transfer-1","attributes":{"state":"WRONG"}},{"type":"appTransferRequests","id":"transfer-1","attributes":{"state":"FUTURE_APPLE_STATE","newField":123}}]`, "present", "transfer-1", "FUTURE_APPLE_STATE"},
		{"missing included", `"appTransferRequest":{"data":{"type":"appTransferRequests","id":"transfer-1"}}`, `[]`, "present", "transfer-1", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"data":{"type":"apps","id":"app-1","relationships":{` + tc.relationship + `}},"included":` + tc.included + `,"meta":{"future":"preserved"}}`
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/apps/app-1" || r.URL.Query().Get("include") != "appTransferRequest" || len(r.URL.Query()) != 1 || r.ContentLength > 0 {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			got, err := testWebClient(server).GetAppTransferStatus(context.Background(), "app-1")
			if err != nil {
				t.Fatal(err)
			}
			if got.AppID != "app-1" || got.Presence != tc.presence || got.RequestID != tc.id || got.State != tc.state {
				t.Fatalf("unexpected summary %+v", got)
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var wantJSON, gotJSON any
			_ = json.Unmarshal([]byte(body), &wantJSON)
			_ = json.Unmarshal(encoded, &gotJSON)
			if !reflect.DeepEqual(wantJSON, gotJSON) {
				t.Fatalf("envelope changed: %s", encoded)
			}
		})
	}
}

func TestGetAppTransferStatusRejectsInvalidResponse(t *testing.T) {
	for _, body := range []string{
		`{"data":{"type":"apps","id":"other"}}`,
		`{"data":{"type":"other","id":"app-1"}}`,
		`{"data":null}`,
		`{"data":{"type":"apps","id":"app-1","relationships":{"appTransferRequest":{"data":[]}}}}`,
		`{"data":{"type":"apps","id":"app-1","relationships":{"appTransferRequest":{"data":{}}}}}`,
		`not-json`,
	} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			if _, err := testWebClient(server).GetAppTransferStatus(context.Background(), "app-1"); err == nil {
				t.Fatal("expected invalid response error")
			}
		})
	}
}

func TestGetAppTransferStatusRequiresAppAndPropagatesError(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":[{"status":"403","code":"FORBIDDEN","title":"Forbidden"}]}`))
	}))
	defer server.Close()
	client := testWebClient(server)
	if _, err := client.GetAppTransferStatus(context.Background(), " "); err == nil || !strings.Contains(err.Error(), "app id is required") {
		t.Fatalf("err=%v", err)
	}
	if requests != 0 {
		t.Fatal("empty app made a request")
	}
	if _, err := client.GetAppTransferStatus(context.Background(), "app-1"); err == nil {
		t.Fatal("expected HTTP error")
	}
}
