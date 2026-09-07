package asc

import "fmt"

// WebTransactionTaxDownloadResult is the local receipt for a downloaded
// Transaction Tax Report archive. It intentionally excludes provider/account
// identifiers, generated job IDs, signed URLs, and finance values.
type WebTransactionTaxDownloadResult struct {
	Date         string `json:"date"`
	Path         string `json:"path"`
	BytesWritten int64  `json:"bytesWritten"`
	ContentType  string `json:"contentType,omitempty"`
	PollStatus   string `json:"pollStatus"`
	Verified     bool   `json:"verified"`
}

func webTransactionTaxDownloadRows(result *WebTransactionTaxDownloadResult) ([]string, [][]string) {
	headers := []string{"Date", "Path", "Bytes", "Content Type", "Poll Status", "Verified"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.Date,
		result.Path,
		fmt.Sprintf("%d", result.BytesWritten),
		result.ContentType,
		result.PollStatus,
		fmt.Sprintf("%t", result.Verified),
	}}
}
