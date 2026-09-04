package secureopen

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTempNoFollowInRootUsesGeneratedNameWithDefaultCreator(t *testing.T) {
	directory := t.TempDir()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatalf("OpenRoot() error = %v", err)
	}
	defer root.Close()

	file, name, err := CreateTempNoFollowInRoot(root, ".", ".asc-test-*.tmp", 0o600)
	if err != nil {
		t.Fatalf("CreateTempNoFollowInRoot() error = %v", err)
	}
	defer file.Close()
	defer func() { _ = root.Remove(name) }()

	if name == "" || filepath.Base(name) != name {
		t.Fatalf("generated name = %q, want a root-relative basename", name)
	}
	if _, err := os.Stat(filepath.Join(directory, name)); err != nil {
		t.Fatalf("generated file is not visible beneath root: %v", err)
	}
}

func TestCreateTempNoFollowInRootCreatorReceivesGeneratedName(t *testing.T) {
	directory := t.TempDir()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatalf("OpenRoot() error = %v", err)
	}
	defer root.Close()

	called := false
	file, name, err := CreateTempNoFollowInRootWithCreator(root, ".", ".asc-test-*.tmp", 0o600, func(gotRoot *os.Root, gotName string, perm os.FileMode) (*os.File, error) {
		called = true
		if gotRoot != root {
			t.Fatalf("creator root = %p, want %p", gotRoot, root)
		}
		if gotName == "" || filepath.Base(gotName) != gotName {
			t.Fatalf("creator name = %q, want a generated basename", gotName)
		}
		return OpenNewFileNoFollowInRoot(gotRoot, gotName, perm)
	})
	if err != nil {
		t.Fatalf("CreateTempNoFollowInRootWithCreator() error = %v", err)
	}
	defer file.Close()
	defer func() { _ = root.Remove(name) }()

	if !called {
		t.Fatal("creator was not called")
	}
	if name == "" {
		t.Fatal("generated name is empty")
	}
}

func TestCreateTempNoFollowInRootClosesInvalidCreatorHandle(t *testing.T) {
	directory := t.TempDir()
	root, err := os.OpenRoot(directory)
	if err != nil {
		t.Fatalf("OpenRoot() error = %v", err)
	}
	defer root.Close()

	creatorErr := errors.New("creator returned an invalid handle")
	_, _, err = CreateTempNoFollowInRootWithCreator(root, ".", ".asc-test-*", 0o600, func(*os.Root, string, os.FileMode) (*os.File, error) {
		return nil, creatorErr
	})
	if !errors.Is(err, creatorErr) {
		t.Fatalf("CreateTempNoFollowInRootWithCreator() error = %v, want %v", err, creatorErr)
	}
}
