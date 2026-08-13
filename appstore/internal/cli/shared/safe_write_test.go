package shared

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

func TestSafeWriteFileNoSymlinkRejectsTrailingSeparatorBeforeSideEffects(t *testing.T) {
	separators := []string{string(os.PathSeparator)}
	if os.PathSeparator != '/' {
		separators = append(separators, "/")
	}

	for _, separator := range separators {
		t.Run(strconv.Quote(separator), func(t *testing.T) {
			parent := t.TempDir()
			createdPath := filepath.Join(parent, "result")
			destination := createdPath + separator
			callbackCalled := false

			_, err := SafeWriteFileNoSymlink(
				destination,
				0o600,
				false,
				".safe-write-*",
				".safe-write-backup-*",
				func(file *os.File) (int64, error) {
					callbackCalled = true
					return 0, nil
				},
			)
			if err == nil || !strings.Contains(err.Error(), strconv.Quote(destination)) {
				t.Fatalf("SafeWriteFileNoSymlink() error = %v, want exact destination %s", err, strconv.Quote(destination))
			}
			if callbackCalled {
				t.Fatal("write callback was called")
			}
			if _, statErr := os.Lstat(createdPath); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("destination-shaped directory was created, stat error = %v", statErr)
			}
		})
	}
}

func TestSafeWriteFileNoSymlinkNoOverwriteRemovesPartialFileAfterCallbackFailure(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	writeErr := errors.New("simulated write failure")

	written, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			written, err := file.Write([]byte("partial"))
			if err != nil {
				return int64(written), err
			}
			return int64(written), writeErr
		},
	)
	if !errors.Is(err, writeErr) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want %v", err, writeErr)
	}
	if written != int64(len("partial")) {
		t.Fatalf("SafeWriteFileNoSymlink() written = %d, want %d", written, len("partial"))
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected partial destination to be removed, stat error = %v", err)
	}
}

func TestSafeWriteFileNoSymlinkNoOverwriteReportsDestinationForCallbackPathError(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	writeErr := errors.New("simulated write failure")

	_, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			return 0, &os.PathError{Op: "write", Path: file.Name(), Err: writeErr}
		},
	)
	if !errors.Is(err, writeErr) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want %v", err, writeErr)
	}
	if !strings.Contains(err.Error(), destination) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want destination %q", err, destination)
	}
	if strings.Contains(err.Error(), ".safe-write-") {
		t.Fatalf("SafeWriteFileNoSymlink() exposed temporary path: %v", err)
	}
}

func TestSafeWriteFileNoSymlinkNoOverwriteReportsDestinationForStagingCreationError(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	callbackCalled := false

	_, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		filepath.Join("missing", ".safe-write-*"),
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			callbackCalled = true
			return 0, nil
		},
	)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want os.ErrNotExist", err)
	}
	if !strings.Contains(err.Error(), destination) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want destination %q", err, destination)
	}
	if strings.Contains(err.Error(), ".safe-write-") {
		t.Fatalf("SafeWriteFileNoSymlink() exposed temporary path: %v", err)
	}
	if callbackCalled {
		t.Fatal("write callback was called")
	}
}

func TestSafeWriteFileNoSymlinkNoOverwritePreservesDestinationCreatedDuringWrite(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "artifact.bin")
	content := []byte("complete")
	concurrent := []byte("concurrent")

	written, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			written, err := file.Write(content)
			if err != nil {
				return int64(written), err
			}
			if err := os.WriteFile(destination, concurrent, 0o600); err != nil {
				return int64(written), err
			}
			return int64(written), nil
		},
	)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want os.ErrExist", err)
	}
	if written != int64(len(content)) {
		t.Fatalf("SafeWriteFileNoSymlink() written = %d, want %d", written, len(content))
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(concurrent) {
		t.Fatalf("destination content = %q, want %q", got, concurrent)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Fatalf("directory entries = %v, want only %q", entries, filepath.Base(destination))
	}
}

func TestSafeWriteFileNoSymlinkNoOverwritePreservesReplacementAfterCallbackFailure(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "artifact.bin")
	displaced := filepath.Join(directory, "displaced.bin")
	replacement := []byte("replacement")
	writeErr := errors.New("simulated write failure")

	_, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			written, err := file.Write([]byte("partial"))
			if err != nil {
				return int64(written), err
			}

			// Renaming an open file is supported on Unix. Windows may require the
			// handle to be closed first, so retry after closing it there.
			if err := os.Rename(destination, displaced); err != nil && !errors.Is(err, os.ErrNotExist) {
				if closeErr := file.Close(); closeErr != nil {
					return int64(written), errors.Join(err, closeErr)
				}
				if err := os.Rename(destination, displaced); err != nil {
					return int64(written), err
				}
			}
			if err := os.WriteFile(destination, replacement, 0o600); err != nil {
				return int64(written), err
			}
			return int64(written), writeErr
		},
	)
	if !errors.Is(err, writeErr) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want %v", err, writeErr)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(replacement) {
		t.Fatalf("destination content = %q, want %q", got, replacement)
	}
}

func TestCleanupIncompleteFilePreservesPrimaryAndCleanupErrors(t *testing.T) {
	primaryErr := errors.New("sync failure")
	closeErr := errors.New("close failure")
	removeErr := errors.New("remove failure")
	closeCalled := false
	removeCalled := false

	err := cleanupIncompleteFile(
		"artifact.bin",
		primaryErr,
		func() error {
			closeCalled = true
			return closeErr
		},
		func(path string) error {
			if !closeCalled {
				t.Fatal("remove called before close")
			}
			if path != "artifact.bin" {
				t.Fatalf("remove path = %q, want artifact.bin", path)
			}
			removeCalled = true
			return removeErr
		},
	)
	if !closeCalled {
		t.Fatal("close was not called")
	}
	if !removeCalled {
		t.Fatal("remove was not called")
	}
	for _, wantErr := range []error{primaryErr, closeErr, removeErr} {
		if !errors.Is(err, wantErr) {
			t.Fatalf("cleanupIncompleteFile() error = %v, want errors.Is(_, %v)", err, wantErr)
		}
	}
}

func TestWriteNewFileNoSymlinkRemovesFileAfterSyncFailure(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	file, err := OpenNewFileNoFollow(destination, 0o600)
	if err != nil {
		t.Fatalf("OpenNewFileNoFollow() error = %v", err)
	}
	syncErr := errors.New("simulated sync failure")

	_, err = writeNewFileNoSymlink(
		destination,
		file,
		func(file *os.File) (int64, error) {
			written, err := file.Write([]byte("complete"))
			return int64(written), err
		},
		newFileWriteOps{
			syncFile:   func() error { return syncErr },
			closeFile:  file.Close,
			removeFile: os.Remove,
		},
	)
	if !errors.Is(err, syncErr) {
		t.Fatalf("writeNewFileNoSymlink() error = %v, want %v", err, syncErr)
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected incomplete destination to be removed, stat error = %v", err)
	}
}

func TestWriteNewFileNoSymlinkRemovesFileAfterCloseFailure(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	file, err := OpenNewFileNoFollow(destination, 0o600)
	if err != nil {
		t.Fatalf("OpenNewFileNoFollow() error = %v", err)
	}
	closeErr := errors.New("simulated close failure")

	_, err = writeNewFileNoSymlink(
		destination,
		file,
		func(file *os.File) (int64, error) {
			written, err := file.Write([]byte("complete"))
			return int64(written), err
		},
		newFileWriteOps{
			syncFile: file.Sync,
			closeFile: func() error {
				if err := file.Close(); err != nil {
					return err
				}
				return closeErr
			},
			removeFile: os.Remove,
		},
	)
	if !errors.Is(err, closeErr) {
		t.Fatalf("writeNewFileNoSymlink() error = %v, want %v", err, closeErr)
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected incomplete destination to be removed, stat error = %v", err)
	}
}

func TestPublishStagedFileNoReplaceTreatsPostPublishCleanupAsBestEffort(t *testing.T) {
	cleanupErr := errors.New("simulated cleanup failure")
	linkCalled := false
	removeCalls := 0

	err := publishStagedFileNoReplace(
		".safe-write-staged",
		"artifact.bin",
		stagedFilePublishOps{
			renameFile: func(string, string) error {
				return secureopen.ErrRenameNoReplaceUnsupported
			},
			linkFile: func(oldName, newName string) error {
				linkCalled = true
				if oldName != ".safe-write-staged" || newName != "artifact.bin" {
					t.Fatalf("link names = %q, %q", oldName, newName)
				}
				return nil
			},
			removeFile: func(name string) error {
				removeCalls++
				if name != ".safe-write-staged" {
					t.Fatalf("remove name = %q", name)
				}
				if removeCalls == 1 {
					return cleanupErr
				}
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("publishStagedFileNoReplace() error = %v", err)
	}
	if !linkCalled {
		t.Fatal("link was not called")
	}
	if removeCalls != 2 {
		t.Fatalf("remove calls = %d, want immediate attempt and deferred retry", removeCalls)
	}
}

func TestPublishStagedFileNoReplaceUsesRenameFirst(t *testing.T) {
	renameCalled := false
	linkCalled := false
	removeCalled := false

	err := publishStagedFileNoReplace(
		".safe-write-staged",
		"artifact.bin",
		stagedFilePublishOps{
			renameFile: func(oldName, newName string) error {
				renameCalled = true
				if oldName != ".safe-write-staged" || newName != "artifact.bin" {
					t.Fatalf("rename names = %q, %q", oldName, newName)
				}
				return nil
			},
			linkFile: func(string, string) error {
				linkCalled = true
				return nil
			},
			removeFile: func(string) error {
				removeCalled = true
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("publishStagedFileNoReplace() error = %v", err)
	}
	if !renameCalled {
		t.Fatal("rename was not called")
	}
	if linkCalled {
		t.Fatal("link fallback was called after successful rename")
	}
	if removeCalled {
		t.Fatal("remove was called after rename consumed the staged path")
	}
}

func TestPublishStagedFileNoReplaceFallsBackOnlyWhenRenameIsUnsupported(t *testing.T) {
	linkCalled := false
	removeCalled := false

	err := publishStagedFileNoReplace(
		".safe-write-staged",
		"artifact.bin",
		stagedFilePublishOps{
			renameFile: func(string, string) error {
				return secureopen.ErrRenameNoReplaceUnsupported
			},
			linkFile: func(oldName, newName string) error {
				linkCalled = true
				if oldName != ".safe-write-staged" || newName != "artifact.bin" {
					t.Fatalf("link names = %q, %q", oldName, newName)
				}
				return nil
			},
			removeFile: func(name string) error {
				removeCalled = true
				if name != ".safe-write-staged" {
					t.Fatalf("remove name = %q", name)
				}
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("publishStagedFileNoReplace() error = %v", err)
	}
	if !linkCalled {
		t.Fatal("link fallback was not called")
	}
	if !removeCalled {
		t.Fatal("staged hard-link name was not removed")
	}
}

func TestPublishStagedFileNoReplaceDoesNotFallbackWhenDestinationExists(t *testing.T) {
	linkCalled := false
	removeCalled := false

	err := publishStagedFileNoReplace(
		".safe-write-staged",
		"artifact.bin",
		stagedFilePublishOps{
			renameFile: func(string, string) error {
				return os.ErrExist
			},
			linkFile: func(string, string) error {
				linkCalled = true
				return nil
			},
			removeFile: func(name string) error {
				removeCalled = true
				if name != ".safe-write-staged" {
					t.Fatalf("remove name = %q", name)
				}
				return nil
			},
		},
	)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("publishStagedFileNoReplace() error = %v, want os.ErrExist", err)
	}
	if linkCalled {
		t.Fatal("link fallback was called for an existing destination")
	}
	if !removeCalled {
		t.Fatal("staged path was not cleaned up")
	}
}

func TestPublishStagedFileNoReplaceCopiesWhenRenameAndLinkAreUnsupported(t *testing.T) {
	copyCalled := false
	removeCalled := false
	linkErr := fmt.Errorf("linkat: %w", syscall.ENOTSUP)

	err := publishStagedFileNoReplace(
		".safe-write-staged",
		"artifact.bin",
		stagedFilePublishOps{
			renameFile: func(string, string) error {
				return secureopen.ErrRenameNoReplaceUnsupported
			},
			linkFile: func(string, string) error {
				return linkErr
			},
			copyFile: func(oldName, newName string) error {
				copyCalled = true
				if oldName != ".safe-write-staged" || newName != "artifact.bin" {
					t.Fatalf("copy names = %q, %q", oldName, newName)
				}
				return nil
			},
			removeFile: func(name string) error {
				removeCalled = true
				if name != ".safe-write-staged" {
					t.Fatalf("remove name = %q", name)
				}
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("publishStagedFileNoReplace() error = %v", err)
	}
	if !copyCalled {
		t.Fatal("copy fallback was not called")
	}
	if !removeCalled {
		t.Fatal("staged file was not removed after the copy fallback")
	}
}

func TestPublishStagedFileNoReplaceDoesNotCopyWhenLinkFindsExistingDestination(t *testing.T) {
	copyCalled := false
	removeCalled := false

	err := publishStagedFileNoReplace(
		".safe-write-staged",
		"artifact.bin",
		stagedFilePublishOps{
			renameFile: func(string, string) error {
				return secureopen.ErrRenameNoReplaceUnsupported
			},
			linkFile: func(string, string) error {
				return os.ErrExist
			},
			copyFile: func(string, string) error {
				copyCalled = true
				return nil
			},
			removeFile: func(string) error {
				removeCalled = true
				return nil
			},
		},
	)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("publishStagedFileNoReplace() error = %v, want os.ErrExist", err)
	}
	if copyCalled {
		t.Fatal("copy fallback was called for an existing destination")
	}
	if !removeCalled {
		t.Fatal("staged path was not cleaned up")
	}
}

func TestPublishStagedFileNoReplaceReportsCopyFailure(t *testing.T) {
	copyErr := errors.New("simulated copy failure")
	removeCalled := false

	err := publishStagedFileNoReplace(
		".safe-write-staged",
		"artifact.bin",
		stagedFilePublishOps{
			renameFile: func(string, string) error {
				return secureopen.ErrRenameNoReplaceUnsupported
			},
			linkFile: func(string, string) error {
				return syscall.ENOTSUP
			},
			copyFile: func(string, string) error {
				return copyErr
			},
			removeFile: func(string) error {
				removeCalled = true
				return nil
			},
		},
	)
	if !errors.Is(err, copyErr) {
		t.Fatalf("publishStagedFileNoReplace() error = %v, want %v", err, copyErr)
	}
	if !removeCalled {
		t.Fatal("staged path was not cleaned up after the copy failure")
	}
}

func TestSafeWriteFileNoSymlinkNoOverwritePublishesWithoutRenameOrLinkSupport(t *testing.T) {
	stubUnsupportedPublishPrimitives(t)

	directory := t.TempDir()
	destination := filepath.Join(directory, "artifact.bin")
	content := []byte("complete")

	written, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			written, err := file.Write(content)
			return int64(written), err
		},
	)
	if err != nil {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v", err)
	}
	if written != int64(len(content)) {
		t.Fatalf("SafeWriteFileNoSymlink() written = %d, want %d", written, len(content))
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("destination content = %q, want %q", got, content)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Fatalf("directory entries = %v, want only %q", entries, filepath.Base(destination))
	}
}

func TestSafeWriteFileNoSymlinkNoOverwriteCopyFallbackPreservesExistingDestination(t *testing.T) {
	stubUnsupportedPublishPrimitives(t)

	directory := t.TempDir()
	destination := filepath.Join(directory, "artifact.bin")
	concurrent := []byte("concurrent")

	_, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			written, err := file.Write([]byte("complete"))
			if err != nil {
				return int64(written), err
			}
			if err := os.WriteFile(destination, concurrent, 0o600); err != nil {
				return int64(written), err
			}
			return int64(written), nil
		},
	)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want os.ErrExist", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(concurrent) {
		t.Fatalf("destination content = %q, want %q", got, concurrent)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Fatalf("directory entries = %v, want only %q", entries, filepath.Base(destination))
	}
}

func TestSafeWriteFileNoSymlinkNoOverwriteReportsPublishFailureAgainstDestination(t *testing.T) {
	previousRename := renameNoReplaceInRoot
	t.Cleanup(func() {
		renameNoReplaceInRoot = previousRename
	})
	renameNoReplaceInRoot = func(_ *os.Root, oldName, newName string) error {
		return &os.LinkError{Op: "renameat2", Old: oldName, New: newName, Err: syscall.EIO}
	}

	destination := filepath.Join(t.TempDir(), "artifact.bin")

	_, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			written, err := file.Write([]byte("complete"))
			return int64(written), err
		},
	)
	if !errors.Is(err, syscall.EIO) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want syscall.EIO", err)
	}
	message := err.Error()
	if !strings.Contains(message, fmt.Sprintf("%q", destination)) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want destination %q", err, destination)
	}
	if count := strings.Count(message, filepath.Base(destination)); count != 1 {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want destination named once, got %d occurrences", err, count)
	}
	if strings.Contains(message, "renameat2") {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, exposed the publication syscall", err)
	}
	if strings.Contains(message, ".safe-write-") {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, exposed the staged path", err)
	}
}

func TestSafeWriteFileNoSymlinkNoOverwriteReportsExistingDestinationAgainstOutputPath(t *testing.T) {
	t.Run("before staging", func(t *testing.T) {
		destination := filepath.Join(t.TempDir(), "artifact.bin")
		if err := os.WriteFile(destination, []byte("original"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		_, err := SafeWriteFileNoSymlink(
			destination,
			0o600,
			false,
			".safe-write-*",
			".safe-write-backup-*",
			func(file *os.File) (int64, error) {
				t.Fatal("write callback was called for an existing destination")
				return 0, nil
			},
		)
		assertExistingDestinationError(t, err, destination)
	})

	t.Run("during publication", func(t *testing.T) {
		destination := filepath.Join(t.TempDir(), "artifact.bin")

		_, err := SafeWriteFileNoSymlink(
			destination,
			0o600,
			false,
			".safe-write-*",
			".safe-write-backup-*",
			func(file *os.File) (int64, error) {
				written, err := file.Write([]byte("complete"))
				if err != nil {
					return int64(written), err
				}
				if err := os.WriteFile(destination, []byte("concurrent"), 0o600); err != nil {
					return int64(written), err
				}
				return int64(written), nil
			},
		)
		assertExistingDestinationError(t, err, destination)
	})
}

func assertExistingDestinationError(t *testing.T, err error, destination string) {
	t.Helper()

	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want os.ErrExist", err)
	}
	message := err.Error()
	if !strings.Contains(message, "output file already exists") {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want existing-output message", err)
	}
	if !strings.Contains(message, fmt.Sprintf("%q", destination)) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want destination %q", err, destination)
	}
	if count := strings.Count(message, filepath.Base(destination)); count != 1 {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want destination named once, got %d occurrences", err, count)
	}
	if strings.Contains(message, ".safe-write-") {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, exposed the staged path", err)
	}
}

// stubUnsupportedPublishPrimitives simulates a destination filesystem such as
// FAT/exFAT, SMB, or FUSE that implements neither an atomic no-replace rename
// nor hard links.
func stubUnsupportedPublishPrimitives(t *testing.T) {
	t.Helper()

	previousRename := renameNoReplaceInRoot
	previousLink := linkInRoot
	t.Cleanup(func() {
		renameNoReplaceInRoot = previousRename
		linkInRoot = previousLink
	})

	renameNoReplaceInRoot = func(_ *os.Root, oldName, newName string) error {
		return &os.LinkError{Op: "renameat2", Old: oldName, New: newName, Err: errors.Join(secureopen.ErrRenameNoReplaceUnsupported, syscall.EINVAL)}
	}
	linkInRoot = func(_ *os.Root, oldName, newName string) error {
		return &os.LinkError{Op: "linkat", Old: oldName, New: newName, Err: syscall.ENOTSUP}
	}
}

func TestSafeWriteFileNoSymlinkNoOverwriteCreatesFileOnSuccess(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	content := []byte("complete")

	written, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			written, err := file.Write(content)
			return int64(written), err
		},
	)
	if err != nil {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v", err)
	}
	if written != int64(len(content)) {
		t.Fatalf("SafeWriteFileNoSymlink() written = %d, want %d", written, len(content))
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("destination content = %q, want %q", got, content)
	}
}

func TestSafeWriteFileNoSymlinkNoOverwritePreservesExistingFile(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "artifact.bin")
	original := []byte("original")
	if err := os.WriteFile(destination, original, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	callbackCalled := false
	_, err := SafeWriteFileNoSymlink(
		destination,
		0o600,
		false,
		".safe-write-*",
		".safe-write-backup-*",
		func(file *os.File) (int64, error) {
			callbackCalled = true
			written, err := file.Write([]byte("replacement"))
			return int64(written), err
		},
	)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("SafeWriteFileNoSymlink() error = %v, want os.ErrExist", err)
	}
	if callbackCalled {
		t.Fatal("write callback was called for an existing destination")
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("destination content = %q, want %q", got, original)
	}
}
