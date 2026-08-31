package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestXcodeToolchainDoctorResultUsesCamelCaseJSONContract(t *testing.T) {
	beta := true
	result := &XcodeToolchainDoctorResult{
		Status:       "warn",
		Source:       "environment",
		DeveloperDir: "/Applications/Xcode-beta.app/Contents/Developer",
		XcodePath:    "/Applications/Xcode-beta.app",
		XcodeVersion: "16.4 beta 2",
		XcodeBuild:   "16F6",
		Beta:         &beta,
		Checks: []XcodeToolchainDoctorCheck{{
			Name:    "developer_dir",
			Status:  "ok",
			Path:    "/Applications/Xcode-beta.app/Contents/Developer",
			Message: "developer directory is available",
		}},
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	for _, field := range []string{"developerDir", "xcodePath", "xcodeVersion", "xcodeBuild"} {
		if _, ok := fields[field]; !ok {
			t.Fatalf("JSON missing camelCase field %q: %s", field, encoded)
		}
	}
	for _, field := range []string{"developer_dir", "xcode_path", "xcode_version", "xcode_build"} {
		if _, ok := fields[field]; ok {
			t.Fatalf("JSON contains internal snake_case field %q: %s", field, encoded)
		}
	}
}

func TestPrintTable_XcodeToolchainDoctorResultUsesRegisteredRenderer(t *testing.T) {
	result := &XcodeToolchainDoctorResult{
		Status:       "fail",
		Source:       "xcode-select",
		DeveloperDir: "/Applications/Missing.xcode/Contents/Developer",
		Checks: []XcodeToolchainDoctorCheck{{
			Name:    "developer_dir",
			Status:  "fail",
			Message: "developer directory is unavailable",
		}},
	}

	output := captureStdout(t, func() error {
		return PrintTable(result)
	})
	for _, expected := range []string{"check", "developer_dir", "fail", "developer directory is unavailable"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("registered renderer output missing %q: %s", expected, output)
		}
	}
}

func TestXcodeToolchainDoctorResultRowsDoesNotDuplicateBetaCheck(t *testing.T) {
	beta := true
	result := &XcodeToolchainDoctorResult{
		Status: "warn",
		Beta:   &beta,
		Checks: []XcodeToolchainDoctorCheck{{
			Name:    "beta",
			Status:  "warn",
			Message: "selected developer directory appears to be a beta Xcode build",
		}},
	}

	_, rows := xcodeToolchainDoctorResultRows(result)
	betaRows := 0
	for _, row := range rows {
		if len(row) > 0 && row[0] == "beta" {
			betaRows++
		}
	}
	if betaRows != 1 {
		t.Fatalf("xcodeToolchainDoctorResultRows() emitted %d beta rows, want exactly one: %v", betaRows, rows)
	}
}
