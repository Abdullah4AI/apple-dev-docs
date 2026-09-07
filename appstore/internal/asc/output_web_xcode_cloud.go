package asc

import (
	"fmt"
	"strconv"
)

// WebXcodeCloudVersionAlias is the safe, scalar summary of one custom version alias.
type WebXcodeCloudVersionAlias struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Locked         bool   `json:"locked"`
	BuildName      string `json:"buildName,omitempty"`
	BuildSupported bool   `json:"buildSupported"`
}

// WebXcodeCloudVersionAliasesResult is the computed alias list for one product.
type WebXcodeCloudVersionAliasesResult struct {
	ProductID      string                      `json:"productId"`
	VersionAliases []WebXcodeCloudVersionAlias `json:"versionAliases"`
}

// WebXcodeCloudVersionAliasResult is the safe scalar result for mutation
// receipts and human-readable rendering of one custom version alias. The
// private API's nested build and workflow-summary values are omitted here.
type WebXcodeCloudVersionAliasResult struct {
	ProductID      string `json:"productId"`
	Action         string `json:"action,omitempty"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Locked         bool   `json:"locked"`
	BuildName      string `json:"buildName,omitempty"`
	BuildSupported bool   `json:"buildSupported"`
}

// WebXcodeCloudVersionAliasDeleteResult is the receipt for deleting one
// custom version alias after a successful post-delete verification read.
type WebXcodeCloudVersionAliasDeleteResult struct {
	ProductID string `json:"productId"`
	ID        string `json:"id"`
	Deleted   bool   `json:"deleted"`
}

func webXcodeCloudVersionAliasesRows(r *WebXcodeCloudVersionAliasesResult) ([]string, [][]string) {
	h := []string{"ID", "Name", "Type", "Locked", "Build name", "Build supported"}
	if r == nil {
		return h, nil
	}
	rows := make([][]string, 0, len(r.VersionAliases))
	for _, a := range r.VersionAliases {
		rows = append(rows, []string{a.ID, a.Name, a.Type, strconv.FormatBool(a.Locked), a.BuildName, strconv.FormatBool(a.BuildSupported)})
	}
	return h, rows
}

func webXcodeCloudVersionAliasRows(r *WebXcodeCloudVersionAliasResult) ([]string, [][]string) {
	h := []string{"Product ID", "Action", "ID", "Name", "Type", "Locked", "Build name", "Build supported"}
	if r == nil {
		return h, nil
	}
	return h, [][]string{{
		r.ProductID,
		r.Action,
		r.ID,
		r.Name,
		r.Type,
		strconv.FormatBool(r.Locked),
		r.BuildName,
		strconv.FormatBool(r.BuildSupported),
	}}
}

func webXcodeCloudVersionAliasDeleteRows(r *WebXcodeCloudVersionAliasDeleteResult) ([]string, [][]string) {
	h := []string{"Product ID", "ID", "Deleted"}
	if r == nil {
		return h, nil
	}
	return h, [][]string{{r.ProductID, r.ID, strconv.FormatBool(r.Deleted)}}
}

// WebXcodeCloudNextBuildNumberResult is the current next-build-number setting
// and, after a mutation, its previously observed value. Apple's TestFlight URL
// is intentionally omitted because it may carry sensitive query parameters.
type WebXcodeCloudNextBuildNumberResult struct {
	ProductID               string `json:"productId"`
	PreviousNextBuildNumber *int   `json:"previousNextBuildNumber,omitempty"`
	NextBuildNumber         int    `json:"nextBuildNumber"`
	Updated                 bool   `json:"updated,omitempty"`
}

func webXcodeCloudNextBuildNumberRows(result *WebXcodeCloudNextBuildNumberResult) ([]string, [][]string) {
	if result == nil {
		result = &WebXcodeCloudNextBuildNumberResult{}
	}
	if result.PreviousNextBuildNumber == nil {
		return []string{"Product ID", "Next Build Number"}, [][]string{{
			result.ProductID,
			strconv.Itoa(result.NextBuildNumber),
		}}
	}
	return []string{"Product ID", "Previous Next Build Number", "Next Build Number", "Updated"}, [][]string{{
		result.ProductID,
		strconv.Itoa(*result.PreviousNextBuildNumber),
		strconv.Itoa(result.NextBuildNumber),
		fmt.Sprintf("%t", result.Updated),
	}}
}

// WebXcodeCloudWorkflowsListResult is the computed list of Xcode Cloud workflows
// for a product. The web CI list model only exposes id, name, and description.
type WebXcodeCloudWorkflowsListResult struct {
	ProductID string                          `json:"productId"`
	Workflows []WebXcodeCloudWorkflowListItem `json:"workflows"`
}

// WebXcodeCloudWorkflowListItem is one workflow from the web CI list endpoint.
type WebXcodeCloudWorkflowListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func webXcodeCloudWorkflowsListRows(result *WebXcodeCloudWorkflowsListResult) ([]string, [][]string) {
	headers := []string{"Workflow ID", "Name", "Description"}
	if result == nil {
		return headers, nil
	}
	rows := make([][]string, 0, len(result.Workflows))
	for _, item := range result.Workflows {
		rows = append(rows, []string{item.ID, item.Name, item.Description})
	}
	return headers, rows
}
