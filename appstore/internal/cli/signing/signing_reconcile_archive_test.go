package signing

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"howett.net/plist"
)

func TestReadCodesignEntitlementsUsesValidatedOpenHandle(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("codesign is available only on macOS")
	}
	if _, err := os.Stat("/usr/bin/codesign"); err != nil {
		t.Skip("codesign is unavailable")
	}

	dir := t.TempDir()
	original := filepath.Join(dir, "Original")
	replacement := filepath.Join(dir, "Replacement")
	copyAndSignTestMachO(t, "/usr/bin/true", original, "ORIGINALTEAM")
	copyAndSignTestMachO(t, "/usr/bin/true", replacement, "REPLACEMENTTEAM")

	handle, err := shared.OpenExistingNoFollow(original)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	validated := filepath.Join(dir, "Validated")
	if err := os.Rename(original, validated); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(replacement, original); err != nil {
		t.Fatal(err)
	}

	entitlements, err := readCodesignEntitlements(handle)
	if err != nil {
		t.Fatalf("readCodesignEntitlements() error = %v", err)
	}
	if got := plistString(entitlements["com.apple.developer.team-identifier"]); got != "ORIGINALTEAM" {
		t.Fatalf("team entitlement = %q, want ORIGINALTEAM", got)
	}
}

func TestReadCodesignEntitlementsRejectsUnsupportedHostAtCodesignBoundary(t *testing.T) {
	executablePath := filepath.Join(t.TempDir(), "Executable")
	if err := os.WriteFile(executablePath, []byte("signed executable"), 0o700); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Open(executablePath)
	if err != nil {
		t.Fatal(err)
	}
	defer executable.Close()

	unsupported := errors.New("codesign unavailable")
	originalPlatformCheck := signingReconcilePlatformCheck
	signingReconcilePlatformCheck = func() error { return unsupported }
	t.Cleanup(func() { signingReconcilePlatformCheck = originalPlatformCheck })

	if _, err := readCodesignEntitlements(executable); !errors.Is(err, unsupported) {
		t.Fatalf("readCodesignEntitlements() error = %v, want unsupported host", err)
	}
}

func TestSigningArchiveReadersRejectSymlinkedComponents(t *testing.T) {
	archive := t.TempDir()
	external := t.TempDir()
	plistBytes, err := plist.Marshal(map[string]any{"CFBundleIdentifier": "com.example.external"}, plist.XMLFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(external, "Info.plist"), plistBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(archive, "External.app")); err != nil {
		t.Fatal(err)
	}
	root, err := rootfs.New(archive)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := readSigningPlist(root, "External.app/Info.plist"); err == nil {
		t.Fatal("readSigningPlist() followed a symlinked parent")
	}
	if _, err := listBundleDirectories(root, "External.app", ".app"); err == nil {
		t.Fatal("listBundleDirectories() followed a symlinked directory")
	}
	if err := os.Symlink(filepath.Join(external, "Info.plist"), filepath.Join(archive, "Info.plist")); err != nil {
		t.Fatal(err)
	}
	if _, err := readSigningPlist(root, "Info.plist"); err == nil {
		t.Fatal("readSigningPlist() followed a symlinked final file")
	}
}

func TestValidateTargetApplicationIdentifierAllowsLegacyPrefixAndRejectsMismatch(t *testing.T) {
	target := signingTarget{BundleID: "com.example.app", Entitlements: map[string]any{
		"application-identifier": "LEGACYSEED.com.example.app",
	}}
	if err := validateTargetApplicationIdentifier(target); err != nil {
		t.Fatalf("legacy App ID prefix rejected: %v", err)
	}
	target.Entitlements["application-identifier"] = "LEGACYSEED.com.other.app"
	if err := validateTargetApplicationIdentifier(target); err == nil {
		t.Fatal("mismatched application identifier accepted")
	}
	target.Entitlements["application-identifier"] = "LEGACYSEED.com.example.app"
	target.Entitlements["com.apple.application-identifier"] = "OTHER.com.example.app"
	if err := validateTargetApplicationIdentifier(target); err == nil {
		t.Fatal("conflicting application identifiers accepted")
	}
}

func TestValidateSigningArchivePlatformAcceptsIOSAndRejectsNonIOS(t *testing.T) {
	tests := []struct {
		name    string
		info    map[string]any
		wantErr bool
	}{
		{
			name: "iOS",
			info: map[string]any{
				"CFBundleSupportedPlatforms": []any{"iPhoneOS"},
				"DTPlatformName":             "iphoneos",
			},
		},
		{
			name: "iOS supported platforms only",
			info: map[string]any{
				"CFBundleSupportedPlatforms": []any{"iPhoneOS"},
			},
		},
		{
			name: "iOS platform name only",
			info: map[string]any{
				"DTPlatformName": "iphoneos",
			},
		},
		{
			name: "macOS",
			info: map[string]any{
				"CFBundleSupportedPlatforms": []any{"MacOSX"},
				"DTPlatformName":             "macosx",
			},
			wantErr: true,
		},
		{
			name: "tvOS",
			info: map[string]any{
				"CFBundleSupportedPlatforms": []any{"AppleTVOS"},
				"DTPlatformName":             "appletvos",
			},
			wantErr: true,
		},
		{
			name: "conflicting platform metadata",
			info: map[string]any{
				"CFBundleSupportedPlatforms": []any{"iPhoneOS"},
				"DTPlatformName":             "macosx",
			},
			wantErr: true,
		},
		{
			name:    "missing platform metadata",
			info:    map[string]any{},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSigningArchivePlatform(test.info)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateSigningArchivePlatform() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestInspectSigningArchiveRejectsMacOSBeforeTargetInspection(t *testing.T) {
	originalPlatformCheck := signingReconcilePlatformCheck
	platformChecked := false
	signingReconcilePlatformCheck = func() error {
		platformChecked = true
		return errors.New("codesign unavailable")
	}
	t.Cleanup(func() { signingReconcilePlatformCheck = originalPlatformCheck })

	archive := t.TempDir()
	applicationPath := filepath.Join("Applications", "Example.app")
	applicationDirectory := filepath.Join(archive, "Products", applicationPath)
	if err := os.MkdirAll(applicationDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	archiveInfo, err := plist.Marshal(map[string]any{
		"ApplicationProperties": map[string]any{
			"ApplicationPath": filepath.ToSlash(applicationPath),
			"Team":            "TEAM1",
		},
	}, plist.XMLFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archive, "Info.plist"), archiveInfo, 0o600); err != nil {
		t.Fatal(err)
	}
	applicationInfo, err := plist.Marshal(map[string]any{
		"CFBundleSupportedPlatforms": []string{"MacOSX"},
		"DTPlatformName":             "macosx",
	}, plist.XMLFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(applicationDirectory, "Info.plist"), applicationInfo, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = inspectSigningArchive(archive)
	if err == nil || !strings.Contains(err.Error(), "supports only iOS archives") {
		t.Fatalf("inspectSigningArchive() error = %v, want non-iOS rejection", err)
	}
	if platformChecked {
		t.Fatal("inspectSigningArchive() checked codesign support before rejecting non-iOS evidence")
	}
}

func copyAndSignTestMachO(t *testing.T, source, destination, teamID string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, 0o700); err != nil {
		t.Fatal(err)
	}
	entitlements, err := plist.Marshal(map[string]any{
		"com.apple.developer.team-identifier": teamID,
	}, plist.XMLFormat)
	if err != nil {
		t.Fatal(err)
	}
	entitlementsPath := destination + ".entitlements"
	if err := os.WriteFile(entitlementsPath, entitlements, 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", "--entitlements", entitlementsPath, destination)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("codesign test Mach-O: %v: %s", err, output)
	}
}
