package asc

import "fmt"

// WebSandboxDeleteResult is the receipt for verified private web sandbox
// tester deletions.
type WebSandboxDeleteResult struct {
	IDs     []string `json:"ids"`
	Deleted bool     `json:"deleted"`
}

func webSandboxDeleteResultRows(result *WebSandboxDeleteResult) ([]string, [][]string) {
	headers := []string{"ID", "Deleted"}
	if result == nil {
		return headers, nil
	}
	rows := make([][]string, 0, len(result.IDs))
	for _, id := range result.IDs {
		rows = append(rows, []string{id, fmt.Sprintf("%t", result.Deleted)})
	}
	return headers, rows
}
