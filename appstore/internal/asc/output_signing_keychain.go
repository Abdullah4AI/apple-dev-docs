package asc

// SigningKeychainInstallResult is the public, non-secret receipt for a
// persistent signing keychain installation.
type SigningKeychainInstallResult struct {
	Action            string `json:"action"`
	KeychainPath      string `json:"keychainPath"`
	CertificateSHA256 string `json:"certificateSha256"`
	CertificateSHA1   string `json:"certificateSha1"`
	TeamID            string `json:"teamId"`
	SearchListUpdated bool   `json:"searchListUpdated"`
}

func signingKeychainInstallRows(result *SigningKeychainInstallResult) ([]string, [][]string) {
	headers := []string{"Action", "Keychain Path", "Certificate SHA-256", "Certificate SHA-1", "Team ID", "Search List Updated"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.Action,
		result.KeychainPath,
		result.CertificateSHA256,
		result.CertificateSHA1,
		result.TeamID,
		formatBool(result.SearchListUpdated),
	}}
}
