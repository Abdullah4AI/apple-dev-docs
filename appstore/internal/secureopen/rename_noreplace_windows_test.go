//go:build windows

package secureopen

import (
	"slices"
	"testing"
	"unicode/utf16"
	"unsafe"
)

func TestBuildFileRenameInformationUsesSameDirectoryNoReplace(t *testing.T) {
	const name = "artifact-界-🚀.bin"
	wantName := utf16.Encode([]rune(name))
	buffer, err := buildFileRenameInformation(name)
	if err != nil {
		t.Fatalf("buildFileRenameInformation() error = %v", err)
	}
	info := (*fileRenameInformation)(unsafe.Pointer(&buffer[0]))
	if info.ReplaceIfExists != 0 {
		t.Fatalf("ReplaceIfExists = %d, want 0", info.ReplaceIfExists)
	}
	if info.RootDirectory != 0 {
		t.Fatalf("RootDirectory = %v, want null for same-directory and SMB renames", info.RootDirectory)
	}
	wantNameLength := len(wantName) * 2
	if info.FileNameLength != uint32(wantNameLength) {
		t.Fatalf("FileNameLength = %d, want %d", info.FileNameLength, wantNameLength)
	}
	gotName := unsafe.Slice(&info.FileName[0], len(wantName))
	if !slices.Equal(gotName, wantName) {
		t.Fatalf("FileName = %v, want UTF-16 units %v", gotName, wantName)
	}
	var layout fileRenameInformation
	wantBufferLength := int(unsafe.Offsetof(layout.FileName)) + wantNameLength
	if len(buffer) != wantBufferLength {
		t.Fatalf("buffer length = %d, want %d", len(buffer), wantBufferLength)
	}
}
