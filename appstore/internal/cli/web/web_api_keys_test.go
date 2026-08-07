package web

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

const apiKeyFixtureP8 = "-----BEGIN PRIVATE KEY-----\nfixture-secret\n-----END PRIVATE KEY-----\n"

func TestWebAPIKeysCreateSavesP8AndPrintsMetadata(t *testing.T) {
	restore := installWebAPIKeyCreateFakes(t)
	t.Cleanup(restore)

	outputDir := t.TempDir()
	cmd := WebAPIKeysCreateCommand()
	if err := cmd.FlagSet.Parse([]string{
		"--name", "Release automation",
		"--role", "APP_MANAGER",
		"--output-dir", outputDir,
		"--output", "json",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, stderr := captureOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, want := range []string{
		`"keyId":"ABC123XYZ"`,
		`"issuerId":"69a6de00-aaaa-bbbb-cccc-123456789abc"`,
		`"roles":["APP_MANAGER"]`,
		`"p8Path":`,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected stdout to contain %s, got %q", want, stdout)
		}
	}
	if strings.Contains(stdout, "fixture-secret") || strings.Contains(stderr, "fixture-secret") {
		t.Fatal("expected P8 contents to stay out of command output")
	}

	path := filepath.Join(outputDir, "AuthKey_ABC123XYZ.p8")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved P8: %v", err)
	}
	if string(contents) != apiKeyFixtureP8 {
		t.Fatalf("unexpected P8 contents: %q", string(contents))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved P8: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected mode 0600, got %04o", info.Mode().Perm())
	}
}

func TestWebAPIKeysCreateValidatesBeforeResolvingSession(t *testing.T) {
	resolveCalled := false
	restoreSession := SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		resolveCalled = true
		return &webcore.AuthSession{}, "cache", nil
	})
	t.Cleanup(restoreSession)

	cmd := WebAPIKeysCreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--name", "Release automation", "--role", "NOT_A_ROLE"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected usage error, got %v", err)
	}
	if resolveCalled {
		t.Fatal("did not expect session resolution before role validation")
	}
}

func TestDownloadWebAPIKeyWithRetryRetriesPropagationErrors(t *testing.T) {
	originalDownload := downloadWebAPIKeyFn
	originalWait := waitWebAPIKeyRetryFn
	t.Cleanup(func() {
		downloadWebAPIKeyFn = originalDownload
		waitWebAPIKeyRetryFn = originalWait
	})

	attempts := 0
	downloadWebAPIKeyFn = func(ctx context.Context, client *webcore.Client, keyID string) ([]byte, error) {
		attempts++
		if attempts < 3 {
			return nil, &webcore.APIError{Status: 404}
		}
		return []byte(apiKeyFixtureP8), nil
	}
	var waits []time.Duration
	waitWebAPIKeyRetryFn = func(ctx context.Context, delay time.Duration) error {
		waits = append(waits, delay)
		return nil
	}

	got, err := downloadWebAPIKeyWithRetry(context.Background(), &webcore.Client{}, "ABC123XYZ")
	if err != nil {
		t.Fatalf("downloadWebAPIKeyWithRetry() error: %v", err)
	}
	if string(got) != apiKeyFixtureP8 {
		t.Fatalf("unexpected P8 contents: %q", string(got))
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if len(waits) != 2 || waits[0] != time.Second || waits[1] != 2*time.Second {
		t.Fatalf("unexpected waits: %#v", waits)
	}
}

func TestDownloadWebAPIKeyWithRetryDoesNotRetryInvalidResponse(t *testing.T) {
	originalDownload := downloadWebAPIKeyFn
	originalWait := waitWebAPIKeyRetryFn
	t.Cleanup(func() {
		downloadWebAPIKeyFn = originalDownload
		waitWebAPIKeyRetryFn = originalWait
	})

	attempts := 0
	downloadWebAPIKeyFn = func(ctx context.Context, client *webcore.Client, keyID string) ([]byte, error) {
		attempts++
		return nil, fmt.Errorf("decode failed: %w", webcore.ErrAPIKeyResponseInvalid)
	}
	waits := 0
	waitWebAPIKeyRetryFn = func(ctx context.Context, delay time.Duration) error {
		waits++
		return nil
	}

	_, err := downloadWebAPIKeyWithRetry(context.Background(), &webcore.Client{}, "ABC123XYZ")
	if !errors.Is(err, webcore.ErrAPIKeyResponseInvalid) {
		t.Fatalf("expected invalid response error, got %v", err)
	}
	if attempts != 1 || waits != 0 {
		t.Fatalf("expected one attempt and no waits, got %d attempts and %d waits", attempts, waits)
	}
}

func TestWebAPIKeysCreateDoesNotReplaceExistingP8(t *testing.T) {
	restore := installWebAPIKeyCreateFakes(t)
	t.Cleanup(restore)
	downloadCalls := 0
	downloadWebAPIKeyFn = func(ctx context.Context, client *webcore.Client, keyID string) ([]byte, error) {
		downloadCalls++
		return []byte(apiKeyFixtureP8), nil
	}

	outputDir := t.TempDir()
	path := filepath.Join(outputDir, "AuthKey_ABC123XYZ.p8")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	cmd := WebAPIKeysCreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--name", "Release automation", "--output-dir", outputDir, "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var execErr error
	stdout, stderr := captureOutput(t, func() {
		execErr = cmd.Exec(context.Background(), nil)
	})
	if execErr == nil {
		t.Fatal("expected existing destination error")
	}
	if stdout != "" || strings.Contains(stderr, "fixture-secret") || strings.Contains(execErr.Error(), "fixture-secret") {
		t.Fatalf("expected no key material in output; stdout=%q stderr=%q err=%v", stdout, stderr, execErr)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}
	if string(contents) != "existing" {
		t.Fatalf("existing file was replaced: %q", string(contents))
	}
	if downloadCalls != 0 {
		t.Fatalf("expected destination collision before one-time download, got %d download calls", downloadCalls)
	}
}

func TestWebAPIKeysCreateRemovesReservationWhenDownloadFails(t *testing.T) {
	restore := installWebAPIKeyCreateFakes(t)
	t.Cleanup(restore)

	downloadErr := &webcore.APIError{Status: 403}
	downloadWebAPIKeyFn = func(ctx context.Context, client *webcore.Client, keyID string) ([]byte, error) {
		return nil, downloadErr
	}

	outputDir := t.TempDir()
	cmd := WebAPIKeysCreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--name", "Release automation", "--output-dir", outputDir}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, downloadErr) {
		t.Fatalf("expected download error, got %v", err)
	}
	path := filepath.Join(outputDir, "AuthKey_ABC123XYZ.p8")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected failed-download reservation to be removed, stat error = %v", statErr)
	}
}

func TestWebAPIKeysCreatePreservesP8WhenIssuerLookupFails(t *testing.T) {
	restore := installWebAPIKeyCreateFakes(t)
	t.Cleanup(restore)

	lookupErr := errors.New("lookup failed")
	getWebAPIKeyFn = func(ctx context.Context, client *webcore.Client, keyID string) (*webcore.APIKey, error) {
		return nil, lookupErr
	}
	outputDir := t.TempDir()
	cmd := WebAPIKeysCreateCommand()
	if err := cmd.FlagSet.Parse([]string{"--name", "Release automation", "--output-dir", outputDir}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	err := cmd.Exec(context.Background(), nil)
	if !errors.Is(err, lookupErr) {
		t.Fatalf("expected issuer lookup error, got %v", err)
	}
	path := filepath.Join(outputDir, "AuthKey_ABC123XYZ.p8")
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("expected recovery path in error, got %v", err)
	}
	contents, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("expected saved P8 after lookup failure: %v", readErr)
	}
	if string(contents) != apiKeyFixtureP8 {
		t.Fatalf("unexpected saved P8: %q", string(contents))
	}
}

func installWebAPIKeyCreateFakes(t *testing.T) func() {
	t.Helper()
	restoreSession := SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{TeamID: "TEAM123"}, "cache", nil
	})
	originalNewClient := newWebAPIKeyClientFn
	originalCreate := createWebAPIKeyFn
	originalDownload := downloadWebAPIKeyFn
	originalGet := getWebAPIKeyFn
	originalWait := waitWebAPIKeyRetryFn

	newWebAPIKeyClientFn = func(session *webcore.AuthSession) *webcore.Client {
		return &webcore.Client{}
	}
	createWebAPIKeyFn = func(ctx context.Context, client *webcore.Client, attrs webcore.APIKeyCreateAttributes) (*webcore.APIKey, error) {
		if attrs.Nickname != "Release automation" {
			t.Fatalf("unexpected nickname %q", attrs.Nickname)
		}
		return &webcore.APIKey{
			KeyID:          "ABC123XYZ",
			Name:           attrs.Nickname,
			Roles:          []string{attrs.Role},
			Active:         true,
			AllAppsVisible: true,
			CanDownload:    true,
			KeyType:        "PUBLIC_API",
		}, nil
	}
	downloadWebAPIKeyFn = func(ctx context.Context, client *webcore.Client, keyID string) ([]byte, error) {
		return []byte(apiKeyFixtureP8), nil
	}
	getWebAPIKeyFn = func(ctx context.Context, client *webcore.Client, keyID string) (*webcore.APIKey, error) {
		return &webcore.APIKey{
			KeyID:          keyID,
			Name:           "Release automation",
			IssuerID:       "69a6de00-aaaa-bbbb-cccc-123456789abc",
			Roles:          []string{"APP_MANAGER"},
			Active:         true,
			AllAppsVisible: true,
			KeyType:        "PUBLIC_API",
		}, nil
	}
	waitWebAPIKeyRetryFn = func(ctx context.Context, delay time.Duration) error { return nil }

	return func() {
		restoreSession()
		newWebAPIKeyClientFn = originalNewClient
		createWebAPIKeyFn = originalCreate
		downloadWebAPIKeyFn = originalDownload
		getWebAPIKeyFn = originalGet
		waitWebAPIKeyRetryFn = originalWait
	}
}
