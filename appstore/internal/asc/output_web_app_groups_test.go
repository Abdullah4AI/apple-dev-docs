package asc

import (
	"strings"
	"testing"
)

func TestPrintTableWebAppGroupReceiptsUseRegistry(t *testing.T) {
	tests := []struct {
		name   string
		result any
		want   []string
	}{
		{
			name:   "delete",
			result: &WebAppGroupDeleteResult{GroupID: "GROUP1", Name: "Shared", Identifier: "group.com.example.shared", Deleted: true, Status: "deleted"},
			want:   []string{"Group ID", "Name", "Identifier", "Deleted", "Status", "GROUP1", "Shared", "group.com.example.shared", "true", "deleted"},
		},
		{
			name:   "unassign",
			result: &WebAppGroupUnassignResult{BundleID: "bundle-1", GroupID: "GROUP1", RemainingGroupIDs: []string{"GROUP2", "GROUP3"}, Changed: true, Status: "unassigned"},
			want:   []string{"Bundle ID", "Group ID", "Remaining Group IDs", "Changed", "Status", "bundle-1", "GROUP1", "GROUP2, GROUP3", "true", "unassigned"},
		},
		{
			name:   "unassign with no remaining groups",
			result: &WebAppGroupUnassignResult{BundleID: "bundle-1", GroupID: "GROUP1", RemainingGroupIDs: []string{}, Changed: true, Status: "unassigned"},
			want:   []string{"bundle-1", "GROUP1", "-", "unassigned"},
		},
		{
			name:   "set",
			result: &WebAppGroupSetResult{BundleID: "bundle-1", GroupIDs: []string{"GROUP2", "GROUP3"}, Added: []string{"GROUP3"}, Removed: []string{"GROUP1"}, Changed: true, Status: "updated"},
			want:   []string{"Bundle ID", "Group IDs", "Added", "Removed", "Changed", "Status", "bundle-1", "GROUP2, GROUP3", "GROUP3", "GROUP1", "true", "updated"},
		},
		{
			name:   "set unchanged",
			result: &WebAppGroupSetResult{BundleID: "bundle-1", GroupIDs: []string{"GROUP2"}, Added: []string{}, Removed: []string{}, Changed: false, Status: "unchanged"},
			want:   []string{"bundle-1", "GROUP2", "-", "false", "unchanged"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			table := captureStdout(t, func() error { return PrintTable(test.result) })
			for _, want := range test.want {
				if !strings.Contains(table, want) {
					t.Fatalf("table output missing %q: %q", want, table)
				}
			}
			markdown := captureStdout(t, func() error { return PrintMarkdown(test.result) })
			for _, want := range test.want {
				if !strings.Contains(markdown, want) {
					t.Fatalf("markdown output missing %q: %q", want, markdown)
				}
			}
		})
	}
}
