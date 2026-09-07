package asc

import (
	"strings"
	"testing"
)

func TestPrintTableWebXcodeCloudWorkflowsListUsesRegistry(t *testing.T) {
	result := &WebXcodeCloudWorkflowsListResult{
		ProductID: "prod-1",
		Workflows: []WebXcodeCloudWorkflowListItem{
			{ID: "wf-1", Name: "TestFlight Deploy", Description: "Build on main"},
			{ID: "wf-2", Name: "PR Check"},
		},
	}

	output := captureStdout(t, func() error { return PrintTable(result) })
	for _, want := range []string{"Workflow ID", "Name", "Description", "wf-1", "TestFlight Deploy", "Build on main", "wf-2", "PR Check"} {
		if !strings.Contains(output, want) {
			t.Fatalf("table output missing %q: %q", want, output)
		}
	}
}

func TestPrintMarkdownWebXcodeCloudWorkflowsListUsesRegistry(t *testing.T) {
	result := &WebXcodeCloudWorkflowsListResult{
		ProductID: "prod-1",
		Workflows: []WebXcodeCloudWorkflowListItem{
			{ID: "wf-1", Name: "Nightly"},
		},
	}

	output := captureStdout(t, func() error { return PrintMarkdown(result) })
	for _, want := range []string{"Workflow ID", "Name", "Description", "wf-1", "Nightly"} {
		if !strings.Contains(output, want) {
			t.Fatalf("markdown output missing %q: %q", want, output)
		}
	}
}

func TestPrintWebXcodeCloudNextBuildNumberUsesRegistry(t *testing.T) {
	previous := 101
	result := &WebXcodeCloudNextBuildNumberResult{
		ProductID:               "prod-1",
		PreviousNextBuildNumber: &previous,
		NextBuildNumber:         102,
		Updated:                 true,
	}

	table := captureStdout(t, func() error { return PrintTable(result) })
	markdown := captureStdout(t, func() error { return PrintMarkdown(result) })
	for name, output := range map[string]string{"table": table, "markdown": markdown} {
		for _, want := range []string{"Product ID", "Previous Next Build Number", "Next Build Number", "Updated", "prod-1", "101", "102", "true"} {
			if !strings.Contains(output, want) {
				t.Fatalf("%s output missing %q: %q", name, want, output)
			}
		}
	}
}

func TestPrintWebXcodeCloudVersionAliasesUsesRegistry(t *testing.T) {
	aliases := &WebXcodeCloudVersionAliasesResult{
		ProductID: "prod-1",
		VersionAliases: []WebXcodeCloudVersionAlias{
			{ID: "alias-1", Name: "Release", Type: "CUSTOM", Locked: true, BuildName: "42", BuildSupported: true},
		},
	}
	outputs := map[string]string{
		"list table":    captureStdout(t, func() error { return PrintTable(aliases) }),
		"list markdown": captureStdout(t, func() error { return PrintMarkdown(aliases) }),
	}
	for name, output := range outputs {
		for _, want := range []string{"ID", "Name", "Type", "Locked", "Build name", "Build supported", "alias-1", "Release", "CUSTOM", "42", "true"} {
			if !strings.Contains(output, want) {
				t.Fatalf("%s output missing %q: %q", name, want, output)
			}
		}
	}
}

func TestPrintWebXcodeCloudVersionAliasUsesRegistry(t *testing.T) {
	result := &WebXcodeCloudVersionAliasResult{
		ProductID:      "prod-1",
		Action:         "updated",
		ID:             "alias-1",
		Name:           "Release",
		Type:           "xcode_version",
		Locked:         true,
		BuildName:      "42",
		BuildSupported: true,
	}
	outputs := map[string]string{
		"table":    captureStdout(t, func() error { return PrintTable(result) }),
		"markdown": captureStdout(t, func() error { return PrintMarkdown(result) }),
	}
	for name, output := range outputs {
		for _, want := range []string{"Product ID", "Action", "ID", "Name", "Type", "Locked", "Build name", "Build supported", "prod-1", "updated", "alias-1", "Release", "xcode_version", "42", "true"} {
			if !strings.Contains(output, want) {
				t.Fatalf("%s output missing %q: %q", name, want, output)
			}
		}
	}
}

func TestPrintWebXcodeCloudVersionAliasDeleteUsesRegistry(t *testing.T) {
	result := &WebXcodeCloudVersionAliasDeleteResult{ProductID: "prod-1", ID: "alias-1", Deleted: true}
	outputs := map[string]string{
		"table":    captureStdout(t, func() error { return PrintTable(result) }),
		"markdown": captureStdout(t, func() error { return PrintMarkdown(result) }),
	}
	for name, output := range outputs {
		for _, want := range []string{"Product ID", "ID", "Deleted", "prod-1", "alias-1", "true"} {
			if !strings.Contains(output, want) {
				t.Fatalf("%s output missing %q: %q", name, want, output)
			}
		}
	}
}
