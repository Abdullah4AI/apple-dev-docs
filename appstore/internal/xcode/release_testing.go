package xcode

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"howett.net/plist"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
)

var (
	sha1IdentitySelectorPattern      = regexp.MustCompile(`(?i)^[0-9a-f]{40}$`)
	profileUUIDPattern               = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	processEnvironmentName           = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	afterReleaseTestingOptionsReadFn = func() {}
)

const releaseTestingExportOptionsSizeLimit = 1024 * 1024

// ManualReleaseTestingExportOptions is the exact signing selection for an
// agent-owned release-testing export. Unlike GenerateExportOptions in manual
// mode, this service never discovers or heuristically selects installed
// signing assets.
type ManualReleaseTestingExportOptions struct {
	OutputPath           string
	TeamID               string
	SigningCertificate   string
	ProvisioningProfiles map[string]string
}

// ManualReleaseTestingExportOptionsResult identifies the immutable plist
// artifact written for a release-testing export.
type ManualReleaseTestingExportOptionsResult struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// WriteManualReleaseTestingExportOptions writes a create-only, private
// ExportOptions.plist with fixed release-testing semantics. The caller must
// provide the exact certificate selector and every bundle-to-profile mapping.
func WriteManualReleaseTestingExportOptions(ctx context.Context, opts ManualReleaseTestingExportOptions) (*ManualReleaseTestingExportOptionsResult, error) {
	opts = normalizeManualReleaseTestingExportOptions(opts)
	if err := validateManualReleaseTestingExportOptions(opts); err != nil {
		return nil, err
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}

	data, err := marshalManualReleaseTestingExportOptions(opts)
	if err != nil {
		return nil, err
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}

	root, err := rootfs.New(filepath.Dir(opts.OutputPath))
	if err != nil {
		return nil, fmt.Errorf("open export options root: %w", err)
	}
	defer root.Close()
	if err := root.CreateNewFileAtomic(filepath.Base(opts.OutputPath), data, 0o600); err != nil {
		return nil, fmt.Errorf("write release-testing export options: %w", err)
	}
	digest := sha256.Sum256(data)
	return &ManualReleaseTestingExportOptionsResult{
		Path:   opts.OutputPath,
		SHA256: hex.EncodeToString(digest[:]),
	}, nil
}

// ValidateManualReleaseTestingExportOptions reopens an existing create-only
// artifact and proves its bytes equal the exact plist this request would write.
func ValidateManualReleaseTestingExportOptions(ctx context.Context, opts ManualReleaseTestingExportOptions) (*ManualReleaseTestingExportOptionsResult, error) {
	opts = normalizeManualReleaseTestingExportOptions(opts)
	if err := validateManualReleaseTestingExportOptions(opts); err != nil {
		return nil, err
	}
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	want, err := marshalManualReleaseTestingExportOptions(opts)
	if err != nil {
		return nil, err
	}
	root, err := rootfs.New(filepath.Dir(opts.OutputPath))
	if err != nil {
		return nil, fmt.Errorf("open export options root: %w", err)
	}
	defer root.Close()
	file, err := root.OpenFile(filepath.Base(opts.OutputPath))
	if err != nil {
		return nil, fmt.Errorf("open release-testing export options: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Mode().Perm() != 0o600 || validateExactIPAFileInfo(info, "release-testing export options") != nil {
		return nil, fmt.Errorf("release-testing export options must be a private regular file")
	}
	got, err := io.ReadAll(io.LimitReader(file, releaseTestingExportOptionsSizeLimit+1))
	if err != nil || len(got) > releaseTestingExportOptionsSizeLimit {
		return nil, fmt.Errorf("read release-testing export options")
	}
	if !bytes.Equal(got, want) {
		return nil, fmt.Errorf("release-testing export options differ from the exact planned signing selection")
	}
	after, err := file.Stat()
	if err != nil || validateStableExactIPA(info, after, "release-testing export options") != nil {
		return nil, fmt.Errorf("release-testing export options changed during validation")
	}
	digest := sha256.Sum256(got)
	return &ManualReleaseTestingExportOptionsResult{Path: opts.OutputPath, SHA256: hex.EncodeToString(digest[:])}, nil
}

func marshalManualReleaseTestingExportOptions(opts ManualReleaseTestingExportOptions) ([]byte, error) {
	payload := map[string]any{
		"method":                     exportOptionsMethodReleaseTesting,
		"destination":                exportOptionsDestinationExport,
		"signingStyle":               exportOptionsSigningStyleManual,
		"teamID":                     opts.TeamID,
		"signingCertificate":         opts.SigningCertificate,
		"provisioningProfiles":       cloneProvisioningProfiles(opts.ProvisioningProfiles),
		"iCloudContainerEnvironment": "Production",
	}
	data, err := plist.MarshalIndent(payload, plist.XMLFormat, "\t")
	if err != nil {
		return nil, fmt.Errorf("serialize release-testing export options: %w", err)
	}
	if err := validateManualReleaseTestingPayload(data); err != nil {
		return nil, fmt.Errorf("validate release-testing export options: %w", err)
	}
	return data, nil
}

// ReleaseTestingExportOptions describes a fixed release-testing export run.
// Environment is required and becomes the exact environment of every
// xcodebuild subprocess, including the Xcode version preflight.
type ReleaseTestingExportOptions struct {
	ArchivePath         string
	ExportOptionsPath   string
	ExportOptionsSHA256 string
	IPAPath             string
	Environment         []string
	LogWriter           io.Writer
}

// ExportReleaseTesting verifies the immutable options artifact and exports to
// a new IPA path. It does not expose overwrite or xcodebuild passthrough flags,
// keeping retries safe to resume from an already-published stage artifact.
func ExportReleaseTesting(ctx context.Context, opts ReleaseTestingExportOptions) (*ExportResult, error) {
	opts.ArchivePath = normalizeDirectoryPath(opts.ArchivePath)
	opts.ExportOptionsPath = strings.TrimSpace(opts.ExportOptionsPath)
	opts.ExportOptionsSHA256 = strings.ToLower(strings.TrimSpace(opts.ExportOptionsSHA256))
	opts.IPAPath = strings.TrimSpace(opts.IPAPath)
	if opts.ArchivePath == "" {
		return nil, fmt.Errorf("archive path is required")
	}
	if opts.ExportOptionsPath == "" {
		return nil, fmt.Errorf("export options path is required")
	}
	if opts.IPAPath == "" {
		return nil, fmt.Errorf("IPA path is required")
	}
	if opts.Environment == nil {
		return nil, fmt.Errorf("sanitized xcodebuild environment is required")
	}
	if err := validateProcessEnvironment(opts.Environment); err != nil {
		return nil, err
	}
	if len(opts.ExportOptionsSHA256) != sha256.Size*2 {
		return nil, fmt.Errorf("export options SHA-256 must be 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(opts.ExportOptionsSHA256); err != nil {
		return nil, fmt.Errorf("export options SHA-256 must be 64 hexadecimal characters")
	}
	optionsRoot, err := rootfs.New(filepath.Dir(opts.ExportOptionsPath))
	if err != nil {
		return nil, fmt.Errorf("open release-testing export options root: %w", err)
	}
	defer optionsRoot.Close()
	data, err := optionsRoot.ReadFileLimited(filepath.Base(opts.ExportOptionsPath), releaseTestingExportOptionsSizeLimit)
	if err != nil {
		return nil, fmt.Errorf("read release-testing export options: %w", err)
	}
	digest := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), opts.ExportOptionsSHA256) {
		return nil, fmt.Errorf("release-testing export options SHA-256 does not match the planned artifact")
	}
	if err := validateManualReleaseTestingPayload(data); err != nil {
		return nil, err
	}
	afterReleaseTestingOptionsReadFn()

	// xcodebuild accepts only a pathname for ExportOptions.plist. Snapshot the
	// already-verified bytes under a private operation-owned directory so a
	// replacement of the caller's path cannot alter signing or destination.
	stagingDir, err := os.MkdirTemp("", ".asc-release-testing-options-*")
	if err != nil {
		return nil, fmt.Errorf("create private export options directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)
	if err := os.Chmod(stagingDir, 0o700); err != nil {
		return nil, fmt.Errorf("secure private export options directory: %w", err)
	}
	stagingRoot, err := rootfs.New(stagingDir)
	if err != nil {
		return nil, fmt.Errorf("open private export options root: %w", err)
	}
	defer stagingRoot.Close()
	const snapshotName = "ExportOptions.plist"
	if err := stagingRoot.CreateNewFileAtomic(snapshotName, data, 0o600); err != nil {
		return nil, fmt.Errorf("snapshot verified release-testing export options: %w", err)
	}

	return Export(ctx, ExportOptions{
		ArchivePath:             opts.ArchivePath,
		ExportOptions:           filepath.Join(stagingDir, snapshotName),
		IPAPath:                 opts.IPAPath,
		Environment:             cloneEnvironment(opts.Environment),
		LogWriter:               opts.LogWriter,
		terminateProcessGroup:   true,
		strictExportedIPASource: true,
	})
}

func normalizeManualReleaseTestingExportOptions(opts ManualReleaseTestingExportOptions) ManualReleaseTestingExportOptions {
	opts.OutputPath = strings.TrimSpace(opts.OutputPath)
	if opts.OutputPath != "" {
		opts.OutputPath = filepath.Clean(opts.OutputPath)
	}
	opts.TeamID = strings.TrimSpace(opts.TeamID)
	opts.SigningCertificate = strings.TrimSpace(opts.SigningCertificate)
	if sha1IdentitySelectorPattern.MatchString(opts.SigningCertificate) {
		opts.SigningCertificate = strings.ToUpper(opts.SigningCertificate)
	}
	opts.ProvisioningProfiles = cloneProvisioningProfiles(opts.ProvisioningProfiles)
	return opts
}

func validateManualReleaseTestingExportOptions(opts ManualReleaseTestingExportOptions) error {
	if opts.OutputPath == "" {
		return fmt.Errorf("output path is required")
	}
	if !strings.EqualFold(filepath.Ext(opts.OutputPath), ".plist") {
		return fmt.Errorf("output path must end with .plist")
	}
	if opts.TeamID == "" {
		return fmt.Errorf("team ID is required")
	}
	if opts.SigningCertificate == "" {
		return fmt.Errorf("signing certificate is required")
	}
	if !sha1IdentitySelectorPattern.MatchString(opts.SigningCertificate) {
		return fmt.Errorf("signing certificate must be an exact 40-character SHA-1 fingerprint")
	}
	for bundleID, profile := range opts.ProvisioningProfiles {
		if strings.TrimSpace(bundleID) == "" {
			return fmt.Errorf("manual export options contain an empty bundle identifier")
		}
		profile = strings.TrimSpace(profile)
		if profile == "" {
			return fmt.Errorf("manual export options contain an empty provisioning profile for bundle %q", bundleID)
		}
		if !profileUUIDPattern.MatchString(profile) {
			return fmt.Errorf("provisioning profile for bundle %q must be an exact UUID", bundleID)
		}
	}
	return validateManualExportOptions(manualExportOptions{
		SigningCertificate:   opts.SigningCertificate,
		ProvisioningProfiles: opts.ProvisioningProfiles,
	})
}

func validateManualReleaseTestingPayload(data []byte) error {
	var payload map[string]any
	if _, err := plist.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode release-testing export options: %w", err)
	}
	allowed := map[string]struct{}{
		"method": {}, "destination": {}, "signingStyle": {}, "teamID": {},
		"signingCertificate": {}, "provisioningProfiles": {}, "iCloudContainerEnvironment": {},
	}
	for key := range payload {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("release-testing export options contain unsupported key %q", key)
		}
	}
	if exportOptionsString(payload["method"]) != exportOptionsMethodReleaseTesting {
		return fmt.Errorf("release-testing export options require method=release-testing")
	}
	if exportOptionsString(payload["destination"]) != exportOptionsDestinationExport {
		return fmt.Errorf("release-testing export options require destination=export")
	}
	if exportOptionsString(payload["signingStyle"]) != exportOptionsSigningStyleManual {
		return fmt.Errorf("release-testing export options require signingStyle=manual")
	}
	if exportOptionsString(payload["teamID"]) == "" {
		return fmt.Errorf("release-testing export options require teamID")
	}
	if exportOptionsString(payload["iCloudContainerEnvironment"]) != "Production" {
		return fmt.Errorf("release-testing export options require iCloudContainerEnvironment=Production")
	}
	profiles, err := provisioningProfilesFromPayload(payload["provisioningProfiles"])
	if err != nil {
		return err
	}
	manual := manualExportOptions{
		SigningCertificate:   exportOptionsString(payload["signingCertificate"]),
		ProvisioningProfiles: profiles,
	}
	if !sha1IdentitySelectorPattern.MatchString(manual.SigningCertificate) {
		return fmt.Errorf("release-testing export options require a 40-character signing certificate SHA-1 fingerprint")
	}
	for bundleID, profile := range profiles {
		if !profileUUIDPattern.MatchString(strings.TrimSpace(profile)) {
			return fmt.Errorf("release-testing export options require an exact provisioning profile UUID for bundle %q", bundleID)
		}
	}
	return validateManualExportOptions(manual)
}

func validateProcessEnvironment(environment []string) error {
	seen := make(map[string]struct{}, len(environment))
	for _, entry := range environment {
		if strings.ContainsRune(entry, 0) {
			return fmt.Errorf("xcodebuild environment entry contains a NUL byte")
		}
		name, _, ok := strings.Cut(entry, "=")
		if !ok || !processEnvironmentName.MatchString(name) {
			return fmt.Errorf("invalid xcodebuild environment entry %q", entry)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("duplicate xcodebuild environment variable %q", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func cloneEnvironment(environment []string) []string {
	if environment == nil {
		return nil
	}
	cloned := make([]string, len(environment))
	copy(cloned, environment)
	return cloned
}

func newDestinationExistsError(path string, err error) error {
	if errors.Is(err, os.ErrExist) {
		return fmt.Errorf("--ipa-path already exists: %s (use --overwrite to replace it): %w", path, os.ErrExist)
	}
	return err
}
