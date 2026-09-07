package asc

import "fmt"

// WebSessionExportResult is the receipt for `asc web auth export`. It
// deliberately reports only where the bundle went and what it covers; cookie
// values stay in the exported file.
type WebSessionExportResult struct {
	Path        string `json:"path"`
	AppleID     string `json:"appleId"`
	CookieCount int    `json:"cookieCount"`
	ExportedAt  string `json:"exportedAt"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
	Overwritten bool   `json:"overwritten"`
}

// WebSessionImportResult is the receipt for `asc web auth import`.
type WebSessionImportResult struct {
	Path                  string `json:"path"`
	AppleID               string `json:"appleId"`
	CookieCount           int    `json:"cookieCount"`
	SkippedExpiredCookies int    `json:"skippedExpiredCookies"`
	ExpiresAt             string `json:"expiresAt,omitempty"`
	Imported              bool   `json:"imported"`
}

func webSessionExportRows(result *WebSessionExportResult) ([]string, [][]string) {
	if result == nil {
		result = &WebSessionExportResult{}
	}
	return []string{"Path", "Apple ID", "Cookies", "Exported At", "Expires At", "Overwritten"}, [][]string{{
		result.Path,
		result.AppleID,
		fmt.Sprintf("%d", result.CookieCount),
		result.ExportedAt,
		result.ExpiresAt,
		fmt.Sprintf("%t", result.Overwritten),
	}}
}

func webSessionImportRows(result *WebSessionImportResult) ([]string, [][]string) {
	if result == nil {
		result = &WebSessionImportResult{}
	}
	return []string{"Path", "Apple ID", "Cookies", "Skipped Expired", "Expires At", "Imported"}, [][]string{{
		result.Path,
		result.AppleID,
		fmt.Sprintf("%d", result.CookieCount),
		fmt.Sprintf("%d", result.SkippedExpiredCookies),
		result.ExpiresAt,
		fmt.Sprintf("%t", result.Imported),
	}}
}
