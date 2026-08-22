package asc

import (
	"fmt"
	"strings"
)

// DeveloperSystemStatusReport is a computed snapshot of Apple's public
// Developer System Status feed.
type DeveloperSystemStatusReport struct {
	Source   string                         `json:"source"`
	Summary  DeveloperSystemStatusSummary   `json:"summary"`
	Message  string                         `json:"message,omitempty"`
	Services []DeveloperSystemStatusService `json:"services"`
}

// DeveloperSystemStatusSummary describes the selected services in a system
// status report. Counts describe all services matched by --service even when
// --issues-only omits healthy services from Services.
type DeveloperSystemStatusSummary struct {
	Status              string `json:"status"`
	TotalServices       int    `json:"totalServices"`
	OperationalServices int    `json:"operationalServices"`
	AffectedServices    int    `json:"affectedServices"`
	ActiveIncidents     int    `json:"activeIncidents"`
}

// DeveloperSystemStatusService is one Apple developer service and its recent
// status events.
type DeveloperSystemStatusService struct {
	Name   string                       `json:"name"`
	Status string                       `json:"status"`
	URL    string                       `json:"url,omitempty"`
	Events []DeveloperSystemStatusEvent `json:"events"`
}

// DeveloperSystemStatusEvent preserves the useful fields exposed by Apple's
// public status feed and adds Active for deterministic filtering.
type DeveloperSystemStatusEvent struct {
	MessageID        string `json:"messageId"`
	StatusType       string `json:"statusType,omitempty"`
	EventStatus      string `json:"eventStatus,omitempty"`
	Message          string `json:"message,omitempty"`
	DatePosted       string `json:"datePosted,omitempty"`
	StartDate        string `json:"startDate,omitempty"`
	EndDate          string `json:"endDate,omitempty"`
	EpochStartDate   int64  `json:"epochStartDate,omitempty"`
	EpochEndDate     *int64 `json:"epochEndDate,omitempty"`
	UsersAffected    string `json:"usersAffected,omitempty"`
	AffectedServices string `json:"affectedServices,omitempty"`
	Active           bool   `json:"active"`
}

func developerSystemStatusSummaryRows(report *DeveloperSystemStatusReport) ([]string, [][]string) {
	if report == nil {
		report = &DeveloperSystemStatusReport{}
	}
	return []string{"Status", "Services", "Operational", "Affected", "Active Incidents", "Message", "Source"}, [][]string{{
		report.Summary.Status,
		fmt.Sprintf("%d", report.Summary.TotalServices),
		fmt.Sprintf("%d", report.Summary.OperationalServices),
		fmt.Sprintf("%d", report.Summary.AffectedServices),
		fmt.Sprintf("%d", report.Summary.ActiveIncidents),
		report.Message,
		report.Source,
	}}
}

func developerSystemStatusServiceRows(report *DeveloperSystemStatusReport) ([]string, [][]string) {
	headers := []string{"Service", "Status", "Active Incidents", "Details", "Updated", "Link"}
	if report == nil {
		return headers, nil
	}

	rows := make([][]string, 0, len(report.Services))
	for _, service := range report.Services {
		activeCount := 0
		details := ""
		updated := ""
		for _, event := range service.Events {
			if !event.Active {
				continue
			}
			activeCount++
			if details == "" {
				details = strings.TrimSpace(strings.Join([]string{event.StatusType, event.Message}, ": "))
				details = strings.Trim(details, ": ")
				updated = event.DatePosted
			}
		}
		rows = append(rows, []string{
			service.Name,
			service.Status,
			fmt.Sprintf("%d", activeCount),
			details,
			updated,
			service.URL,
		})
	}
	return headers, rows
}
