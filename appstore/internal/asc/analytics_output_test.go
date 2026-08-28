package asc

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestAnalyticsReportRequestOutputUsesCurrentSchema(t *testing.T) {
	var response AnalyticsReportRequestResponse
	if err := json.Unmarshal([]byte(`{
		"data": {
			"type": "analyticsReportRequests",
			"id": "request-1",
			"attributes": {
				"accessType": "ONGOING",
				"state": "PROCESSING",
				"createdDate": "2026-08-10T00:00:00Z",
				"stoppedDueToInactivity": false
			}
		}
	}`), &response); err != nil {
		t.Fatalf("unmarshal analytics request: %v", err)
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal analytics request: %v", err)
	}
	for _, removed := range []string{`"state"`, `"createdDate"`} {
		if strings.Contains(string(encoded), removed) {
			t.Fatalf("current analytics request output contains removed field %s: %s", removed, encoded)
		}
	}
	for _, retained := range []string{`"accessType":"ONGOING"`, `"stoppedDueToInactivity":false`} {
		if !strings.Contains(string(encoded), retained) {
			t.Fatalf("current analytics request output is missing %s: %s", retained, encoded)
		}
	}

	falseValue := false
	tests := []struct {
		name        string
		render      func() ([]string, [][]string)
		wantHeaders []string
		wantRow     []string
	}{
		{
			name: "created request",
			render: func() ([]string, [][]string) {
				return analyticsReportRequestResultRows(&AnalyticsReportRequestResult{
					RequestID:              "request-1",
					AppID:                  "app-1",
					AccessType:             "ONGOING",
					StoppedDueToInactivity: &falseValue,
				})
			},
			wantHeaders: []string{"Request ID", "App ID", "Access Type", "Stopped Due To Inactivity"},
			wantRow:     []string{"request-1", "app-1", "ONGOING", "false"},
		},
		{
			name: "reused request",
			render: func() ([]string, [][]string) {
				return analyticsReportRequestReuseResultRows(&AnalyticsReportRequestReuseResult{
					RequestID:              "request-1",
					AppID:                  "app-1",
					AccessType:             "ONGOING",
					StoppedDueToInactivity: &falseValue,
					Created:                false,
				})
			},
			wantHeaders: []string{"Request ID", "App ID", "Access Type", "Stopped Due To Inactivity", "Created"},
			wantRow:     []string{"request-1", "app-1", "ONGOING", "false", "false"},
		},
		{
			name: "request list",
			render: func() ([]string, [][]string) {
				return analyticsReportRequestsRows(&AnalyticsReportRequestsResponse{Data: []AnalyticsReportRequestResource{{
					ID: "request-1",
					Attributes: AnalyticsReportRequestAttributes{
						AccessType:             AnalyticsAccessTypeOngoing,
						StoppedDueToInactivity: &falseValue,
					},
				}}})
			},
			wantHeaders: []string{"ID", "Access Type", "Stopped Due To Inactivity", "App ID"},
			wantRow:     []string{"request-1", "ONGOING", "false", ""},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			headers, rows := test.render()
			if !reflect.DeepEqual(headers, test.wantHeaders) {
				t.Fatalf("headers = %#v, want %#v", headers, test.wantHeaders)
			}
			if len(rows) != 1 || !reflect.DeepEqual(rows[0], test.wantRow) {
				t.Fatalf("rows = %#v, want %#v", rows, [][]string{test.wantRow})
			}
		})
	}
}

func TestSalesReportResultRowsShowsMissingAvailabilityWithoutFileColumns(t *testing.T) {
	available := false
	headers, rows := salesReportResultRows(&SalesReportResult{
		VendorNumber:  "12345678",
		ReportType:    "SALES",
		ReportSubType: "SUMMARY",
		Frequency:     "DAILY",
		ReportDate:    "2026-08-18",
		Version:       "1_0",
		Available:     &available,
	})
	if len(headers) != 7 || headers[6] != "Available" {
		t.Fatalf("headers = %v, want availability metadata without file columns", headers)
	}
	if len(rows) != 1 || len(rows[0]) != 7 || rows[0][6] != "false" {
		t.Fatalf("rows = %v, want available=false", rows)
	}
}

func TestSalesReportResultRowsPreservesDownloadedFileColumns(t *testing.T) {
	headers, rows := salesReportResultRows(&SalesReportResult{FilePath: "report.tsv.gz"})
	if len(headers) != 10 || headers[6] != "Compressed File" {
		t.Fatalf("headers = %v, want existing download columns", headers)
	}
	if len(rows) != 1 || len(rows[0]) != 10 || rows[0][6] != "report.tsv.gz" {
		t.Fatalf("rows = %v, want existing download row", rows)
	}
}
