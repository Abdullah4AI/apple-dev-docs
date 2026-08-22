package asc

import (
	"fmt"
	"strconv"
	"strings"
)

// XcodeCloudDoctorResult is the computed diagnosis for an Xcode Cloud build run.
type XcodeCloudDoctorResult struct {
	Run              *XcodeCloudStatusResult           `json:"run"`
	Summary          XcodeCloudDoctorSummary           `json:"summary"`
	Actions          []XcodeCloudDoctorAction          `json:"actions"`
	LogBundles       []XcodeCloudDoctorLogBundle       `json:"logBundles"`
	CoverageWarnings []XcodeCloudDoctorCoverageWarning `json:"coverageWarnings"`
	Conclusion       string                            `json:"conclusion"`
	NextAction       string                            `json:"nextAction"`
}

// XcodeCloudDoctorSummary contains aggregate counts for a doctor report.
type XcodeCloudDoctorSummary struct {
	TotalActions        int `json:"totalActions"`
	FailedActions       int `json:"failedActions"`
	CanceledActions     int `json:"canceledActions"`
	SkippedActions      int `json:"skippedActions"`
	Errors              int `json:"errors"`
	Warnings            int `json:"warnings"`
	Artifacts           int `json:"artifacts"`
	LogBundles          int `json:"logBundles"`
	LogBundlesInspected int `json:"logBundlesInspected"`
}

// XcodeCloudDoctorAction contains the issues and artifacts for one build action.
type XcodeCloudDoctorAction struct {
	ID                string                     `json:"id"`
	Name              string                     `json:"name,omitempty"`
	ActionType        string                     `json:"actionType,omitempty"`
	ExecutionProgress string                     `json:"executionProgress,omitempty"`
	CompletionStatus  string                     `json:"completionStatus,omitempty"`
	IsRequiredToPass  *bool                      `json:"isRequiredToPass,omitempty"`
	Issues            []XcodeCloudDoctorIssue    `json:"issues"`
	Artifacts         []XcodeCloudDoctorArtifact `json:"artifacts"`
}

// XcodeCloudDoctorIssue is an issue reported for a build action.
type XcodeCloudDoctorIssue struct {
	ID         string        `json:"id"`
	IssueType  string        `json:"issueType,omitempty"`
	Category   string        `json:"category,omitempty"`
	Message    string        `json:"message,omitempty"`
	FileSource *FileLocation `json:"fileSource,omitempty"`
}

// XcodeCloudDoctorArtifact is an artifact produced by a build action.
type XcodeCloudDoctorArtifact struct {
	ID       string `json:"id"`
	FileType string `json:"fileType,omitempty"`
	FileName string `json:"fileName,omitempty"`
	FileSize int    `json:"fileSize,omitempty"`
}

// XcodeCloudDoctorCoverageWarning explains a diagnostic coverage limitation.
type XcodeCloudDoctorCoverageWarning struct {
	ID          string `json:"id"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

// XcodeCloudDoctorLogBundle records inspection results for a log bundle.
type XcodeCloudDoctorLogBundle struct {
	ArtifactID           string                          `json:"artifactId"`
	ActionID             string                          `json:"actionId"`
	FileName             string                          `json:"fileName,omitempty"`
	FileSize             int                             `json:"fileSize,omitempty"`
	Inspected            bool                            `json:"inspected"`
	SavedPath            string                          `json:"savedPath,omitempty"`
	ExportStatus         string                          `json:"exportStatus,omitempty"`
	Diagnostics          []XcodeCloudDoctorLogDiagnostic `json:"diagnostics"`
	DiagnosticsTruncated bool                            `json:"diagnosticsTruncated,omitempty"`
}

// XcodeCloudDoctorLogDiagnostic is an App Store import diagnostic extracted from a log bundle.
type XcodeCloudDoctorLogDiagnostic struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	SourceFile string `json:"sourceFile,omitempty"`
}

func xcodeCloudDoctorResultTables(result *XcodeCloudDoctorResult, render func([]string, [][]string)) error {
	if result == nil {
		result = &XcodeCloudDoctorResult{}
	}
	run := result.Run
	if run == nil {
		run = &XcodeCloudStatusResult{}
	}

	buildNumber := "n/a"
	if run.BuildNumber > 0 {
		buildNumber = strconv.Itoa(run.BuildNumber)
	}
	render([]string{"Summary field", "Summary value"}, [][]string{
		{"runId", run.BuildRunID},
		{"buildNumber", buildNumber},
		{"executionProgress", xcodeCloudDoctorOrNA(run.ExecutionProgress)},
		{"completionStatus", xcodeCloudDoctorOrNA(run.CompletionStatus)},
		{"totalActions", strconv.Itoa(result.Summary.TotalActions)},
		{"failedActions", strconv.Itoa(result.Summary.FailedActions)},
		{"canceledActions", strconv.Itoa(result.Summary.CanceledActions)},
		{"skippedActions", strconv.Itoa(result.Summary.SkippedActions)},
		{"errors", strconv.Itoa(result.Summary.Errors)},
		{"warnings", strconv.Itoa(result.Summary.Warnings)},
		{"artifacts", strconv.Itoa(result.Summary.Artifacts)},
		{"logBundles", strconv.Itoa(result.Summary.LogBundles)},
		{"logBundlesInspected", strconv.Itoa(result.Summary.LogBundlesInspected)},
		{"conclusion", result.Conclusion},
		{"nextAction", result.NextAction},
	})

	actionRows := make([][]string, 0, len(result.Actions))
	issueRows := make([][]string, 0, result.Summary.Errors+result.Summary.Warnings)
	artifactRows := make([][]string, 0, result.Summary.Artifacts)
	for _, action := range result.Actions {
		actionRows = append(actionRows, []string{
			action.ID,
			action.Name,
			action.ActionType,
			action.ExecutionProgress,
			action.CompletionStatus,
			strconv.Itoa(len(action.Issues)),
			strconv.Itoa(len(action.Artifacts)),
		})
		for _, issue := range action.Issues {
			location := ""
			if issue.FileSource != nil {
				location = issue.FileSource.Path
				if issue.FileSource.LineNumber > 0 {
					location = fmt.Sprintf("%s:%d", location, issue.FileSource.LineNumber)
				}
			}
			issueRows = append(issueRows, []string{action.ID, issue.IssueType, issue.Category, issue.Message, location})
		}
		for _, artifact := range action.Artifacts {
			artifactRows = append(artifactRows, []string{
				action.ID,
				artifact.ID,
				artifact.FileType,
				artifact.FileName,
				strconv.Itoa(artifact.FileSize),
			})
		}
	}
	render([]string{"Actions ID", "Name", "Type", "Progress", "Status", "Issues", "Artifacts"}, actionRows)
	render([]string{"Issues action ID", "Type", "Category", "Message", "Location"}, issueRows)
	render([]string{"Artifacts action ID", "ID", "Type", "Name", "Bytes"}, artifactRows)

	logRows := make([][]string, 0, len(result.LogBundles))
	diagnosticRows := make([][]string, 0)
	for _, bundle := range result.LogBundles {
		codes := make([]string, 0, len(bundle.Diagnostics))
		for _, diagnostic := range bundle.Diagnostics {
			codes = append(codes, diagnostic.Code)
		}
		logRows = append(logRows, []string{
			bundle.ActionID,
			bundle.ArtifactID,
			strconv.FormatBool(bundle.Inspected),
			xcodeCloudDoctorOrNA(bundle.ExportStatus),
			strings.Join(codes, ","),
			strconv.FormatBool(bundle.DiagnosticsTruncated),
			bundle.SavedPath,
		})
		for _, diagnostic := range bundle.Diagnostics {
			diagnosticRows = append(diagnosticRows, []string{
				bundle.ActionID,
				bundle.ArtifactID,
				diagnostic.Code,
				diagnostic.Message,
				diagnostic.SourceFile,
			})
		}
	}
	render([]string{"Log bundles action ID", "Artifact ID", "Inspected", "Export status", "Diagnostics", "Diagnostics truncated", "Saved path"}, logRows)
	render([]string{"Log diagnostics action ID", "Artifact ID", "Code", "Message", "Source file"}, diagnosticRows)

	coverageRows := make([][]string, 0, len(result.CoverageWarnings))
	for _, warning := range result.CoverageWarnings {
		coverageRows = append(coverageRows, []string{warning.ID, warning.Message, warning.Remediation})
	}
	render([]string{"Coverage warning ID", "Message", "Remediation"}, coverageRows)
	return nil
}

func xcodeCloudDoctorOrNA(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "n/a"
	}
	return value
}
