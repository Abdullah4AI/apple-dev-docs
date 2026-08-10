package notarization

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

type notaryRoundTripper func(*http.Request) (*http.Response, error)

func (fn notaryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestNotarizationSubmitValidation(t *testing.T) {
	cmd := submitCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}
}

func TestNotarizationSubmitUploadsOpenedArchiveAfterPathReplacement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not permit replacing an open file")
	}

	originalContents := []byte("AAAAAAAAAAAAAAAA")
	replacementContents := []byte("BBBBBBBBBBBBBBBB")

	tests := []struct {
		name        string
		replacePath func(t *testing.T, archivePath, preservedPath string, replacement []byte)
	}{
		{
			name: "regular file",
			replacePath: func(t *testing.T, archivePath, preservedPath string, replacement []byte) {
				t.Helper()
				if err := os.Rename(archivePath, preservedPath); err != nil {
					t.Errorf("rename original archive: %v", err)
					return
				}
				if err := os.WriteFile(archivePath, replacement, 0o600); err != nil {
					t.Errorf("write replacement archive: %v", err)
				}
			},
		},
		{
			name: "symlink",
			replacePath: func(t *testing.T, archivePath, preservedPath string, replacement []byte) {
				t.Helper()
				replacementPath := filepath.Join(filepath.Dir(archivePath), "replacement.zip")
				if err := os.WriteFile(replacementPath, replacement, 0o600); err != nil {
					t.Errorf("write symlink target: %v", err)
					return
				}
				if err := os.Rename(archivePath, preservedPath); err != nil {
					t.Errorf("rename original archive: %v", err)
					return
				}
				if err := os.Symlink(replacementPath, archivePath); err != nil {
					t.Errorf("create replacement symlink: %v", err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			archivePath := filepath.Join(dir, "archive.zip")
			preservedPath := filepath.Join(dir, "original.zip")
			if err := os.WriteFile(archivePath, originalContents, 0o600); err != nil {
				t.Fatalf("write original archive: %v", err)
			}

			expectedHashBytes := sha256.Sum256(originalContents)
			expectedHash := hex.EncodeToString(expectedHashBytes[:])
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPost || req.URL.Path != "/notary/v2/submissions" {
					t.Errorf("notary request = %s %s, want POST /notary/v2/submissions", req.Method, req.URL.Path)
					http.Error(w, "unexpected request", http.StatusBadRequest)
					return
				}

				var payload asc.NotarySubmissionRequest
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Errorf("decode notary request: %v", err)
					http.Error(w, "invalid request", http.StatusBadRequest)
					return
				}
				if payload.Sha256 != expectedHash {
					t.Errorf("submitted hash = %q, want %q", payload.Sha256, expectedHash)
				}

				test.replacePath(t, archivePath, preservedPath, replacementContents)

				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(asc.NotarySubmissionResponse{
					Data: asc.NotarySubmissionResponseData{
						Type: "newSubmissions",
						ID:   "submission-1",
						Attributes: asc.NotarySubmissionResponseAttributes{
							AwsAccessKeyID:     "access-key",
							AwsSecretAccessKey: "secret-key",
							AwsSessionToken:    "session-token",
							Bucket:             "notary-bucket",
							Object:             "notary-object",
						},
					},
				}); err != nil {
					t.Errorf("encode notary response: %v", err)
				}
			}))
			t.Cleanup(server.Close)

			client := newNotarizationTestClient(t, server)
			t.Cleanup(shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
				return client, nil
			}))

			var uploadedContents []byte
			originalTransport := http.DefaultClient.Transport
			http.DefaultClient.Transport = notaryRoundTripper(func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				uploadedContents = body
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(bytes.NewReader(nil)),
					Request:    req,
				}, nil
			})
			t.Cleanup(func() {
				http.DefaultClient.Transport = originalTransport
			})

			cmd := submitCommand()
			if err := cmd.FlagSet.Parse([]string{"--file", archivePath, "--output", "json"}); err != nil {
				t.Fatalf("parse command: %v", err)
			}
			if err := cmd.Exec(context.Background(), nil); err != nil {
				t.Fatalf("submit command: %v", err)
			}
			if !bytes.Equal(uploadedContents, originalContents) {
				t.Fatalf("uploaded contents = %q, want originally opened archive %q", uploadedContents, originalContents)
			}
		})
	}
}

func newNotarizationTestClient(t *testing.T, server *httptest.Server) *asc.Client {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	keyPath := filepath.Join(t.TempDir(), "AuthKey_TEST.p8")
	keyData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(keyPath, keyData, 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	client, err := asc.NewClientWithHTTPClient("TEST_KEY", "TEST_ISSUER", keyPath, server.Client())
	if err != nil {
		t.Fatalf("create test client: %v", err)
	}
	client.SetNotaryBaseURL(server.URL)
	return client
}
