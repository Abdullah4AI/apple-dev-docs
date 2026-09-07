package web

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestClientGetWebUserSendsExactIdentityPath(t *testing.T) {
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/iris/v1/users/USER-UUID" {
			t.Fatalf("path = %s, want /iris/v1/users/USER-UUID", r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Fatalf("query = %q, want empty", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":{"type":"users","id":"USER-UUID","attributes":{"username":"owner@example.com"}}}`)
	}))

	user, err := client.GetWebUser(context.Background(), "USER-UUID")
	if err != nil {
		t.Fatalf("GetWebUser() error: %v", err)
	}
	if user == nil || user.ID != "USER-UUID" || user.Username != "owner@example.com" {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func TestClientListIndividualAPIKeysForUserSendsActorFilter(t *testing.T) {
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/iris/v2/apiKeys" {
			t.Fatalf("request = %s %s, want GET /iris/v2/apiKeys", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("include") != "createdByActor,revokedByActor" {
			t.Fatalf("include = %q", query.Get("include"))
		}
		if query.Get("filter[createdByActor]") != "USER:USER-UUID" {
			t.Fatalf("createdByActor filter = %q", query.Get("filter[createdByActor]"))
		}
		if len(query) != 2 {
			t.Fatalf("unexpected query parameters: %#v", query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[{"type":"apiKeys","id":"IND-1","attributes":{"isActive":true,"publicKey":null},"relationships":{"createdByActor":{"data":{"type":"actors","id":"USER-UUID"}}}}]}`)
	}))

	keys, err := client.ListIndividualAPIKeysForUser(context.Background(), "USER-UUID")
	if err != nil {
		t.Fatalf("ListIndividualAPIKeysForUser() error: %v", err)
	}
	if len(keys) != 1 || keys[0].KeyID != "IND-1" || !keys[0].Active || keys[0].PublicKeyPresent {
		t.Fatalf("unexpected keys: %#v", keys)
	}
}

func TestClientListIndividualAPIKeysForUserFollowsNextPage(t *testing.T) {
	page := 0
	publicPEM := testIndividualPublicKeyPEM(t)
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/iris/v2/apiKeys" {
			t.Fatalf("request = %s %s, want GET /iris/v2/apiKeys", r.Method, r.URL.Path)
		}
		page++
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case 1:
			if r.URL.Query().Get("filter[createdByActor]") != "USER:USER-UUID" {
				t.Fatalf("first page filter = %q", r.URL.Query().Get("filter[createdByActor]"))
			}
			_, _ = io.WriteString(w, `{"data":[{"type":"apiKeys","id":"IND-INACTIVE","attributes":{"isActive":false,"publicKey":null}}],"links":{"next":"/iris/v2/apiKeys?page=2"}}`)
		case 2:
			query := r.URL.Query()
			if query.Get("page") != "2" {
				t.Fatalf("second page query = %s, want page=2", r.URL.RawQuery)
			}
			response := map[string]any{"data": []any{map[string]any{
				"type": "apiKeys", "id": "IND-ACTIVE",
				"attributes": map[string]any{"isActive": true, "publicKey": publicPEM},
			}}}
			_ = json.NewEncoder(w).Encode(response)
		default:
			t.Fatalf("unexpected page %d, query %s", page, r.URL.RawQuery)
		}
	}))

	keys, err := client.ListIndividualAPIKeysForUser(context.Background(), "USER-UUID")
	if err != nil {
		t.Fatalf("ListIndividualAPIKeysForUser() error: %v", err)
	}
	if page != 2 || len(keys) != 2 || !keys[1].Active || !keys[1].PublicKeyPresent {
		t.Fatalf("unexpected paginated keys: pages=%d keys=%#v", page, keys)
	}
}

func TestClientListIndividualAPIKeysForUserRejectsCrossHostNextPage(t *testing.T) {
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[],"links":{"next":"https://attacker.invalid/iris/v2/apiKeys?page=2"}}`)
	}))

	_, err := client.ListIndividualAPIKeysForUser(context.Background(), "USER-UUID")
	if err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("expected cross-host pagination error, got %v", err)
	}
}

func TestClientCreateIndividualAPIKeySendsEmptyPayload(t *testing.T) {
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/iris/v2/apiKeys" {
			t.Fatalf("request = %s %s, want POST /iris/v2/apiKeys", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		want := map[string]any{"data": map[string]any{"type": "apiKeys"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("request body = %#v, want %#v", got, want)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	err := client.CreateIndividualAPIKey(context.Background())
	if err != nil {
		t.Fatalf("CreateIndividualAPIKey() error: %v", err)
	}
}

func TestClientRegisterIndividualAPIKeySendsPublicKeyPatch(t *testing.T) {
	publicPEM := "-----BEGIN PUBLIC KEY-----\npublic\n-----END PUBLIC KEY-----\n"
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/iris/v2/apiKeys/IND-1" {
			t.Fatalf("request = %s %s, want PATCH /iris/v2/apiKeys/IND-1", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		want := map[string]any{
			"data": map[string]any{
				"type": "apiKeys",
				"id":   "IND-1",
				"attributes": map[string]any{
					"publicKey": publicPEM,
				},
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("request body = %#v, want %#v", got, want)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	if err := client.RegisterIndividualAPIKey(context.Background(), "IND-1", publicPEM); err != nil {
		t.Fatalf("RegisterIndividualAPIKey() error: %v", err)
	}
}

func TestClientRegisterIndividualAPIKeyRejectsBlankInputs(t *testing.T) {
	client := newAPIKeyHTTPTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("did not expect a request for invalid input")
	}))
	for _, test := range []struct {
		name, keyID, publicPEM string
	}{
		{name: "missing key id", publicPEM: "PUBLIC"},
		{name: "missing public key", keyID: "IND-1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := client.RegisterIndividualAPIKey(context.Background(), test.keyID, test.publicPEM)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if bytes.Contains([]byte(err.Error()), []byte("PUBLIC")) {
				t.Fatal("error leaked public key material")
			}
		})
	}
}

func testIndividualPublicKeyPEM(t *testing.T) string {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal test public key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}
