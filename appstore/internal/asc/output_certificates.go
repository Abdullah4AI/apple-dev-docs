package asc

import (
	"fmt"
)

// CertificateExportResult is the metadata-only result for certificates export.
// It intentionally contains no certificate, CSR, private-key, or password data.
type CertificateExportResult struct {
	Operation         string `json:"operation"`
	CertificatePath   string `json:"certificatePath"`
	PrivateKeyPath    string `json:"privateKeyPath"`
	CSRPath           string `json:"csrPath,omitempty"`
	P12Out            string `json:"p12Out"`
	CertificateSHA256 string `json:"certificateSha256"`
	NotBefore         string `json:"notBefore"`
	NotAfter          string `json:"notAfter"`
	KeyType           string `json:"keyType"`
	KeySize           int    `json:"keySize"`
	PrivateKeyMatched bool   `json:"privateKeyMatched"`
	CSRMatched        *bool  `json:"csrMatched,omitempty"`
}

func certificateExportResultRows(result *CertificateExportResult) ([]string, [][]string) {
	rows := [][]string{
		{"operation", result.Operation},
		{"certificate_path", result.CertificatePath},
		{"private_key_path", result.PrivateKeyPath},
		{"p12_out", result.P12Out},
		{"certificate_sha256", result.CertificateSHA256},
		{"not_before", result.NotBefore},
		{"not_after", result.NotAfter},
		{"key_type", result.KeyType},
		{"key_size", fmt.Sprintf("%d", result.KeySize)},
		{"private_key_matched", fmt.Sprintf("%t", result.PrivateKeyMatched)},
	}
	if result.CSRPath != "" {
		rows = append(
			rows,
			[]string{"csr_path", result.CSRPath},
			[]string{"csr_matched", fmt.Sprintf("%t", result.CSRMatched != nil && *result.CSRMatched)},
		)
	}
	return []string{"field", "value"}, rows
}
