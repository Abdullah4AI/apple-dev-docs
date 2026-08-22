package asc

import (
	"fmt"
)

// DeviceLocalUDIDResult represents CLI output for local device UDID lookup.
type DeviceLocalUDIDResult struct {
	UDID     string `json:"udid"`
	Platform string `json:"platform"`
}

// DeviceBatchRegistrationItem describes one row processed by a batch registration.
type DeviceBatchRegistrationItem struct {
	Row      int    `json:"row"`
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	UDID     string `json:"udid"`
	Platform string `json:"platform"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Error    string `json:"error,omitempty"`
}

// DeviceBatchRegistrationSummary represents CLI output for batch device registration.
type DeviceBatchRegistrationSummary struct {
	InputFile       string                        `json:"inputFile"`
	DryRun          bool                          `json:"dryRun"`
	ContinueOnError bool                          `json:"continueOnError"`
	Total           int                           `json:"total"`
	Processed       int                           `json:"processed"`
	Registered      int                           `json:"registered"`
	Planned         int                           `json:"planned"`
	Skipped         int                           `json:"skipped"`
	Failed          int                           `json:"failed"`
	Results         []DeviceBatchRegistrationItem `json:"results"`
}

func deviceLocalUDIDRows(result *DeviceLocalUDIDResult) ([]string, [][]string) {
	headers := []string{"UDID", "Platform"}
	rows := [][]string{{result.UDID, result.Platform}}
	return headers, rows
}

func deviceBatchRegistrationSummaryRows(summary *DeviceBatchRegistrationSummary) ([]string, [][]string) {
	headers := []string{"Input File", "Dry Run", "Total", "Processed", "Registered", "Planned", "Skipped", "Failed"}
	rows := [][]string{{
		compactWhitespace(summary.InputFile),
		fmt.Sprintf("%t", summary.DryRun),
		fmt.Sprintf("%d", summary.Total),
		fmt.Sprintf("%d", summary.Processed),
		fmt.Sprintf("%d", summary.Registered),
		fmt.Sprintf("%d", summary.Planned),
		fmt.Sprintf("%d", summary.Skipped),
		fmt.Sprintf("%d", summary.Failed),
	}}
	return headers, rows
}

func deviceBatchRegistrationItemRows(items []DeviceBatchRegistrationItem) ([]string, [][]string) {
	headers := []string{"Line", "ID", "Name", "UDID", "Platform", "Status", "Reason", "Error"}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			fmt.Sprintf("%d", item.Row),
			compactWhitespace(item.ID),
			compactWhitespace(item.Name),
			compactWhitespace(item.UDID),
			compactWhitespace(item.Platform),
			compactWhitespace(item.Status),
			compactWhitespace(item.Reason),
			compactWhitespace(item.Error),
		})
	}
	return headers, rows
}

func devicesRows(resp *DevicesResponse) ([]string, [][]string) {
	headers := []string{"ID", "Name", "UDID", "Platform", "Status", "Class", "Model", "Added"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{
			item.ID,
			compactWhitespace(item.Attributes.Name),
			compactWhitespace(item.Attributes.UDID),
			compactWhitespace(string(item.Attributes.Platform)),
			compactWhitespace(string(item.Attributes.Status)),
			compactWhitespace(string(item.Attributes.DeviceClass)),
			compactWhitespace(item.Attributes.Model),
			compactWhitespace(item.Attributes.AddedDate),
		})
	}
	return headers, rows
}
