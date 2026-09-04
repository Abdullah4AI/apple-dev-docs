package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCertificateExportResultUsesCamelCaseJSONFields(t *testing.T) {
	csrMatched := true
	result := CertificateExportResult{
		Operation:         "certificates export",
		CertificatePath:   "certificate.cer",
		PrivateKeyPath:    "private.key",
		CSRPath:           "request.csr",
		P12Out:            "identity.p12",
		CertificateSHA256: "ABC123",
		NotBefore:         "2026-08-30T00:00:00Z",
		NotAfter:          "2027-08-30T00:00:00Z",
		KeyType:           "RSA",
		KeySize:           2048,
		PrivateKeyMatched: true,
		CSRMatched:        &csrMatched,
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	for _, field := range []string{
		`"certificatePath"`,
		`"privateKeyPath"`,
		`"csrPath"`,
		`"p12Out"`,
		`"certificateSha256"`,
		`"notBefore"`,
		`"notAfter"`,
		`"keyType"`,
		`"keySize"`,
		`"privateKeyMatched"`,
		`"csrMatched"`,
	} {
		if !strings.Contains(string(encoded), field) {
			t.Fatalf("JSON output missing %s: %s", field, encoded)
		}
	}
}

func TestCertificateExportResultRegisteredTableAndMarkdownOutput(t *testing.T) {
	result := &CertificateExportResult{
		Operation:         "certificates export",
		CertificatePath:   "certificate.cer",
		PrivateKeyPath:    "private.key",
		P12Out:            "identity.p12",
		CertificateSHA256: "ABC123",
		NotBefore:         "2026-08-30T00:00:00Z",
		NotAfter:          "2027-08-30T00:00:00Z",
		KeyType:           "RSA",
		KeySize:           2048,
		PrivateKeyMatched: true,
	}

	ensureOutputRegistryPopulated()
	if !isRegistryTypeRegistered(typeForPtr[CertificateExportResult]()) {
		t.Fatal("CertificateExportResult is not registered with the output renderer")
	}

	table := captureStdout(t, func() error { return PrintTable(result) })
	for _, want := range []string{"operation", "certificate_path", "private_key_path", "p12_out", "private_key_matched"} {
		if !strings.Contains(table, want) {
			t.Fatalf("table output missing %q: %s", want, table)
		}
	}

	markdown := captureStdout(t, func() error { return PrintMarkdown(result) })
	for _, want := range []string{"| operation", "| certificate_path", "| private_key_path", "| p12_out", "| private_key_matched"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("Markdown output missing %q: %s", want, markdown)
		}
	}
}
