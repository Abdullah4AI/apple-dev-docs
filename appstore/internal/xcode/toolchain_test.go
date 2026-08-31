package xcode

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestToolchainReportCarriesNoJSONContract guards against the probe-layer
// structs re-establishing a serialization contract. JSON output for
// `asc xcode doctor` must flow exclusively through the exported camelCase
// structs in internal/asc/output_xcode_toolchain.go.
func TestToolchainReportCarriesNoJSONContract(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(ToolchainReport{}),
		reflect.TypeOf(ToolchainCheck{}),
	} {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if tag, ok := field.Tag.Lookup("json"); ok {
				t.Fatalf("%s.%s carries json tag %q; serialize through internal/asc output structs instead", typ.Name(), field.Name, tag)
			}
		}
	}
}

func TestParseXcodeVersion(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantVersion string
		wantBuild   string
		wantErr     string
	}{
		{
			name:        "stable",
			output:      "Xcode 16.4\nBuild version 16F6\n",
			wantVersion: "16.4",
			wantBuild:   "16F6",
		},
		{
			name:        "warning before version",
			output:      "warning\nXcode 27.0 beta 4\nBuild version 27A5228h\n",
			wantVersion: "27.0 beta 4",
			wantBuild:   "27A5228h",
		},
		{
			name:    "empty",
			wantErr: `unexpected xcodebuild -version output: "empty output"`,
		},
		{
			name:    "missing build",
			output:  "Xcode 16.4\n",
			wantErr: "missing Build version",
		},
		{
			name:    "malformed",
			output:  "Command Line Tools 16.0\n",
			wantErr: "unexpected xcodebuild -version output",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseXcodeVersion(test.output)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("parseXcodeVersion() error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseXcodeVersion() error = %v", err)
			}
			if got.Version != test.wantVersion || got.Build != test.wantBuild {
				t.Fatalf("parseXcodeVersion() = %+v, want version=%q build=%q", got, test.wantVersion, test.wantBuild)
			}
		})
	}
}

func TestInspectToolchainUsesExplicitDeveloperDirAndSDK(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	developerDir := filepath.Join(t.TempDir(), "Xcode.app", "Contents", "Developer")
	if err := os.MkdirAll(developerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	installToolchainXcodebuild(t, developerDir)
	originalDeveloperDir := filepath.Join(t.TempDir(), "OtherXcode.app", "Contents", "Developer")
	t.Setenv("DEVELOPER_DIR", originalDeveloperDir)
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = func(name string) (string, error) {
		switch name {
		case "xcodebuild":
			return "/usr/bin/xcodebuild", nil
		case "xcrun":
			return "/usr/bin/xcrun", nil
		default:
			return "", exec.ErrNotFound
		}
	}
	activeDeveloperDirFn = func(context.Context) (string, error) {
		t.Fatal("active developer directory lookup must not run for an explicit candidate")
		return "", nil
	}

	report, err := InspectToolchain(context.Background(), ToolchainOptions{
		DeveloperDir: filepath.Dir(filepath.Dir(developerDir)),
		SDK:          "iphonesimulator",
		LogWriter:    io.Discard,
	})
	if err != nil {
		t.Fatalf("InspectToolchain() error = %v", err)
	}
	if report == nil {
		t.Fatal("InspectToolchain() report = nil")
	}
	if report.Status != ToolchainStatusOK || report.Source != ToolchainSourceFlag {
		t.Fatalf("InspectToolchain() report status/source = %q/%q, want ok/flag: %+v", report.Status, report.Source, report)
	}
	if report.DeveloperDir != developerDir {
		t.Fatalf("DeveloperDir = %q, want %q", report.DeveloperDir, developerDir)
	}
	if report.XcodeVersion != "16.4" || report.XcodeBuild != "16F6" {
		t.Fatalf("version/build = %q/%q, want 16.4/16F6", report.XcodeVersion, report.XcodeBuild)
	}
	if check, ok := toolchainReportCheck(report, "xcodebuild"); !ok || check.Path != filepath.Join(developerDir, "usr", "bin", "xcodebuild") {
		t.Fatalf("xcodebuild check = %+v, want selected-toolchain path", check)
	}
	if report.XcodePath != filepath.Dir(filepath.Dir(developerDir)) {
		t.Fatalf("XcodePath = %q, want %q", report.XcodePath, filepath.Dir(filepath.Dir(developerDir)))
	}
	if report.Beta == nil || *report.Beta {
		t.Fatal("stable Xcode candidate unexpectedly reported as beta")
	}
	if !toolchainReportHasCheck(report, "sdk:iphonesimulator", ToolchainCheckStatusOK) {
		t.Fatalf("SDK check missing or not OK: %+v", report.Checks)
	}
	if got := os.Getenv("DEVELOPER_DIR"); got != originalDeveloperDir {
		t.Fatalf("parent DEVELOPER_DIR changed to %q, want %q", got, originalDeveloperDir)
	}
}

func TestInspectToolchainUsesXcrunResolvedPathForVersionProbe(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	developerDir := filepath.Join(t.TempDir(), "Xcode.app", "Contents", "Developer")
	if err := os.MkdirAll(developerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	resolvedPath := installToolchainXcodebuild(t, developerDir)
	pathShadow := filepath.Join(t.TempDir(), "shadow", "xcodebuild")
	t.Setenv("DEVELOPER_DIR", "")
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = func(name string) (string, error) {
		switch name {
		case "xcodebuild":
			return pathShadow, nil
		case "xcrun":
			return "/tmp/shadow/xcrun", nil
		default:
			return "", exec.ErrNotFound
		}
	}

	var invoked []string
	originalCommandContext := commandContextFn
	commandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		invoked = append(invoked, name)
		return originalCommandContext(ctx, name, args...)
	}

	report, err := InspectToolchain(context.Background(), ToolchainOptions{
		DeveloperDir: developerDir,
		LogWriter:    io.Discard,
	})
	if err != nil {
		t.Fatalf("InspectToolchain() error = %v", err)
	}
	if report == nil || report.Status != ToolchainStatusOK {
		t.Fatalf("InspectToolchain() report = %+v, want ok", report)
	}
	check, ok := toolchainReportCheck(report, "xcodebuild")
	if !ok || check.Path != resolvedPath || check.Status != ToolchainCheckStatusOK {
		t.Fatalf("xcodebuild check = %+v (found=%t), want resolved path %q", check, ok, resolvedPath)
	}
	for _, name := range invoked {
		if filepath.Base(name) == "xcodebuild" && name != resolvedPath {
			t.Fatalf("version probe invoked %q, want exact xcrun-resolved path %q", name, resolvedPath)
		}
		if name == pathShadow {
			t.Fatalf("version probe used PATH-shadowed xcodebuild %q", name)
		}
	}
}

func TestInspectToolchainRejectsXcrunResolvedPathOutsideCandidate(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	developerDir := filepath.Join(t.TempDir(), "Xcode.app", "Contents", "Developer")
	if err := os.MkdirAll(developerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	installToolchainXcodebuild(t, developerDir)
	shadowPath := filepath.Join(t.TempDir(), "OtherXcode.app", "Contents", "Developer", "usr", "bin", "xcodebuild")
	t.Setenv("DEVELOPER_DIR", "")
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	t.Setenv("GO_TOOLCHAIN_DOCTOR_XCRUN_PATH", shadowPath)
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = toolchainDoctorLookPath

	report, err := InspectToolchain(context.Background(), ToolchainOptions{
		DeveloperDir: developerDir,
		LogWriter:    io.Discard,
	})
	if err == nil || report == nil || report.Status != ToolchainStatusFail {
		t.Fatalf("InspectToolchain() report/error = %+v/%v, want failed mismatch", report, err)
	}
	if check, ok := toolchainReportCheck(report, "xcrun"); !ok || check.Status != ToolchainCheckStatusFail || !strings.Contains(check.Message, "outside selected developer directory") {
		t.Fatalf("xcrun check = %+v (found=%t), want rejected resolved path", check, ok)
	}
	if check, ok := toolchainReportCheck(report, "xcodebuild"); !ok || check.Status != ToolchainCheckStatusFail {
		t.Fatalf("xcodebuild check = %+v (found=%t), want failed mismatch", check, ok)
	}
}

func TestInspectToolchainRejectsXcrunResolvedSymlinkOutsideCandidate(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	root := t.TempDir()
	developerDir := filepath.Join(root, "Xcode.app", "Contents", "Developer")
	if err := os.MkdirAll(developerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	outsideDeveloperDir := filepath.Join(root, "OtherXcode.app", "Contents", "Developer")
	outsidePath := installToolchainXcodebuild(t, outsideDeveloperDir)
	shadowPath := filepath.Join(developerDir, "usr", "bin", "xcodebuild")
	if err := os.MkdirAll(filepath.Dir(shadowPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.Symlink(outsidePath, shadowPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	t.Setenv("DEVELOPER_DIR", "")
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = toolchainDoctorLookPath

	report, err := InspectToolchain(context.Background(), ToolchainOptions{
		DeveloperDir: developerDir,
		LogWriter:    io.Discard,
	})
	if err == nil || report == nil || report.Status != ToolchainStatusFail {
		t.Fatalf("InspectToolchain() report/error = %+v/%v, want failed symlink escape", report, err)
	}
	if check, ok := toolchainReportCheck(report, "xcrun"); !ok || check.Status != ToolchainCheckStatusFail || !strings.Contains(check.Message, "outside selected developer directory") {
		t.Fatalf("xcrun check = %+v (found=%t), want rejected symlink target", check, ok)
	}
}

// TestValidateResolvedXcodebuildPathAcceptsAlternateCasingOfSelectedDirectory
// covers case-insensitive volumes (the macOS default), where a selector such
// as /applications/Xcode.app/Contents/Developer and an xcrun result under
// /Applications/Xcode.app/Contents/Developer name the same toolchain.
// Containment must compare filesystem identity, not lexical spellings.
func TestValidateResolvedXcodebuildPathAcceptsAlternateCasingOfSelectedDirectory(t *testing.T) {
	root := t.TempDir()
	developerDir := filepath.Join(root, "Xcode.app", "Contents", "Developer")
	toolPath := installToolchainXcodebuild(t, developerDir)

	alternateDeveloperDir := filepath.Join(root, "xCODE.APP", "Contents", "Developer")
	alternateInfo, err := os.Stat(alternateDeveloperDir)
	if err != nil {
		t.Skipf("volume is case-sensitive; alternate-case spelling unavailable: %v", err)
	}
	canonicalInfo, err := os.Stat(developerDir)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if !os.SameFile(canonicalInfo, alternateInfo) {
		t.Skip("volume treats the alternate casing as a distinct directory")
	}

	validated, err := validateResolvedXcodebuildPath(toolPath, alternateDeveloperDir)
	if err != nil {
		t.Fatalf("validateResolvedXcodebuildPath() error = %v, want acceptance for the same physical toolchain", err)
	}
	if validated != toolPath {
		t.Fatalf("validateResolvedXcodebuildPath() = %q, want xcrun spelling %q", validated, toolPath)
	}
}

// requireSamePathSpelling skips the calling test when the volume does not
// treat the alternate-cased spelling as the same directory entry, so the
// case-insensitive regressions run on default macOS volumes and skip on
// case-sensitive filesystems.
func requireSamePathSpelling(t *testing.T, canonicalPath, alternatePath string) {
	t.Helper()
	alternateInfo, err := os.Stat(alternatePath)
	if err != nil {
		t.Skipf("volume is case-sensitive; alternate-case spelling unavailable: %v", err)
	}
	canonicalInfo, err := os.Stat(canonicalPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if !os.SameFile(canonicalInfo, alternateInfo) {
		t.Skip("volume treats the alternate casing as a distinct directory")
	}
}

func TestNormalizeToolchainDeveloperDirHandlesAlternateCasedSpellings(t *testing.T) {
	t.Run("bundle developer components", func(t *testing.T) {
		root := t.TempDir()
		canonicalDeveloperDir := filepath.Join(root, "Xcode.app", "Contents", "Developer")
		if err := os.MkdirAll(canonicalDeveloperDir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		selector := filepath.Join(root, "Xcode.app", "contents", "developer")
		requireSamePathSpelling(t, canonicalDeveloperDir, selector)

		developerDir, xcodePath, commandLineTools, err := normalizeToolchainDeveloperDir(selector)
		if err != nil || commandLineTools {
			t.Fatalf("normalizeToolchainDeveloperDir() = commandLineTools=%t err=%v, want plain developer directory", commandLineTools, err)
		}
		if developerDir != selector {
			t.Fatalf("developerDir = %q, want selected spelling %q", developerDir, selector)
		}
		if want := filepath.Join(root, "Xcode.app"); xcodePath != want {
			t.Fatalf("xcodePath = %q, want %q despite alternate-cased bundle components", xcodePath, want)
		}
	})

	t.Run("app extension", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "Xcode.app", "Contents", "Developer"), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		selector := filepath.Join(root, "xcode.APP")
		requireSamePathSpelling(t, filepath.Join(root, "Xcode.app"), selector)

		developerDir, xcodePath, commandLineTools, err := normalizeToolchainDeveloperDir(selector)
		if err != nil || commandLineTools {
			t.Fatalf("normalizeToolchainDeveloperDir() = commandLineTools=%t err=%v, want app-bundle normalization", commandLineTools, err)
		}
		if want := filepath.Join(selector, "Contents", "Developer"); developerDir != want {
			t.Fatalf("developerDir = %q, want %q", developerDir, want)
		}
		if xcodePath != selector {
			t.Fatalf("xcodePath = %q, want selected spelling %q", xcodePath, selector)
		}
	})

	t.Run("command line tools", func(t *testing.T) {
		root := t.TempDir()
		canonicalCLT := filepath.Join(root, "Library", "Developer", "CommandLineTools")
		if err := os.MkdirAll(canonicalCLT, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		selector := filepath.Join(root, "library", "developer", "COMMANDLINETOOLS")
		requireSamePathSpelling(t, canonicalCLT, selector)

		_, _, commandLineTools, err := normalizeToolchainDeveloperDir(selector)
		if err != nil {
			t.Fatalf("normalizeToolchainDeveloperDir() error = %v", err)
		}
		if !commandLineTools {
			t.Fatal("commandLineTools = false, want Command Line Tools classification for alternate casing")
		}
	})
}

func TestValidateResolvedXcodebuildPathAcceptsAlternateCasedExecutableSpelling(t *testing.T) {
	root := t.TempDir()
	developerDir := filepath.Join(root, "Xcode.app", "Contents", "Developer")
	toolPath := installToolchainXcodebuild(t, developerDir)

	alternateToolPath := filepath.Join(developerDir, "usr", "bin", "XCODEBUILD")
	requireSamePathSpelling(t, toolPath, alternateToolPath)

	validated, err := validateResolvedXcodebuildPath(alternateToolPath, developerDir)
	if err != nil {
		t.Fatalf("validateResolvedXcodebuildPath() error = %v, want acceptance for the same physical executable", err)
	}
	if validated != alternateToolPath {
		t.Fatalf("validateResolvedXcodebuildPath() = %q, want xcrun spelling %q", validated, alternateToolPath)
	}
}

func TestValidateResolvedXcodebuildPathReportsMissingToolInsideAlternateCasedDirectory(t *testing.T) {
	// Resolve the temporary root so the missing-path diagnostic exercises the
	// casing dimension alone; a nonexistent path cannot be symlink-resolved,
	// so the fallback comparison is lexical by design.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	developerDir := filepath.Join(root, "Xcode.app", "Contents", "Developer")
	if err := os.MkdirAll(developerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	alternateSpelling := filepath.Join(root, "xcode.app", "Contents", "Developer")
	requireSamePathSpelling(t, developerDir, alternateSpelling)

	missingTool := filepath.Join(alternateSpelling, "usr", "bin", "xcodebuild")
	_, err = validateResolvedXcodebuildPath(missingTool, developerDir)
	if err == nil {
		t.Fatal("validateResolvedXcodebuildPath() error = nil, want unavailable failure")
	}
	if strings.Contains(err.Error(), "outside selected developer directory") {
		t.Fatalf("validateResolvedXcodebuildPath() error = %v, want unavailable diagnostic for the same directory", err)
	}
	if !strings.Contains(err.Error(), "is unavailable") {
		t.Fatalf("validateResolvedXcodebuildPath() error = %v, want unavailable diagnostic", err)
	}
}

func TestInspectToolchainClassifiesBetaFromCanonicalSelectedSymlink(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	root := t.TempDir()
	canonicalApp := filepath.Join(root, "Xcode-beta.app")
	canonicalDeveloperDir := filepath.Join(canonicalApp, "Contents", "Developer")
	installToolchainXcodebuild(t, canonicalDeveloperDir)

	selectedApp := filepath.Join(root, "Xcode.app")
	if err := os.Symlink(canonicalApp, selectedApp); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	selectedDeveloperDir := filepath.Join(selectedApp, "Contents", "Developer")
	t.Setenv("DEVELOPER_DIR", "")
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = toolchainDoctorLookPath

	report, err := InspectToolchain(context.Background(), ToolchainOptions{
		DeveloperDir: selectedApp,
		LogWriter:    io.Discard,
	})
	if err != nil {
		t.Fatalf("InspectToolchain() error = %v", err)
	}
	if report == nil || report.Status != ToolchainStatusWarn {
		t.Fatalf("InspectToolchain() report = %+v, want warning for canonical beta target", report)
	}
	if report.Beta == nil || !*report.Beta {
		t.Fatalf("InspectToolchain() beta = %v, want true from canonical target", report.Beta)
	}
	if !toolchainReportHasCheck(report, "beta", ToolchainCheckStatusWarn) {
		t.Fatalf("missing beta warning: %+v", report.Checks)
	}
	if report.DeveloperDir != selectedDeveloperDir {
		t.Fatalf("DeveloperDir = %q, want selected spelling %q", report.DeveloperDir, selectedDeveloperDir)
	}
	if report.XcodePath != selectedApp {
		t.Fatalf("XcodePath = %q, want selected spelling %q", report.XcodePath, selectedApp)
	}
}

func TestClassifyBetaXcodePathFailsClosedWhenCanonicalizationFails(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "Missing.app", "Contents", "Developer")
	beta, err := classifyBetaXcodePath(missing, "")
	if err == nil {
		t.Fatal("classifyBetaXcodePath() error = nil, want canonicalization failure")
	}
	if beta != nil {
		t.Fatalf("classifyBetaXcodePath() beta = %v, want unknown", *beta)
	}
}

func TestInspectToolchainNormalizesExtensionlessXcodeAppSymlinkFromCanonicalTarget(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	root := t.TempDir()
	tests := []struct {
		name        string
		appName     string
		wantStatus  ToolchainStatus
		wantBeta    bool
		wantWarning bool
	}{
		{name: "stable", appName: "Xcode.app", wantStatus: ToolchainStatusOK},
		{name: "beta", appName: "Xcode-beta.app", wantStatus: ToolchainStatusWarn, wantBeta: true, wantWarning: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			canonicalApp := filepath.Join(root, test.name, test.appName)
			canonicalDeveloperDir := filepath.Join(canonicalApp, "Contents", "Developer")
			installToolchainXcodebuild(t, canonicalDeveloperDir)
			selectedApp := filepath.Join(root, test.name, "current-xcode")
			if err := os.Symlink(canonicalApp, selectedApp); err != nil {
				t.Fatalf("Symlink() error = %v", err)
			}

			t.Setenv("DEVELOPER_DIR", "")
			t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
			commandContextFn = toolchainDoctorHelperCommandContext
			lookPathFn = toolchainDoctorLookPath

			report, err := InspectToolchain(context.Background(), ToolchainOptions{
				DeveloperDir: selectedApp,
				LogWriter:    io.Discard,
			})
			if err != nil {
				t.Fatalf("InspectToolchain() error = %v", err)
			}
			if report == nil || report.Status != test.wantStatus {
				t.Fatalf("InspectToolchain() report = %+v, want status %q", report, test.wantStatus)
			}
			if report.DeveloperDir != filepath.Join(selectedApp, "Contents", "Developer") {
				t.Fatalf("DeveloperDir = %q, want selected spelling under %q", report.DeveloperDir, selectedApp)
			}
			if report.XcodePath != selectedApp {
				t.Fatalf("XcodePath = %q, want selected spelling %q", report.XcodePath, selectedApp)
			}
			if report.Beta == nil || *report.Beta != test.wantBeta {
				t.Fatalf("Beta = %v, want %t from canonical target", report.Beta, test.wantBeta)
			}
			if test.wantWarning && !toolchainReportHasCheck(report, "beta", ToolchainCheckStatusWarn) {
				t.Fatalf("missing beta warning: %+v", report.Checks)
			}
			wantXcodebuild := filepath.Join(selectedApp, "Contents", "Developer", "usr", "bin", "xcodebuild")
			if check, ok := toolchainReportCheck(report, "xcodebuild"); !ok || check.Status != ToolchainCheckStatusOK || check.Path != wantXcodebuild {
				t.Fatalf("xcodebuild check = %+v (found=%t), want selected probe path %q", check, ok, wantXcodebuild)
			}
		})
	}
}

func TestInspectToolchainKeepsCanonicalDeveloperDirectorySymlinkAsDeveloperCandidate(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	root := t.TempDir()
	canonicalDeveloperDir := filepath.Join(root, "Xcode-beta.app", "Contents", "Developer")
	installToolchainXcodebuild(t, canonicalDeveloperDir)
	selectedDeveloperDir := filepath.Join(root, "current-developer")
	if err := os.Symlink(canonicalDeveloperDir, selectedDeveloperDir); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	t.Setenv("DEVELOPER_DIR", "")
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = toolchainDoctorLookPath

	report, err := InspectToolchain(context.Background(), ToolchainOptions{
		DeveloperDir: selectedDeveloperDir,
		LogWriter:    io.Discard,
	})
	if err != nil {
		t.Fatalf("InspectToolchain() error = %v", err)
	}
	if report == nil || report.Status != ToolchainStatusWarn {
		t.Fatalf("InspectToolchain() report = %+v, want beta warning", report)
	}
	if report.DeveloperDir != selectedDeveloperDir {
		t.Fatalf("DeveloperDir = %q, want selected spelling %q", report.DeveloperDir, selectedDeveloperDir)
	}
	if report.XcodePath != "" {
		t.Fatalf("XcodePath = %q, want omitted for developer-directory selector", report.XcodePath)
	}
	if report.Beta == nil || !*report.Beta {
		t.Fatalf("Beta = %v, want true from canonical developer target", report.Beta)
	}
	if check, ok := toolchainReportCheck(report, "xcodebuild"); !ok || check.Status != ToolchainCheckStatusOK {
		t.Fatalf("xcodebuild check = %+v (found=%t), want successful probe", check, ok)
	}
}

func TestInspectToolchainFailsClosedForBrokenAndCyclicDeveloperSymlinks(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	root := t.TempDir()
	tests := []struct {
		name   string
		target func(string) string
	}{
		{name: "broken", target: func(root string) string { return filepath.Join(root, "missing") }},
		{name: "cyclic", target: func(root string) string { return filepath.Join(root, "cycle") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selectedDeveloperDir := filepath.Join(root, test.name, "selected")
			if err := os.MkdirAll(filepath.Dir(selectedDeveloperDir), 0o755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			if err := os.Symlink(test.target(filepath.Dir(selectedDeveloperDir)), selectedDeveloperDir); err != nil {
				t.Fatalf("Symlink() error = %v", err)
			}
			if test.name == "cyclic" {
				if err := os.Symlink(selectedDeveloperDir, test.target(filepath.Dir(selectedDeveloperDir))); err != nil {
					t.Fatalf("cycle Symlink() error = %v", err)
				}
			}

			report, err := InspectToolchain(context.Background(), ToolchainOptions{
				DeveloperDir: selectedDeveloperDir,
				LogWriter:    io.Discard,
			})
			if err == nil || report == nil || report.Status != ToolchainStatusFail {
				t.Fatalf("InspectToolchain() report/error = %+v/%v, want failed selection", report, err)
			}
			if report.Beta != nil {
				t.Fatalf("InspectToolchain() beta = %v, want unknown after %s symlink failure", *report.Beta, test.name)
			}
			if !toolchainReportHasCheck(report, "developer_dir", ToolchainCheckStatusFail) {
				t.Fatalf("missing failed developer_dir check: %+v", report.Checks)
			}
		})
	}
}

func TestInspectToolchainClassifiesCommandLineToolsFromCanonicalSelectedSymlink(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	root := t.TempDir()
	canonicalDeveloperDir := filepath.Join(root, "Library", "Developer", "CommandLineTools")
	installToolchainXcodebuild(t, canonicalDeveloperDir)

	selectedDeveloperDir := filepath.Join(root, "SelectedToolchain")
	if err := os.Symlink(canonicalDeveloperDir, selectedDeveloperDir); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	t.Setenv("DEVELOPER_DIR", "")
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = toolchainDoctorLookPath

	report, err := InspectToolchain(context.Background(), ToolchainOptions{
		DeveloperDir: selectedDeveloperDir,
		LogWriter:    io.Discard,
	})
	if err == nil || report == nil || report.Status != ToolchainStatusFail {
		t.Fatalf("InspectToolchain() report/error = %+v/%v, want Command Line Tools failure", report, err)
	}
	if report.DeveloperDir != selectedDeveloperDir {
		t.Fatalf("DeveloperDir = %q, want selected spelling %q", report.DeveloperDir, selectedDeveloperDir)
	}
	if report.XcodePath != "" {
		t.Fatalf("XcodePath = %q, want empty for Command Line Tools", report.XcodePath)
	}
	if report.Beta == nil || *report.Beta {
		t.Fatalf("InspectToolchain() beta = %v, want false for canonical Command Line Tools", report.Beta)
	}
	if !toolchainReportHasCheck(report, "developer_dir", ToolchainCheckStatusFail) {
		t.Fatalf("missing Command Line Tools failure: %+v", report.Checks)
	}
}

func TestInspectToolchainUsesEnvironmentBeforeXcodeSelect(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	developerDir := filepath.Join(t.TempDir(), "Xcode.app", "Contents", "Developer")
	if err := os.MkdirAll(developerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	installToolchainXcodebuild(t, developerDir)
	t.Setenv("DEVELOPER_DIR", developerDir)
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = toolchainDoctorLookPath
	activeDeveloperDirFn = func(context.Context) (string, error) {
		t.Fatal("xcode-select lookup must not run when DEVELOPER_DIR is set")
		return "", nil
	}

	report, err := InspectToolchain(context.Background(), ToolchainOptions{LogWriter: io.Discard})
	if err != nil {
		t.Fatalf("InspectToolchain() error = %v", err)
	}
	if report == nil || report.Source != ToolchainSourceEnvironment || report.DeveloperDir != developerDir {
		t.Fatalf("unexpected environment report: %+v", report)
	}
}

func TestInspectToolchainUsesXcodeSelectWhenEnvironmentUnset(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	developerDir := filepath.Join(t.TempDir(), "Xcode.app", "Contents", "Developer")
	if err := os.MkdirAll(developerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	installToolchainXcodebuild(t, developerDir)
	t.Setenv("DEVELOPER_DIR", "")
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = toolchainDoctorLookPath
	activeDeveloperDirFn = func(context.Context) (string, error) { return developerDir, nil }

	report, err := InspectToolchain(context.Background(), ToolchainOptions{LogWriter: io.Discard})
	if err != nil {
		t.Fatalf("InspectToolchain() error = %v", err)
	}
	if report == nil || report.Source != ToolchainSourceXcodeSelect || report.DeveloperDir != developerDir {
		t.Fatalf("unexpected xcode-select report: %+v", report)
	}
}

func TestInspectToolchainReportsBetaWarningAndCommandLineToolsFailure(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	t.Setenv("DEVELOPER_DIR", "")
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = toolchainDoctorLookPath
	activeDeveloperDirFn = func(context.Context) (string, error) {
		t.Fatal("xcode-select lookup must not run for explicit candidates")
		return "", nil
	}

	stable := false
	beta := true
	tests := []struct {
		name       string
		candidate  func(string) string
		wantStatus ToolchainStatus
		wantCheck  ToolchainCheckStatus
		wantBeta   *bool
	}{
		{
			name: "beta xcode",
			candidate: func(root string) string {
				return filepath.Join(root, "Xcode-beta.app", "Contents", "Developer")
			},
			wantStatus: ToolchainStatusWarn,
			wantCheck:  ToolchainCheckStatusOK,
			wantBeta:   &beta,
		},
		{
			name: "command line tools",
			candidate: func(root string) string {
				return filepath.Join(root, "Library", "Developer", "CommandLineTools")
			},
			wantStatus: ToolchainStatusFail,
			wantCheck:  ToolchainCheckStatusFail,
			wantBeta:   &stable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := test.candidate(t.TempDir())
			if err := os.MkdirAll(candidate, 0o755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			installToolchainXcodebuild(t, candidate)

			report, err := InspectToolchain(context.Background(), ToolchainOptions{
				DeveloperDir: candidate,
				LogWriter:    io.Discard,
			})
			if test.wantStatus == ToolchainStatusFail && err == nil {
				t.Fatalf("InspectToolchain() error = nil, want Command Line Tools failure")
			}
			if test.wantStatus != ToolchainStatusFail && err != nil {
				t.Fatalf("InspectToolchain() error = %v", err)
			}
			if report == nil || report.Status != test.wantStatus || !sameOptionalBool(report.Beta, test.wantBeta) {
				t.Fatalf("unexpected report: %+v", report)
			}
			if check, ok := toolchainReportCheck(report, "developer_dir"); !ok || check.Status != test.wantCheck {
				t.Fatalf("developer_dir check = %+v (found=%t), want %q", check, ok, test.wantCheck)
			}
			if test.wantBeta != nil && *test.wantBeta && !toolchainReportHasCheck(report, "beta", ToolchainCheckStatusWarn) {
				t.Fatalf("missing beta warning: %+v", report.Checks)
			}
		})
	}
}

func TestInspectToolchainReturnsFailureForUnavailableCandidate(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	lookPathFn = toolchainDoctorLookPath
	report, err := InspectToolchain(context.Background(), ToolchainOptions{
		DeveloperDir: filepath.Join(t.TempDir(), "Missing.xcode", "Contents", "Developer"),
		LogWriter:    io.Discard,
	})
	if err == nil {
		t.Fatal("InspectToolchain() error = nil, want unavailable candidate error")
	}
	if report == nil || report.Status != ToolchainStatusFail {
		t.Fatalf("InspectToolchain() report = %+v, error = %v; want failed report", report, err)
	}
	if !toolchainReportHasCheck(report, "developer_dir", ToolchainCheckStatusFail) {
		t.Fatalf("missing failed developer_dir check: %+v", report.Checks)
	}
	if report.Beta != nil {
		t.Fatalf("failed developer-directory selection reported beta=%t, want unknown", *report.Beta)
	}
}

func TestInspectToolchainRejectsNonDirectoryCandidate(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	candidate := filepath.Join(t.TempDir(), "Xcode.app")
	if err := os.WriteFile(candidate, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	report, err := InspectToolchain(context.Background(), ToolchainOptions{DeveloperDir: candidate, LogWriter: io.Discard})
	if err == nil || report == nil || report.Status != ToolchainStatusFail {
		t.Fatalf("InspectToolchain() report/error = %+v/%v, want failed report", report, err)
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("InspectToolchain() error = %v, want non-directory detail", err)
	}
}

func TestInspectToolchainReportsUnsupportedPlatform(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "linux"
	report, err := InspectToolchain(context.Background(), ToolchainOptions{DeveloperDir: t.TempDir(), LogWriter: io.Discard})
	if report != nil {
		t.Fatalf("InspectToolchain() report = %+v, want nil on unsupported platform", report)
	}
	if err == nil || !strings.Contains(err.Error(), "supported on macOS only") {
		t.Fatalf("InspectToolchain() error = %v, want macOS-only error", err)
	}
}

func toolchainReportHasCheck(report *ToolchainReport, name string, status ToolchainCheckStatus) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}

func toolchainReportCheck(report *ToolchainReport, name string) (ToolchainCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return ToolchainCheck{}, false
}

func sameOptionalBool(got, want *bool) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return *got == *want
}

func toolchainDoctorLookPath(name string) (string, error) {
	switch name {
	case "xcodebuild":
		return "/usr/bin/xcodebuild", nil
	case "xcrun":
		return "/usr/bin/xcrun", nil
	default:
		return "", exec.ErrNotFound
	}
}

func installToolchainXcodebuild(t *testing.T, developerDir string) string {
	t.Helper()
	pathValue := filepath.Join(developerDir, "usr", "bin", "xcodebuild")
	if err := os.MkdirAll(filepath.Dir(pathValue), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(pathValue, []byte("fake xcodebuild"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return pathValue
}

func toolchainDoctorHelperCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	commandArgs := []string{"-test.run=TestToolchainDoctorHelperProcess", "--", name}
	commandArgs = append(commandArgs, args...)
	return exec.CommandContext(ctx, os.Args[0], commandArgs...)
}

func TestToolchainDoctorHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER") != "1" {
		return
	}

	args := os.Args
	separator := -1
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 || separator+1 >= len(args) {
		fmt.Fprintln(os.Stderr, "missing helper command")
		os.Exit(2)
	}
	command := args[separator+1:]
	commandName := ""
	if len(command) > 0 {
		commandName = filepath.Base(command[0])
	}
	switch {
	case len(command) >= 2 && commandName == "xcodebuild" && command[1] == "-version":
		if os.Getenv("GO_TOOLCHAIN_DOCTOR_FAIL_XCODEBUILD") == "1" {
			fmt.Fprintln(os.Stderr, strings.Repeat("toolchain diagnostic ", 400))
			os.Exit(7)
		}
		fmt.Fprintln(os.Stdout, "Xcode 16.4")
		fmt.Fprintln(os.Stdout, "Build version 16F6")
		os.Exit(0)
	case len(command) >= 3 && commandName == "xcrun" && command[1] == "--find" && command[2] == "xcodebuild":
		if os.Getenv("GO_TOOLCHAIN_DOCTOR_FAIL_XCRUN_FIND") == "1" {
			fmt.Fprintln(os.Stderr, "xcrun could not resolve xcodebuild")
			os.Exit(8)
		}
		resolvedPath := os.Getenv("GO_TOOLCHAIN_DOCTOR_XCRUN_PATH")
		if resolvedPath == "" {
			resolvedPath = filepath.Join(os.Getenv("DEVELOPER_DIR"), "usr", "bin", "xcodebuild")
		}
		fmt.Fprintln(os.Stdout, resolvedPath)
		os.Exit(0)
	case len(command) >= 4 && commandName == "xcrun" && command[1] == "--sdk" && command[3] == "--show-sdk-path":
		if os.Getenv("GO_TOOLCHAIN_DOCTOR_FAIL_SDK") == "1" {
			fmt.Fprintln(os.Stderr, "requested SDK is not installed")
			os.Exit(9)
		}
		fmt.Fprintln(os.Stdout, filepath.Join(os.Getenv("DEVELOPER_DIR"), "Platforms", "iPhoneSimulator.platform", "Developer", "SDKs", command[2]+".sdk"))
		os.Exit(0)
	case len(command) >= 2 && commandName == "xcode-select":
		fmt.Fprintln(os.Stderr, "xcode-select must not be used for an explicit toolchain candidate")
		os.Exit(3)
	default:
		fmt.Fprintf(os.Stderr, "unexpected helper command %q\n", command)
		os.Exit(2)
	}
}

func TestInspectToolchainPreservesCancellation(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)
	runtimeGOOS = "darwin"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := InspectToolchain(ctx, ToolchainOptions{DeveloperDir: t.TempDir(), LogWriter: io.Discard})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("InspectToolchain() error = %v, want context.Canceled", err)
	}
}

func TestInspectToolchainBoundsProbeDiagnostics(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	developerDir := filepath.Join(t.TempDir(), "Xcode.app", "Contents", "Developer")
	if err := os.MkdirAll(developerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	installToolchainXcodebuild(t, developerDir)
	t.Setenv("DEVELOPER_DIR", "")
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	t.Setenv("GO_TOOLCHAIN_DOCTOR_FAIL_XCODEBUILD", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = toolchainDoctorLookPath

	var logs strings.Builder
	report, err := InspectToolchain(context.Background(), ToolchainOptions{
		DeveloperDir: developerDir,
		LogWriter:    &logs,
	})
	if err == nil || report == nil || report.Status != ToolchainStatusFail {
		t.Fatalf("InspectToolchain() report/error = %+v/%v, want failed report", report, err)
	}
	check, ok := toolchainReportCheck(report, "xcodebuild")
	if !ok || !strings.Contains(check.Message, "xcodebuild -version failed") {
		t.Fatalf("xcodebuild check = %+v, want bounded probe failure", check)
	}
	if len(check.Message) > 512 {
		t.Fatalf("xcodebuild diagnostic length = %d, want bounded message", len(check.Message))
	}
	if !strings.Contains(logs.String(), "toolchain diagnostic") {
		t.Fatalf("probe log = %q, want child diagnostic", logs.String())
	}
	if len(logs.String()) > toolchainProbeDiagnosticLimit*2+2 {
		t.Fatalf("probe log length = %d, want bounded output", len(logs.String()))
	}
}

func TestInspectToolchainReportsProbeFailures(t *testing.T) {
	restore := overrideTestEnvironment(t)
	t.Cleanup(restore)

	runtimeGOOS = "darwin"
	developerDir := filepath.Join(t.TempDir(), "Xcode.app", "Contents", "Developer")
	if err := os.MkdirAll(developerDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	installToolchainXcodebuild(t, developerDir)
	t.Setenv("DEVELOPER_DIR", "")
	t.Setenv("GO_WANT_TOOLCHAIN_DOCTOR_HELPER", "1")
	commandContextFn = toolchainDoctorHelperCommandContext
	lookPathFn = toolchainDoctorLookPath

	for _, test := range []struct {
		name      string
		env       string
		sdk       string
		checkName string
	}{
		{name: "xcodebuild", env: "GO_TOOLCHAIN_DOCTOR_FAIL_XCODEBUILD", checkName: "xcodebuild"},
		{name: "xcrun", env: "GO_TOOLCHAIN_DOCTOR_FAIL_XCRUN_FIND", checkName: "xcrun"},
		{name: "sdk", env: "GO_TOOLCHAIN_DOCTOR_FAIL_SDK", sdk: "iphonesimulator", checkName: "sdk:iphonesimulator"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.env, "1")
			report, err := InspectToolchain(context.Background(), ToolchainOptions{
				DeveloperDir: developerDir,
				SDK:          test.sdk,
				LogWriter:    io.Discard,
			})
			if err == nil || report == nil || report.Status != ToolchainStatusFail {
				t.Fatalf("InspectToolchain() report/error = %+v/%v, want failed report", report, err)
			}
			if !toolchainReportHasCheck(report, test.checkName, ToolchainCheckStatusFail) {
				t.Fatalf("missing failed %s check: %+v", test.checkName, report.Checks)
			}
		})
	}
}
