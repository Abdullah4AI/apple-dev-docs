package asc

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The sentinels stand in for the capability-bearing parts of a presigned URL.
const (
	presignedSignatureSentinel = "asc-red-sentinel-presigned-sig-8e15fa"
	presignedTokenSentinel     = "asc-red-sentinel-presigned-token-c40d27"
	presignedUserinfoSentinel  = "asc-red-sentinel-presigned-userinfo-5a9b36"
)

type failingUploadTransport struct {
	err     error
	request *http.Request
}

func (t *failingUploadTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.request = req
	return nil, t.err
}

func writeUploadFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "asset.bin")
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func presignedUploadURL() string {
	return "https://" + presignedUserinfoSentinel + ":secret@upload.example.com/v1/assets/chunk-1" +
		"?X-Amz-Signature=" + presignedSignatureSentinel +
		"&token=" + presignedTokenSentinel +
		"#" + presignedTokenSentinel
}

func assertNoPresignedCredentials(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an upload error")
	}
	message := err.Error()
	for _, sentinel := range []string{presignedSignatureSentinel, presignedTokenSentinel, presignedUserinfoSentinel} {
		if strings.Contains(message, sentinel) {
			t.Fatalf("upload error leaked presigned credentials: %q", message)
		}
	}
	if !strings.Contains(message, "upload.example.com") {
		t.Fatalf("upload error dropped safe operation context: %q", message)
	}
}

func TestUploadAssetFromFileSanitizesTransportErrorURL(t *testing.T) {
	path := writeUploadFixture(t)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer file.Close()

	transportErr := errors.New("dial tcp: connection refused")
	transport := &failingUploadTransport{err: transportErr}
	t.Cleanup(swapUploadTransport(t, transport))

	uploadErr := UploadAssetFromFile(context.Background(), file, 3, []UploadOperation{
		{Method: http.MethodPut, URL: presignedUploadURL(), Length: 3, Offset: 0},
	})

	assertNoPresignedCredentials(t, uploadErr)
	if !strings.Contains(uploadErr.Error(), "upload operation 0") {
		t.Fatalf("upload error dropped the operation index: %q", uploadErr.Error())
	}
	if !errors.Is(uploadErr, transportErr) {
		t.Fatalf("errors.Is lost the wrapped transport error: %v", uploadErr)
	}
	if transport.request == nil || transport.request.URL.Query().Get("X-Amz-Signature") != presignedSignatureSentinel {
		t.Fatal("the real presigned URL must still be used for the request")
	}
}

func TestUploadAssetFromFileSanitizesTimeoutErrorURL(t *testing.T) {
	// A timed-out PUT is retried, so this also covers redaction of the error
	// that survives retry exhaustion.
	setFastAssetUploadRetries(t, "1")

	path := writeUploadFixture(t)
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer file.Close()

	transport := &failingUploadTransport{err: context.DeadlineExceeded}
	t.Cleanup(swapUploadTransport(t, transport))

	uploadErr := UploadAssetFromFile(context.Background(), file, 3, []UploadOperation{
		{Method: http.MethodPut, URL: presignedUploadURL(), Length: 3, Offset: 0},
	})

	assertNoPresignedCredentials(t, uploadErr)
	if !errors.Is(uploadErr, context.DeadlineExceeded) {
		t.Fatalf("errors.Is lost the deadline classification: %v", uploadErr)
	}
	if !strings.Contains(uploadErr.Error(), "timeout") {
		t.Fatalf("upload error dropped timeout context: %q", uploadErr.Error())
	}
}

func TestExecuteUploadOperationsSanitizesTransportErrorURL(t *testing.T) {
	path := writeUploadFixture(t)

	transportErr := errors.New("tls: handshake failure")
	client := &http.Client{Transport: &failingUploadTransport{err: transportErr}}

	uploadErr := ExecuteUploadOperations(
		context.Background(),
		path,
		[]UploadOperation{{Method: http.MethodPost, URL: presignedUploadURL(), Length: 3, Offset: 0}},
		WithUploadHTTPClient(client),
	)

	assertNoPresignedCredentials(t, uploadErr)
	if !errors.Is(uploadErr, transportErr) {
		t.Fatalf("errors.Is lost the wrapped transport error: %v", uploadErr)
	}
}

func swapUploadTransport(t *testing.T, transport http.RoundTripper) func() {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = transport
	return func() {
		http.DefaultTransport = original
	}
}
