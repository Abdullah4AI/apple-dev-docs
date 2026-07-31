package asc

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestDoStreamNoAuthRefusesRedirectWithoutForwardingReferer(t *testing.T) {
	var redirectedRequests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedRequests.Add(1)
		t.Errorf("redirect target received request with referer %q", r.Referer())
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/capture", http.StatusFound)
	}))
	defer source.Close()

	client := &Client{httpClient: source.Client()}
	rawURL := source.URL + "/download?Signature=signed-secret"
	_, err := client.doStreamNoAuth(context.Background(), rawURL, "application/octet-stream")
	if err == nil {
		t.Fatal("expected redirect response to fail")
	}
	if got := redirectedRequests.Load(); got != 0 {
		t.Fatalf("redirect target received %d requests, want 0", got)
	}
	if strings.Contains(err.Error(), "signed-secret") {
		t.Fatalf("redirect error leaked signed query: %v", err)
	}
}

func TestDoStreamNoAuthSanitizesTransportErrorAndPreservesCause(t *testing.T) {
	sentinel := errors.New("dial failed for https://download.example/file?Signature=signed-secret")
	client := &Client{httpClient: &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, sentinel
		}),
	}}

	_, err := client.doStreamNoAuth(
		context.Background(),
		"https://download.example/file?Signature=signed-secret",
		"application/octet-stream",
	)
	if err == nil {
		t.Fatal("expected transport error")
	}
	if strings.Contains(err.Error(), "signed-secret") || strings.Contains(err.Error(), "Signature") {
		t.Fatalf("transport error leaked signed query: %v", err)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("transport cause was not preserved: %v", err)
	}
}

func TestDoStreamNoAuthDoesNotEchoCapabilityFromErrorBody(t *testing.T) {
	client := &Client{httpClient: &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "403 Forbidden",
				StatusCode: http.StatusForbidden,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(
					`{"errors":[{"title":"Denied","detail":"https://download.example/file?Signature=signed-secret"}]}`,
				)),
			}, nil
		}),
	}}

	_, err := client.doStreamNoAuth(
		context.Background(),
		"https://download.example/file?Signature=signed-secret",
		"application/octet-stream",
	)
	if err == nil {
		t.Fatal("expected HTTP error")
	}
	if strings.Contains(err.Error(), "signed-secret") || strings.Contains(err.Error(), "Signature") {
		t.Fatalf("HTTP error leaked capability-bearing response text: %v", err)
	}
	if !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("HTTP error lost safe status context: %v", err)
	}
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("HTTP error lost forbidden classification: %v", err)
	}
}
