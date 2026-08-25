package asc

import (
	"strconv"
	"strings"
)

// AgeRatingAuditRow reports one active app info's social-media capability responses.
type AgeRatingAuditRow struct {
	AppID                    string   `json:"appId"`
	AppInfoID                string   `json:"appInfoId,omitempty"`
	AppInfoState             string   `json:"appInfoState,omitempty"`
	Name                     string   `json:"name,omitempty"`
	BundleID                 string   `json:"bundleId,omitempty"`
	SocialMedia              string   `json:"socialMedia"`
	SocialMediaAgeRestricted string   `json:"socialMediaAgeRestricted"`
	MessagingAndChat         string   `json:"messagingAndChat"`
	UserGeneratedContent     string   `json:"userGeneratedContent"`
	AgeAssurance             string   `json:"ageAssurance"`
	MissingResponses         []string `json:"missingResponses"`
	Ready                    bool     `json:"ready"`
	Error                    string   `json:"error,omitempty"`
}

// AgeRatingAuditResult summarizes social-media capability readiness per active app info.
type AgeRatingAuditResult struct {
	Apps         []AgeRatingAuditRow `json:"apps"`
	ReadyCount   int                 `json:"readyCount"`
	MissingCount int                 `json:"missingCount"`
	ErrorCount   int                 `json:"errorCount"`
}

func ageRatingAuditResultRows(result *AgeRatingAuditResult) ([]string, [][]string) {
	headers := []string{"App ID", "App Info ID", "State", "Name", "Social Media", "Age Restricted", "Messaging & Chat", "User Generated Content", "Age Assurance", "Ready", "Missing"}
	rows := make([][]string, 0, len(result.Apps))
	for _, row := range result.Apps {
		missing := strings.Join(row.MissingResponses, ", ")
		if row.Error != "" {
			missing = "error: " + row.Error
		}
		rows = append(rows, []string{
			row.AppID,
			row.AppInfoID,
			row.AppInfoState,
			row.Name,
			row.SocialMedia,
			row.SocialMediaAgeRestricted,
			row.MessagingAndChat,
			row.UserGeneratedContent,
			row.AgeAssurance,
			strconv.FormatBool(row.Ready),
			missing,
		})
	}
	return headers, rows
}
