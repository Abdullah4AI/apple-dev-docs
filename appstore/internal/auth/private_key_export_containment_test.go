package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMigrationPrivateKeyFilenameSanitizesPathBearingKeyID(t *testing.T) {
	name := migrationPrivateKeyFilename("team", "../../escape")
	if strings.ContainsAny(name, `/\`) {
		t.Fatalf("migrationPrivateKeyFilename() = %q, want a single path component", name)
	}
	if strings.Contains(name, "..") {
		t.Fatalf("migrationPrivateKeyFilename() = %q, want no parent traversal", name)
	}
}

func TestWritePrivateKeyPEMFileRejectsPathBearingKeyID(t *testing.T) {
	keyDir := filepath.Join(t.TempDir(), "keys")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	err := writePrivateKeyPEMFile(keyDir, "../escaped.p8", "PRIVATE KEY MATERIAL")
	if err == nil {
		t.Fatal("writePrivateKeyPEMFile() error = nil, want traversal rejection")
	}
	escaped := filepath.Join(filepath.Dir(keyDir), "escaped.p8")
	if _, statErr := os.Stat(escaped); statErr == nil {
		t.Fatalf("private key material was written outside the key directory: %s", escaped)
	}
}

func TestWritePrivateKeyPEMFileRejectsSymlinkedParentDirectory(t *testing.T) {
	keyDir := filepath.Join(t.TempDir(), "keys")
	external := t.TempDir()
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Symlink(external, filepath.Join(keyDir, "nested")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	err := writePrivateKeyPEMFile(keyDir, filepath.Join("nested", "AuthKey.p8"), "PRIVATE KEY MATERIAL")
	if err == nil {
		t.Fatal("writePrivateKeyPEMFile() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("writePrivateKeyPEMFile() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(external, "AuthKey.p8")); statErr == nil {
		t.Fatal("private key material escaped through a symlinked parent")
	}
}

func TestWritePrivateKeyPEMFileRejectsSymlinkedDestination(t *testing.T) {
	keyDir := filepath.Join(t.TempDir(), "keys")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	sentinelPath := filepath.Join(t.TempDir(), "sentinel.txt")
	if err := os.WriteFile(sentinelPath, []byte("original"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Symlink(sentinelPath, filepath.Join(keyDir, "AuthKey.p8")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	err := writePrivateKeyPEMFile(keyDir, "AuthKey.p8", "PRIVATE KEY MATERIAL")
	if err == nil {
		t.Fatal("writePrivateKeyPEMFile() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("writePrivateKeyPEMFile() error = %v, want symlink rejection", err)
	}
	data, readErr := os.ReadFile(sentinelPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(data) != "original" {
		t.Fatalf("sentinel content = %q, want %q", data, "original")
	}
}

func TestWritePrivateKeyPEMFileTightensLoosePermissionsWithoutFollowingSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not reported faithfully on Windows")
	}
	keyDir := filepath.Join(t.TempDir(), "keys")
	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	path := filepath.Join(keyDir, "AuthKey_team_ABC123.p8")
	if err := os.WriteFile(path, []byte("PRIVATE KEY MATERIAL"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := writePrivateKeyPEMFile(keyDir, "AuthKey_team_ABC123.p8", "PRIVATE KEY MATERIAL"); err != nil {
		t.Fatalf("writePrivateKeyPEMFile() error = %v", err)
	}

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key mode = %v, want tightened to 0600", info.Mode().Perm())
	}
}

func TestWritePrivateKeyPEMFileWritesNewKeyWithSecureMode(t *testing.T) {
	keyDir := filepath.Join(t.TempDir(), "keys")

	if err := writePrivateKeyPEMFile(keyDir, "AuthKey_team_ABC123.p8", "PRIVATE KEY MATERIAL"); err != nil {
		t.Fatalf("writePrivateKeyPEMFile() error = %v", err)
	}

	path := filepath.Join(keyDir, "AuthKey_team_ABC123.p8")
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key mode = %v, want 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "PRIVATE KEY MATERIAL" {
		t.Fatalf("key content = %q", data)
	}

	// Re-exporting identical material is idempotent.
	if err := writePrivateKeyPEMFile(keyDir, "AuthKey_team_ABC123.p8", "PRIVATE KEY MATERIAL"); err != nil {
		t.Fatalf("writePrivateKeyPEMFile() rerun error = %v", err)
	}

	// Different material must never silently replace an existing key.
	if err := writePrivateKeyPEMFile(keyDir, "AuthKey_team_ABC123.p8", "OTHER MATERIAL"); err == nil {
		t.Fatal("writePrivateKeyPEMFile() error = nil, want refusal to overwrite existing key")
	}
}
