package asc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestBetaTesterUsagesPageTablesRendersBothDimensionShapes(t *testing.T) {
	tests := []struct {
		name string
		row  string
	}{
		{
			name: "string form",
			row:  `{"dataPoints":[{"start":"2026-08-01T00:00:00Z","end":"2026-08-02T00:00:00Z","values":{"sessionCount":12,"crashCount":1,"feedbackCount":2}}],"dimensions":{"betaTesters":{"data":"tester-1"}}}`,
		},
		{
			name: "object form",
			row:  `{"dataPoints":[{"start":"2026-08-01T00:00:00Z","end":"2026-08-02T00:00:00Z","values":{"sessionCount":12,"crashCount":1,"feedbackCount":2}}],"dimensions":{"betaTesters":{"data":{"type":"betaTesters","id":"tester-1"}}}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page := &BetaTesterUsagesPage{Data: []json.RawMessage{json.RawMessage(test.row)}}
			var headers []string
			var rows [][]string
			if err := betaTesterUsagesPageTables(page, func(h []string, r [][]string) {
				headers = h
				rows = r
			}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertSingleRowEquals(
				t, headers, rows,
				[]string{"Tester ID", "Start", "End", "Sessions", "Crashes", "Feedback"},
				[]string{"tester-1", "2026-08-01T00:00:00Z", "2026-08-02T00:00:00Z", "12", "1", "2"},
			)
		})
	}
}

func TestBetaTesterUsagesPageTablesRejectsMalformedDimensions(t *testing.T) {
	page := &BetaTesterUsagesPage{
		Data: []json.RawMessage{json.RawMessage(`{"dataPoints":[],"dimensions":{"betaTesters":{"data":42}}}`)},
	}
	err := betaTesterUsagesPageTables(page, func([]string, [][]string) {})
	if err == nil {
		t.Fatal("expected error for malformed dimensions")
	}
}

// TestBetaTestersImportSummaryJSONByteCompat pins the exported
// BetaTestersImportSummary JSON to the exact bytes previously produced by the
// unexported testflight.betaTestersImportSummary struct, proving the move to
// internal/asc changed code organization only, not output.
func TestBetaTestersImportSummaryJSONByteCompat(t *testing.T) {
	t.Run("all fields including failures", func(t *testing.T) {
		summary := &BetaTestersImportSummary{
			AppID:           "app-1",
			InputFile:       "testers.csv",
			DryRun:          true,
			Invite:          false,
			SkipExisting:    false,
			ContinueOnError: true,
			AppliedGroup:    "Beta",
			Total:           3,
			Created:         1,
			Existed:         1,
			Updated:         1,
			Invited:         0,
			Failed:          1,
			Failures: []BetaTestersImportFailure{
				{Row: 2, Email: "bad@example.com", Error: "invalid email format"},
			},
		}
		got, err := json.Marshal(summary)
		if err != nil {
			t.Fatalf("marshal import summary: %v", err)
		}
		// Fixture: byte-for-byte JSON emitted by the pre-move unexported struct.
		want := `{"appId":"app-1","inputFile":"testers.csv","dryRun":true,"invite":false,"skipExisting":false,` +
			`"continueOnError":true,"appliedGroup":"Beta","total":3,"created":1,"existed":1,"updated":1,` +
			`"invited":0,"failed":1,"failures":[{"row":2,"email":"bad@example.com","error":"invalid email format"}]}`
		if string(got) != want {
			t.Fatalf("import summary JSON drifted:\n got: %s\nwant: %s", got, want)
		}
	})

	t.Run("omitempty fields stay absent", func(t *testing.T) {
		summary := &BetaTestersImportSummary{
			AppID:           "app-1",
			InputFile:       "testers.csv",
			ContinueOnError: true,
			Total:           1,
			Existed:         1,
		}
		got, err := json.Marshal(summary)
		if err != nil {
			t.Fatalf("marshal import summary: %v", err)
		}
		want := `{"appId":"app-1","inputFile":"testers.csv","dryRun":false,"invite":false,"skipExisting":false,` +
			`"continueOnError":true,"total":1,"created":0,"existed":1,"updated":0,"invited":0,"failed":0}`
		if string(got) != want {
			t.Fatalf("import summary JSON drifted:\n got: %s\nwant: %s", got, want)
		}
	})
}

func TestBetaTestersExportSummaryJSONByteCompat(t *testing.T) {
	summary := &BetaTestersExportSummary{
		AppID:         "app-1",
		OutputFile:    "testers.csv",
		Total:         12,
		IncludeGroups: true,
	}
	got, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal export summary: %v", err)
	}
	want := `{"appId":"app-1","outputFile":"testers.csv","total":12,"includeGroups":true}`
	if string(got) != want {
		t.Fatalf("export summary JSON drifted:\n got: %s\nwant: %s", got, want)
	}
}

func TestTestFlightSyncSummaryJSONByteCompat(t *testing.T) {
	summary := &TestFlightSyncSummary{
		File:    "testflight.yaml",
		App:     "Example App",
		Groups:  2,
		Builds:  3,
		Testers: 4,
	}
	got, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("marshal sync summary: %v", err)
	}
	want := `{"file":"testflight.yaml","app":"Example App","groups":2,"builds":3,"testers":4}`
	if string(got) != want {
		t.Fatalf("sync summary JSON drifted:\n got: %s\nwant: %s", got, want)
	}
}

func TestBetaTestersSummaryRows(t *testing.T) {
	t.Run("export summary", func(t *testing.T) {
		headers, rows := betaTestersExportSummaryRows(&BetaTestersExportSummary{
			AppID:         "app-1",
			OutputFile:    "testers.csv",
			Total:         12,
			IncludeGroups: true,
		})
		wantHeaders := []string{"App ID", "Output File", "Total", "Include Groups"}
		wantRow := []string{"app-1", "testers.csv", "12", "true"}
		assertSingleRowEquals(t, headers, rows, wantHeaders, wantRow)
	})

	t.Run("import summary and failures", func(t *testing.T) {
		summary := &BetaTestersImportSummary{
			AppID:     "app-1",
			InputFile: "testers.csv",
			DryRun:    true,
			Total:     2,
			Created:   1,
			Failed:    1,
			Failures: []BetaTestersImportFailure{
				{Row: 2, Email: "bad@example.com", Error: "invalid email format"},
			},
		}
		headers, rows := betaTestersImportSummaryRows(summary)
		wantHeaders := []string{"App ID", "Input File", "Dry Run", "Total", "Created", "Existed", "Updated", "Invited", "Failed"}
		wantRow := []string{"app-1", "testers.csv", "true", "2", "1", "0", "0", "0", "1"}
		assertSingleRowEquals(t, headers, rows, wantHeaders, wantRow)

		failureHeaders, failureRows := betaTestersImportFailureRows(summary.Failures)
		wantFailureHeaders := []string{"Row", "Email", "Error"}
		wantFailureRow := []string{"2", "bad@example.com", "invalid email format"}
		assertSingleRowEquals(t, failureHeaders, failureRows, wantFailureHeaders, wantFailureRow)
	})

	t.Run("sync summary", func(t *testing.T) {
		headers, rows := testFlightSyncSummaryRows(&TestFlightSyncSummary{
			File:    "testflight.yaml",
			App:     "Example App",
			Groups:  2,
			Builds:  3,
			Testers: 4,
		})
		wantHeaders := []string{"File", "App", "Groups", "Builds", "Testers"}
		wantRow := []string{"testflight.yaml", "Example App", "2", "3", "4"}
		assertSingleRowEquals(t, headers, rows, wantHeaders, wantRow)
	})
}

// TestBetaTestersImportSummaryDirectRendererRegistered proves the moved import
// summary renders both the summary table and the failures table through the
// registry (multi-table direct renderer).
func TestBetaTestersImportSummaryDirectRendererRegistered(t *testing.T) {
	ensureOutputRegistryPopulated()

	summary := &BetaTestersImportSummary{
		AppID:     "app-1",
		InputFile: "testers.csv",
		Total:     1,
		Failed:    1,
		Failures: []BetaTestersImportFailure{
			{Row: 1, Email: "bad@example.com", Error: "invalid email format"},
		},
	}

	var tables [][]string
	err := renderByRegistry(summary, func(headers []string, rows [][]string) {
		tables = append(tables, headers)
	})
	if err != nil {
		t.Fatalf("renderByRegistry returned error: %v", err)
	}
	if len(tables) != 2 {
		t.Fatalf("expected summary and failures tables, got %d tables", len(tables))
	}
	if !reflect.DeepEqual(tables[1], []string{"Row", "Email", "Error"}) {
		t.Fatalf("unexpected failures headers: %v", tables[1])
	}

	t.Run("failures table skipped when empty", func(t *testing.T) {
		var count int
		err := renderByRegistry(&BetaTestersImportSummary{AppID: "app-1"}, func([]string, [][]string) {
			count++
		})
		if err != nil {
			t.Fatalf("renderByRegistry returned error: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected only the summary table, got %d tables", count)
		}
	})
}
