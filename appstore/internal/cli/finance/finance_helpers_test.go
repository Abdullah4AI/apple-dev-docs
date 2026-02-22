package finance

import (
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestNormalizeFinanceReportType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"financial", "financial", "FINANCIAL", false},
		{"finance detail", "finance_detail", "FINANCE_DETAIL", false},
		{"trimmed", " FINANCIAL ", "FINANCIAL", false},
		{"invalid", "invalid", "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reportType, err := normalizeFinanceReportType(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected report type to parse, got %v", err)
			}
			if string(reportType) != test.want {
				t.Fatalf("expected %s, got %q", test.want, reportType)
			}
		})
	}
}

func TestNormalizeFinanceReportDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid", "2025-12", "2025-12", false},
		{"trimmed", " 2025-01 ", "2025-01", false},
		{"month zero", "2025-00", "", true},
		{"month overflow", "2025-13", "", true},
		{"wrong separator", "2025/12", "", true},
		{"with day", "2025-12-01", "", true},
		{"short year", "25-12", "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			date, err := normalizeFinanceReportDate(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected date to parse, got %v", err)
			}
			if date != test.want {
				t.Fatalf("expected date to be %s, got %q", test.want, date)
			}
		})
	}
}

func TestNormalizeFinanceReportRegion(t *testing.T) {
	tests := []struct {
		name       string
		reportType asc.FinanceReportType
		input      string
		want       string
		wantErr    bool
	}{
		{"financial uppercases", asc.FinanceReportTypeFinancial, "us", "US", false},
		{"financial trimmed", asc.FinanceReportTypeFinancial, " eu ", "EU", false},
		{"finance detail z1", asc.FinanceReportTypeFinanceDetail, "z1", "Z1", false},
		{"finance detail invalid", asc.FinanceReportTypeFinanceDetail, "US", "", true},
		{"missing region", asc.FinanceReportTypeFinancial, " ", "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			region, err := normalizeFinanceReportRegion(test.reportType, test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected region to parse, got %v", err)
			}
			if region != test.want {
				t.Fatalf("expected region to be %s, got %q", test.want, region)
			}
		})
	}
}
