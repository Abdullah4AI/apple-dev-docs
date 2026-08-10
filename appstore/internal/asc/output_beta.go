package asc

import (
	"fmt"
	"strings"
)

// BetaTesterInvitationResult represents CLI output for invitations.
type BetaTesterInvitationResult struct {
	InvitationID string `json:"invitationId"`
	TesterID     string `json:"testerId,omitempty"`
	AppID        string `json:"appId,omitempty"`
	Email        string `json:"email,omitempty"`
}

// BetaTesterDeleteResult represents CLI output for deletions.
type BetaTesterDeleteResult struct {
	ID      string `json:"id"`
	Email   string `json:"email,omitempty"`
	Deleted bool   `json:"deleted"`
}

// BetaTesterGroupsUpdateResult represents CLI output for beta tester group updates.
type BetaTesterGroupsUpdateResult struct {
	TesterID string   `json:"testerId"`
	GroupIDs []string `json:"groupIds"`
	Action   string   `json:"action"`
}

// BetaTesterAppsUpdateResult represents CLI output for beta tester app updates.
type BetaTesterAppsUpdateResult struct {
	TesterID string   `json:"testerId"`
	AppIDs   []string `json:"appIds"`
	Action   string   `json:"action"`
}

// BetaTesterBuildsUpdateResult represents CLI output for beta tester build updates.
type BetaTesterBuildsUpdateResult struct {
	TesterID string   `json:"testerId"`
	BuildIDs []string `json:"buildIds"`
	Action   string   `json:"action"`
}

// AppBetaTestersUpdateResult represents CLI output for app beta tester updates.
type AppBetaTestersUpdateResult struct {
	AppID     string   `json:"appId"`
	TesterIDs []string `json:"testerIds"`
	Action    string   `json:"action"`
}

// BuildBetaGroupMembershipResult describes the TestFlight groups that give a
// build access, including explicit relationship membership and all-build access.
type BuildBetaGroupMembershipResult struct {
	BuildID      string                            `json:"buildId"`
	AppID        string                            `json:"appId"`
	Complete     bool                              `json:"complete"`
	LookupMethod string                            `json:"lookupMethod"`
	GroupCount   int                               `json:"groupCount"`
	Groups       []BuildBetaGroupMembershipGroup   `json:"groups"`
	Failures     []BuildBetaGroupMembershipFailure `json:"failures,omitempty"`
}

// BuildBetaGroupMembershipGroup is one group that contains or implicitly
// receives a build.
type BuildBetaGroupMembershipGroup struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Type                 string `json:"type"`
	Membership           string `json:"membership"`
	HasAccessToAllBuilds bool   `json:"hasAccessToAllBuilds"`
}

// BuildBetaGroupMembershipFailure records an inverse relationship lookup that
// could not be completed.
type BuildBetaGroupMembershipFailure struct {
	GroupID   string `json:"groupId"`
	GroupName string `json:"groupName,omitempty"`
	Error     string `json:"error"`
}

// BetaFeedbackSubmissionDeleteResult represents CLI output for beta feedback deletions.
type BetaFeedbackSubmissionDeleteResult struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

func formatBetaTesterName(attr BetaTesterAttributes) string {
	first := strings.TrimSpace(attr.FirstName)
	last := strings.TrimSpace(attr.LastName)
	switch {
	case first == "" && last == "":
		return ""
	case first == "":
		return last
	case last == "":
		return first
	default:
		return first + " " + last
	}
}

func betaGroupsRows(resp *BetaGroupsResponse) ([]string, [][]string) {
	headers := []string{"ID", "Name", "Internal", "Public Link Enabled", "Public Link"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{
			item.ID,
			compactWhitespace(item.Attributes.Name),
			fmt.Sprintf("%t", item.Attributes.IsInternalGroup),
			fmt.Sprintf("%t", item.Attributes.PublicLinkEnabled),
			item.Attributes.PublicLink,
		})
	}
	return headers, rows
}

func buildBetaGroupMembershipRows(result *BuildBetaGroupMembershipResult) ([]string, [][]string) {
	headers := []string{"Build ID", "App ID", "Group ID", "Name", "Type", "Membership", "All Builds", "Complete", "Error"}
	rows := make([][]string, 0, len(result.Groups)+len(result.Failures))
	for _, group := range result.Groups {
		rows = append(rows, []string{
			result.BuildID,
			result.AppID,
			group.ID,
			compactWhitespace(group.Name),
			group.Type,
			group.Membership,
			fmt.Sprintf("%t", group.HasAccessToAllBuilds),
			fmt.Sprintf("%t", result.Complete),
			"",
		})
	}
	for _, failure := range result.Failures {
		rows = append(rows, []string{
			result.BuildID,
			result.AppID,
			failure.GroupID,
			compactWhitespace(failure.GroupName),
			"",
			"error",
			"",
			"false",
			compactWhitespace(failure.Error),
		})
	}
	return headers, rows
}

func betaTestersRows(resp *BetaTestersResponse) ([]string, [][]string) {
	headers := []string{"ID", "Email", "Name", "State", "Invite"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{
			item.ID,
			item.Attributes.Email,
			compactWhitespace(formatBetaTesterName(item.Attributes)),
			string(item.Attributes.State),
			string(item.Attributes.InviteType),
		})
	}
	return headers, rows
}

func betaTesterDeleteResultRows(result *BetaTesterDeleteResult) ([]string, [][]string) {
	headers := []string{"ID", "Email", "Deleted"}
	rows := [][]string{{result.ID, result.Email, fmt.Sprintf("%t", result.Deleted)}}
	return headers, rows
}

func betaTesterGroupsUpdateResultRows(result *BetaTesterGroupsUpdateResult) ([]string, [][]string) {
	headers := []string{"Tester ID", "Group IDs", "Action"}
	rows := [][]string{{result.TesterID, strings.Join(result.GroupIDs, ","), result.Action}}
	return headers, rows
}

func betaTesterAppsUpdateResultRows(result *BetaTesterAppsUpdateResult) ([]string, [][]string) {
	headers := []string{"Tester ID", "App IDs", "Action"}
	rows := [][]string{{result.TesterID, strings.Join(result.AppIDs, ","), result.Action}}
	return headers, rows
}

func betaTesterBuildsUpdateResultRows(result *BetaTesterBuildsUpdateResult) ([]string, [][]string) {
	headers := []string{"Tester ID", "Build IDs", "Action"}
	rows := [][]string{{result.TesterID, strings.Join(result.BuildIDs, ","), result.Action}}
	return headers, rows
}

func appBetaTestersUpdateResultRows(result *AppBetaTestersUpdateResult) ([]string, [][]string) {
	headers := []string{"App ID", "Tester IDs", "Action"}
	rows := [][]string{{result.AppID, strings.Join(result.TesterIDs, ","), result.Action}}
	return headers, rows
}

func betaFeedbackSubmissionDeleteResultRows(result *BetaFeedbackSubmissionDeleteResult) ([]string, [][]string) {
	headers := []string{"ID", "Deleted"}
	rows := [][]string{{result.ID, fmt.Sprintf("%t", result.Deleted)}}
	return headers, rows
}

func betaTesterInvitationResultRows(result *BetaTesterInvitationResult) ([]string, [][]string) {
	headers := []string{"Invitation ID", "Tester ID", "App ID", "Email"}
	rows := [][]string{{result.InvitationID, result.TesterID, result.AppID, result.Email}}
	return headers, rows
}
