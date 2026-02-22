package asc

import (
	"context"
	"net/url"
	"strings"
)

// FinanceReportType represents the finance report type for the App Store Connect API.
//
// The API supports two report types that map to UI options as follows:
//
//   - FINANCIAL: Aggregated monthly financial report.
//     Maps to UI options "All Countries or Regions (Single File)" with region ZZ,
//     or "All Countries or Regions (Multiple Files)" with individual region codes.
//
//   - FINANCE_DETAIL: Detailed report with transaction and settlement dates.
//     Maps to UI option "All Countries or Regions (Detailed)".
//     Requires region code Z1.
//
// Note: Transaction Tax reports shown in the App Store Connect UI are NOT available
// via the API. They must be downloaded manually from the web interface.
type FinanceReportType string

const (
	// FinanceReportTypeFinancial is for aggregated monthly financial reports.
	// Use with region codes: US, EU, ZZ (consolidated), or other individual/regional codes.
	FinanceReportTypeFinancial FinanceReportType = "FINANCIAL"

	// FinanceReportTypeFinanceDetail is for detailed reports with transaction dates.
	// Must be used with region code Z1 only.
	FinanceReportTypeFinanceDetail FinanceReportType = "FINANCE_DETAIL"
)

// FinanceReportParams describes finance report query parameters.
type FinanceReportParams struct {
	VendorNumber string
	ReportType   FinanceReportType
	RegionCode   string
	ReportDate   string
}

func buildFinanceReportQuery(params FinanceReportParams) string {
	values := url.Values{}
	if strings.TrimSpace(params.VendorNumber) != "" {
		values.Set("filter[vendorNumber]", strings.TrimSpace(params.VendorNumber))
	}
	if params.ReportType != "" {
		values.Set("filter[reportType]", string(params.ReportType))
	}
	if strings.TrimSpace(params.RegionCode) != "" {
		values.Set("filter[regionCode]", strings.TrimSpace(params.RegionCode))
	}
	if strings.TrimSpace(params.ReportDate) != "" {
		values.Set("filter[reportDate]", strings.TrimSpace(params.ReportDate))
	}
	return values.Encode()
}

// DownloadFinanceReport retrieves a finance report as a gzip stream.
func (c *Client) DownloadFinanceReport(ctx context.Context, params FinanceReportParams) (*ReportDownload, error) {
	path := "/v1/financeReports"
	if queryString := buildFinanceReportQuery(params); queryString != "" {
		path += "?" + queryString
	}

	resp, err := c.doStream(ctx, path, "application/a-gzip")
	if err != nil {
		return nil, err
	}
	return &ReportDownload{Body: resp.Body, ContentLength: resp.ContentLength}, nil
}
