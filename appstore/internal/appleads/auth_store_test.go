package appleads

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99designs/keyring"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/config"
)

func TestStoreCredentialsConfigUsesActiveConfigPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "active-config.json")
	t.Setenv("ASC_CONFIG_PATH", configPath)

	if err := StoreCredentialsConfig("ads", testAdsCredentials()); err != nil {
		t.Fatalf("StoreCredentialsConfig() error: %v", err)
	}

	cfg, err := config.LoadAt(configPath)
	if err != nil {
		t.Fatalf("LoadAt(active) error: %v", err)
	}
	if len(cfg.Ads.Keys) != 1 || cfg.Ads.Keys[0].Name != "ads" {
		t.Fatalf("active config ads keys = %+v, want ads profile", cfg.Ads.Keys)
	}
}

func TestLoadConfigWithPathDoesNotFallbackToGlobalWhenASCConfigPathSet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	globalPath, err := config.GlobalPath()
	if err != nil {
		t.Fatalf("GlobalPath() error: %v", err)
	}
	if err := StoreCredentialsConfigAt("global", testAdsCredentials(), globalPath); err != nil {
		t.Fatalf("StoreCredentialsConfigAt(global) error: %v", err)
	}

	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	_, _, err = loadConfigWithPath()
	if !errors.Is(err, config.ErrNotFound) {
		t.Fatalf("loadConfigWithPath() error = %v, want ErrNotFound", err)
	}
}

func TestGetCredentialsFallsBackToConfigDefaultWhenKeychainHasNoDefault(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("ASC_CONFIG_PATH", configPath)
	if err := StoreCredentialsConfigAt("config-default", testAdsCredentials(), configPath); err != nil {
		t.Fatalf("StoreCredentialsConfigAt() error: %v", err)
	}

	keychainPayload, err := json.Marshal(credentialPayload{
		ClientID:       "KEYCHAIN_CLIENT",
		TeamID:         "KEYCHAIN_TEAM",
		KeyID:          "KEYCHAIN_KEY",
		PrivateKeyPath: "keychain-private-key.pem",
		OrgID:          "999999",
	})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	original := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return fakeAdsKeyring{
			items: map[string]keyring.Item{
				keyringKey("keychain-only"): {
					Key:  keyringKey("keychain-only"),
					Data: keychainPayload,
				},
				keyringKey("keychain-other"): {
					Key:  keyringKey("keychain-other"),
					Data: keychainPayload,
				},
			},
		}, nil
	}
	t.Cleanup(func() { openKeyring = original })

	credentials, source, err := GetCredentialsWithSource("")
	if err != nil {
		t.Fatalf("GetCredentialsWithSource() error: %v", err)
	}
	if source != "config" || credentials.Profile != "config-default" || credentials.ClientID != "CLIENT" {
		t.Fatalf("credentials = %+v source = %q, want config default profile", credentials, source)
	}
}

func TestGetCredentialsPrefersConfigDefaultOverSingleKeychainFallback(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("ASC_CONFIG_PATH", configPath)
	configCreds := testAdsCredentials()
	configCreds.ClientID = "CONFIG_CLIENT"
	if err := StoreCredentialsConfigAt("config-default", configCreds, configPath); err != nil {
		t.Fatalf("StoreCredentialsConfigAt() error: %v", err)
	}

	keychainPayload, err := json.Marshal(credentialPayload{
		ClientID:       "KEYCHAIN_CLIENT",
		TeamID:         "KEYCHAIN_TEAM",
		KeyID:          "KEYCHAIN_KEY",
		PrivateKeyPath: "keychain-private-key.pem",
		OrgID:          "999999",
	})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	original := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return fakeAdsKeyring{
			items: map[string]keyring.Item{
				keyringKey("keychain-only"): {
					Key:  keyringKey("keychain-only"),
					Data: keychainPayload,
				},
			},
		}, nil
	}
	t.Cleanup(func() { openKeyring = original })

	credentials, source, err := GetCredentialsWithSource("")
	if err != nil {
		t.Fatalf("GetCredentialsWithSource() error: %v", err)
	}
	if source != "config" || credentials.Profile != "config-default" || credentials.ClientID != "CONFIG_CLIENT" {
		t.Fatalf("credentials = %+v source = %q, want config default over keychain fallback", credentials, source)
	}
}

func TestGetCredentialsPrefersActiveConfigProfileOverSameNamedKeychainProfile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("ASC_CONFIG_PATH", configPath)
	configCreds := testAdsCredentials()
	configCreds.ClientID = "CONFIG_CLIENT"
	configCreds.OrgID = "CONFIG_ORG"
	if err := StoreCredentialsConfigAt("shared", configCreds, configPath); err != nil {
		t.Fatalf("StoreCredentialsConfigAt() error: %v", err)
	}

	keychainPayload, err := json.Marshal(credentialPayload{
		ClientID:       "KEYCHAIN_CLIENT",
		TeamID:         "KEYCHAIN_TEAM",
		KeyID:          "KEYCHAIN_KEY",
		PrivateKeyPath: "keychain-private-key.pem",
		OrgID:          "KEYCHAIN_ORG",
	})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}
	original := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		return fakeAdsKeyring{
			items: map[string]keyring.Item{
				keyringKey("shared"): {
					Key:  keyringKey("shared"),
					Data: keychainPayload,
				},
			},
		}, nil
	}
	t.Cleanup(func() { openKeyring = original })

	credentials, source, err := GetCredentialsWithSource("shared")
	if err != nil {
		t.Fatalf("GetCredentialsWithSource() error: %v", err)
	}
	if source != "config" || credentials.ClientID != "CONFIG_CLIENT" || credentials.OrgID != "CONFIG_ORG" {
		t.Fatalf("credentials = %+v source = %q, want active config profile", credentials, source)
	}
}

func TestBypassKeychainRemovalSkipsKeychain(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_ADS_BYPASS_KEYCHAIN", "1")
	if err := StoreCredentialsConfigAt("ads", testAdsCredentials(), configPath); err != nil {
		t.Fatalf("StoreCredentialsConfigAt() error: %v", err)
	}

	called := false
	original := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		called = true
		return nil, errors.New("keychain should be bypassed")
	}
	t.Cleanup(func() { openKeyring = original })

	if err := RemoveCredentials("ads"); err != nil {
		t.Fatalf("RemoveCredentials() error: %v", err)
	}
	if called {
		t.Fatal("RemoveCredentials opened keychain despite ASC_ADS_BYPASS_KEYCHAIN")
	}

	cfg, err := config.LoadAt(configPath)
	if err != nil {
		t.Fatalf("LoadAt() error: %v", err)
	}
	if len(cfg.Ads.Keys) != 0 {
		t.Fatalf("ads config after removal = %+v, want no keys", cfg.Ads.Keys)
	}
}

func TestShouldBypassKeychainAcceptsDocumentedTruthyValues(t *testing.T) {
	for _, value := range []string{"1", "true", "yes", "y", "on"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("ASC_ADS_BYPASS_KEYCHAIN", value)
			if !ShouldBypassKeychain() {
				t.Fatalf("ShouldBypassKeychain() = false for %q, want true", value)
			}
		})
	}
}

func TestRemoveCredentialsReturnsNotFoundWhenProfileMissing(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_ADS_BYPASS_KEYCHAIN", "1")
	if err := StoreCredentialsConfigAt("ads", testAdsCredentials(), configPath); err != nil {
		t.Fatalf("StoreCredentialsConfigAt() error: %v", err)
	}

	err := RemoveCredentials("missing")
	if !errors.Is(err, keyring.ErrKeyNotFound) {
		t.Fatalf("RemoveCredentials() error = %v, want key not found", err)
	}
}

func TestBypassKeychainRemoveAllSkipsKeychain(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_ADS_BYPASS_KEYCHAIN", "1")
	if err := StoreCredentialsConfigAt("ads", testAdsCredentials(), configPath); err != nil {
		t.Fatalf("StoreCredentialsConfigAt() error: %v", err)
	}

	called := false
	original := openKeyring
	openKeyring = func() (keyring.Keyring, error) {
		called = true
		return nil, errors.New("keychain should be bypassed")
	}
	t.Cleanup(func() { openKeyring = original })

	if err := RemoveAllCredentials(); err != nil {
		t.Fatalf("RemoveAllCredentials() error: %v", err)
	}
	if called {
		t.Fatal("RemoveAllCredentials opened keychain despite ASC_ADS_BYPASS_KEYCHAIN")
	}

	cfg, err := config.LoadAt(configPath)
	if err != nil {
		t.Fatalf("LoadAt() error: %v", err)
	}
	if len(cfg.Ads.Keys) != 0 || strings.TrimSpace(cfg.Ads.DefaultKeyName) != "" {
		t.Fatalf("ads config after clear = %+v, want empty credentials", cfg.Ads)
	}
}

type fakeAdsKeyring struct {
	items map[string]keyring.Item
}

func (f fakeAdsKeyring) Get(key string) (keyring.Item, error) {
	item, ok := f.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (f fakeAdsKeyring) GetMetadata(key string) (keyring.Metadata, error) {
	item, ok := f.items[key]
	if !ok {
		return keyring.Metadata{}, keyring.ErrKeyNotFound
	}
	return keyring.Metadata{
		Item: &keyring.Item{
			Key:         item.Key,
			Label:       item.Label,
			Description: item.Description,
		},
	}, nil
}

func (f fakeAdsKeyring) Set(item keyring.Item) error {
	f.items[item.Key] = item
	return nil
}

func (f fakeAdsKeyring) Remove(key string) error {
	if _, ok := f.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(f.items, key)
	return nil
}

func (f fakeAdsKeyring) Keys() ([]string, error) {
	keys := make([]string, 0, len(f.items))
	for key := range f.items {
		keys = append(keys, key)
	}
	return keys, nil
}

func testAdsCredentials() Credentials {
	return Credentials{
		ClientID:       "CLIENT",
		TeamID:         "TEAM",
		KeyID:          "KEY",
		PrivateKeyPath: "private-key.pem",
		OrgID:          "123456",
	}
}
