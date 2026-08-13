package signing

import (
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestSigningSyncCommandLongHelpUsesOutputDirExample(t *testing.T) {
	cmd := SigningSyncCommand()
	if !strings.Contains(cmd.LongHelp, "--output-dir ./signing") {
		t.Fatalf("expected long help to document --output-dir, got %q", cmd.LongHelp)
	}
	if strings.Contains(cmd.LongHelp, "--output ./signing") {
		t.Fatalf("expected long help to avoid --output path example, got %q", cmd.LongHelp)
	}
}

func TestSigningSyncPushHelpDocumentsDeviceTransition(t *testing.T) {
	deviceFlag := syncPushCommand().FlagSet.Lookup("device")
	if deviceFlag == nil {
		t.Fatal("expected --device flag")
	}
	if !strings.Contains(deviceFlag.Usage, "--create-missing") ||
		!strings.Contains(deviceFlag.Usage, "deprecated") ||
		!strings.Contains(deviceFlag.Usage, "5.0.0") {
		t.Fatalf("--device usage = %q, want the transition and rejection release", deviceFlag.Usage)
	}
}

func TestSigningSyncPreparesRepositoryOnceInAssetOrder(t *testing.T) {
	tests := []struct {
		name       string
		hasProfile bool
		wantEvents []string
	}{
		{
			name:       "existing profile",
			hasProfile: true,
			wantEvents: []string{
				"GET /v1/bundleIds/bundle-main/profiles",
				"GET /v1/profiles/profile-main/certificates",
				"clone repository",
			},
		},
		{
			name: "missing profile",
			wantEvents: []string{
				"GET /v1/bundleIds/bundle-main/profiles",
				"GET /v1/certificates",
				"clone repository",
				"POST /v1/profiles",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := []string{}
			client := newSigningFetchTestClient(t, func(req *http.Request) *http.Response {
				events = append(events, req.Method+" "+req.URL.Path)
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/v1/bundleIds/bundle-main/profiles":
					if tt.hasProfile {
						return signingFetchJSONResponse(http.StatusOK, `{"data":[{"type":"profiles","id":"profile-main","attributes":{"profileType":"IOS_APP_STORE","profileState":"ACTIVE"}}]}`)
					}
					return signingFetchJSONResponse(http.StatusOK, `{"data":[]}`)
				case req.Method == http.MethodGet && req.URL.Path == "/v1/profiles/profile-main/certificates":
					return signingFetchJSONResponse(http.StatusOK, `{"data":[{"type":"certificates","id":"cert-1","attributes":{"certificateType":"IOS_DISTRIBUTION"}}]}`)
				case req.Method == http.MethodGet && req.URL.Path == "/v1/certificates":
					return signingFetchJSONResponse(http.StatusOK, `{"data":[{"type":"certificates","id":"cert-1","attributes":{"certificateType":"IOS_DISTRIBUTION","activated":true,"expirationDate":"2100-01-01T00:00:00Z"}}]}`)
				case req.Method == http.MethodPost && req.URL.Path == "/v1/profiles":
					return signingFetchJSONResponse(http.StatusCreated, `{"data":{"type":"profiles","id":"profile-created","attributes":{"profileType":"IOS_APP_STORE","profileState":"ACTIVE"}}}`)
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return signingFetchJSONResponse(http.StatusInternalServerError, `{}`)
				}
			})

			cloneCount := 0
			prepareRepository := onceAfterSuccess(func() error {
				cloneCount++
				events = append(events, "clone repository")
				return nil
			})
			_, _, _, err := resolveSigningAssets(
				context.Background(),
				client,
				signingAssetsOptions{
					BundleIDResourceID: "bundle-main",
					BundleIdentifier:   "com.example.signing.profile",
					ProfileType:        "IOS_APP_STORE",
					CreateMissing:      !tt.hasProfile,
					BeforeCreate: func(profileCreatePlan) error {
						return prepareRepository()
					},
				},
			)
			if err != nil {
				t.Fatalf("resolveSigningAssets() error: %v", err)
			}
			if err := prepareRepository(); err != nil {
				t.Fatalf("prepareRepository() error: %v", err)
			}
			if cloneCount != 1 {
				t.Fatalf("repository clone count = %d, want 1", cloneCount)
			}
			if strings.Join(events, ",") != strings.Join(tt.wantEvents, ",") {
				t.Fatalf("unexpected operation order: got %v, want %v", events, tt.wantEvents)
			}
		})
	}
}

func TestSigningSyncPushWarnsForDeviceWithoutCreateMissing(t *testing.T) {
	clientCalls := 0
	t.Cleanup(shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
		clientCalls++
		return nil, errors.New("client reached after validation")
	}))

	cmd := syncPushCommand()
	cmd.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := cmd.Parse([]string{
			"--bundle-id", "com.example.app",
			"--profile-type", "IOS_APP_DEVELOPMENT",
			"--repo", "git@github.com:team/certs.git",
			"--password", "secret",
			"--device", "DEVICE1",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = cmd.Run(context.Background())
	})

	if runErr == nil || runErr.Error() != "signing sync push: client reached after validation" {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("deprecated input must not return a usage error: %v", runErr)
	}
	if clientCalls != 1 {
		t.Fatalf("client factory calls = %d, want 1", clientCalls)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	wantWarning := "Warning: --device without --create-missing is deprecated and ignored because device IDs are only applied when creating a profile. Add --create-missing so they can be applied if a profile must be created. This combination will be rejected in 5.0.0.\n"
	if stderr != wantWarning {
		t.Fatalf("stderr = %q, want %q", stderr, wantWarning)
	}
}

func TestSanitizeRepoURLForOutputRedactsCredentials(t *testing.T) {
	raw := "https://token:secret@example.com/org/repo.git?access_token=abc123"
	got := sanitizeRepoURLForOutput(raw)

	if strings.Contains(got, "token:secret@") || strings.Contains(got, "secret") || strings.Contains(got, "abc123") {
		t.Fatalf("expected credentials to be redacted, got %q", got)
	}
	if !strings.Contains(got, "%5BREDACTED%5D") {
		t.Fatalf("expected sanitized marker, got %q", got)
	}
}

func TestSigningCommandLongHelpUsesOutputDirForSyncPull(t *testing.T) {
	cmd := SigningCommand()
	if !strings.Contains(cmd.LongHelp, "asc signing sync pull --repo git@github.com:team/certs.git --output-dir ./signing") {
		t.Fatalf("expected top-level help to use --output-dir for sync pull, got %q", cmd.LongHelp)
	}
	if strings.Contains(cmd.LongHelp, "asc signing sync pull --repo git@github.com:team/certs.git --output ./signing") {
		t.Fatalf("expected top-level help to avoid --output for sync pull, got %q", cmd.LongHelp)
	}
}

func TestSigningSyncCommandLongHelpPullExampleOmitsUnsupportedFlags(t *testing.T) {
	cmd := SigningSyncCommand()
	if strings.Contains(cmd.LongHelp, "asc signing sync pull --bundle-id") {
		t.Fatalf("expected pull example to omit --bundle-id, got %q", cmd.LongHelp)
	}
	if strings.Contains(cmd.LongHelp, "asc signing sync pull --profile-type") {
		t.Fatalf("expected pull example to omit --profile-type, got %q", cmd.LongHelp)
	}
}

func TestWriteDecryptedOutputFileWritesPlaintext(t *testing.T) {
	outDir := t.TempDir()
	relPath := filepath.Join("profiles", "appstore", "app.mobileprovision")
	plaintext := []byte("profile-data")

	if err := writeDecryptedOutputFile(outDir, relPath, plaintext); err != nil {
		t.Fatalf("writeDecryptedOutputFile: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(outDir, relPath))
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("output mismatch: got %q, want %q", got, plaintext)
	}
}

func TestWriteDecryptedOutputFileRejectsSymlinkTarget(t *testing.T) {
	outDir := t.TempDir()
	targetDir := t.TempDir()
	relPath := filepath.Join("profiles", "appstore", "app.mobileprovision")
	destPath := filepath.Join(outDir, relPath)
	targetPath := filepath.Join(targetDir, "app.mobileprovision")

	if err := os.WriteFile(targetPath, []byte("original"), 0o600); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		t.Fatalf("mkdir output parent: %v", err)
	}
	if err := os.Symlink(targetPath, destPath); err != nil {
		t.Fatalf("create destination symlink: %v", err)
	}

	err := writeDecryptedOutputFile(outDir, relPath, []byte("updated"))
	if err == nil {
		t.Fatal("expected symlink rejection error, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection error, got %v", err)
	}

	got, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("read target file: %v", readErr)
	}
	if string(got) != "original" {
		t.Fatalf("did not expect write through symlink target, got %q", got)
	}
}
