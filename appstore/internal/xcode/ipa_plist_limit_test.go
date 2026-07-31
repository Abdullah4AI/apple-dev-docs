package xcode

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"howett.net/plist"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/infoplist"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/infoplist/infoplisttest"
)

func TestReadIPABundleInfoAtInfoPlistSizeLimit(t *testing.T) {
	ipaPath := writeIPAWithRawInfoPlist(t, buildSizedAppInfoPlist(t, infoplist.MaxBytes))

	info, err := readIPABundleInfo(ipaPath)
	if err != nil {
		t.Fatalf("readIPABundleInfo() error: %v", err)
	}
	if info.BundleID != "com.example.demo" || info.Version != "1.2.3" || info.BuildNumber != "42" {
		t.Fatalf("unexpected bundle info at size limit: %+v", info)
	}
	if info.Platform != "IOS" {
		t.Fatalf("expected platform IOS, got %q", info.Platform)
	}
}

func TestReadIPABundleInfoRejectsInfoPlistOneByteOverLimit(t *testing.T) {
	ipaPath := writeIPAWithRawInfoPlist(t, buildSizedAppInfoPlist(t, infoplist.MaxBytes+1))

	_, err := readIPABundleInfo(ipaPath)
	if err == nil {
		t.Fatal("expected oversized Info.plist rejection, got nil")
	}
	if !strings.Contains(err.Error(), "Info.plist limit") {
		t.Fatalf("expected Info.plist limit error, got %v", err)
	}
}

func TestReadIPABundleInfoRejectsHighlyCompressibleInfoPlist(t *testing.T) {
	ipaPath := writeIPAWithRawInfoPlist(t, bytes.Repeat([]byte("A"), infoplist.MaxBytes*2))

	_, err := readIPABundleInfo(ipaPath)
	if err == nil {
		t.Fatal("expected compressible oversized Info.plist rejection, got nil")
	}
	if !strings.Contains(err.Error(), "declared uncompressed size") {
		t.Fatalf("expected declared-size rejection, got %v", err)
	}
}

// A member that understates its uncompressed size must not slip past the
// declared-size check and then expand freely. archive/zip refuses a member as
// soon as it streams past its advertised size, and the bounded read is the
// backstop if it ever does not, so the caller always gets an error instead of
// an unbounded expansion.
func TestReadIPABundleInfoRejectsUnderstatedInfoPlistSize(t *testing.T) {
	ipaPath := writeIPAWithRawInfoPlist(t, bytes.Repeat([]byte("A"), infoplist.MaxBytes*2))
	infoplisttest.ForgeZipDeclaredUncompressedSize(t, ipaPath, 1024)

	if _, err := readIPABundleInfo(ipaPath); err == nil {
		t.Fatal("expected rejection for forged ZIP size metadata, got nil")
	}
}

func writeIPAWithRawInfoPlist(t *testing.T, plistData []byte) string {
	t.Helper()

	ipaPath := filepath.Join(t.TempDir(), "Demo.ipa")
	file, err := os.Create(ipaPath)
	if err != nil {
		t.Fatalf("create IPA: %v", err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	entry, err := writer.Create("Payload/Demo.app/Info.plist")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := entry.Write(plistData); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return ipaPath
}

// buildSizedAppInfoPlist returns a valid XML app Info.plist whose serialized
// length is exactly totalSize, padded with a filler key so boundary tests can
// hit the documented limit precisely.
func buildSizedAppInfoPlist(t *testing.T, totalSize int) []byte {
	t.Helper()

	padding := 0
	for range 8 {
		data, err := plist.Marshal(map[string]any{
			"CFBundleIdentifier":         "com.example.demo",
			"CFBundleShortVersionString": "1.2.3",
			"CFBundleVersion":            "42",
			"DTPlatformName":             "iphoneos",
			"Padding":                    strings.Repeat("a", padding),
		}, plist.XMLFormat)
		if err != nil {
			t.Fatalf("plist.Marshal() error: %v", err)
		}
		if len(data) == totalSize {
			return data
		}
		padding += totalSize - len(data)
		if padding < 0 {
			t.Fatalf("cannot build an Info.plist as small as %d bytes", totalSize)
		}
	}
	t.Fatalf("failed to build an Info.plist of exactly %d bytes", totalSize)
	return nil
}
