package asc

import (
	"strings"
	"testing"
)

func TestPrintTableWebAppDeclarationListUsesRegistry(t *testing.T) {
	result := WebAppDeclarationList{
		{
			AppID:           "app-1",
			RequirementID:   "req-1",
			RequirementName: "MEDICAL_DEVICE",
			Status:          "COLLECTED",
			FormID:          "form-1",
			Required:        true,
		},
		{
			AppID:           "app-1",
			RequirementID:   "req-2",
			RequirementName: "OTHER_REQUIREMENT",
			Status:          "PENDING_COLLECTION",
			Required:        false,
		},
	}

	output := captureStdout(t, func() error { return PrintTable(&result) })
	for _, want := range []string{
		"Requirement", "Status", "Required", "Requirement ID", "Form ID",
		"MEDICAL_DEVICE", "COLLECTED", "true", "req-1", "form-1",
		"OTHER_REQUIREMENT", "PENDING_COLLECTION", "false", "req-2", "n/a",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("table output missing %q: %q", want, output)
		}
	}
}

func TestPrintTableWebMedicalDeviceDeclarationStateUsesRegistry(t *testing.T) {
	result := &WebMedicalDeviceDeclarationState{
		AppID:              "app-1",
		RequirementName:    "MEDICAL_DEVICE",
		Declaration:        "no",
		Status:             "COLLECTED",
		Required:           true,
		CountriesOrRegions: []string{"US", "EEA"},
	}

	output := captureStdout(t, func() error { return PrintTable(result) })
	for _, want := range []string{"app-1", "MEDICAL_DEVICE", "no", "COLLECTED", "true", "US,EEA"} {
		if !strings.Contains(output, want) {
			t.Fatalf("table output missing %q: %q", want, output)
		}
	}
}

func TestPrintMarkdownWebMedicalDeviceDeclarationStateOmitsBlankDeclaration(t *testing.T) {
	result := &WebMedicalDeviceDeclarationState{
		AppID:           "app-1",
		RequirementName: "MEDICAL_DEVICE",
		Required:        true,
	}

	output := captureStdout(t, func() error { return PrintMarkdown(result) })
	for _, want := range []string{"app-1", "MEDICAL_DEVICE", "n/a"} {
		if !strings.Contains(output, want) {
			t.Fatalf("markdown output missing %q: %q", want, output)
		}
	}
}

func TestPrintTableWebMedicalDeviceDeclarationResultUsesRegistry(t *testing.T) {
	result := &WebMedicalDeviceDeclarationResult{
		AppID:              "app-1",
		RequirementName:    "MEDICAL_DEVICE",
		Declared:           false,
		Changed:            false,
		Status:             "COLLECTED",
		CountriesOrRegions: []string{"US", "EEA"},
	}

	output := captureStdout(t, func() error { return PrintTable(result) })
	for _, want := range []string{"app-1", "MEDICAL_DEVICE", "false", "COLLECTED", "US,EEA"} {
		if !strings.Contains(output, want) {
			t.Fatalf("table output missing %q: %q", want, output)
		}
	}
}
