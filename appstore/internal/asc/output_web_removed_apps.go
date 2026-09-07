package asc

import "fmt"

// WebRemovedAppRestoreResult is the restore mutation receipt.
type WebRemovedAppRestoreResult struct {
	AppID             string `json:"appId"`
	Access            string `json:"access"`
	Removed           bool   `json:"removed"`
	PermissionWritten bool   `json:"permissionWritten"`
}

func webRemovedAppRestoreRows(r *WebRemovedAppRestoreResult) ([]string, [][]string) {
	return []string{"App ID", "Access", "Removed", "Permission Written"}, [][]string{{r.AppID, r.Access, fmt.Sprint(r.Removed), fmt.Sprint(r.PermissionWritten)}}
}
