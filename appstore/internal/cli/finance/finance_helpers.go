package finance

import (
	"fmt"
	"strings"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func normalizeFinanceReportType(value string) (asc.FinanceReportType, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case string(asc.FinanceReportTypeFinancial):
		return asc.FinanceReportTypeFinancial, nil
	case string(asc.FinanceReportTypeFinanceDetail):
		return asc.FinanceReportTypeFinanceDetail, nil
	default:
		return "", fmt.Errorf("--report-type must be FINANCIAL or FINANCE_DETAIL")
	}
}

func normalizeFinanceReportDate(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("--date is required")
	}
	parsed, err := time.Parse("2006-01", trimmed)
	if err != nil {
		return "", fmt.Errorf("--date must be in YYYY-MM format")
	}
	return parsed.Format("2006-01"), nil
}

func normalizeFinanceReportRegion(reportType asc.FinanceReportType, value string) (string, error) {
	regionCode := strings.ToUpper(strings.TrimSpace(value))
	if regionCode == "" {
		return "", fmt.Errorf("--region is required")
	}
	if reportType == asc.FinanceReportTypeFinanceDetail && regionCode != "Z1" {
		return "", fmt.Errorf("--region must be Z1 for FINANCE_DETAIL reports")
	}
	return regionCode, nil
}
