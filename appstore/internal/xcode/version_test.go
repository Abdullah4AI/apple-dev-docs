package xcode

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetVersion_NotMacOS(t *testing.T) {
	projectDir := writeLegacyVersionProject(t)
	prev := runtimeGOOS
	runtimeGOOS = "linux"
	defer func() { runtimeGOOS = prev }()

	_, err := GetVersion(context.Background(), projectDir, "")
	if err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Fatalf("expected macOS error, got: %v", err)
	}
}

func TestSetVersion_NotMacOS(t *testing.T) {
	projectDir := writeLegacyVersionProject(t)
	prev := runtimeGOOS
	runtimeGOOS = "linux"
	defer func() { runtimeGOOS = prev }()

	_, err := SetVersion(context.Background(), SetVersionOptions{ProjectDir: projectDir, Version: "1.0.0"})
	if err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Fatalf("expected macOS error, got: %v", err)
	}
}

func TestBumpVersion_NotMacOS(t *testing.T) {
	projectDir := writeLegacyVersionProject(t)
	prev := runtimeGOOS
	runtimeGOOS = "linux"
	defer func() { runtimeGOOS = prev }()

	_, err := BumpVersion(context.Background(), BumpVersionOptions{ProjectDir: projectDir, BumpType: BumpPatch})
	if err == nil || !strings.Contains(err.Error(), "macOS") {
		t.Fatalf("expected macOS error, got: %v", err)
	}
}

func TestGetVersion_MissingAgvtool(t *testing.T) {
	projectDir := writeLegacyVersionProject(t)
	prev := lookPathFn
	lookPathFn = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}
	defer func() { lookPathFn = prev }()

	prevOS := runtimeGOOS
	runtimeGOOS = "darwin"
	defer func() { runtimeGOOS = prevOS }()

	_, err := GetVersion(context.Background(), projectDir, "")
	if err == nil || !strings.Contains(err.Error(), "agvtool") {
		t.Fatalf("expected agvtool not found error, got: %v", err)
	}
}

func TestBumpVersionType_Validate(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"major", true},
		{"minor", true},
		{"patch", true},
		{"build", true},
		{"MAJOR", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		_, err := ParseBumpType(tt.input)
		if tt.valid && err != nil {
			t.Errorf("ParseBumpType(%q) unexpected error: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("ParseBumpType(%q) expected error", tt.input)
		}
	}
}

func TestIsVariableReference(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"$(MARKETING_VERSION)", true},
		{"1.2.3", false},
		{"$(CURRENT_PROJECT_VERSION)", true},
		{"", false},
	}

	for _, tt := range tests {
		if got := isVariableReference(tt.input); got != tt.want {
			t.Errorf("isVariableReference(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestIncrementBuildString(t *testing.T) {
	tests := []struct {
		current string
		want    string
		wantErr bool
	}{
		{"42", "43", false},
		{"1", "2", false},
		{"1.2.3", "1.2.4", false},
		{"", "", true},
		{"abc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.current, func(t *testing.T) {
			got, err := incrementBuildString(tt.current)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("incrementBuildString(%q) = %q, want %q", tt.current, got, tt.want)
			}
		})
	}
}

func TestParseAgvtoolVersionOutput_PreservesVariableAndLiteralValues(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
		modern bool
	}{
		{name: "modern", output: "App=$(MARKETING_VERSION)\n", want: "$(MARKETING_VERSION)", modern: true},
		{name: "literal", output: "App=1.2.3\n", want: "1.2.3", modern: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAgvtoolVersionOutput(tt.output, "")
			if err != nil {
				t.Fatalf("parseAgvtoolVersionOutput() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseAgvtoolVersionOutput() = %q, want %q", got, tt.want)
			}
			if modern := isVariableReference(got); modern != tt.modern {
				t.Fatalf("isVariableReference(%q) = %t, want %t", got, modern, tt.modern)
			}
		})
	}
}

func TestParseAgvtoolVersionOutput_TargetFilter(t *testing.T) {
	multiTargetOutput := "App=1.2.3\nExtension=2.0.0\n"

	got, err := parseAgvtoolVersionOutput(multiTargetOutput, "Extension")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2.0.0" {
		t.Fatalf("expected Extension version, got %q", got)
	}
}

func TestParseAgvtoolVersionOutput_RequiresTargetForAmbiguousOutput(t *testing.T) {
	_, err := parseAgvtoolVersionOutput("App=1.2.3\nExtension=2.0.0\n", "")
	if err == nil || !strings.Contains(err.Error(), "use --target") {
		t.Fatalf("expected ambiguous target error, got %v", err)
	}
}

func TestParseAgvtoolVersionOutput_MissingTargetErrors(t *testing.T) {
	_, err := parseAgvtoolVersionOutput("App=1.2.3\nExtension=2.0.0\n", "Widget")
	if err == nil || !strings.Contains(err.Error(), `target "Widget" not found`) {
		t.Fatalf("expected missing target error, got %v", err)
	}
}

func TestParseAgvtoolVersionOutput_RejectsDivergentConfigurationsForTarget(t *testing.T) {
	_, err := parseAgvtoolVersionOutput("App=1.2.3\nApp=2.0.0\n", "App")
	if err == nil || !strings.Contains(err.Error(), "differing values") {
		t.Fatalf("expected divergent configuration error, got %v", err)
	}
}

func TestParseAgvtoolVersionOutput_RejectsDivergentConfigurationsWithoutTarget(t *testing.T) {
	_, err := parseAgvtoolVersionOutput("App=1.2.3\nApp=2.0.0\n", "")
	if err == nil || !strings.Contains(err.Error(), "differing values") {
		t.Fatalf("expected divergent configuration error, got %v", err)
	}
}

func TestParseAgvtoolBuildOutput_TargetFilter(t *testing.T) {
	got, err := parseAgvtoolBuildOutput("App=41\nExtension=7\n", "App")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "41" {
		t.Fatalf("expected App build number, got %q", got)
	}
}

func TestBuildSettingsTargetNames(t *testing.T) {
	output := `
Build settings for action build and target App:
    MARKETING_VERSION = 1.2.3

Build settings for action build and target App:
    CURRENT_PROJECT_VERSION = 42

Build settings for action build and target Extension:
    MARKETING_VERSION = 2.0.0
`

	targets := buildSettingsTargetNames(output)
	if len(targets) != 2 || targets[0] != "App" || targets[1] != "Extension" {
		t.Fatalf("expected target names [App Extension], got %v", targets)
	}
}

func TestFindXcodeprojAcceptsExplicitProjectPath(t *testing.T) {
	tempDir := t.TempDir()
	appProject := filepath.Join(tempDir, "App.xcodeproj")
	podsProject := filepath.Join(tempDir, "Pods.xcodeproj")
	if err := os.MkdirAll(appProject, 0o755); err != nil {
		t.Fatalf("mkdir app project: %v", err)
	}
	if err := os.MkdirAll(podsProject, 0o755); err != nil {
		t.Fatalf("mkdir pods project: %v", err)
	}

	got, err := findXcodeproj(appProject)
	if err != nil {
		t.Fatalf("expected explicit project path to succeed, got %v", err)
	}
	if got != appProject {
		t.Fatalf("expected %q, got %q", appProject, got)
	}
}

func TestFindXcodeprojDoesNotRetargetTrailingWhitespaceProjectPath(t *testing.T) {
	tempDir := t.TempDir()
	exactProject := filepath.Join(tempDir, "Foo.xcodeproj")
	whitespaceProject := exactProject + " "
	if err := os.MkdirAll(exactProject, 0o755); err != nil {
		t.Fatalf("mkdir exact project: %v", err)
	}
	if err := os.MkdirAll(whitespaceProject, 0o755); err != nil {
		t.Fatalf("mkdir whitespace project: %v", err)
	}

	got, err := findXcodeproj(whitespaceProject)
	if err == nil {
		t.Fatalf("findXcodeproj(%q) = %q, want no nested project error", whitespaceProject, got)
	}
	if got == exactProject {
		t.Fatalf("trailing-whitespace path was retargeted to %q", exactProject)
	}
}

func TestFindXcodeprojPreservesWhitespaceProjectDirectory(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, " Project Root ")
	projectPath := filepath.Join(projectDir, "Demo.xcodeproj")
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatalf("mkdir whitespace project directory: %v", err)
	}

	got, err := findXcodeproj(projectDir)
	if err != nil {
		t.Fatalf("findXcodeproj(%q) error = %v", projectDir, err)
	}
	if got != projectPath {
		t.Fatalf("findXcodeproj(%q) = %q, want %q", projectDir, got, projectPath)
	}
}

func TestFindXcodeprojMultipleProjectsSuggestsProjectFlag(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "App.xcodeproj"), 0o755); err != nil {
		t.Fatalf("mkdir app project: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "Pods.xcodeproj"), 0o755); err != nil {
		t.Fatalf("mkdir pods project: %v", err)
	}

	_, err := findXcodeproj(tempDir)
	if err == nil || !strings.Contains(err.Error(), "use --project to pick one") {
		t.Fatalf("expected actionable multiple project error, got %v", err)
	}
}

func TestGetVersionDiscoveryErrorDoesNotFallBackToAgvtool(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "commands.log")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	commandContextFn = helperCommandContext(t, logPath)
	t.Cleanup(restore)

	_, err := GetVersionScoped(context.Background(), GetVersionOptions{ProjectDir: tempDir})
	if err == nil || !strings.Contains(err.Error(), "no .xcodeproj found") {
		t.Fatalf("expected project discovery error, got %v", err)
	}
	if logData, readErr := os.ReadFile(logPath); readErr == nil && strings.Contains(string(logData), "agvtool") {
		t.Fatalf("project discovery error fell back to agvtool: %q", logData)
	}
}

func TestGetVersionScopedNeverDoesNotLaunchXcodebuildSettingsFallback(t *testing.T) {
	projectDir := writeLegacyVersionProject(t)
	logPath := filepath.Join(t.TempDir(), "commands.log")
	var diagnostic bytes.Buffer

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	commandContextFn = helperCommandContext(t, logPath)
	t.Setenv("ASC_XCODE_HELPER_VARIABLE_VERSION", "1")
	t.Setenv("ASC_XCODE_HELPER_SINGLE_TARGET", "1")
	t.Cleanup(restore)

	_, err := GetVersionScoped(context.Background(), GetVersionOptions{
		ProjectDir:              projectDir,
		BuildSettingsLookup:     BuildSettingsLookupNever,
		BuildSettingsDiagnostic: &diagnostic,
	})
	if err == nil || !strings.Contains(err.Error(), "--xcodebuild-settings-lookup never") {
		t.Fatalf("expected disabled fallback error, got %v", err)
	}
	if !strings.Contains(err.Error(), marketingVersionSetting) || !strings.Contains(err.Error(), currentProjectSetting) {
		t.Fatalf("expected setting remediation in error, got %v", err)
	}
	if diagnostic.String() != "" {
		t.Fatalf("disabled fallback emitted a warning: %q", diagnostic.String())
	}
	logData, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("read helper log: %v", readErr)
	}
	if strings.Contains(string(logData), "xcodebuild|-showBuildSettings") {
		t.Fatalf("never policy launched xcodebuild: %q", logData)
	}
}

func TestGetVersionScopedAutoWarnsBeforeXcodebuildSettingsFallback(t *testing.T) {
	projectDir := writeLegacyVersionProject(t)
	logPath := filepath.Join(t.TempDir(), "commands.log")
	var diagnostic bytes.Buffer

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	helpers := helperCommandContext(t, logPath)
	commandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "xcodebuild" && len(args) > 0 && args[0] == "-showBuildSettings" {
			warning := diagnostic.String()
			for _, want := range []string{marketingVersionSetting, currentProjectSetting, "--xcodebuild-settings-lookup never"} {
				if !strings.Contains(warning, want) {
					t.Fatalf("warning was not emitted before xcodebuild command creation or omitted %q: %q", want, warning)
				}
			}
		}
		return helpers(ctx, name, args...)
	}
	t.Setenv("ASC_XCODE_HELPER_VARIABLE_VERSION", "1")
	t.Setenv("ASC_XCODE_HELPER_SINGLE_TARGET", "1")
	t.Cleanup(restore)

	result, err := GetVersionScoped(context.Background(), GetVersionOptions{
		ProjectDir:              projectDir,
		BuildSettingsLookup:     BuildSettingsLookupAuto,
		BuildSettingsDiagnostic: &diagnostic,
	})
	if err != nil {
		t.Fatalf("GetVersionScoped() error = %v", err)
	}
	if result.Version != "4.5.6" || result.BuildNumber != "99" {
		t.Fatalf("fallback result = %#v", result)
	}
	logData, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("read helper log: %v", readErr)
	}
	if !strings.Contains(string(logData), "xcodebuild|-showBuildSettings") {
		t.Fatalf("auto policy did not launch fallback: %q", logData)
	}
}

func TestGetVersionScopedStructuredParsingRemainsSilent(t *testing.T) {
	projectPath := writeStructuredVersionProject(t, true)
	var diagnostic bytes.Buffer
	called := false

	restore := overrideTestEnvironment(t)
	commandContextFn = func(context.Context, string, ...string) *exec.Cmd {
		called = true
		return nil
	}
	t.Cleanup(restore)

	result, err := GetVersionScoped(context.Background(), GetVersionOptions{
		ProjectDir:              projectPath,
		BuildSettingsLookup:     BuildSettingsLookupAuto,
		BuildSettingsDiagnostic: &diagnostic,
	})
	if err != nil {
		t.Fatalf("GetVersionScoped() error = %v", err)
	}
	if result.Version == "" || result.BuildNumber == "" {
		t.Fatalf("structured result = %#v", result)
	}
	if called {
		t.Fatal("structured parsing created a subprocess")
	}
	if diagnostic.String() != "" {
		t.Fatalf("structured parsing emitted a diagnostic: %q", diagnostic.String())
	}
}

func TestBumpVersionLegacyBuildWarnsOnceAcrossPostMutationReread(t *testing.T) {
	projectDir := writeLegacyVersionProject(t)
	logPath := filepath.Join(t.TempDir(), "commands.log")
	var diagnostic bytes.Buffer

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	commandContextFn = helperCommandContext(t, logPath)
	t.Setenv("ASC_XCODE_HELPER_VARIABLE_VERSION", "1")
	t.Setenv("ASC_XCODE_HELPER_SINGLE_TARGET", "1")
	t.Cleanup(restore)

	_, err := BumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir:              projectDir,
		BumpType:                BumpBuild,
		BuildSettingsLookup:     BuildSettingsLookupAuto,
		BuildSettingsDiagnostic: &diagnostic,
	})
	if err != nil {
		t.Fatalf("BumpVersion() error = %v", err)
	}
	if got := strings.Count(diagnostic.String(), "Warning: structured project parsing"); got != 1 {
		t.Fatalf("fallback warning count = %d, want 1; stderr = %q", got, diagnostic.String())
	}
	logData, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("read helper log: %v", readErr)
	}
	if got := strings.Count(string(logData), "xcodebuild|-showBuildSettings"); got != 2 {
		t.Fatalf("xcodebuild fallback count = %d, want 2 fresh reads across mutation; log = %q", got, logData)
	}
}

func TestBuildSettingsLookupIsSharedAcrossPreMutationBumpPhases(t *testing.T) {
	projectDir := writeLegacyVersionProject(t)
	logPath := filepath.Join(t.TempDir(), "commands.log")
	var diagnostic bytes.Buffer

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	commandContextFn = helperCommandContext(t, logPath)
	t.Setenv("ASC_XCODE_HELPER_VARIABLE_VERSION", "1")
	t.Setenv("ASC_XCODE_HELPER_SINGLE_TARGET", "1")
	t.Cleanup(restore)
	lookupSession := NewBuildSettingsLookupSession(BuildSettingsLookupAuto, &diagnostic)

	if err := ValidateBumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir:              projectDir,
		BumpType:                BumpBuild,
		BuildNumber:             "108",
		BuildSettingsLookup:     BuildSettingsLookupAuto,
		BuildSettingsDiagnostic: &diagnostic,
		BuildSettingsSession:    lookupSession,
	}); err != nil {
		t.Fatalf("ValidateBumpVersion() error = %v", err)
	}
	if _, err := GetConsistentMarketingVersion(context.Background(), GetVersionOptions{
		ProjectDir:              projectDir,
		BuildSettingsLookup:     BuildSettingsLookupAuto,
		BuildSettingsDiagnostic: &diagnostic,
		BuildSettingsSession:    lookupSession,
	}); err != nil {
		t.Fatalf("GetConsistentMarketingVersion() error = %v", err)
	}
	if _, err := BumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir:              projectDir,
		BumpType:                BumpBuild,
		BuildNumber:             "108",
		BuildSettingsLookup:     BuildSettingsLookupAuto,
		BuildSettingsDiagnostic: &diagnostic,
		BuildSettingsSession:    lookupSession,
	}); err != nil {
		t.Fatalf("BumpVersion() error = %v", err)
	}

	if got := strings.Count(diagnostic.String(), "Warning: structured project parsing"); got != 1 {
		t.Fatalf("fallback warning count = %d, want 1; stderr = %q", got, diagnostic.String())
	}
	logData, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("read helper log: %v", readErr)
	}
	if got := strings.Count(string(logData), "xcodebuild|-showBuildSettings"); got != 1 {
		t.Fatalf("xcodebuild fallback count = %d, want 1 shared pre-mutation read; log = %q", got, logData)
	}
}

func TestSetVersionTargetedWritesRequireStructuredProject(t *testing.T) {
	prevOS := runtimeGOOS
	prevLookPath := lookPathFn
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	defer func() {
		runtimeGOOS = prevOS
		lookPathFn = prevLookPath
	}()

	_, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: writeLegacyVersionProject(t),
		Target:     "App",
		Version:    "1.2.3",
	})
	if err == nil || !strings.Contains(err.Error(), "scoped edits require structured Xcode build settings") {
		t.Fatalf("expected structured-project requirement, got %v", err)
	}
}

func TestSetVersionLegacyMultiTargetUsesProjectWideWrite(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := writeLegacyVersionProject(t)
	logPath := filepath.Join(tempDir, "commands.log")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	commandContextFn = helperCommandContext(t, logPath)
	t.Cleanup(restore)

	result, err := SetVersion(context.Background(), SetVersionOptions{
		ProjectDir: projectDir,
		Version:    "1.3.0",
	})
	if err != nil {
		t.Fatalf("expected project-wide edit to succeed, got %v", err)
	}
	if result.Version != "1.3.0" {
		t.Fatalf("expected edited version in result, got %#v", result)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read helper log: %v", err)
	}
	logText := string(logData)
	if !strings.Contains(logText, "agvtool|new-marketing-version|1.3.0") {
		t.Fatalf("expected project-wide marketing version update, got %q", logText)
	}
	if strings.Contains(logText, "agvtool|what-version|-terse") {
		t.Fatalf("expected edit path to avoid ambiguous build-number reads, got %q", logText)
	}
}

func TestBumpVersionLegacyTargetScopeIsRejectedBeforeProjectWideWrite(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := writeLegacyVersionProject(t)
	logPath := filepath.Join(tempDir, "commands.log")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	commandContextFn = helperCommandContext(t, logPath)
	t.Cleanup(restore)

	_, err := BumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir: projectDir,
		Target:     "Extension",
		BumpType:   BumpPatch,
	})
	if err == nil || !strings.Contains(err.Error(), "scoped bumps require structured") {
		t.Fatalf("expected structured-project requirement, got %v", err)
	}

	if logData, readErr := os.ReadFile(logPath); readErr == nil && strings.Contains(string(logData), "new-marketing-version") {
		t.Fatalf("legacy target scope performed a project-wide write: %q", logData)
	}
}

func TestBumpVersionLegacyRemoteBuildUsesRequestedNumber(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := writeLegacyVersionProject(t)
	logPath := filepath.Join(tempDir, "commands.log")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}
	commandContextFn = helperCommandContext(t, logPath)
	t.Setenv("ASC_XCODE_HELPER_SINGLE_TARGET", "1")
	t.Cleanup(restore)

	result, err := BumpVersion(context.Background(), BumpVersionOptions{
		ProjectDir:  projectDir,
		BumpType:    BumpBuild,
		BuildNumber: "108",
	})
	if err != nil {
		t.Fatalf("BumpVersion() error = %v", err)
	}
	if result.OldBuild != "41" || result.NewBuild != "108" {
		t.Fatalf("unexpected result: %#v", result)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read helper log: %v", err)
	}
	logText := string(logData)
	if !strings.Contains(logText, "agvtool|new-version|-all|108") {
		t.Fatalf("expected requested legacy build update, got %q", logText)
	}
	if strings.Contains(logText, "agvtool|next-version|-all") {
		t.Fatalf("remote build bump incremented locally: %q", logText)
	}
}

func TestGetConsistentMarketingVersionLegacyRejectsDivergentConfigurations(t *testing.T) {
	projectDir := writeLegacyVersionProject(t)
	logPath := filepath.Join(t.TempDir(), "commands.log")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(file string) (string, error) { return "/usr/bin/" + file, nil }
	commandContextFn = helperCommandContext(t, logPath)
	t.Setenv("ASC_XCODE_HELPER_DIVERGENT_CONFIGURATIONS", "1")
	t.Setenv("ASC_XCODE_HELPER_SINGLE_TARGET", "1")
	t.Cleanup(restore)

	_, err := GetConsistentMarketingVersion(context.Background(), GetVersionOptions{ProjectDir: projectDir})
	if err == nil || !strings.Contains(err.Error(), "differing values") {
		t.Fatalf("expected divergent legacy configuration error, got %v", err)
	}
}

func writeLegacyVersionProject(t *testing.T) string {
	t.Helper()
	projectPath := writeStructuredVersionProject(t, false)
	pbxprojPath := filepath.Join(projectPath, "project.pbxproj")
	contents, err := os.ReadFile(pbxprojPath)
	if err != nil {
		t.Fatalf("read legacy project fixture: %v", err)
	}
	legacy := strings.ReplaceAll(string(contents), marketingVersionSetting, "LEGACY_MARKETING_VERSION")
	legacy = strings.ReplaceAll(legacy, currentProjectSetting, "LEGACY_CURRENT_PROJECT_VERSION")
	if err := os.WriteFile(pbxprojPath, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy project fixture: %v", err)
	}
	return filepath.Dir(projectPath)
}

func TestBumpVersionString(t *testing.T) {
	tests := []struct {
		current  string
		bumpType BumpType
		want     string
		wantErr  bool
	}{
		{"1.2.3", BumpPatch, "1.2.4", false},
		{"1.2.3", BumpMinor, "1.3.0", false},
		{"1.2.3", BumpMajor, "2.0.0", false},
		{"1.0", BumpPatch, "", true},
		{"1.0", BumpMinor, "1.1", false},
		{"1.0", BumpMajor, "2.0", false},
		{"bad", BumpPatch, "", true},
		{"", BumpPatch, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_"+string(tt.bumpType), func(t *testing.T) {
			got, err := bumpVersionString(tt.current, tt.bumpType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("bumpVersionString(%q, %s) = %q, want %q", tt.current, tt.bumpType, got, tt.want)
			}
		})
	}
}
