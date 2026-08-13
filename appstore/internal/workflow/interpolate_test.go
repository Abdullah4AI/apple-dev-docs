package workflow

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractDeclaredOutputs_PreservesJSONNumberForm(t *testing.T) {
	tests := []struct {
		name   string
		stdout string
		want   string
	}{
		{name: "integer", stdout: `{"v":42}`, want: "42"},
		{name: "zero", stdout: `{"v":0}`, want: "0"},
		{name: "negative integer", stdout: `{"v":-7}`, want: "-7"},
		{name: "fraction", stdout: `{"v":1.5}`, want: "1.5"},
		{name: "trailing zero fraction", stdout: `{"v":1.50}`, want: "1.50"},
		{name: "large integer beyond float64", stdout: `{"v":9007199254740993}`, want: "9007199254740993"},
		{name: "very small", stdout: `{"v":0.0000001}`, want: "0.0000001"},
		{name: "exponent", stdout: `{"v":1e21}`, want: "1e21"},
		{name: "build number", stdout: `{"v":142}`, want: "142"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outputs, err := extractDeclaredOutputs(map[string]string{"V": "$.v"}, []byte(tc.stdout))
			if err != nil {
				t.Fatalf("extractDeclaredOutputs: %v", err)
			}
			if outputs["V"] != tc.want {
				t.Fatalf("expected V=%q, got %q", tc.want, outputs["V"])
			}
		})
	}
}

func TestExtractDeclaredOutputs_PreservesNestedAndContainerNumbers(t *testing.T) {
	stdout := []byte(`{"nested":{"n":7},"list":[1,2.5,9007199254740993]}`)

	outputs, err := extractDeclaredOutputs(map[string]string{"N": "$.nested.n", "LIST": "$.list"}, stdout)
	if err != nil {
		t.Fatalf("extractDeclaredOutputs: %v", err)
	}
	if outputs["N"] != "7" {
		t.Fatalf("expected N=7, got %q", outputs["N"])
	}
	if outputs["LIST"] != "[1,2.5,9007199254740993]" {
		t.Fatalf("expected LIST to keep JSON number form, got %q", outputs["LIST"])
	}
}

func TestExtractDeclaredOutputs_RejectsTrailingData(t *testing.T) {
	tests := []struct {
		name    string
		stdout  string
		wantErr string
	}{
		{name: "text", stdout: `{"v":1} trailing`, wantErr: "parse command stdout as JSON"},
		{name: "closing brace", stdout: `{"v":1}}`, wantErr: "parse command stdout as JSON"},
		{name: "closing bracket", stdout: `{"v":1}]`, wantErr: "parse command stdout as JSON"},
		{name: "second top-level value", stdout: `{"v":1} {"extra":2}`, wantErr: "unexpected trailing top-level value"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := extractDeclaredOutputs(map[string]string{"V": "$.v"}, []byte(tc.stdout))
			if err == nil {
				t.Fatal("expected trailing data after the JSON value to be rejected")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestRun_NumericOutputInterpolatesUnchanged(t *testing.T) {
	def := &Definition{
		Workflows: map[string]Workflow{
			"release": {
				Steps: []Step{
					{
						Name: "resolve_next_build",
						Run:  `printf '{"nextBuildNumber":42}'`,
						Outputs: map[string]string{
							"BUILD_NUMBER": "$.nextBuildNumber",
						},
					},
					{
						Name: "archive",
						Run:  `echo CURRENT_PROJECT_VERSION=${steps.resolve_next_build.BUILD_NUMBER}`,
					},
				},
			},
		},
	}

	opts := runOpts("release")
	opts.WorkflowFile = filepath.Join(t.TempDir(), "workflow.json")
	opts.StateDir = filepath.Join(t.TempDir(), "runs")

	result, err := Run(context.Background(), def, opts)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := result.Outputs["resolve_next_build"]["BUILD_NUMBER"]; got != "42" {
		t.Fatalf("expected BUILD_NUMBER=42, got %q", got)
	}
	stdout := opts.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "CURRENT_PROJECT_VERSION=42\n") {
		t.Fatalf("expected interpolated command to receive 42, got %q", stdout)
	}
}
