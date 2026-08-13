package secureopen

import (
	"errors"
	"os"
	"testing"
)

// The platform rename implementations share name validation and error wrapping,
// but each implementation is only compiled on its own GOOS. These tests carry no
// build tag so the shared contract is exercised on every platform, including the
// ones whose rename syscall path a given CI runner cannot execute.

func TestValidateRenameNoReplaceNames(t *testing.T) {
	testCases := []struct {
		name      string
		oldName   string
		newName   string
		wantError bool
	}{
		{name: "valid names", oldName: ".staged-output", newName: "artifact.bin"},
		{name: "empty source", oldName: "", newName: "artifact.bin", wantError: true},
		{name: "empty destination", oldName: ".staged-output", newName: "", wantError: true},
		{name: "current directory", oldName: ".", newName: "artifact.bin", wantError: true},
		{name: "parent directory", oldName: ".staged-output", newName: "..", wantError: true},
		{name: "nested source", oldName: "nested/.staged-output", newName: "artifact.bin", wantError: true},
		{name: "nested destination", oldName: ".staged-output", newName: "nested/artifact.bin", wantError: true},
		{name: "trailing separator", oldName: ".staged-output", newName: "artifact.bin/", wantError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateRenameNoReplaceNames(testCase.oldName, testCase.newName)
			if testCase.wantError {
				if err == nil {
					t.Fatalf("validateRenameNoReplaceNames(%q, %q) = nil, want error", testCase.oldName, testCase.newName)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateRenameNoReplaceNames(%q, %q) error = %v", testCase.oldName, testCase.newName, err)
			}
		})
	}
}

func TestRenameNoReplaceErrorKeepsOperands(t *testing.T) {
	if err := renameNoReplaceError("rename", ".staged-output", "artifact.bin", nil); err != nil {
		t.Fatalf("renameNoReplaceError(nil) = %v, want nil", err)
	}

	cause := errors.New("simulated rename failure")
	err := renameNoReplaceError("rename", ".staged-output", "artifact.bin", cause)
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		t.Fatalf("renameNoReplaceError() = %v, want *os.LinkError", err)
	}
	if linkErr.Op != "rename" || linkErr.Old != ".staged-output" || linkErr.New != "artifact.bin" {
		t.Fatalf("link error = %+v, want rename .staged-output artifact.bin", linkErr)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("renameNoReplaceError() = %v, want wrapped cause", err)
	}
	if errors.Is(err, ErrRenameNoReplaceUnsupported) {
		t.Fatalf("renameNoReplaceError() = %v, want no unsupported marker", err)
	}
}

func TestUnsupportedRenameNoReplaceErrorMarksFallback(t *testing.T) {
	err := unsupportedRenameNoReplaceError("rename", ".staged-output", "artifact.bin", nil)
	if !errors.Is(err, ErrRenameNoReplaceUnsupported) {
		t.Fatalf("unsupportedRenameNoReplaceError(nil) = %v, want ErrRenameNoReplaceUnsupported", err)
	}

	cause := errors.New("simulated filesystem rejection")
	err = unsupportedRenameNoReplaceError("rename", ".staged-output", "artifact.bin", cause)
	if !errors.Is(err, ErrRenameNoReplaceUnsupported) {
		t.Fatalf("unsupportedRenameNoReplaceError() = %v, want ErrRenameNoReplaceUnsupported", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("unsupportedRenameNoReplaceError() = %v, want wrapped cause", err)
	}
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		t.Fatalf("unsupportedRenameNoReplaceError() = %v, want *os.LinkError", err)
	}
	if linkErr.Old != ".staged-output" || linkErr.New != "artifact.bin" {
		t.Fatalf("link error = %+v, want .staged-output artifact.bin", linkErr)
	}
}
