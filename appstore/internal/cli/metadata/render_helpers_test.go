package metadata

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestBuildPlanRowsNoChangesPlaceholder(t *testing.T) {
	rows := buildPlanRows(PushPlanResult{})
	if len(rows) != 1 {
		t.Fatalf("expected one placeholder row, got %d", len(rows))
	}
	if rows[0][0] != "none" || rows[0][6] != "no changes" {
		t.Fatalf("unexpected placeholder row: %v", rows[0])
	}
}

func TestSanitizePlanCellNormalizesAndTruncates(t *testing.T) {
	input := "line1\n" + strings.Repeat("x", 120)
	got := sanitizePlanCell(input)

	if strings.Contains(got, "\n") {
		t.Fatalf("expected no literal newlines, got %q", got)
	}
	if !strings.Contains(got, "\\n") {
		t.Fatalf("expected escaped newline marker, got %q", got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated output to end with ellipsis, got %q", got)
	}
}

func TestBuildAPICallSummarySorted(t *testing.T) {
	summary := buildAPICallSummary(
		scopeCallCounts{create: 1, update: 2},
		scopeCallCounts{create: 1, delete: 1},
	)
	if len(summary) != 4 {
		t.Fatalf("expected 4 summary items, got %d", len(summary))
	}

	for i := 1; i < len(summary); i++ {
		prev := summary[i-1]
		curr := summary[i]
		if prev.Scope > curr.Scope {
			t.Fatalf("summary not sorted by scope: %v", summary)
		}
		if prev.Scope == curr.Scope && prev.Operation > curr.Operation {
			t.Fatalf("summary not sorted by operation within scope: %v", summary)
		}
	}
}

func TestRenderHelpersProduceOutput(t *testing.T) {
	pushResult := PushPlanResult{
		AppID:    "app-1",
		Version:  "1.2.3",
		Dir:      "/tmp/meta",
		DryRun:   true,
		Applied:  true,
		Adds:     []PlanItem{{Key: "app-info:en-US:name", Scope: appInfoDirName, Locale: "en-US", Field: "name", Reason: "add", To: "Local"}},
		Updates:  []PlanItem{{Key: "version:1.2.3:en-US:description", Scope: versionDirName, Locale: "en-US", Version: "1.2.3", Field: "description", Reason: "update", From: "Old", To: "New"}},
		Deletes:  []PlanItem{{Key: "version:1.2.3:fr:keywords", Scope: versionDirName, Locale: "fr", Version: "1.2.3", Field: "keywords", Reason: "delete", From: "remote"}},
		APICalls: []PlanAPICall{{Operation: "update_localization", Scope: appInfoDirName, Count: 1}},
		Actions:  []ApplyAction{{Scope: appInfoDirName, Locale: "en-US", Action: "update", LocalizationID: "loc-1"}},
	}
	validateResult := ValidateResult{
		Dir:          "/tmp/meta",
		FilesScanned: 2,
		Issues: []ValidateIssue{
			{Scope: appInfoDirName, File: "/tmp/meta/app-info/en-US.json", Locale: "en-US", Field: "name", Severity: issueSeverityError, Message: "name is required"},
		},
		ErrorCount: 1,
		Valid:      false,
	}
	pullResult := PullResult{
		AppID:     "app-1",
		Version:   "1.2.3",
		Dir:       "/tmp/meta",
		Includes:  []string{"localizations"},
		FileCount: 1,
		Files:     []string{"/tmp/meta/app-info/en-US.json"},
	}

	tableOut := captureStdout(t, func() {
		if err := printPullResultTable(pullResult); err != nil {
			t.Fatalf("printPullResultTable() error: %v", err)
		}
		if err := printPushPlanTable(pushResult); err != nil {
			t.Fatalf("printPushPlanTable() error: %v", err)
		}
		if err := printValidateResultTable(validateResult); err != nil {
			t.Fatalf("printValidateResultTable() error: %v", err)
		}
	})
	if !strings.Contains(tableOut, "Dry Run: true") || !strings.Contains(tableOut, "Files Scanned: 2") {
		t.Fatalf("expected key table render text, got %q", tableOut)
	}

	markdownOut := captureStdout(t, func() {
		if err := printPullResultMarkdown(pullResult); err != nil {
			t.Fatalf("printPullResultMarkdown() error: %v", err)
		}
		if err := printPushPlanMarkdown(pushResult); err != nil {
			t.Fatalf("printPushPlanMarkdown() error: %v", err)
		}
		if err := printValidateResultMarkdown(validateResult); err != nil {
			t.Fatalf("printValidateResultMarkdown() error: %v", err)
		}
	})
	if !strings.Contains(markdownOut, "**Dry Run:** true") || !strings.Contains(markdownOut, "**Files Scanned:** 2") {
		t.Fatalf("expected key markdown render text, got %q", markdownOut)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}
	os.Stdout = writePipe

	fn()

	if err := writePipe.Close(); err != nil {
		t.Fatalf("close write pipe: %v", err)
	}
	os.Stdout = oldStdout

	data, err := io.ReadAll(readPipe)
	if err != nil {
		t.Fatalf("read stdout pipe: %v", err)
	}
	return string(data)
}
