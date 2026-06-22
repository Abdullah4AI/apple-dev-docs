package storekit

import (
	"path/filepath"
	"strings"
	"testing"

	storekitapi "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/storekit"
)

func TestResolveCredentialsWithExplicitProfileIgnoresIncompleteEnvironmentCredentials(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_STOREKIT_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_STOREKIT_KEY_ID", "stray-key-id")
	stored := storekitapi.Credentials{
		KeyID:          "PROFILE_KEY",
		IssuerID:       "PROFILE_ISSUER",
		PrivateKeyPath: "/profiles/SubscriptionKey.p8",
		BundleID:       "com.example.profile",
	}
	if err := storekitapi.StoreCredentialsConfigAt("saved", stored, configPath); err != nil {
		t.Fatalf("StoreCredentialsConfigAt() error = %v", err)
	}
	profile := "saved"

	credentials, source, err := resolveCredentialsWithSource(commonFlags{Profile: &profile})
	if err != nil {
		t.Fatalf("resolveCredentialsWithSource() error = %v", err)
	}
	if source != "--storekit-profile" || credentials.KeyID != stored.KeyID || credentials.BundleID != stored.BundleID {
		t.Fatalf("credentials = %#v source=%q", credentials, source)
	}
}

func TestResolveCredentialsWithExplicitProfileStrictAuthRejectsIncompleteEnvironmentCredentials(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_STOREKIT_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_STOREKIT_KEY_ID", "stray-key-id")
	t.Setenv("ASC_STOREKIT_STRICT_AUTH", "1")
	stored := storekitapi.Credentials{
		KeyID:          "PROFILE_KEY",
		IssuerID:       "PROFILE_ISSUER",
		PrivateKeyPath: "/profiles/SubscriptionKey.p8",
		BundleID:       "com.example.profile",
	}
	if err := storekitapi.StoreCredentialsConfigAt("saved", stored, configPath); err != nil {
		t.Fatalf("StoreCredentialsConfigAt() error = %v", err)
	}
	profile := "saved"

	_, _, err := resolveCredentialsWithSource(commonFlags{Profile: &profile})
	if err == nil || !strings.Contains(err.Error(), "mixed StoreKit authentication sources") {
		t.Fatalf("resolveCredentialsWithSource() error = %v", err)
	}
}

func TestResolveCredentialsWithoutProfileRejectsIncompleteEnvironmentCredentials(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("ASC_STOREKIT_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_STOREKIT_KEY_ID", "partial-key-id")

	_, _, err := resolveCredentialsWithSource(commonFlags{})
	if err == nil || !strings.Contains(err.Error(), "incomplete StoreKit environment credentials") {
		t.Fatalf("resolveCredentialsWithSource() error = %v", err)
	}
}
