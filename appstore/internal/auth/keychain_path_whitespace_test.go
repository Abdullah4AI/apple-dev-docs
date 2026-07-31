package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMigrationPathsPreserveExactWhitespace(t *testing.T) {
	t.Run("config path", func(t *testing.T) {
		base := t.TempDir()
		spaced := filepath.Join(base, " config.json ")
		unspaced := filepath.Join(base, "config.json")

		got, err := resolveMigrationConfigPath(spaced)
		if err != nil {
			t.Fatalf("resolveMigrationConfigPath() error = %v", err)
		}
		if got != spaced {
			t.Fatalf("resolveMigrationConfigPath() = %q, want exact path %q (not sibling %q)", got, spaced, unspaced)
		}
	})

	t.Run("private key directory", func(t *testing.T) {
		base := t.TempDir()
		spaced := filepath.Join(base, " keys ")
		unspaced := filepath.Join(base, "keys")

		got, err := resolveMigrationPrivateKeyDir(spaced, filepath.Join(base, "config.json"))
		if err != nil {
			t.Fatalf("resolveMigrationPrivateKeyDir() error = %v", err)
		}
		if got != spaced {
			t.Fatalf("resolveMigrationPrivateKeyDir() = %q, want exact path %q (not sibling %q)", got, spaced, unspaced)
		}
	})
}

func TestResolveMigrationPathsTreatAllSpaceNamesAsPaths(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	work := t.TempDir()
	if err := os.Chdir(work); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore Chdir() error = %v", err)
		}
	})

	const allSpaceName = "   "
	want, err := filepath.Abs(allSpaceName)
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}

	configPath, err := resolveMigrationConfigPath(allSpaceName)
	if err != nil {
		t.Fatalf("resolveMigrationConfigPath() error = %v", err)
	}
	if configPath != want {
		t.Fatalf("resolveMigrationConfigPath() = %q, want %q", configPath, want)
	}

	keyDir, err := resolveMigrationPrivateKeyDir(allSpaceName, filepath.Join(work, "config.json"))
	if err != nil {
		t.Fatalf("resolveMigrationPrivateKeyDir() error = %v", err)
	}
	if keyDir != want {
		t.Fatalf("resolveMigrationPrivateKeyDir() = %q, want %q", keyDir, want)
	}
}

func TestStoredPrivateKeyPathPreservesExactWhitespace(t *testing.T) {
	base := t.TempDir()
	spaced := filepath.Join(base, " AuthKey.p8 ")
	unspaced := filepath.Join(base, "AuthKey.p8")
	if err := os.WriteFile(spaced, []byte("spaced-key"), 0o600); err != nil {
		t.Fatalf("WriteFile(spaced) error = %v", err)
	}
	if err := os.WriteFile(unspaced, []byte("wrong-sibling"), 0o600); err != nil {
		t.Fatalf("WriteFile(unspaced) error = %v", err)
	}

	pem, err := loadPrivateKeyPEMForStorage(spaced)
	if err != nil {
		t.Fatalf("loadPrivateKeyPEMForStorage() error = %v", err)
	}
	if pem != "spaced-key" {
		t.Fatalf("loadPrivateKeyPEMForStorage() = %q, want content from %q", pem, spaced)
	}

	got, exported, err := migrationPrivateKeyPath(Credential{
		Name:           "team",
		PrivateKeyPath: spaced,
	}, filepath.Join(base, "exports"), "team")
	if err != nil {
		t.Fatalf("migrationPrivateKeyPath() error = %v", err)
	}
	if got != spaced || exported {
		t.Fatalf("migrationPrivateKeyPath() = (%q, %t), want (%q, false)", got, exported, spaced)
	}
}

func TestStoredPrivateKeyPathAcceptsAllSpaceFilename(t *testing.T) {
	base := t.TempDir()
	allSpacePath := filepath.Join(base, "   ")
	if err := os.WriteFile(allSpacePath, []byte("space-key"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	pem, err := loadPrivateKeyPEMForStorage(allSpacePath)
	if err != nil {
		t.Fatalf("loadPrivateKeyPEMForStorage() error = %v", err)
	}
	if pem != "space-key" {
		t.Fatalf("loadPrivateKeyPEMForStorage() = %q, want %q", pem, "space-key")
	}

	got, exported, err := migrationPrivateKeyPath(Credential{
		Name:           "team",
		PrivateKeyPath: allSpacePath,
	}, filepath.Join(base, "exports"), "team")
	if err != nil {
		t.Fatalf("migrationPrivateKeyPath() error = %v", err)
	}
	if got != allSpacePath || exported {
		t.Fatalf("migrationPrivateKeyPath() = (%q, %t), want (%q, false)", got, exported, allSpacePath)
	}
}

func TestPrivateKeyExportRootPreservesExactWhitespace(t *testing.T) {
	base := t.TempDir()
	spacedKeyDir := filepath.Join(base, " keys ")
	unspacedKeyDir := filepath.Join(base, "keys")
	if err := os.MkdirAll(unspacedKeyDir, 0o700); err != nil {
		t.Fatalf("MkdirAll(unspaced) error = %v", err)
	}
	const name = "AuthKey_team.p8"

	if err := writePrivateKeyPEMFile(spacedKeyDir, name, "spaced-root-key"); err != nil {
		t.Fatalf("writePrivateKeyPEMFile() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(spacedKeyDir, name))
	if err != nil {
		t.Fatalf("ReadFile(spaced) error = %v", err)
	}
	if string(got) != "spaced-root-key" {
		t.Fatalf("spaced root key = %q, want %q", got, "spaced-root-key")
	}
	if _, err := os.Stat(filepath.Join(unspacedKeyDir, name)); !os.IsNotExist(err) {
		t.Fatalf("unspaced sibling Stat() error = %v, want not-exist", err)
	}

	allSpaceKeyDir := filepath.Join(base, "   ")
	if err := writePrivateKeyPEMFile(allSpaceKeyDir, name, "all-space-root-key"); err != nil {
		t.Fatalf("writePrivateKeyPEMFile(all-space root) error = %v", err)
	}
	got, err = os.ReadFile(filepath.Join(allSpaceKeyDir, name))
	if err != nil {
		t.Fatalf("ReadFile(all-space root) error = %v", err)
	}
	if string(got) != "all-space-root-key" {
		t.Fatalf("all-space root key = %q, want %q", got, "all-space-root-key")
	}
}
