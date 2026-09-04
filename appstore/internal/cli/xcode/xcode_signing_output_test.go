package xcode

import (
	"encoding/json"
	"reflect"
	"testing"

	localxcode "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/xcode"
)

func TestXcodeSigningPlanOutputPreservesArtifactJSONShape(t *testing.T) {
	value := "Manual"
	oldValue := "Automatic"
	plan := &localxcode.SigningPlan{
		SchemaVersion:         1,
		Command:               "asc xcode signing plan",
		GeneratedAt:           "2026-08-30T00:00:00Z",
		PlanHash:              "plan-hash",
		Ready:                 true,
		ProjectPath:           "/tmp/Demo.xcodeproj",
		SettingsFilePath:      "/tmp/settings.json",
		PlanPath:              "/tmp/plan.json",
		ReceiptPath:           "/tmp/receipt.json",
		AllowExternalXCConfig: true,
		Desired: []localxcode.SigningPlanTarget{{
			Target: "Demo",
			Configurations: []localxcode.SigningPlanConfiguration{{
				Name:     "Release",
				Settings: []localxcode.SigningPlanSetting{{Key: "CODE_SIGN_STYLE", Value: &value}},
			}},
		}},
		Files: []localxcode.SigningPlanFile{{
			Path:   "/tmp/Demo.xcodeproj/project.pbxproj",
			SHA256: "source-hash",
			Source: "pbxproj",
		}},
		Changes: []localxcode.SigningSettingChange{{
			Target:        "Demo",
			Configuration: "Release",
			Setting:       "CODE_SIGN_STYLE",
			Operation:     "set",
			Resolution:    "direct",
			OldValue:      &oldValue,
			NewValue:      &value,
			Path:          "/tmp/Demo.xcodeproj/project.pbxproj",
			Source:        "pbxproj",
		}},
		MissingOptionalIncludes: []string{"/tmp/Configs/Missing.xcconfig"},
		Blockers:                []string{},
		Warnings:                []string{"warning"},
	}

	output := newXcodeSigningPlanOutput(plan)
	if output == nil {
		t.Fatal("newXcodeSigningPlanOutput returned nil")
	}
	if reflect.TypeOf(output) == reflect.TypeOf(plan) {
		t.Fatal("expected outward plan output to be distinct from the artifact type")
	}

	assertEquivalentSigningJSON(t, plan, output)
}

func TestXcodeSigningApplyOutputPreservesReceiptJSONShape(t *testing.T) {
	oldValue := "Automatic"
	newValue := "Manual"
	result := &localxcode.SigningApplyResult{
		SchemaVersion: 1,
		AppliedAt:     "2026-08-30T00:00:00Z",
		Completed:     true,
		PlanHash:      "plan-hash",
		PlanPath:      "/tmp/plan.json",
		ReceiptPath:   "/tmp/receipt.json",
		ChangedFiles:  []string{"/tmp/Demo.xcodeproj/project.pbxproj"},
		Files: []localxcode.SigningFileChange{{
			Path:         "/tmp/Demo.xcodeproj/project.pbxproj",
			Source:       "pbxproj",
			BeforeSHA256: "before",
			AfterSHA256:  "after",
		}},
		Changes: []localxcode.SigningSettingChange{{
			Target:        "Demo",
			Configuration: "Release",
			Setting:       "CODE_SIGN_STYLE",
			Operation:     "set",
			Resolution:    "direct",
			OldValue:      &oldValue,
			NewValue:      &newValue,
			Path:          "/tmp/Demo.xcodeproj/project.pbxproj",
			Source:        "pbxproj",
		}},
	}

	output := newXcodeSigningApplyOutput(result)
	if output == nil {
		t.Fatal("newXcodeSigningApplyOutput returned nil")
	}
	assertEquivalentSigningJSON(t, result, output)
}

func TestXcodeSigningOutputConvertersReturnNilForNilInput(t *testing.T) {
	if output := newXcodeSigningPlanOutput(nil); output != nil {
		t.Fatalf("newXcodeSigningPlanOutput(nil) = %v, want nil", output)
	}
	if output := newXcodeSigningApplyOutput(nil); output != nil {
		t.Fatalf("newXcodeSigningApplyOutput(nil) = %v, want nil", output)
	}
}

func TestXcodeSigningOutputConvertersDoNotShareMutableState(t *testing.T) {
	value := "Manual"
	plan := &localxcode.SigningPlan{
		Blockers:                []string{"blocker"},
		Warnings:                []string{"warning"},
		MissingOptionalIncludes: []string{"/tmp/Missing.xcconfig"},
		Changes: []localxcode.SigningSettingChange{{
			Setting:  "CODE_SIGN_STYLE",
			NewValue: &value,
		}},
	}

	output := newXcodeSigningPlanOutput(plan)
	plan.Blockers[0] = "mutated"
	plan.Warnings[0] = "mutated"
	value = "mutated"

	if output.Blockers[0] != "blocker" {
		t.Fatalf("blockers alias the artifact slice: %q", output.Blockers[0])
	}
	if output.Warnings[0] != "warning" {
		t.Fatalf("warnings alias the artifact slice: %q", output.Warnings[0])
	}
	plan.MissingOptionalIncludes[0] = "mutated"
	if output.MissingOptionalIncludes[0] != "/tmp/Missing.xcconfig" {
		t.Fatalf("missing optional includes alias the artifact slice: %q", output.MissingOptionalIncludes[0])
	}
	if got := *output.Changes[0].NewValue; got != "Manual" {
		t.Fatalf("change value aliases the artifact pointer: %q", got)
	}
}

func assertEquivalentSigningJSON(t *testing.T, want, got any) {
	t.Helper()

	var wantValue any
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want JSON: %v", err)
	}
	if err := json.Unmarshal(wantJSON, &wantValue); err != nil {
		t.Fatalf("decode want JSON: %v", err)
	}

	var gotValue any
	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal got JSON: %v", err)
	}
	if err := json.Unmarshal(gotJSON, &gotValue); err != nil {
		t.Fatalf("decode got JSON: %v", err)
	}

	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("outward JSON changed artifact shape:\nwant: %s\n got: %s", wantJSON, gotJSON)
	}
}
