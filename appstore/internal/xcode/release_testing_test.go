package xcode

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"howett.net/plist"
)

const (
	testSigningCertificateSHA1  = "0123456789ABCDEF0123456789ABCDEF01234567"
	testProvisioningProfileUUID = "12345678-1234-1234-1234-123456789ABC"
)

func TestWriteManualReleaseTestingExportOptionsWritesExactCreateOnlyArtifact(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "state", "ExportOptions.plist")
	result, err := WriteManualReleaseTestingExportOptions(context.Background(), ManualReleaseTestingExportOptions{
		OutputPath:         outputPath,
		TeamID:             "TEAM123",
		SigningCertificate: strings.ToLower(testSigningCertificateSHA1),
		ProvisioningProfiles: map[string]string{
			"com.example.demo": testProvisioningProfileUUID,
		},
	})
	if err != nil {
		t.Fatalf("WriteManualReleaseTestingExportOptions() error: %v", err)
	}
	if result.Path != outputPath {
		t.Fatalf("result path = %q, want %q", result.Path, outputPath)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	digest := sha256.Sum256(data)
	if result.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("result sha256 = %q, want %q", result.SHA256, hex.EncodeToString(digest[:]))
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}

	var payload map[string]any
	if _, err := plist.Unmarshal(data, &payload); err != nil {
		t.Fatalf("plist.Unmarshal() error: %v", err)
	}
	for key, want := range map[string]string{
		"method":                     "release-testing",
		"destination":                "export",
		"signingStyle":               "manual",
		"teamID":                     "TEAM123",
		"signingCertificate":         "0123456789ABCDEF0123456789ABCDEF01234567",
		"iCloudContainerEnvironment": "Production",
	} {
		if got := exportOptionsString(payload[key]); got != want {
			t.Errorf("payload[%q] = %q, want %q", key, got, want)
		}
	}
	profiles, err := provisioningProfilesFromPayload(payload["provisioningProfiles"])
	if err != nil {
		t.Fatalf("provisioningProfilesFromPayload() error: %v", err)
	}
	if got := profiles["com.example.demo"]; got != testProvisioningProfileUUID {
		t.Fatalf("profile = %q, want %q", got, testProvisioningProfileUUID)
	}
}

func TestWriteManualReleaseTestingExportOptionsRejectsInvalidInputWithoutArtifact(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*ManualReleaseTestingExportOptions)
		message string
	}{
		{name: "missing output", mutate: func(o *ManualReleaseTestingExportOptions) { o.OutputPath = "" }, message: "output path is required"},
		{name: "missing team", mutate: func(o *ManualReleaseTestingExportOptions) { o.TeamID = "" }, message: "team ID is required"},
		{name: "missing certificate", mutate: func(o *ManualReleaseTestingExportOptions) { o.SigningCertificate = "" }, message: "signing certificate is required"},
		{name: "certificate name", mutate: func(o *ManualReleaseTestingExportOptions) { o.SigningCertificate = "Apple Distribution: Example" }, message: "40-character SHA-1 fingerprint"},
		{name: "empty profiles", mutate: func(o *ManualReleaseTestingExportOptions) { o.ProvisioningProfiles = nil }, message: "provisioning profile mappings"},
		{name: "empty bundle", mutate: func(o *ManualReleaseTestingExportOptions) {
			o.ProvisioningProfiles = map[string]string{"": testProvisioningProfileUUID}
		}, message: "empty bundle identifier"},
		{name: "empty profile", mutate: func(o *ManualReleaseTestingExportOptions) {
			o.ProvisioningProfiles = map[string]string{"com.example": ""}
		}, message: "empty provisioning profile"},
		{name: "profile name", mutate: func(o *ManualReleaseTestingExportOptions) {
			o.ProvisioningProfiles = map[string]string{"com.example": "Ad Hoc Profile"}
		}, message: "must be an exact UUID"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outputPath := filepath.Join(t.TempDir(), "state", "ExportOptions.plist")
			opts := ManualReleaseTestingExportOptions{
				OutputPath:         outputPath,
				TeamID:             "TEAM123",
				SigningCertificate: testSigningCertificateSHA1,
				ProvisioningProfiles: map[string]string{
					"com.example.demo": testProvisioningProfileUUID,
				},
			}
			test.mutate(&opts)
			_, err := WriteManualReleaseTestingExportOptions(context.Background(), opts)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error = %v, want containing %q", err, test.message)
			}
			if opts.OutputPath != "" {
				if _, statErr := os.Lstat(outputPath); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("invalid input created artifact: %v", statErr)
				}
			}
		})
	}
}

func TestWriteManualReleaseTestingExportOptionsRefusesExistingDestination(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "ExportOptions.plist")
	if err := os.WriteFile(outputPath, []byte("existing"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}
	_, err := WriteManualReleaseTestingExportOptions(context.Background(), ManualReleaseTestingExportOptions{
		OutputPath:         outputPath,
		TeamID:             "TEAM123",
		SigningCertificate: testSigningCertificateSHA1,
		ProvisioningProfiles: map[string]string{
			"com.example.demo": testProvisioningProfileUUID,
		},
	})
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("error = %v, want os.ErrExist", err)
	}
	if got, readErr := os.ReadFile(outputPath); readErr != nil || string(got) != "existing" {
		t.Fatalf("existing destination changed: data=%q err=%v", got, readErr)
	}
}

func TestValidateManualReleaseTestingExportOptionsRequiresExactPrivateBytes(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "ExportOptions.plist")
	opts := ManualReleaseTestingExportOptions{
		OutputPath: outputPath, TeamID: "TEAM123", SigningCertificate: testSigningCertificateSHA1,
		ProvisioningProfiles: map[string]string{"com.example.demo": testProvisioningProfileUUID},
	}
	written, err := WriteManualReleaseTestingExportOptions(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	validated, err := ValidateManualReleaseTestingExportOptions(context.Background(), opts)
	if err != nil || validated.SHA256 != written.SHA256 {
		t.Fatalf("exact validation result=%#v error=%v", validated, err)
	}
	tampered := opts
	tampered.TeamID = "OTHERTEAM"
	if _, err := ValidateManualReleaseTestingExportOptions(context.Background(), tampered); err == nil || !strings.Contains(err.Error(), "differ") {
		t.Fatalf("different signing selection accepted: %v", err)
	}
	if err := os.Chmod(outputPath, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateManualReleaseTestingExportOptions(context.Background(), opts); err == nil || !strings.Contains(err.Error(), "private") {
		t.Fatalf("non-private export options accepted: %v", err)
	}
}

func TestExportReleaseTestingUsesExactCallerEnvironmentAndCreateOnlyDestination(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "Demo.xcarchive")
	if err := writeArchiveInfoPlist(archivePath); err != nil {
		t.Fatalf("writeArchiveInfoPlist() error: %v", err)
	}
	exportOptionsPath := filepath.Join(tempDir, "ExportOptions.plist")
	writeExportOptionsPlist(t, exportOptionsPath, map[string]any{
		"method":                     "release-testing",
		"destination":                "export",
		"signingStyle":               "manual",
		"teamID":                     "TEAM123",
		"signingCertificate":         testSigningCertificateSHA1,
		"provisioningProfiles":       map[string]string{"com.example.demo": testProvisioningProfileUUID},
		"iCloudContainerEnvironment": "Production",
	})
	commandLog := filepath.Join(tempDir, "commands.log")
	environmentLog := filepath.Join(tempDir, "environment.log")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(string) (string, error) { return "/usr/bin/xcodebuild", nil }
	commandContextFn = helperCommandContext(t, commandLog)
	t.Cleanup(restore)
	t.Setenv("ASC_SHOULD_NOT_LEAK", "host-secret")

	ipaPath := filepath.Join(tempDir, "artifacts", "Demo.ipa")
	environment := []string{
		"GO_WANT_XCODE_HELPER_PROCESS=1",
		"ASC_XCODE_HELPER_LOG=" + commandLog,
		"ASC_XCODE_HELPER_ENV_LOG=" + environmentLog,
		"ASC_ONLY=ephemeral",
	}
	result, err := ExportReleaseTesting(context.Background(), ReleaseTestingExportOptions{
		ArchivePath:         archivePath,
		ExportOptionsPath:   exportOptionsPath,
		ExportOptionsSHA256: fileSHA256(t, exportOptionsPath),
		IPAPath:             ipaPath,
		Environment:         environment,
	})
	if err != nil {
		t.Fatalf("ExportReleaseTesting() error: %v", err)
	}
	if result.IPAPath != ipaPath {
		t.Fatalf("IPA path = %q, want %q", result.IPAPath, ipaPath)
	}
	if info, err := os.Stat(ipaPath); err != nil {
		t.Fatalf("Stat(IPA) error: %v", err)
	} else if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("IPA mode = %o, want 600", got)
	}
	data, err := os.ReadFile(environmentLog)
	if err != nil {
		t.Fatalf("ReadFile(environment log) error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("environment log = %q, want version and export entries", data)
	}
	for _, line := range lines {
		if !strings.Contains(line, "ASC_ONLY=ephemeral") || !strings.Contains(line, "ASC_SHOULD_NOT_LEAK=") || strings.Contains(line, "host-secret") {
			t.Fatalf("caller environment not exact: %q", line)
		}
	}
}

func TestExportReleaseTestingDoesNotReplaceDestinationCreatedDuringExport(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "Demo.xcarchive")
	if err := writeArchiveInfoPlist(archivePath); err != nil {
		t.Fatalf("writeArchiveInfoPlist() error: %v", err)
	}
	exportOptionsPath := filepath.Join(tempDir, "ExportOptions.plist")
	writeExportOptionsPlist(t, exportOptionsPath, map[string]any{
		"method":                     "release-testing",
		"destination":                "export",
		"signingStyle":               "manual",
		"teamID":                     "TEAM123",
		"signingCertificate":         testSigningCertificateSHA1,
		"provisioningProfiles":       map[string]string{"com.example.demo": testProvisioningProfileUUID},
		"iCloudContainerEnvironment": "Production",
	})
	commandLog := filepath.Join(tempDir, "commands.log")
	itaPath := filepath.Join(tempDir, "Demo.ipa")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(string) (string, error) { return "/usr/bin/xcodebuild", nil }
	commandContextFn = helperCommandContext(t, commandLog)
	t.Cleanup(restore)

	_, err := ExportReleaseTesting(context.Background(), ReleaseTestingExportOptions{
		ArchivePath:         archivePath,
		ExportOptionsPath:   exportOptionsPath,
		ExportOptionsSHA256: fileSHA256(t, exportOptionsPath),
		IPAPath:             itaPath,
		Environment: []string{
			"GO_WANT_XCODE_HELPER_PROCESS=1",
			"ASC_XCODE_HELPER_LOG=" + commandLog,
			"ASC_XCODE_HELPER_RACING_IPA_PATH=" + itaPath,
		},
	})
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("error = %v, want os.ErrExist", err)
	}
	data, readErr := os.ReadFile(itaPath)
	if readErr != nil || string(data) != "racing ipa" {
		t.Fatalf("racing destination changed: data=%q err=%v", data, readErr)
	}
}

func TestExportReleaseTestingUsesVerifiedSnapshotAfterSourcePathReplacement(t *testing.T) {
	tempDir := t.TempDir()
	archivePath, exportOptionsPath := writeReleaseTestingExportFixture(t, tempDir)
	commandLog := filepath.Join(tempDir, "commands.log")

	restoreEnvironment := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(string) (string, error) { return "/usr/bin/xcodebuild", nil }
	commandContextFn = helperCommandContext(t, commandLog)
	t.Cleanup(restoreEnvironment)

	originalHook := afterReleaseTestingOptionsReadFn
	afterReleaseTestingOptionsReadFn = func() {
		writeExportOptionsPlist(t, exportOptionsPath, map[string]any{
			"method":       "app-store-connect",
			"destination":  "upload",
			"signingStyle": "automatic",
		})
	}
	t.Cleanup(func() { afterReleaseTestingOptionsReadFn = originalHook })

	ipaPath := filepath.Join(tempDir, "Demo.ipa")
	result, err := ExportReleaseTesting(context.Background(), ReleaseTestingExportOptions{
		ArchivePath:         archivePath,
		ExportOptionsPath:   exportOptionsPath,
		ExportOptionsSHA256: fileSHA256(t, exportOptionsPath),
		IPAPath:             ipaPath,
		Environment: []string{
			"GO_WANT_XCODE_HELPER_PROCESS=1",
			"ASC_XCODE_HELPER_LOG=" + commandLog,
		},
	})
	if err != nil {
		t.Fatalf("ExportReleaseTesting() error: %v", err)
	}
	if result.IPAPath != ipaPath {
		t.Fatalf("IPA path = %q, want %q", result.IPAPath, ipaPath)
	}
	commands, err := os.ReadFile(commandLog)
	if err != nil {
		t.Fatalf("ReadFile(command log) error: %v", err)
	}
	if strings.Contains(string(commands), "|-exportOptionsPlist|"+exportOptionsPath) {
		t.Fatalf("xcodebuild received replaceable source path: %q", commands)
	}
}

func TestExportReleaseTestingTerminatesDescendantsAfterXcodebuildExits(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	archivePath, exportOptionsPath := writeReleaseTestingExportFixture(t, tempDir)
	commandLog := filepath.Join(tempDir, "commands.log")
	readyPath := filepath.Join(tempDir, "descendant.ready")
	canaryPath := filepath.Join(tempDir, "descendant.canary")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(string) (string, error) { return "/usr/bin/xcodebuild", nil }
	commandContextFn = helperCommandContext(t, commandLog)
	t.Cleanup(restore)

	_, err := ExportReleaseTesting(context.Background(), ReleaseTestingExportOptions{
		ArchivePath:         archivePath,
		ExportOptionsPath:   exportOptionsPath,
		ExportOptionsSHA256: fileSHA256(t, exportOptionsPath),
		IPAPath:             filepath.Join(tempDir, "Demo.ipa"),
		Environment: []string{
			"GO_WANT_XCODE_HELPER_PROCESS=1",
			"ASC_XCODE_HELPER_LOG=" + commandLog,
			"ASC_XCODE_HELPER_DESCENDANT_READY=" + readyPath,
			"ASC_XCODE_HELPER_DESCENDANT_CANARY=" + canaryPath,
			"ASC_XCODE_HELPER_CANARY_DELAY=4s",
		},
	})
	if err != nil {
		t.Fatalf("ExportReleaseTesting() error: %v", err)
	}
	if _, err := os.Stat(readyPath); err != nil {
		t.Fatalf("descendant did not start: %v", err)
	}
	time.Sleep(4200 * time.Millisecond)
	if _, err := os.Stat(canaryPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("xcodebuild descendant survived successful parent exit: %v", err)
	}
	assertNoReleaseTestingResidues(t, tempDir)
}

func TestExportReleaseTestingTerminatesDescendantsOnContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	archivePath, exportOptionsPath := writeReleaseTestingExportFixture(t, tempDir)
	commandLog := filepath.Join(tempDir, "commands.log")
	readyPath := filepath.Join(tempDir, "descendant.ready")
	canaryPath := filepath.Join(tempDir, "descendant.canary")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(string) (string, error) { return "/usr/bin/xcodebuild", nil }
	commandContextFn = helperCommandContext(t, commandLog)
	t.Cleanup(restore)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		for {
			if _, err := os.Stat(readyPath); err == nil {
				cancel()
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Millisecond):
			}
		}
	}()
	_, err := ExportReleaseTesting(ctx, ReleaseTestingExportOptions{
		ArchivePath:         archivePath,
		ExportOptionsPath:   exportOptionsPath,
		ExportOptionsSHA256: fileSHA256(t, exportOptionsPath),
		IPAPath:             filepath.Join(tempDir, "Demo.ipa"),
		Environment: []string{
			"GO_WANT_XCODE_HELPER_PROCESS=1",
			"ASC_XCODE_HELPER_LOG=" + commandLog,
			"ASC_XCODE_HELPER_DESCENDANT_READY=" + readyPath,
			"ASC_XCODE_HELPER_DESCENDANT_CANARY=" + canaryPath,
			"ASC_XCODE_HELPER_CANARY_DELAY=4s",
			"ASC_XCODE_HELPER_PARENT_DELAY=2s",
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ExportReleaseTesting() error = %v, want context canceled", err)
	}
	if _, err := os.Stat(readyPath); err != nil {
		t.Fatalf("descendant did not start before cancellation: %v", err)
	}
	time.Sleep(4200 * time.Millisecond)
	if _, err := os.Stat(canaryPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("xcodebuild descendant survived context cancellation: %v", err)
	}
	assertNoReleaseTestingResidues(t, tempDir)
}

func TestExportReleaseTestingRemovesPrivateOptionsAfterExportFailure(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	archivePath, exportOptionsPath := writeReleaseTestingExportFixture(t, tempDir)
	commandLog := filepath.Join(tempDir, "commands.log")

	restore := overrideTestEnvironment(t)
	runtimeGOOS = "darwin"
	lookPathFn = func(string) (string, error) { return "/usr/bin/xcodebuild", nil }
	commandContextFn = helperCommandContext(t, commandLog)
	t.Cleanup(restore)

	_, err := ExportReleaseTesting(context.Background(), ReleaseTestingExportOptions{
		ArchivePath:         archivePath,
		ExportOptionsPath:   exportOptionsPath,
		ExportOptionsSHA256: fileSHA256(t, exportOptionsPath),
		IPAPath:             filepath.Join(tempDir, "Demo.ipa"),
		Environment: []string{
			"GO_WANT_XCODE_HELPER_PROCESS=1",
			"ASC_XCODE_HELPER_LOG=" + commandLog,
			"ASC_XCODE_HELPER_EXPORT_FAIL=1",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "xcodebuild export failed") {
		t.Fatalf("ExportReleaseTesting() error = %v, want export failure", err)
	}
	assertNoReleaseTestingResidues(t, tempDir)
}

func TestExportReleaseTestingRejectsUntrustedXcodebuildIPAOutputs(t *testing.T) {
	tests := []struct {
		name      string
		malicious string
		message   string
	}{
		{name: "symlink", malicious: "symlink", message: "symlink"},
		{name: "hard link", malicious: "hardlink", message: "multiple hard links"},
		{name: "directory", malicious: "directory", message: "not a regular file"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			archivePath, exportOptionsPath := writeReleaseTestingExportFixture(t, tempDir)
			commandLog := filepath.Join(tempDir, "commands.log")

			restore := overrideTestEnvironment(t)
			runtimeGOOS = "darwin"
			lookPathFn = func(string) (string, error) { return "/usr/bin/xcodebuild", nil }
			commandContextFn = helperCommandContext(t, commandLog)
			t.Cleanup(restore)

			ipaPath := filepath.Join(tempDir, "final", "Demo.ipa")
			_, err := ExportReleaseTesting(context.Background(), ReleaseTestingExportOptions{
				ArchivePath:         archivePath,
				ExportOptionsPath:   exportOptionsPath,
				ExportOptionsSHA256: fileSHA256(t, exportOptionsPath),
				IPAPath:             ipaPath,
				Environment: []string{
					"GO_WANT_XCODE_HELPER_PROCESS=1",
					"ASC_XCODE_HELPER_LOG=" + commandLog,
					"ASC_XCODE_HELPER_MALICIOUS_IPA=" + test.malicious,
				},
			})
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("ExportReleaseTesting() error = %v, want containing %q", err, test.message)
			}
			if _, statErr := os.Lstat(ipaPath); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("untrusted output produced final IPA: %v", statErr)
			}
		})
	}
}

func TestFinalizeExactExportedIPARejectsSourceMutationDuringCopyVerification(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "Exported.ipa")
	if err := writeTestIPA(sourcePath); err != nil {
		t.Fatalf("writeTestIPA() error: %v", err)
	}
	destinationPath := filepath.Join(directory, "final", "Demo.ipa")
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}

	originalHook := afterExactIPACopyFn
	afterExactIPACopyFn = func(path string) {
		if err := os.WriteFile(path, []byte("changed after copy"), 0o600); err != nil {
			t.Fatalf("WriteFile(mutated source) error: %v", err)
		}
	}
	t.Cleanup(func() { afterExactIPACopyFn = originalHook })

	_, err := finalizeExactExportedIPA(sourcePath, destinationPath)
	if err == nil || !strings.Contains(err.Error(), "changed during verification") {
		t.Fatalf("finalizeExactExportedIPA() error = %v, want source stability rejection", err)
	}
	if _, statErr := os.Lstat(destinationPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("mutated source produced final IPA: %v", statErr)
	}
}

func TestFinalizeExactExportedIPADoesNotReportSuccessWhenDirectorySyncFails(t *testing.T) {
	directory := t.TempDir()
	sourcePath := filepath.Join(directory, "Exported.ipa")
	if err := writeTestIPA(sourcePath); err != nil {
		t.Fatalf("writeTestIPA() error: %v", err)
	}
	destinationPath := filepath.Join(directory, "final", "Demo.ipa")
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}

	wantErr := errors.New("forced destination directory sync failure")
	originalSync := syncExactIPADestinationDirFn
	syncExactIPADestinationDirFn = func(*os.Root) error { return wantErr }
	t.Cleanup(func() { syncExactIPADestinationDirFn = originalSync })

	result, err := finalizeExactExportedIPA(sourcePath, destinationPath)
	if !errors.Is(err, wantErr) {
		t.Fatalf("finalizeExactExportedIPA() error = %v, want sync failure", err)
	}
	if result != (bundleInfo{}) {
		t.Fatalf("finalizeExactExportedIPA() result = %+v, want zero result", result)
	}
	if _, statErr := os.Stat(destinationPath); statErr != nil {
		t.Fatalf("ambiguous published IPA should remain for resume verification: %v", statErr)
	}
}

func assertNoReleaseTestingResidues(t *testing.T, directory string) {
	t.Helper()
	for _, pattern := range []string{".asc-release-testing-options-*", ".asc-xcode-export-*", ".asc-verified-ipa-*"} {
		matches, err := filepath.Glob(filepath.Join(directory, pattern))
		if err != nil {
			t.Fatalf("Glob(%q) error: %v", pattern, err)
		}
		if len(matches) != 0 {
			t.Fatalf("temporary residues for %q: %v", pattern, matches)
		}
	}
}

func writeReleaseTestingExportFixture(t *testing.T, directory string) (string, string) {
	t.Helper()
	archivePath := filepath.Join(directory, "Demo.xcarchive")
	if err := writeArchiveInfoPlist(archivePath); err != nil {
		t.Fatalf("writeArchiveInfoPlist() error: %v", err)
	}
	exportOptionsPath := filepath.Join(directory, "ExportOptions.plist")
	writeExportOptionsPlist(t, exportOptionsPath, map[string]any{
		"method":                     "release-testing",
		"destination":                "export",
		"signingStyle":               "manual",
		"teamID":                     "TEAM123",
		"signingCertificate":         testSigningCertificateSHA1,
		"provisioningProfiles":       map[string]string{"com.example.demo": testProvisioningProfileUUID},
		"iCloudContainerEnvironment": "Production",
	})
	return archivePath, exportOptionsPath
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
