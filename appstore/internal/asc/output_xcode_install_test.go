package asc

import (
	"strings"
	"testing"
)

func TestXcodeInstallResultUsesRegisteredRows(t *testing.T) {
	ensureOutputRegistryPopulated()
	result := &XcodeInstallResult{
		SchemaVersion: 1, Operation: "xcode.install", Success: true, Installed: true, Verified: true,
		IPA:    XcodeInstallArtifact{BundleID: "com.example.demo", Version: "1.2.3", BuildNumber: "45", SizeBytes: 4, SHA256: strings.Repeat("c", 64)},
		Device: &XcodeInstallDevice{IdentifierSHA256: strings.Repeat("a", 64), Platform: "IOS", PairingState: "paired", ConnectionState: "connected"},
	}
	handled, ok := outputRegistry[typeForPtr[XcodeInstallResult]()]
	if !ok {
		t.Fatal("XcodeInstallResult is not registered")
	}
	headers, rows, err := handled(result)
	if err != nil {
		t.Fatalf("registered rows error = %v", err)
	}
	if len(headers) != 2 || len(rows) != 15 || !strings.Contains(strings.Join(rows[0], " "), "1") {
		t.Fatalf("headers/rows = %#v/%#v", headers, rows)
	}
}
