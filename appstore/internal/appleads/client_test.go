package appleads

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
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

func TestDoPreservesRequiresOrgForExplicitCampaignVersion(t *testing.T) {
	client, err := NewClient(Credentials{AccessToken: "ACCESS", OrgID: "123456"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("X-AP-Context"); got != "orgId=123456" {
				t.Fatalf("X-AP-Context = %q, want org context", got)
			}
			return jsonResponse(200, `{}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.Do(context.Background(), EndpointSpec{
		Method:      http.MethodGet,
		Path:        "v5/campaigns",
		Version:     APIVersionCampaignV5,
		RequiresOrg: true,
	}, nil, nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
}

func TestRequestForVersionRoutesPlatformAPIAndAdAccountContext(t *testing.T) {
	seen := []string{}
	client, err := NewClient(
		Credentials{AccessToken: "ACCESS", OrgID: "LEGACY_ORG", AdAccountID: "AD_ACCOUNT"},
		WithPlatformBaseURL("https://api.ads.apple.com/v1/"),
		WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seen = append(seen, req.Method+" "+req.URL.String()+" "+req.Header.Get("X-AP-Context"))
			response := jsonResponse(200, `{"result":{}}`)
			response.Header.Set("RateLimit-Limit", "100")
			response.Header.Set("RateLimit-Remaining", "99")
			return response, nil
		})}),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	if _, err := client.RequestForVersion(context.Background(), APIVersionPlatformV1, http.MethodGet, "v1/me", nil, nil, ContextNone); err != nil {
		t.Fatalf("platform me request error: %v", err)
	}
	if _, err := client.Do(context.Background(), EndpointSpec{
		Method:  http.MethodGet,
		Path:    "v1/campaigns/{campaignId}",
		Version: APIVersionPlatformV1,
		Context: ContextAdAccount,
	}, map[string]string{"campaignId": "123"}, url.Values{"include": {"budget"}}, nil); err != nil {
		t.Fatalf("platform campaign request error: %v", err)
	}

	want := []string{
		"GET https://api.ads.apple.com/v1/me ",
		"GET https://api.ads.apple.com/v1/campaigns/123?include=budget adAccountId=AD_ACCOUNT;",
	}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s\nwant:\n%s", strings.Join(seen, "\n"), strings.Join(want, "\n"))
	}
	if rate := client.LastRateLimit(); rate.Limit != "100" || rate.Remaining != "99" {
		t.Fatalf("LastRateLimit() = %+v", rate)
	}
}

func TestNewClientRejectsUnsafeAdAccountIDs(t *testing.T) {
	for _, value := range []string{
		"123;orgId=999",
		"123\n456",
		"123\r456",
		"123\t456",
		"\n123",
	} {
		t.Run(value, func(t *testing.T) {
			_, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: value})
			if err == nil || !strings.Contains(err.Error(), "invalid ad account ID") {
				t.Fatalf("NewClient() error = %v, want invalid ad account ID", err)
			}
		})
	}
}

func TestNewClientRejectsUnsafeOrgIDs(t *testing.T) {
	for _, value := range []string{
		"123;adAccountId=999",
		"123\n456",
		"123\r456",
		"123\t456",
		"\n123",
	} {
		t.Run(value, func(t *testing.T) {
			_, err := NewClient(Credentials{AccessToken: "ACCESS", OrgID: value})
			if err == nil || !strings.Contains(err.Error(), "invalid organization ID") {
				t.Fatalf("NewClient() error = %v, want invalid organization ID", err)
			}
		})
	}
}

func TestContextHeaderRejectsUnsafeOrgIDBeforeHTTP(t *testing.T) {
	requests := 0
	client, err := NewClient(Credentials{AccessToken: "ACCESS"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			return jsonResponse(http.StatusOK, `{}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	client.credentials.OrgID = "123;adAccountId=999"

	_, err = client.Request(context.Background(), http.MethodGet, "v5/campaigns", nil, nil, true)
	if err == nil || !strings.Contains(err.Error(), "invalid organization ID") {
		t.Fatalf("Request() error = %v, want invalid organization ID", err)
	}
	if requests != 0 {
		t.Fatalf("HTTP requests = %d, want 0", requests)
	}
}

func TestContextHeaderRejectsUnsafeAdAccountIDBeforeHTTP(t *testing.T) {
	requests := 0
	client, err := NewClient(Credentials{AccessToken: "ACCESS"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			return jsonResponse(http.StatusOK, `{}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	client.credentials.AdAccountID = "123;orgId=999"

	_, err = client.RequestForVersion(context.Background(), APIVersionPlatformV1, http.MethodGet, "v1/campaigns", nil, nil, ContextAdAccount)
	if err == nil || !strings.Contains(err.Error(), "invalid ad account ID") {
		t.Fatalf("RequestForVersion() error = %v, want invalid ad account ID", err)
	}
	if requests != 0 {
		t.Fatalf("HTTP requests = %d, want 0", requests)
	}
}

func TestWithBlankPlatformBaseURLFallsBackToDefault(t *testing.T) {
	client, err := NewClient(
		Credentials{AccessToken: "ACCESS"},
		WithPlatformBaseURL("   "),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	if client.platformBaseURL != PlatformBaseURL {
		t.Fatalf("platformBaseURL = %q, want default %q", client.platformBaseURL, PlatformBaseURL)
	}
}

func TestRequestForVersionRejectsCrossVersionTargetsBeforeHTTP(t *testing.T) {
	requests := 0
	client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "AD_ACCOUNT"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			return jsonResponse(200, `{}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	for _, path := range []string{
		"v5/me",
		"https://api.searchads.apple.com/api/v5/me",
		"https://api.ads.apple.com/v1/../campaigns",
		"https://api.ads.apple.com/v1/%2e%2e/campaigns",
		"v1/%2e%2e/campaigns",
		"v1//example.com/campaigns",
		"v1/https://example.com/campaigns",
		"https://example.com@api.ads.apple.com/v1/me",
		"https://example.com/v1/me",
	} {
		if _, err := client.RequestForVersion(context.Background(), APIVersionPlatformV1, http.MethodGet, path, nil, nil, ContextNone); err == nil {
			t.Fatalf("RequestForVersion(%q) unexpectedly succeeded", path)
		}
	}
	if requests != 0 {
		t.Fatalf("HTTP requests = %d, want 0", requests)
	}
}

func TestRequestForVersionRejectsCrossVersionContextBeforeHTTP(t *testing.T) {
	requests := 0
	client, err := NewClient(Credentials{AccessToken: "ACCESS", OrgID: "ORG", AdAccountID: "ACCOUNT"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			return jsonResponse(200, `{}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	tests := []struct {
		version APIVersion
		path    string
		context ContextKind
	}{
		{version: APIVersionPlatformV1, path: "v1/me", context: ContextOrg},
		{version: APIVersionCampaignV5, path: "v5/me", context: ContextAdAccount},
		{version: APIVersionCampaignV5, path: "v5/me", context: ContextAdAccountOptional},
	}
	for _, tt := range tests {
		if _, err := client.RequestForVersion(context.Background(), tt.version, http.MethodGet, tt.path, nil, nil, tt.context); err == nil {
			t.Fatalf("RequestForVersion(%s, %d) unexpectedly succeeded", tt.version, tt.context)
		}
	}
	if requests != 0 {
		t.Fatalf("HTTP requests = %d, want 0", requests)
	}
}

func TestRequestForVersionRequiresAdAccountBeforeTokenResolution(t *testing.T) {
	requests := 0
	client, err := NewClient(Credentials{
		ClientID:       "CLIENT",
		TeamID:         "TEAM",
		KeyID:          "KEY",
		PrivateKeyPath: "missing.pem",
	}, WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return jsonResponse(200, `{}`), nil
	})}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.RequestForVersion(context.Background(), APIVersionPlatformV1, http.MethodGet, "v1/campaigns/123", nil, nil, ContextAdAccount)
	if err == nil || !strings.Contains(err.Error(), "ad account ID is required") {
		t.Fatalf("RequestForVersion() error = %v, want missing ad account", err)
	}
	if requests != 0 {
		t.Fatalf("HTTP requests = %d, want 0", requests)
	}
}

func TestRequestParsesPlatformErrorAndRateLimitHeaders(t *testing.T) {
	requests := 0
	client, err := NewClient(Credentials{AccessToken: "ACCESS"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			response := jsonResponse(429, `{"error":{"code":"RATE_LIMITED","message":"Slow down","details":[{"code":"ACCOUNT_LIMIT","message":"Try later","info":{"policy":"account"}}]}}`)
			response.Header.Set("RateLimit-Limit", "100")
			response.Header.Set("RateLimit-Remaining", "0")
			response.Header.Set("RateLimit-Reset", "42")
			return response, nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.RequestForVersion(context.Background(), APIVersionPlatformV1, http.MethodPost, "v1/ad-accounts", nil, nil, ContextNone)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("RequestForVersion() error = %v, want *APIError", err)
	}
	if apiErr.Version != APIVersionPlatformV1 || apiErr.Code != "RATE_LIMITED" || apiErr.Message != "Slow down" {
		t.Fatalf("APIError = %+v", apiErr)
	}
	if len(apiErr.Details) != 1 || apiErr.Details[0].Code != "ACCOUNT_LIMIT" || apiErr.Details[0].Info["policy"] != "account" {
		t.Fatalf("APIError details = %+v", apiErr.Details)
	}
	if apiErr.RateLimit.Limit != "100" || apiErr.RateLimit.Remaining != "0" || apiErr.RateLimit.Reset != "42" {
		t.Fatalf("APIError rate limit = %+v", apiErr.RateLimit)
	}
	if requests != 1 {
		t.Fatalf("mutation requests = %d, want 1", requests)
	}
}

func TestLegacyGetDoesNotGainPlatformRetryBehavior(t *testing.T) {
	requests := 0
	client, err := NewClient(Credentials{AccessToken: "ACCESS"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			return jsonResponse(429, `{"error":{"errors":[{"message":"Slow down","messageCode":"RATE_LIMITED"}]}}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.Request(context.Background(), http.MethodGet, "v5/me", nil, nil, false)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Version != APIVersionCampaignV5 {
		t.Fatalf("legacy error = %v, want v5 APIError", err)
	}
	if requests != 1 {
		t.Fatalf("legacy requests = %d, want 1", requests)
	}
}

func TestPlatformGetRetriesTransientFailure(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")
	asc.ResetConfigCacheForTest()
	t.Cleanup(asc.ResetConfigCacheForTest)

	requests := 0
	client, err := NewClient(Credentials{AccessToken: "ACCESS"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			if requests == 1 {
				return jsonResponse(http.StatusServiceUnavailable, `{"error":{"code":"UNAVAILABLE","message":"Try again"}}`), nil
			}
			return jsonResponse(http.StatusOK, `{}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	if _, err := client.RequestForVersion(context.Background(), APIVersionPlatformV1, http.MethodGet, "v1/me", nil, nil, ContextNone); err != nil {
		t.Fatalf("RequestForVersion() error: %v", err)
	}
	if requests != 2 {
		t.Fatalf("platform GET requests = %d, want 2", requests)
	}
}

func TestPlatformRetrySafeQueryPostRetriesRateLimit(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")
	asc.ResetConfigCacheForTest()
	t.Cleanup(asc.ResetConfigCacheForTest)

	attempts := 0
	client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "123"}, WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return jsonResponse(http.StatusTooManyRequests, `{"code":"RATE_LIMITED","message":"Slow down"}`), nil
		}
		return jsonResponse(http.StatusOK, `{"result":[]}`), nil
	})}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	client.now = func() time.Time { return time.Unix(0, 0) }
	spec := EndpointSpec{
		Method:    http.MethodPost,
		Path:      "v1/resources/query",
		Version:   APIVersionPlatformV1,
		Context:   ContextAdAccount,
		RetrySafe: true,
	}

	if _, err := client.Do(context.Background(), spec, nil, nil, json.RawMessage(`{}`)); err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestParsePlatformErrorAcceptsDirectEnvelopeAndStructuredInfo(t *testing.T) {
	err := parseErrorForVersion(
		[]byte(`{"code":"BAD_REQUEST","message":"Invalid request","details":[{"code":"BAD_FILTER","message":"Invalid filter","info":{"index":2,"values":["A","B"]}}]}`),
		400,
		http.Header{"Ratelimit-Limit": {"50"}},
		APIVersionPlatformV1,
	)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiErr.Code != "BAD_REQUEST" || len(apiErr.Details) != 1 || apiErr.Details[0].Info["index"] != float64(2) {
		t.Fatalf("APIError = %+v", apiErr)
	}
	if apiErr.RateLimit.Limit != "50" {
		t.Fatalf("rate limit = %+v", apiErr.RateLimit)
	}
}

func TestParsePlatformErrorSurfacesDetailsOnlyEnvelope(t *testing.T) {
	err := parseErrorForVersion(
		[]byte(`{"details":[{"code":"BAD_FILTER","message":"Invalid filter"}]}`),
		400,
		nil,
		APIVersionPlatformV1,
	)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiErr.Field != "BAD_FILTER" || apiErr.Detail != "Invalid filter" {
		t.Fatalf("APIError = %+v, want first detail surfaced", apiErr)
	}
	if got := apiErr.Error(); !strings.Contains(got, "BAD_FILTER") || !strings.Contains(got, "Invalid filter") {
		t.Fatalf("APIError.Error() = %q, want structured detail", got)
	}
}

func TestEmptySuccessBodyIsVersionSpecific(t *testing.T) {
	client, err := NewClient(Credentials{AccessToken: "ACCESS"}, WithHTTPClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(200, ""), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	legacy, err := client.Request(context.Background(), http.MethodGet, "v5/me", nil, nil, false)
	if err != nil {
		t.Fatalf("legacy request error: %v", err)
	}
	platform, err := client.RequestForVersion(context.Background(), APIVersionPlatformV1, http.MethodGet, "v1/me", nil, nil, ContextNone)
	if err != nil {
		t.Fatalf("platform request error: %v", err)
	}
	if string(legacy) != `{"data":null}` || string(platform) != `{}` {
		t.Fatalf("empty bodies = legacy %s platform %s", legacy, platform)
	}
}

func TestAdsRetryDelayPrefersRetryAfterThenRateLimitReset(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		headers  http.Header
		maxDelay time.Duration
		want     time.Duration
	}{
		{name: "retry after seconds", headers: http.Header{"Retry-After": {"3"}, "RateLimit-Reset": {"9"}}, maxDelay: 30 * time.Second, want: 3 * time.Second},
		{name: "retry after date", headers: http.Header{"Retry-After": {now.Add(4 * time.Second).Format(http.TimeFormat)}}, maxDelay: 30 * time.Second, want: 4 * time.Second},
		{name: "rate limit reset delta", headers: http.Header{"Retry-After": {"invalid"}, "RateLimit-Reset": {"5"}}, maxDelay: 30 * time.Second, want: 5 * time.Second},
		{name: "zero ignored", headers: http.Header{"Retry-After": {"0"}, "RateLimit-Reset": {"0"}}, maxDelay: 30 * time.Second, want: 0},
		{name: "negative ignored", headers: http.Header{"RateLimit-Reset": {"-1"}}, maxDelay: 30 * time.Second, want: 0},
		{name: "retry after capped", headers: http.Header{"Retry-After": {"31"}}, maxDelay: 5 * time.Second, want: 5 * time.Second},
		{name: "rate limit reset capped", headers: http.Header{"RateLimit-Reset": {"31"}}, maxDelay: 5 * time.Second, want: 5 * time.Second},
		{name: "retry after overflow capped", headers: http.Header{"Retry-After": {"9223372036854775807"}}, maxDelay: 5 * time.Second, want: 5 * time.Second},
		{name: "rate limit reset overflow capped", headers: http.Header{"RateLimit-Reset": {"9223372036854775807"}}, maxDelay: 5 * time.Second, want: 5 * time.Second},
		{name: "overflow remains bounded without cap", headers: http.Header{"Retry-After": {"9223372036854775807"}}, want: time.Duration(1<<63 - 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := adsRetryDelay(tt.headers, now, tt.maxDelay); got != tt.want {
				t.Fatalf("adsRetryDelay() = %s, want %s", got, tt.want)
			}
		})
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

func TestPaginateAllAggregatesPlatformV1Envelope(t *testing.T) {
	requests := []string{}
	client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "123"}, WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.URL.Query().Get("offset"))
		if req.URL.Query().Get("offset") == "0" {
			return jsonResponse(200, `{"result":[{"adamId":1},{"adamId":2}],"pagination":{"totalCount":3,"offset":0,"pageSize":2}}`), nil
		}
		return jsonResponse(200, `{"result":[{"adamId":3}],"pagination":{"totalCount":3,"offset":2,"pageSize":2}}`), nil
	})}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	spec := EndpointSpec{Version: APIVersionPlatformV1, Context: ContextAdAccount, Method: "GET", Path: "v1/search/apps"}
	raw, err := client.PaginateAll(context.Background(), spec, nil, url.Values{"query": {"test"}}, 0, 2, nil)
	if err != nil {
		t.Fatalf("PaginateAll() error = %v", err)
	}
	var got struct {
		Result []map[string]int `json:"result"`
	}
	if err := json.Unmarshal(raw, &got); err != nil || len(got.Result) != 3 || got.Result[2]["adamId"] != 3 {
		t.Fatalf("result = %+v error = %v", got.Result, err)
	}
	if strings.Join(requests, ",") != "0,2" {
		t.Fatalf("offsets = %v", requests)
	}
}

func TestPaginateAllPlatformV1ToleratesMissingTotalCount(t *testing.T) {
	requests := []string{}
	client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "123"}, WithHTTPClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		offset := req.URL.Query().Get("offset")
		requests = append(requests, offset)
		switch offset {
		case "0":
			return jsonResponse(200, `{"result":[{"adamId":1},{"adamId":2}],"pagination":{"offset":0,"pageSize":2}}`), nil
		case "2":
			return jsonResponse(200, `{"result":[{"adamId":3}],"pagination":{"offset":2,"pageSize":2}}`), nil
		default:
			t.Fatalf("unexpected offset %q", offset)
			return nil, nil
		}
	})}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	spec := EndpointSpec{Version: APIVersionPlatformV1, Context: ContextAdAccount, Method: "GET", Path: "v1/search/apps"}
	raw, err := client.PaginateAll(context.Background(), spec, nil, url.Values{"query": {"test"}}, 0, 2, nil)
	if err != nil {
		t.Fatalf("PaginateAll() error = %v", err)
	}
	var got struct {
		Result     []map[string]int `json:"result"`
		Pagination map[string]any   `json:"pagination"`
	}
	if err := json.Unmarshal(raw, &got); err != nil || len(got.Result) != 3 || got.Result[2]["adamId"] != 3 {
		t.Fatalf("result = %+v error = %v", got.Result, err)
	}
	if _, present := got.Pagination["totalCount"]; present {
		t.Fatalf("pagination unexpectedly invented totalCount: %+v", got.Pagination)
	}
	if strings.Join(requests, ",") != "0,2" {
		t.Fatalf("offsets = %v", requests)
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
