package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestFinanceReportsValidationErrors(t *testing.T) {
	t.Setenv("ASC_VENDOR_NUMBER", "")
	t.Setenv("ASC_ANALYTICS_VENDOR_NUMBER", "")

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing vendor",
			args:    []string{"finance", "reports", "--report-type", "FINANCIAL", "--region", "US", "--date", "2025-12"},
			wantErr: "--vendor is required",
		},
		{
			name:    "missing report type",
			args:    []string{"finance", "reports", "--vendor", "12345678", "--region", "US", "--date", "2025-12"},
			wantErr: "--report-type is required",
		},
		{
			name:    "missing region",
			args:    []string{"finance", "reports", "--vendor", "12345678", "--report-type", "FINANCIAL", "--date", "2025-12"},
			wantErr: "--region is required",
		},
		{
			name:    "missing date",
			args:    []string{"finance", "reports", "--vendor", "12345678", "--report-type", "FINANCIAL", "--region", "US"},
			wantErr: "--date is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := root.Run(context.Background())
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected ErrHelp, got %v", err)
				}
			})

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected error %q, got %q", test.wantErr, stderr)
			}
		})
	}
}

func TestFinanceReportsInvalidFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "invalid report type",
			args:    []string{"finance", "reports", "--vendor", "12345678", "--report-type", "INVALID", "--region", "US", "--date", "2025-12"},
			wantErr: "--report-type must be FINANCIAL or FINANCE_DETAIL",
		},
		{
			name:    "invalid date format",
			args:    []string{"finance", "reports", "--vendor", "12345678", "--report-type", "FINANCIAL", "--region", "US", "--date", "2025-13"},
			wantErr: "--date must be in YYYY-MM format",
		},
		{
			name:    "invalid finance detail region",
			args:    []string{"finance", "reports", "--vendor", "12345678", "--report-type", "FINANCE_DETAIL", "--region", "US", "--date", "2025-12"},
			wantErr: "--region must be Z1 for FINANCE_DETAIL reports",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := root.Run(context.Background())
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected non-help error, got %v", err)
				}
			})

			_ = stdout
			_ = stderr
		})
	}
}

func TestFinanceRegionsCommandOutputsJSON(t *testing.T) {
	root := RootCommand("1.2.3")

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"finance", "regions"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var payload struct {
		Regions []struct {
			RegionCode string `json:"regionCode"`
		} `json:"regions"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}
	if len(payload.Regions) == 0 {
		t.Fatal("expected regions in output")
	}
	found := false
	for _, region := range payload.Regions {
		if region.RegionCode == "Z1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected region Z1 in output")
	}
}
