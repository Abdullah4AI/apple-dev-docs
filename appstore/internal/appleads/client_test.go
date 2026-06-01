package appleads

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func testPrivateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}
	return key
}

func marshalECPrivateKeyPEM(t *testing.T, key *ecdsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey() error: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}))
}

func TestGenerateClientSecretClaims(t *testing.T) {
	now := time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	tokenString, err := GenerateClientSecret("KEY123", "TEAM123", "CLIENT123", testPrivateKey(t), now, 10*time.Minute)
	if err != nil {
		t.Fatalf("GenerateClientSecret() error: %v", err)
	}

	claims := jwt.MapClaims{}
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, claims)
	if err != nil {
		t.Fatalf("ParseUnverified() error: %v", err)
	}
	if got := token.Header["kid"]; got != "KEY123" {
		t.Fatalf("kid = %v, want KEY123", got)
	}
	if got := token.Method.Alg(); got != "ES256" {
		t.Fatalf("alg = %q, want ES256", got)
	}
	assertClaim(t, claims, "iss", "TEAM123")
	assertClaim(t, claims, "aud", "https://appleid.apple.com")
	assertClaim(t, claims, "sub", "CLIENT123")
	if got, want := int64(claims["exp"].(float64)-claims["iat"].(float64)), int64((10 * time.Minute).Seconds()); got != want {
		t.Fatalf("exp-iat = %d, want %d", got, want)
	}
}

func TestAccessTokenUsesAppleOAuthClientCredentialsRequest(t *testing.T) {
	requests := 0
	client, err := NewClient(Credentials{
		ClientID:      "CLIENT123",
		TeamID:        "TEAM123",
		KeyID:         "KEY123",
		PrivateKeyPEM: marshalECPrivateKeyPEM(t, testPrivateKey(t)),
	}, WithTokenURL("https://appleid.test/auth/oauth2/token"), WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			if req.Method != http.MethodPost {
				t.Fatalf("token method = %s, want POST", req.Method)
			}
			if got := req.URL.String(); got != "https://appleid.test/auth/oauth2/token" {
				t.Fatalf("token URL = %s", got)
			}
			if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
				t.Fatalf("Content-Type = %q, want form encoded", got)
			}
			data, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("ReadAll(body) error: %v", err)
			}
			form, err := url.ParseQuery(string(data))
			if err != nil {
				t.Fatalf("ParseQuery() error: %v", err)
			}
			if form.Get("grant_type") != "client_credentials" {
				t.Fatalf("grant_type = %q", form.Get("grant_type"))
			}
			if form.Get("client_id") != "CLIENT123" {
				t.Fatalf("client_id = %q", form.Get("client_id"))
			}
			if form.Get("scope") != "searchadsorg" {
				t.Fatalf("scope = %q", form.Get("scope"))
			}
			if form.Get("client_secret") == "" {
				t.Fatal("client_secret is empty")
			}
			return jsonResponse(200, `{"access_token":"ACCESS","token_type":"Bearer","expires_in":3600}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	token, err := client.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("AccessToken() error: %v", err)
	}
	if token != "ACCESS" {
		t.Fatalf("token = %q, want ACCESS", token)
	}
	token, err = client.AccessToken(context.Background())
	if err != nil {
		t.Fatalf("second AccessToken() error: %v", err)
	}
	if token != "ACCESS" || requests != 1 {
		t.Fatalf("token cache got token %q requests %d, want ACCESS requests 1", token, requests)
	}
}

func TestRequestSetsBearerAndOrganizationContext(t *testing.T) {
	seen := []string{}
	client, err := NewClient(Credentials{AccessToken: "ACCESS", OrgID: "123456"}, WithBaseURL("https://api.searchads.apple.com/api/"), WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.Method+" "+req.URL.Path+" "+req.Header.Get("X-AP-Context"))
			if got := req.Header.Get("Authorization"); got != "Bearer ACCESS" {
				t.Fatalf("Authorization = %q, want bearer token", got)
			}
			return jsonResponse(200, `{"data":[]}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	me, ok := EndpointByCommandPath("me", "view")
	if !ok {
		t.Fatal("missing me view endpoint")
	}
	if _, err := client.Do(context.Background(), me, nil, nil, nil); err != nil {
		t.Fatalf("me Do() error: %v", err)
	}
	campaigns, ok := EndpointByCommandPath("campaigns", "list")
	if !ok {
		t.Fatal("missing campaigns list endpoint")
	}
	if _, err := client.Do(context.Background(), campaigns, nil, url.Values{"limit": {"1"}}, nil); err != nil {
		t.Fatalf("campaigns Do() error: %v", err)
	}
	want := []string{
		"GET /api/v5/me ",
		"GET /api/v5/campaigns orgId=123456",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s\nwant:\n%s", strings.Join(seen, "\n"), strings.Join(want, "\n"))
	}
}

func TestRequestParsesAppleAdsErrorEnvelope(t *testing.T) {
	client, err := NewClient(Credentials{AccessToken: "ACCESS", OrgID: "123456"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(400, `{"error":{"errors":[{"field":"campaignId","message":"Invalid campaign","messageCode":"INVALID_INPUT"}]}}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	campaigns, ok := EndpointByCommandPath("campaigns", "view")
	if !ok {
		t.Fatal("missing campaigns view endpoint")
	}
	_, err = client.Do(context.Background(), campaigns, map[string]string{"campaignId": "1"}, nil, nil)
	if err == nil {
		t.Fatal("expected API error")
	}
	if got := err.Error(); !strings.Contains(got, "HTTP 400: INVALID_INPUT: campaignId: Invalid campaign") {
		t.Fatalf("error = %q", got)
	}
}

func TestPaginateAllAggregatesOffsetPages(t *testing.T) {
	requestOffsets := []string{}
	client, err := NewClient(Credentials{AccessToken: "ACCESS", OrgID: "123456"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			offset := req.URL.Query().Get("offset")
			requestOffsets = append(requestOffsets, offset)
			switch offset {
			case "0":
				return jsonResponse(200, `{"data":[{"id":1},{"id":2}],"pagination":{"itemsPerPage":2,"startIndex":0,"totalResults":3}}`), nil
			case "2":
				return jsonResponse(200, `{"data":[{"id":3}],"pagination":{"itemsPerPage":2,"startIndex":2,"totalResults":3}}`), nil
			default:
				t.Fatalf("unexpected offset %q", offset)
				return nil, nil
			}
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	campaigns, ok := EndpointByCommandPath("campaigns", "list")
	if !ok {
		t.Fatal("missing campaigns list endpoint")
	}
	raw, err := client.PaginateAll(context.Background(), campaigns, nil, nil, 0, 2, nil)
	if err != nil {
		t.Fatalf("PaginateAll() error: %v", err)
	}
	var parsed struct {
		Data       []map[string]int `json:"data"`
		Pagination PageDetail       `json:"pagination"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	if len(parsed.Data) != 3 || parsed.Data[2]["id"] != 3 {
		t.Fatalf("aggregated data = %+v, want 3 rows ending with id=3", parsed.Data)
	}
	if got := strings.Join(requestOffsets, ","); got != "0,2" {
		t.Fatalf("offsets = %q, want 0,2", got)
	}
}

func TestPaginateAllReportsClampedStartOffset(t *testing.T) {
	client, err := NewClient(Credentials{AccessToken: "ACCESS", OrgID: "123456"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.URL.Query().Get("offset"); got != "0" {
				t.Fatalf("offset = %q, want 0", got)
			}
			return jsonResponse(200, `{"data":[{"id":1}],"pagination":{"itemsPerPage":1,"startIndex":0,"totalResults":1}}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	campaigns, ok := EndpointByCommandPath("campaigns", "list")
	if !ok {
		t.Fatal("missing campaigns list endpoint")
	}
	raw, err := client.PaginateAll(context.Background(), campaigns, nil, nil, -10, 1, nil)
	if err != nil {
		t.Fatalf("PaginateAll() error: %v", err)
	}
	var parsed struct {
		Pagination PageDetail `json:"pagination"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	if parsed.Pagination.StartIndex != 0 {
		t.Fatalf("StartIndex = %d, want 0", parsed.Pagination.StartIndex)
	}
}

func assertClaim(t *testing.T, claims jwt.MapClaims, name, want string) {
	t.Helper()
	if got := claims[name]; got != want {
		t.Fatalf("%s = %v, want %s", name, got, want)
	}
}
