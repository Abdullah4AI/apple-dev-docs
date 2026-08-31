//go:build darwin

package signing

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigurePersistentSigningKeychainCleansUpWithIndependentContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	configureErr := errors.New("configuration failed")
	deleted := false
	err := configurePersistentSigningKeychain(
		ctx,
		"/tmp/persistent.keychain-db",
		func(runCtx context.Context, _ []byte, _ ...string) ([]byte, []byte, error) {
			cancel()
			return nil, nil, configureErr
		},
		func(cleanupCtx context.Context, path string) error {
			if cleanupCtx.Err() != nil {
				t.Fatalf("cleanup context is canceled: %v", cleanupCtx.Err())
			}
			if path != "/tmp/persistent.keychain-db" {
				t.Fatalf("cleanup path = %q", path)
			}
			deleted = true
			return nil
		},
	)
	if !errors.Is(err, configureErr) || !deleted {
		t.Fatalf("configure error = %v, deleted = %v", err, deleted)
	}
}

func TestPersistentSigningProbeUsesPrivateTemporaryDirectory(t *testing.T) {
	operatorDir := t.TempDir()
	operatorProbe := filepath.Join(operatorDir, "codesign-probe")
	if err := os.WriteFile(operatorProbe, []byte("operator-owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	keychainPath := filepath.Join(operatorDir, "persistent.keychain-db")
	verified := false
	err := withPersistentSigningProbe(
		context.Background(),
		keychainPath,
		strings.Repeat("A", 40),
		createSigningRunTempDir,
		removeSigningRunTempDir,
		func(_ context.Context, probeDir, gotKeychainPath, _ string) error {
			if probeDir == operatorDir || filepath.Dir(probeDir) != filepath.Clean(os.TempDir()) {
				t.Fatalf("probe directory = %q, operator directory = %q", probeDir, operatorDir)
			}
			if gotKeychainPath != keychainPath {
				t.Fatalf("keychain path = %q", gotKeychainPath)
			}
			verified = true
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Fatal("codesign probe did not run")
	}
	data, err := os.ReadFile(operatorProbe)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "operator-owned" {
		t.Fatalf("operator probe content = %q", data)
	}
}

func TestValidatePersistentSigningCertificateFingerprintsAllowsCertificateChain(t *testing.T) {
	leaf := strings.Repeat("A", 40)
	chain := []string{strings.Repeat("B", 40), leaf, strings.Repeat("C", 40)}
	if err := validatePersistentSigningCertificateFingerprints(chain, strings.ToLower(leaf)); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePersistentSigningCertificateFingerprintsRequiresLeafExactlyOnce(t *testing.T) {
	leaf := strings.Repeat("A", 40)
	for _, certificates := range [][]string{
		{strings.Repeat("B", 40)},
		{leaf, leaf},
	} {
		if err := validatePersistentSigningCertificateFingerprints(certificates, leaf); err == nil {
			t.Fatalf("certificates = %v, want exact-leaf-count failure", certificates)
		}
	}
}
