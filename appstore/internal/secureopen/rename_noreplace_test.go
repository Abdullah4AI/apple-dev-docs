//go:build darwin || linux || windows

package secureopen

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRenameNoReplaceInRootPublishesStagedFile(t *testing.T) {
	directory := t.TempDir()
	stagedName := ".staged-output"
	destinationName := "artifact.bin"
	if err := os.WriteFile(filepath.Join(directory, stagedName), []byte("complete"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatalf("OpenRoot() error = %v", err)
	}
	defer root.Close()

	if err := RenameNoReplaceInRoot(root, stagedName, destinationName); err != nil {
		t.Fatalf("RenameNoReplaceInRoot() error = %v", err)
	}
	if _, err := root.Lstat(stagedName); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staged path still exists, Lstat() error = %v", err)
	}
	got, err := root.ReadFile(destinationName)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != "complete" {
		t.Fatalf("destination content = %q, want complete", got)
	}
}

func TestRenameNoReplaceInRootPreservesExistingDestination(t *testing.T) {
	directory := t.TempDir()
	stagedName := ".staged-output"
	destinationName := "artifact.bin"
	if err := os.WriteFile(filepath.Join(directory, stagedName), []byte("complete"), 0o600); err != nil {
		t.Fatalf("WriteFile(staged) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, destinationName), []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile(destination) error = %v", err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatalf("OpenRoot() error = %v", err)
	}
	defer root.Close()

	err = RenameNoReplaceInRoot(root, stagedName, destinationName)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("RenameNoReplaceInRoot() error = %v, want os.ErrExist", err)
	}
	got, err := root.ReadFile(destinationName)
	if err != nil {
		t.Fatalf("ReadFile(destination) error = %v", err)
	}
	if string(got) != "existing" {
		t.Fatalf("destination content = %q, want existing", got)
	}
	got, err = root.ReadFile(stagedName)
	if err != nil {
		t.Fatalf("ReadFile(staged) error = %v", err)
	}
	if string(got) != "complete" {
		t.Fatalf("staged content = %q, want complete", got)
	}
}

func TestRenameNoReplaceInRootPublishesDirectory(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := root.Mkdir("staged", 0o700); err != nil {
		t.Fatal(err)
	}
	staged, err := root.OpenRoot("staged")
	if err != nil {
		t.Fatal(err)
	}
	file, err := OpenNewFileNoFollowInRoot(staged, "bundle.json", 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("complete")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := staged.Close(); err != nil {
		t.Fatal(err)
	}
	if err := RenameNoReplaceInRoot(root, "staged", "published"); err != nil {
		t.Fatalf("RenameNoReplaceInRoot() directory error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "published", "bundle.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "complete" {
		t.Fatalf("published data = %q", data)
	}
}
