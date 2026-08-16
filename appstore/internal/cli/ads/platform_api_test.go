package ads

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/appleads"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

type platformRoundTripFunc func(*http.Request) (*http.Response, error)

func (f platformRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func platformJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestRawPlatformRequestPreservesRetrySafetyForKnownQueriesOnly(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")
	asc.ResetConfigCacheForTest()
	t.Cleanup(asc.ResetConfigCacheForTest)

	queryAttempts := 0
	client, err := appleads.NewClient(appleads.Credentials{AccessToken: "ACCESS", AdAccountID: "123"}, appleads.WithHTTPClient(&http.Client{
		Transport: platformRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			queryAttempts++
			if queryAttempts == 1 {
				return platformJSONResponse(http.StatusTooManyRequests, `{"code":"RATE_LIMITED"}`), nil
			}
			return platformJSONResponse(http.StatusOK, `{"result":[]}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	if _, err := requestRawPlatformEndpoint(context.Background(), client, http.MethodPost, "v1/eligibilities/apps/query", json.RawMessage(`{}`), appleads.ContextAdAccount); err != nil {
		t.Fatalf("known query request error: %v", err)
	}
	if queryAttempts != 2 {
		t.Fatalf("known query attempts = %d, want 2", queryAttempts)
	}

	mutationAttempts := 0
	client, err = appleads.NewClient(appleads.Credentials{AccessToken: "ACCESS", AdAccountID: "123"}, appleads.WithHTTPClient(&http.Client{
		Transport: platformRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			mutationAttempts++
			return platformJSONResponse(http.StatusServiceUnavailable, `{"code":"UNAVAILABLE"}`), nil
		}),
	}))
	if err != nil {
		t.Fatalf("NewClient() for mutation error: %v", err)
	}

	if _, err := requestRawPlatformEndpoint(context.Background(), client, http.MethodPost, "v1/creatives", json.RawMessage(`{}`), appleads.ContextAdAccount); err == nil {
		t.Fatal("known mutation unexpectedly succeeded")
	}
	if mutationAttempts != 1 {
		t.Fatalf("known mutation attempts = %d, want 1", mutationAttempts)
	}
}
