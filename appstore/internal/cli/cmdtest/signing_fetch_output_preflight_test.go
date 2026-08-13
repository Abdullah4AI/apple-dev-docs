package cmdtest

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// signingFetchStubAPI counts the App Store Connect calls signing fetch makes
// while resolving and creating a profile.
type signingFetchStubAPI struct {
	BundleLookups      atomic.Int32
	ProfileLookups     atomic.Int32
	CertificateLookups atomic.Int32
	ProfileCreates     atomic.Int32
}

// startSigningFetchStubAPI serves a bundle ID with no matching profile and a
// single eligible certificate, so signing fetch takes the --create-missing path.
func startSigningFetchStubAPI(t *testing.T) *signingFetchStubAPI {
	t.Helper()

	stub := &signingFetchStubAPI{}
	certificateContent := base64.StdEncoding.EncodeToString([]byte("certificate"))
	profileContent := base64.StdEncoding.EncodeToString([]byte("profile"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/bundleIds":
			stub.BundleLookups.Add(1)
			if got := req.URL.Query().Get("filter[identifier]"); got != "com.example.app" {
				t.Errorf("bundle identifier filter = %q, want com.example.app", got)
			}
			writeSigningFetchOutputJSON(t, w, http.StatusOK, `{"data":[{"type":"bundleIds","id":"bundle-main","attributes":{"identifier":"com.example.app"}}]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/bundleIds/bundle-main/profiles":
			stub.ProfileLookups.Add(1)
			writeSigningFetchOutputJSON(t, w, http.StatusOK, `{"data":[]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/certificates":
			stub.CertificateLookups.Add(1)
			writeSigningFetchOutputJSON(t, w, http.StatusOK, fmt.Sprintf(
				`{"data":[{"type":"certificates","id":"cert-1","attributes":{"certificateType":"IOS_DISTRIBUTION","serialNumber":"CERT1","certificateContent":%q,"activated":true,"expirationDate":"2100-01-01T00:00:00Z"}}]}`,
				certificateContent,
			))
		case req.Method == http.MethodPost && req.URL.Path == "/v1/profiles":
			stub.ProfileCreates.Add(1)
			writeSigningFetchOutputJSON(t, w, http.StatusCreated, fmt.Sprintf(
				`{"data":{"type":"profiles","id":"profile-created","attributes":{"name":"Created Profile","profileType":"IOS_APP_STORE","profileState":"ACTIVE","profileContent":%q}}}`,
				profileContent,
			))
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.String())
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	serverTransport := server.Client().Transport
	client, err := asc.NewClientWithHTTPClient(
		os.Getenv("ASC_KEY_ID"),
		os.Getenv("ASC_ISSUER_ID"),
		os.Getenv("ASC_PRIVATE_KEY_PATH"),
		&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			cloned := req.Clone(req.Context())
			cloned.URL.Scheme = serverURL.Scheme
			cloned.URL.Host = serverURL.Host
			return serverTransport.RoundTrip(cloned)
		})},
	)
	if err != nil {
		t.Fatalf("create signing fetch test client: %v", err)
	}
	t.Cleanup(shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
		return client, nil
	}))

	return stub
}

func runSigningFetchCreateMissing(t *testing.T, outputPath string) (string, string, error) {
	t.Helper()

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"signing", "fetch",
			"--bundle-id", "com.example.app",
			"--profile-type", "IOS_APP_STORE",
			"--create-missing",
			"--output", outputPath,
			"--format", "json",
		}); err != nil {
			t.Fatalf("parse signing fetch: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	return stdout, stderr, runErr
}

func TestSigningFetchPreflightsOutputBeforeProfileCreation(t *testing.T) {
	setupAuth(t)

	outputPath := filepath.Join(t.TempDir(), "not-a-directory")
	sentinel := []byte("keep this file")
	if err := os.WriteFile(outputPath, sentinel, 0o600); err != nil {
		t.Fatalf("write output sentinel: %v", err)
	}

	stub := startSigningFetchStubAPI(t)

	stdout, _, runErr := runSigningFetchCreateMissing(t, outputPath)

	if runErr == nil {
		t.Fatal("signing fetch succeeded with a regular file as its output directory")
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty output", stdout)
	}
	if got := stub.ProfileCreates.Load(); got != 0 {
		t.Fatalf("profile create requests = %d, want 0", got)
	}
	if got := stub.BundleLookups.Load(); got != 1 {
		t.Fatalf("bundle ID lookups = %d, want 1", got)
	}
	if got := stub.ProfileLookups.Load(); got != 1 {
		t.Fatalf("profile lookups = %d, want 1", got)
	}
	if got := stub.CertificateLookups.Load(); got != 1 {
		t.Fatalf("certificate lookups = %d, want 1", got)
	}
	gotSentinel, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output sentinel: %v", err)
	}
	if string(gotSentinel) != string(sentinel) {
		t.Fatalf("output sentinel = %q, want %q", gotSentinel, sentinel)
	}
}

func TestSigningFetchPreflightsExistingOutputFilesBeforeProfileCreation(t *testing.T) {
	setupAuth(t)

	outputDir := t.TempDir()
	certificatePath := filepath.Join(outputDir, "CERT1.cer")
	sentinel := []byte("certificate from an earlier fetch")
	if err := os.WriteFile(certificatePath, sentinel, 0o600); err != nil {
		t.Fatalf("write existing certificate: %v", err)
	}

	stub := startSigningFetchStubAPI(t)

	stdout, stderr, runErr := runSigningFetchCreateMissing(t, outputDir)

	if runErr == nil {
		t.Fatal("signing fetch overwrote or ignored an existing output file")
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty output", stdout)
	}
	if got := stub.ProfileCreates.Load(); got != 0 {
		t.Fatalf("profile create requests = %d, want 0 (a created profile would be orphaned): stderr=%q", got, stderr)
	}

	gotSentinel, err := os.ReadFile(certificatePath)
	if err != nil {
		t.Fatalf("read existing certificate: %v", err)
	}
	if string(gotSentinel) != string(sentinel) {
		t.Fatalf("existing certificate = %q, want %q", gotSentinel, sentinel)
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("read output directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "CERT1.cer" {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("output directory entries = %v, want only the pre-existing certificate", names)
	}
}

func writeSigningFetchOutputJSON(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, body); err != nil {
		t.Errorf("write JSON response: %v", err)
	}
}
