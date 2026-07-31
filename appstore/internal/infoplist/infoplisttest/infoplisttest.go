// Package infoplisttest provides helpers for tests that exercise the
// Info.plist expansion limits against hand-forged ZIP metadata.
package infoplisttest

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

// ForgeZipDeclaredUncompressedSize rewrites the uncompressed-size field of the
// archive's single central-directory record so the archive understates how
// much data the member expands to.
func ForgeZipDeclaredUncompressedSize(t testing.TB, ipaPath string, declaredSize uint32) {
	t.Helper()

	data, err := os.ReadFile(ipaPath)
	if err != nil {
		t.Fatalf("read IPA: %v", err)
	}
	const centralDirectorySignature = "PK\x01\x02"
	index := bytes.Index(data, []byte(centralDirectorySignature))
	if index < 0 {
		t.Fatal("central directory header not found")
	}
	if bytes.Contains(data[index+len(centralDirectorySignature):], []byte(centralDirectorySignature)) {
		t.Fatal("expected exactly one central directory header")
	}
	binary.LittleEndian.PutUint32(data[index+24:index+28], declaredSize)
	if err := os.WriteFile(ipaPath, data, 0o600); err != nil {
		t.Fatalf("write IPA: %v", err)
	}
}
