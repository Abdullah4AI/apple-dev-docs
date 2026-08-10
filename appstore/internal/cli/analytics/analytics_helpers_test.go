package analytics

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestAnalyticsDownloadDefaultOutputSanitizesInstanceID(t *testing.T) {
	const requestID = "11111111-1111-1111-1111-111111111111"
	tests := []struct {
		name       string
		instanceID string
		want       string
	}{
		{
			name:       "preserves prefixed ID",
			instanceID: "r39-example-instance",
			want:       "analytics_report_11111111-1111-1111-1111-111111111111_r39-example-instance.csv.gz",
		},
		{
			name:       "removes path syntax",
			instanceID: `r39/../../target\segment:part`,
			want:       "analytics_report_11111111-1111-1111-1111-111111111111_r39_.._.._target_segment_part.csv.gz",
		},
		{
			name:       "uses fallback for only path syntax",
			instanceID: `../\:`,
			want:       "analytics_report_11111111-1111-1111-1111-111111111111_instance.csv.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyticsDownloadDefaultOutput(requestID, tt.instanceID)
			if got != tt.want {
				t.Fatalf("analyticsDownloadDefaultOutput() = %q, want %q", got, tt.want)
			}
			if filepath.Base(got) != got || strings.ContainsAny(got, `/\`) {
				t.Fatalf("analyticsDownloadDefaultOutput() returned path syntax: %q", got)
			}
		})
	}
}

func TestResolveReportOutputPaths_Decompress(t *testing.T) {
	compressed, decompressed := shared.ResolveReportOutputPaths("report.tsv.gz", "default.tsv.gz", ".tsv", true)
	if compressed != "report.tsv.gz" {
		t.Fatalf("expected compressed path report.tsv.gz, got %q", compressed)
	}
	if decompressed != "report.tsv" {
		t.Fatalf("expected decompressed path report.tsv, got %q", decompressed)
	}

	compressed, decompressed = shared.ResolveReportOutputPaths("report.tsv", "default.tsv.gz", ".tsv", true)
	if compressed != "report.tsv.gz" {
		t.Fatalf("expected compressed path report.tsv.gz, got %q", compressed)
	}
	if decompressed != "report.tsv" {
		t.Fatalf("expected decompressed path report.tsv, got %q", decompressed)
	}

	compressed, decompressed = shared.ResolveReportOutputPaths("report", "default.tsv.gz", ".tsv", true)
	if compressed != "report" {
		t.Fatalf("expected compressed path report, got %q", compressed)
	}
	if decompressed != "report.tsv" {
		t.Fatalf("expected decompressed path report.tsv, got %q", decompressed)
	}
}

func TestNormalizeReportDate_MonthlyValidation(t *testing.T) {
	_, err := normalizeReportDate("2024/01/02", asc.SalesReportFrequencyMonthly)
	if err == nil {
		t.Fatal("expected error for malformed monthly date")
	}
}

func TestNormalizeReportDate_AllowsOmission(t *testing.T) {
	date, err := normalizeReportDate("", asc.SalesReportFrequencyDaily)
	if err != nil {
		t.Fatalf("expected omitted date to be valid, got %v", err)
	}
	if date != "" {
		t.Fatalf("expected empty normalized date, got %q", date)
	}
}

func TestNormalizeSalesReportTypeSupportsDocumentedValues(t *testing.T) {
	for _, value := range []string{
		"SUBSCRIBER",
		"SUBSCRIPTION_OFFER_CODE_REDEMPTION",
		"INSTALLS",
		"FIRST_ANNUAL",
		"WIN_BACK_ELIGIBILITY",
	} {
		t.Run(value, func(t *testing.T) {
			got, err := normalizeSalesReportType(value)
			if err != nil {
				t.Fatalf("expected %s to be accepted, got %v", value, err)
			}
			if string(got) != value {
				t.Fatalf("expected %s, got %q", value, got)
			}
		})
	}
}

func TestNormalizeSalesReportSubTypeSupportsDocumentedValues(t *testing.T) {
	for _, value := range []string{
		"SUMMARY_INSTALL_TYPE",
		"SUMMARY_TERRITORY",
		"SUMMARY_CHANNEL",
	} {
		t.Run(value, func(t *testing.T) {
			got, err := normalizeSalesReportSubType(value)
			if err != nil {
				t.Fatalf("expected %s to be accepted, got %v", value, err)
			}
			if string(got) != value {
				t.Fatalf("expected %s, got %q", value, got)
			}
		})
	}
}

func TestNormalizeReportDate_MonthlyFormat(t *testing.T) {
	date, err := normalizeReportDate("2024-01", asc.SalesReportFrequencyMonthly)
	if err != nil {
		t.Fatalf("expected monthly date to parse, got %v", err)
	}
	if date != "2024-01" {
		t.Fatalf("expected monthly report key 2024-01, got %q", date)
	}
}

func TestNormalizeReportDate_MonthlyFullDate(t *testing.T) {
	for _, input := range []string{"2024-02-15", "2024-02-29"} {
		date, err := normalizeReportDate(input, asc.SalesReportFrequencyMonthly)
		if err != nil {
			t.Fatalf("expected monthly date %s to parse, got %v", input, err)
		}
		if date != "2024-02" {
			t.Fatalf("expected monthly report key 2024-02 for %s, got %q", input, date)
		}
	}
}

func TestNormalizeReportDate_YearlyFormat(t *testing.T) {
	date, err := normalizeReportDate("2024", asc.SalesReportFrequencyYearly)
	if err != nil {
		t.Fatalf("expected yearly date to parse, got %v", err)
	}
	if date != "2024" {
		t.Fatalf("expected yearly report key 2024, got %q", date)
	}
}

func TestNormalizeReportDate_YearlyFullDate(t *testing.T) {
	for _, input := range []string{"2024-06-30", "2024-12-31"} {
		date, err := normalizeReportDate(input, asc.SalesReportFrequencyYearly)
		if err != nil {
			t.Fatalf("expected yearly date %s to parse, got %v", input, err)
		}
		if date != "2024" {
			t.Fatalf("expected yearly report key 2024 for %s, got %q", input, date)
		}
	}
}

func TestNormalizeReportDate_WeeklyMondayConvertsToSunday(t *testing.T) {
	date, err := normalizeReportDate("2026-02-09", asc.SalesReportFrequencyWeekly)
	if err != nil {
		t.Fatalf("expected weekly monday date to parse, got %v", err)
	}
	if date != "2026-02-15" {
		t.Fatalf("expected weekly monday date to normalize to sunday 2026-02-15, got %q", date)
	}
}

func TestNormalizeReportDate_WeeklySundayRemainsSunday(t *testing.T) {
	date, err := normalizeReportDate("2026-02-15", asc.SalesReportFrequencyWeekly)
	if err != nil {
		t.Fatalf("expected weekly sunday date to parse, got %v", err)
	}
	if date != "2026-02-15" {
		t.Fatalf("expected weekly sunday date to remain unchanged, got %q", date)
	}
}

func TestNormalizeReportDate_WeeklyRejectsNonBoundaryDate(t *testing.T) {
	_, err := normalizeReportDate("2026-02-11", asc.SalesReportFrequencyWeekly)
	if err == nil {
		t.Fatal("expected error for weekly non-monday/sunday date")
	}
}

func TestNormalizeSalesReportVersionRejectsVersionOutsideTuple(t *testing.T) {
	_, err := normalizeSalesReportVersion("1_4", asc.SalesReportTypeSales, asc.SalesReportSubTypeSummary, asc.SalesReportFrequencyDaily)
	if err == nil {
		t.Fatal("expected SALES/SUMMARY/DAILY version 1_4 to be rejected")
	}
}

func TestNormalizeSalesReportVersionDefaultsSubscriptionTo1_4(t *testing.T) {
	version, err := normalizeSalesReportVersion("  ", asc.SalesReportTypeSubscription, asc.SalesReportSubTypeSummary, asc.SalesReportFrequencyDaily)
	if err != nil {
		t.Fatalf("expected empty version to use the default, got %v", err)
	}
	if version != asc.SalesReportVersion1_4 {
		t.Fatalf("expected subscription default version 1_4, got %q", version)
	}
}

func TestNormalizeSalesReportVersionPreservesOtherDefaults(t *testing.T) {
	version, err := normalizeSalesReportVersion("  ", asc.SalesReportTypeSales, asc.SalesReportSubTypeSummary, asc.SalesReportFrequencyDaily)
	if err != nil {
		t.Fatalf("expected empty version to use the default, got %v", err)
	}
	if version != asc.SalesReportVersion1_0 {
		t.Fatalf("expected sales default version 1_0, got %q", version)
	}
}

func TestNormalizeSalesReportVersionDefaultsSubscriberTo1_3(t *testing.T) {
	version, err := normalizeSalesReportVersion("", asc.SalesReportTypeSubscriber, asc.SalesReportSubTypeDetailed, asc.SalesReportFrequencyDaily)
	if err != nil {
		t.Fatalf("expected empty version to use the default, got %v", err)
	}
	if version != asc.SalesReportVersion1_3 {
		t.Fatalf("expected subscriber default version 1_3, got %q", version)
	}
}

func TestNormalizeSalesReportVersionDefaultsMonthlyInstallsTo1_2(t *testing.T) {
	version, err := normalizeSalesReportVersion("", asc.SalesReportTypeInstalls, asc.SalesReportSubTypeSummary, asc.SalesReportFrequencyMonthly)
	if err != nil {
		t.Fatalf("expected empty version to use the default, got %v", err)
	}
	if version != asc.SalesReportVersion1_2 {
		t.Fatalf("expected monthly installs default version 1_2, got %q", version)
	}
}

func TestNormalizeSalesReportVersionRejectsInvalidValue(t *testing.T) {
	_, err := normalizeSalesReportVersion("1.4", asc.SalesReportTypeSubscription, asc.SalesReportSubTypeSummary, asc.SalesReportFrequencyDaily)
	if err == nil {
		t.Fatal("expected invalid version error")
	}
}

func TestNormalizeSalesReportVersionRejectsUndocumentedFutureVersion(t *testing.T) {
	_, err := normalizeSalesReportVersion(" 1_5 ", asc.SalesReportTypeSubscription, asc.SalesReportSubTypeSummary, asc.SalesReportFrequencyDaily)
	if err == nil {
		t.Fatal("expected undocumented version 1_5 to be rejected")
	}
}

func TestValidateSalesReportTupleRejectsUnsupportedCombination(t *testing.T) {
	err := validateSalesReportTuple(
		asc.SalesReportTypeWinBackEligibility,
		asc.SalesReportSubTypeDetailed,
		asc.SalesReportFrequencyWeekly,
	)
	if err == nil {
		t.Fatal("expected unsupported WIN_BACK_ELIGIBILITY/DETAILED/WEEKLY tuple to be rejected")
	}
}

func TestDecompressGzipFile(t *testing.T) {
	tempDir := t.TempDir()
	source := filepath.Join(tempDir, "source.tsv.gz")
	dest := filepath.Join(tempDir, "dest.tsv")

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte("hello")); err != nil {
		t.Fatalf("failed to write gzip: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("failed to close gzip: %v", err)
	}
	if err := os.WriteFile(source, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("failed to write source gzip: %v", err)
	}

	size, err := shared.DecompressGzipFile(source, dest)
	if err != nil {
		t.Fatalf("decompressGzipFile() error: %v", err)
	}
	if size == 0 {
		t.Fatalf("expected non-zero decompressed size")
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read dest file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected decompressed content to be hello, got %q", string(data))
	}
}
