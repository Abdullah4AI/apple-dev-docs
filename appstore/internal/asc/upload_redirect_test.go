package asc

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestExecuteUploadOperationsRefusesRedirectWithoutForwardingBodyOrReferer(t *testing.T) {
	var redirectedRequests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedRequests.Add(1)
		body, _ := io.ReadAll(r.Body)
		t.Errorf("redirect target received request: method=%s referer=%q body=%q", r.Method, r.Referer(), body)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/capture", http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	filePath := filepath.Join(t.TempDir(), "asset.bin")
	if err := os.WriteFile(filePath, []byte("secret upload body"), 0o600); err != nil {
		t.Fatalf("write upload fixture: %v", err)
	}

	err := ExecuteUploadOperations(
		context.Background(),
		filePath,
		[]UploadOperation{{
			Method: http.MethodPut,
			URL:    source.URL + "/upload?X-Amz-Signature=signed-secret",
			Length: int64(len("secret upload body")),
		}},
		WithUploadHTTPClient(source.Client()),
		WithUploadConcurrency(1),
		withUploadRetryOptions(0),
	)
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

func TestUploadAssetFromFileRefusesRedirectWithoutForwardingBodyOrReferer(t *testing.T) {
	var redirectedRequests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedRequests.Add(1)
		body, _ := io.ReadAll(r.Body)
		t.Errorf("redirect target received request: method=%s referer=%q body=%q", r.Method, r.Referer(), body)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/capture", http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	filePath := filepath.Join(t.TempDir(), "asset.bin")
	if err := os.WriteFile(filePath, []byte("secret upload body"), 0o600); err != nil {
		t.Fatalf("write upload fixture: %v", err)
	}
	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("open upload fixture: %v", err)
	}
	defer file.Close()

	originalTransport := http.DefaultTransport
	http.DefaultTransport = source.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	err = UploadAssetFromFile(context.Background(), file, int64(len("secret upload body")), []UploadOperation{{
		Method: http.MethodPut,
		URL:    source.URL + "/upload?X-Amz-Signature=signed-secret",
		Length: int64(len("secret upload body")),
	}})
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
