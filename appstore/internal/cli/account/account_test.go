package account

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	authsvc "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/auth"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestAuthHealthCheckHonorsRootProfileSelection(t *testing.T) {
	previousProfile := shared.SelectedProfile()
	shared.SetSelectedProfile("work")
	t.Cleanup(func() { shared.SetSelectedProfile(previousProfile) })

	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.json")
	keyPath := filepath.Join(configDir, "work.p8")
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey() error: %v", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	if err := authsvc.StoreCredentialsConfigAt("work", "WORKKEY", "12345678-abcd-1234-abcd-123456789012", keyPath, configPath); err != nil {
		t.Fatalf("StoreCredentialsConfigAt() error: %v", err)
	}

	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_PROFILE", "")
	t.Setenv("ASC_KEY_ID", "ENVKEY")
	t.Setenv("ASC_ISSUER_ID", "12345678-abcd-1234-abcd-123456789012")
	t.Setenv("ASC_KEY_TYPE", "")
	t.Setenv("ASC_PRIVATE_KEY", "")
	t.Setenv("ASC_PRIVATE_KEY_B64", "")
	t.Setenv("ASC_PRIVATE_KEY_PATH", filepath.Join(t.TempDir(), "ignored-missing.p8"))

	check := authHealthCheck()
	if check.Status == "fail" {
		t.Fatalf("expected root profile selection to suppress ignored environment key failure, got %#v", check)
	}
}

func TestSummarizeAccountChecks(t *testing.T) {
	red := summarizeAccountChecks([]accountCheck{
		{Name: "authentication", Status: "fail", Message: "auth broken"},
		{Name: "api_access", Status: "ok", Message: "ok"},
	})
	if red.Health != "red" {
		t.Fatalf("expected red health, got %q", red.Health)
	}
	if red.ErrorCount != 1 {
		t.Fatalf("expected one error, got %d", red.ErrorCount)
	}
	if red.NextAction != "auth broken" {
		t.Fatalf("unexpected next action %q", red.NextAction)
	}

	yellow := summarizeAccountChecks([]accountCheck{
		{Name: "authentication", Status: "ok", Message: "ok"},
		{Name: "agreements", Status: "unavailable", Message: "not available"},
	})
	if yellow.Health != "yellow" {
		t.Fatalf("expected yellow health, got %q", yellow.Health)
	}
	if yellow.WarningCount != 1 {
		t.Fatalf("expected one warning, got %d", yellow.WarningCount)
	}

	green := summarizeAccountChecks([]accountCheck{
		{Name: "authentication", Status: "ok", Message: "ok"},
		{Name: "api_access", Status: "ok", Message: "ok"},
	})
	if green.Health != "green" {
		t.Fatalf("expected green health, got %q", green.Health)
	}
	if green.NextAction != "No action needed." {
		t.Fatalf("unexpected next action %q", green.NextAction)
	}
}
