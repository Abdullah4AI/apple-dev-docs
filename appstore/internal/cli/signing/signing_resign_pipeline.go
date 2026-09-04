package signing

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"howett.net/plist"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/infoplist"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

type signingResignCodePlan struct {
	Path             string
	EntitlementsPath string
}

type signingResignPreparedTree struct {
	Archive      signingResignArchive
	CodePlans    []signingResignCodePlan
	SwiftSupport []signingResignSwiftSupportEntry
}

type signingResignSwiftSupportEntry struct {
	RelativePath string
	SizeBytes    int64
	SHA256       string
	Mode         os.FileMode
}

// ErrSigningResignPublicationAmbiguous means the destination file was created
// but its post-publication validation did not complete successfully. Callers
// must inspect the reported artifact before retrying.
var ErrSigningResignPublicationAmbiguous = errors.New("re-signed IPA publication is ambiguous")

// signingResignBeforePublishedHashFn is a no-op production hook used by the
// package tests to make the post-publication cancellation boundary
// deterministic.
var signingResignBeforePublishedHashFn = func() {}

func executeSigningResignImplementation(ctx context.Context, options signingResignOptions) (result signingResignResult, resultErr error) {
	publicStage := signingResignStagePreparation
	publicCode := signingResignCodeFilesystem
	defer func() {
		if resultErr == nil || isSigningResignUsageError(resultErr) {
			return
		}
		resultErr = wrapSigningResignOperationalError(publicStage, publicCode, resultErr)
	}()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := contextError(ctx); err != nil {
		return result, err
	}
	if runtime.GOOS != "darwin" {
		return result, fmt.Errorf("signing resign is supported only on macOS")
	}
	ctx, stopSignals := platformSigningRunContext(ctx)
	defer stopSignals()
	if err := validateSigningResignOptions(options); err != nil {
		return result, signingResignUsage(err)
	}

	inputPath, err := filepath.Abs(filepath.Clean(options.IPAPath))
	if err != nil {
		return result, fmt.Errorf("resolve IPA input: %w", err)
	}
	outputPath, err := filepath.Abs(filepath.Clean(options.OutputPath))
	if err != nil {
		return result, fmt.Errorf("resolve IPA output: %w", err)
	}
	manifestPath, err := filepath.Abs(filepath.Clean(options.ProfilesManifestPath))
	if err != nil {
		return result, fmt.Errorf("resolve profiles manifest: %w", err)
	}
	if filepath.Clean(inputPath) == filepath.Clean(outputPath) {
		return result, fmt.Errorf("IPA input and output must be different paths")
	}

	inputRoot, err := rootfs.New(filepath.Dir(inputPath))
	if err != nil {
		return result, fmt.Errorf("open IPA input directory: %w", err)
	}
	defer inputRoot.Close()
	source, err := inputRoot.OpenFile(filepath.Base(inputPath))
	if err != nil {
		return result, fmt.Errorf("open IPA input: %w", err)
	}
	defer source.Close()
	sourceInfo, err := source.Stat()
	if err != nil {
		return result, fmt.Errorf("inspect IPA input: %w", err)
	}

	outputRoot, err := rootfs.New(filepath.Dir(outputPath))
	if err != nil {
		return result, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactPublish,
			fmt.Errorf("open IPA output directory: %w", err),
		)
	}
	defer outputRoot.Close()
	if err := outputRoot.CheckCreateNewFile(filepath.Base(outputPath)); err != nil {
		return result, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactPublish,
			fmt.Errorf("preflight IPA output: %w", err),
		)
	}
	if sourceInfo.Mode()&os.ModeSymlink != 0 {
		return result, fmt.Errorf("IPA input is a symbolic link")
	}

	stageDir, err := os.MkdirTemp("", "asc-signing-resign.")
	if err != nil {
		return result, fmt.Errorf("create private re-signing directory: %w", err)
	}
	if err := os.Chmod(stageDir, 0o700); err != nil {
		_ = removeSigningResignStage(stageDir)
		return result, fmt.Errorf("secure private re-signing directory: %w", err)
	}
	defer func() {
		cleanupErr := removeSigningResignStage(stageDir)
		if cleanupErr == nil {
			return
		}
		// A cleanup failure after the artifact reached its create-only
		// destination must keep the publication visible to the caller, exactly
		// like the environment-cleanup-after-publication path below.
		published := result.Output.Path != "" || errors.Is(resultErr, ErrSigningResignPublicationAmbiguous)
		resultErr = errors.Join(resultErr, signingResignStageCleanupFailure(published, cleanupErr))
	}()

	stageRoot, err := rootfs.New(stageDir)
	if err != nil {
		return result, fmt.Errorf("open private re-signing directory: %w", err)
	}
	defer stageRoot.Close()
	stageOS, err := stageRoot.OpenRoot()
	if err != nil {
		return result, fmt.Errorf("open private re-signing root: %w", err)
	}
	defer stageOS.Close()
	if err := stageRoot.MkdirAll("tree", 0o700); err != nil {
		return result, fmt.Errorf("create private IPA staging tree: %w", err)
	}
	treeRoot, err := rootfs.New(filepath.Join(stageDir, "tree"))
	if err != nil {
		return result, fmt.Errorf("open private IPA staging tree: %w", err)
	}
	defer treeRoot.Close()
	treeOS, err := stageOS.OpenRoot("tree")
	if err != nil {
		return result, fmt.Errorf("open private IPA staging tree root: %w", err)
	}
	defer treeOS.Close()

	snapshot, inputDigest, err := snapshotSigningResignIPA(ctx, source, sourceInfo.Size(), stageOS)
	if err != nil {
		return result, err
	}
	defer snapshot.Close()
	snapshotInfo, err := snapshot.Stat()
	if err != nil {
		return result, fmt.Errorf("inspect IPA snapshot: %w", err)
	}
	archiveReader, err := zip.NewReader(snapshot, snapshotInfo.Size())
	if err != nil {
		return result, fmt.Errorf("read IPA archive: %w", err)
	}
	if err := validateSigningResignArchive(ctx, archiveReader); err != nil {
		return result, err
	}
	if err := materializeSigningResignArchive(ctx, archiveReader, treeOS); err != nil {
		return result, fmt.Errorf("materialize IPA: %w", err)
	}
	archive, err := discoverSigningResignArchive(ctx, archiveReader, treeRoot)
	if err != nil {
		return result, err
	}
	if err := validateSigningResignTargetIDs(archive.Targets); err != nil {
		return result, err
	}

	manifest, err := readSigningResignManifest(manifestPath)
	if err != nil {
		return result, err
	}
	targetIDs := buildSigningResignTargetIDs(archive.Targets)
	if err := validateSigningResignManifestTargets(manifest, targetIDs); err != nil {
		return result, err
	}
	profiles, err := readSigningResignProfiles(manifestPath, manifest)
	if err != nil {
		return result, err
	}
	for _, profile := range profiles {
		defer clear(profile.Data)
	}
	identityData, err := readBoundedSigningRunFile(options.IdentityPath, signingRunInputLimit, true)
	if err != nil {
		return result, fmt.Errorf("read signing identity failed")
	}
	defer clear(identityData)
	var passwordData []byte
	if strings.TrimSpace(options.IdentityPasswordPath) != "" {
		passwordData, err = readBoundedSigningRunFile(options.IdentityPasswordPath, signingRunPasswordLimit, true)
		if err != nil {
			return result, fmt.Errorf("read signing identity password failed")
		}
		defer clear(passwordData)
	}
	identityPassword := bytes.TrimSuffix(passwordData, []byte("\n"))
	identityPassword = bytes.TrimSuffix(identityPassword, []byte("\r"))
	identity, err := inspectSigningRunIdentity(identityData, identityPassword, signingRunNowFn())
	if err != nil {
		return result, fmt.Errorf("inspect signing identity: %w", err)
	}
	if err := validateSigningResignProfileSet(profiles, identity); err != nil {
		return result, err
	}
	prepared, err := prepareSigningResignTree(ctx, stageRoot, treeRoot, archive, profiles)
	if err != nil {
		return result, err
	}

	teamID, err := signingRunCertificateTeamID(identity.Certificate)
	if err != nil {
		return result, err
	}
	result = signingResignResult{
		SchemaVersion: 1,
		Command:       "signing resign",
		Input: signingResignInputResult{
			SizeBytes: sourceInfo.Size(),
			SHA256:    strings.ToUpper(inputDigest),
		},
		Identity: signingResignIdentityResult{
			CertificateSHA256: identity.CertificateSHA256,
			TeamID:            teamID,
		},
		Verification: signingResignVerification{
			Status: "pending",
			Scope:  "complete-main-app-code-resources-entitlements-profile-and-certificate-binding",
		},
	}
	result.Targets = make([]signingResignTargetResult, 0, len(prepared.Archive.Targets))
	for _, target := range prepared.Archive.Targets {
		profile := profiles[target.BundleID]
		result.Targets = append(result.Targets, signingResignTargetResult{
			Kind: target.Kind, RelativePath: target.RelativePath, BundleID: target.BundleID,
			ProfileClass: profile.Class, ProfileUUID: profile.UUID, ProfileSHA256: profile.SHA256,
			Status: "pending",
		})
	}

	var outputArtifact signingResignArtifactResult
	publicStage = signingResignStageEnvironment
	publicCode = signingResignCodeEnvironment
	if err := runSigningResignEnvironment(ctx, identity, func(signingContext context.Context, keychainPath string) error {
		publicStage = signingResignStageSigning
		publicCode = signingResignCodeSigning
		if err := signSigningResignTree(signingContext, treeRoot.Path(), prepared, identity.CertificateSHA1, keychainPath); err != nil {
			return wrapSigningResignOperationalError(signingResignStageSigning, signingResignCodeSigning, err)
		}
		publicStage = signingResignStageVerification
		publicCode = signingResignCodeVerification
		if err := verifySigningResignTree(signingContext, treeRoot.Path(), prepared, teamID, identity.CertificateSHA256); err != nil {
			return wrapSigningResignOperationalError(signingResignStageVerification, signingResignCodeVerification, err)
		}
		publicStage = signingResignStageArtifact
		publicCode = signingResignCodeArtifactRead
		packedPath, packedSize, packedDigest, err := repackSigningResignTree(signingContext, stageRoot, treeRoot)
		if err != nil {
			return wrapSigningResignOperationalError(signingResignStageArtifact, signingResignCodeFilesystem, err)
		}
		if err := validatePackedSigningResignIPA(signingContext, packedPath, packedSize); err != nil {
			return wrapSigningResignOperationalError(signingResignStageVerification, signingResignCodeVerification, err)
		}
		if err := verifyPackedSigningResignIPA(signingContext, packedPath, packedSize, stageRoot, treeRoot.Path(), prepared, teamID, identity.CertificateSHA256); err != nil {
			return wrapSigningResignOperationalError(signingResignStageVerification, signingResignCodeVerification, err)
		}
		publicStage = signingResignStageArtifact
		publicCode = signingResignCodeArtifactPublish
		if err := outputRoot.MkdirAll(".", 0o755); err != nil {
			return wrapSigningResignOperationalError(
				signingResignStageArtifact,
				signingResignCodeArtifactPublish,
				fmt.Errorf("create IPA output directory: %w", err),
			)
		}
		if err := outputRoot.CheckCreateNewFile(filepath.Base(outputPath)); err != nil {
			return wrapSigningResignOperationalError(
				signingResignStageArtifact,
				signingResignCodeArtifactPublish,
				fmt.Errorf("preflight IPA output: %w", err),
			)
		}
		outputArtifact, err = publishSigningResignOutput(signingContext, outputRoot, filepath.Base(outputPath), packedPath, packedSize, packedDigest)
		return err
	}); err != nil {
		if outputArtifact.Path != "" {
			return result, fmt.Errorf("%w: re-signed IPA was published but environment cleanup failed: %w", ErrSigningResignPublicationAmbiguous, err)
		}
		return result, err
	}
	result.Output = outputArtifact
	result.Verification.Status = "verified"
	for index := range result.Targets {
		result.Targets[index].Status = "verified"
	}
	return result, nil
}

func validateSigningResignOptions(options signingResignOptions) error {
	required := []struct {
		label string
		value string
	}{
		{label: "IPA input", value: options.IPAPath},
		{label: "IPA output", value: options.OutputPath},
		{label: "signing identity", value: options.IdentityPath},
		{label: "profiles manifest", value: options.ProfilesManifestPath},
	}
	for _, item := range required {
		if strings.TrimSpace(item.value) == "" {
			return fmt.Errorf("%s is required", item.label)
		}
		if strings.ContainsRune(item.value, 0) {
			return fmt.Errorf("%s contains a NUL byte", item.label)
		}
	}
	if strings.ContainsRune(options.IdentityPasswordPath, 0) {
		return fmt.Errorf("identity password path contains a NUL byte")
	}
	return nil
}

func prepareSigningResignTree(ctx context.Context, stageRoot, treeRoot rootfs.Root, archive signingResignArchive, profiles map[string]signingResignProfile) (signingResignPreparedTree, error) {
	if err := contextError(ctx); err != nil {
		return signingResignPreparedTree{}, err
	}
	for _, target := range archive.Targets {
		if err := validateSigningResignExistingEntitlements(target.ExistingEntitlements, target.BundleID); err != nil {
			return signingResignPreparedTree{}, fmt.Errorf("target %s existing entitlements: %w", target.BundleID, err)
		}
	}
	if err := stageRoot.MkdirAll("entitlements", 0o700); err != nil {
		return signingResignPreparedTree{}, fmt.Errorf("create private entitlements directory failed")
	}
	prepared := signingResignPreparedTree{Archive: archive}
	for index := range prepared.Archive.Targets {
		target := &prepared.Archive.Targets[index]
		profile, ok := profiles[target.BundleID]
		if !ok {
			return signingResignPreparedTree{}, fmt.Errorf("missing profile for target %s", target.BundleID)
		}
		entitlements, err := buildSigningResignEntitlementsForProfile(target.ExistingEntitlements, profile)
		if err != nil {
			// An unauthorized-claims refusal is public-safe and actionable;
			// keep it that way through the operational boundary instead of
			// flattening the remediation into a bare stage/code message.
			return signingResignPreparedTree{}, wrapSigningResignPublicDetail(
				fmt.Sprintf("target %s entitlements", target.BundleID),
				err,
			)
		}
		entitlementsData, err := marshalSigningResignEntitlements(entitlements)
		if err != nil {
			return signingResignPreparedTree{}, fmt.Errorf("target %s entitlements: %w", target.BundleID, err)
		}
		entitlementsName := filepath.Join("entitlements", fmt.Sprintf("target-%03d.plist", index))
		if err := stageRoot.WriteFile(entitlementsName, entitlementsData, 0o600); err != nil {
			return signingResignPreparedTree{}, fmt.Errorf("write target %s entitlements failed", target.BundleID)
		}
		profileName := filepath.FromSlash(path.Join(target.RelativePath, "embedded.mobileprovision"))
		profileMode := target.ProfileMode
		if profileMode == 0 {
			profileMode = 0o644
		}
		if err := treeRoot.WriteFile(profileName, profile.Data, profileMode); err != nil {
			return signingResignPreparedTree{}, fmt.Errorf("embed profile for target %s failed", target.BundleID)
		}
		target.Profile = profile
		target.EntitlementsPath = filepath.Join(stageRoot.Path(), entitlementsName)
	}

	codePaths, err := enumerateSigningResignMachOFiles(ctx, treeRoot.Path())
	if err != nil {
		return signingResignPreparedTree{}, fmt.Errorf("enumerate Mach-O code failed")
	}
	mainPrefix := filepath.Join(treeRoot.Path(), filepath.FromSlash(prepared.Archive.MainPath)) + string(filepath.Separator)
	targetExecutablePaths := make(map[string]struct{}, len(prepared.Archive.Targets))
	for _, target := range prepared.Archive.Targets {
		targetExecutablePaths[targetExecutablePath(treeRoot.Path(), target)] = struct{}{}
	}
	if err := validateSigningResignPreservedExternalDirectories(ctx, treeRoot.Path()); err != nil {
		return signingResignPreparedTree{}, err
	}
	prepared.SwiftSupport, err = captureSigningResignPreservedInventory(ctx, treeRoot.Path())
	if err != nil {
		return signingResignPreparedTree{}, fmt.Errorf("capture preserved support inventory: %w", err)
	}
	for index, codePath := range codePaths {
		if err := contextError(ctx); err != nil {
			return signingResignPreparedTree{}, err
		}
		if !strings.HasPrefix(codePath, mainPrefix) {
			if isSigningResignPreservedExternalCodePath(treeRoot.Path(), codePath) {
				// SwiftSupport/iphoneos contains Apple-supplied Swift runtime
				// libraries that are distributed beside the app payload. They
				// were provenance-checked as a complete directory above and
				// remain byte-for-byte untouched.
				continue
			}
			return signingResignPreparedTree{}, fmt.Errorf("Mach-O code exists outside the main app")
		}
		if _, isTargetExecutable := targetExecutablePaths[filepath.Clean(codePath)]; isTargetExecutable {
			continue
		}
		target, ok := signingResignTargetForCodePath(prepared.Archive.Targets, treeRoot.Path(), codePath)
		if !ok {
			return signingResignPreparedTree{}, fmt.Errorf("Mach-O code is not contained by an app-like target")
		}
		if err := validateSigningResignNestedExecutableMode(ctx, treeRoot, codePath); err != nil {
			return signingResignPreparedTree{}, err
		}
		entitlements, err := readSigningResignEntitlements(ctx, codePath)
		if err != nil {
			return signingResignPreparedTree{}, fmt.Errorf("read nested code entitlements failed")
		}
		if err := validateSigningResignNestedEntitlements(entitlements, profiles[target.BundleID].Entitlements); err != nil {
			displayPath, _ := filepath.Rel(treeRoot.Path(), codePath)
			return signingResignPreparedTree{}, fmt.Errorf("nested code %s entitlements: %w", filepath.ToSlash(displayPath), err)
		}
		var entitlementsPath string
		if len(entitlements) > 0 {
			data, err := marshalSigningResignEntitlements(entitlements)
			if err != nil {
				return signingResignPreparedTree{}, err
			}
			name := filepath.Join("entitlements", fmt.Sprintf("code-%06d.plist", index))
			if err := stageRoot.WriteFile(name, data, 0o600); err != nil {
				return signingResignPreparedTree{}, fmt.Errorf("write nested code entitlements failed")
			}
			entitlementsPath = filepath.Join(stageRoot.Path(), name)
		}
		prepared.CodePlans = append(prepared.CodePlans, signingResignCodePlan{Path: codePath, EntitlementsPath: entitlementsPath})
	}
	return prepared, nil
}

func validateSigningResignNestedExecutableMode(ctx context.Context, tree rootfs.Root, codePath string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	relative, err := filepath.Rel(tree.Path(), codePath)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("nested executable is outside the staging tree")
	}
	file, err := tree.OpenFile(relative)
	if err != nil {
		return fmt.Errorf("inspect nested executable mode: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect nested executable mode: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("nested executable is not a regular file")
	}
	if info.Mode().Perm()&0o100 == 0 {
		return fmt.Errorf("nested executable file mode is missing the owner-execute permission")
	}
	if err := contextError(ctx); err != nil {
		return err
	}
	return nil
}

func isSigningResignPreservedExternalCodePath(treeRoot, codePath string) bool {
	relative, err := filepath.Rel(treeRoot, codePath)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return false
	}
	relative = filepath.ToSlash(relative)
	if relative == "WatchKitSupport2/WK" {
		// App Store-exported Watch IPAs carry the distribution-side WK shim
		// binary beside the payload. It is provenance-checked and preserved
		// byte-for-byte, never re-signed.
		return true
	}
	const prefix = "SwiftSupport/iphoneos/"
	if !strings.HasPrefix(relative, prefix) {
		return false
	}
	name := strings.TrimPrefix(relative, prefix)
	return name != "" && !strings.ContainsRune(name, '/') && name != ".dylib" && strings.HasSuffix(name, ".dylib")
}

func verifySigningResignPreservedExternalCode(ctx context.Context, codePath string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if _, err := runSigningResignToolFn(ctx, "/usr/bin/codesign", "--verify", "--strict", "--all-architectures", "-R=anchor apple generic", codePath); err != nil {
		return err
	}
	return nil
}

func validateSigningResignSwiftSupport(ctx context.Context, treeRoot string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	swiftSupportRoot := filepath.Join(treeRoot, "SwiftSupport")
	info, err := os.Lstat(swiftSupportRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect SwiftSupport directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("SwiftSupport is not a regular directory")
	}
	entries, err := os.ReadDir(swiftSupportRoot)
	if err != nil {
		return fmt.Errorf("read SwiftSupport directory: %w", err)
	}
	if len(entries) != 1 || entries[0].Name() != "iphoneos" {
		return fmt.Errorf("SwiftSupport must contain only the iphoneos directory")
	}
	swiftRoot := filepath.Join(swiftSupportRoot, "iphoneos")
	platformInfo, err := os.Lstat(swiftRoot)
	if err != nil {
		return fmt.Errorf("inspect SwiftSupport/iphoneos directory: %w", err)
	}
	if platformInfo.Mode()&os.ModeSymlink != 0 || !platformInfo.IsDir() {
		return fmt.Errorf("SwiftSupport/iphoneos is not a regular directory")
	}
	entries, err = os.ReadDir(swiftRoot)
	if err != nil {
		return fmt.Errorf("read SwiftSupport/iphoneos directory: %w", err)
	}
	for _, entry := range entries {
		if err := contextError(ctx); err != nil {
			return err
		}
		name := entry.Name()
		candidate := filepath.Join(swiftRoot, name)
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("SwiftSupport/iphoneos contains a nested or symbolic-link entry")
		}
		entryInfo, err := entry.Info()
		if err != nil || !entryInfo.Mode().IsRegular() {
			return fmt.Errorf("SwiftSupport/iphoneos contains a non-regular entry")
		}
		if name == ".dylib" || !strings.HasSuffix(name, ".dylib") {
			return fmt.Errorf("SwiftSupport/iphoneos contains an unsupported entry")
		}
		if err := verifySigningResignPreservedExternalCode(ctx, candidate); err != nil {
			return fmt.Errorf("verify preserved SwiftSupport code failed: %w", err)
		}
	}
	return nil
}

// validateSigningResignWatchKitSupport enforces the standard Watch
// distribution layout: an optional WatchKitSupport2 directory containing
// exactly the regular, non-symlink WK binary, whose Apple provenance is
// verified before it is preserved.
func validateSigningResignWatchKitSupport(ctx context.Context, treeRoot string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	watchRoot := filepath.Join(treeRoot, "WatchKitSupport2")
	info, err := os.Lstat(watchRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect WatchKitSupport2 directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("WatchKitSupport2 is not a regular directory")
	}
	entries, err := os.ReadDir(watchRoot)
	if err != nil {
		return fmt.Errorf("read WatchKitSupport2 directory: %w", err)
	}
	if len(entries) != 1 || entries[0].Name() != "WK" {
		return fmt.Errorf("WatchKitSupport2 must contain only the WK binary")
	}
	entry := entries[0]
	if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("WatchKitSupport2 contains a nested or symbolic-link entry")
	}
	entryInfo, err := entry.Info()
	if err != nil || !entryInfo.Mode().IsRegular() {
		return fmt.Errorf("WatchKitSupport2 contains a non-regular entry")
	}
	if entryInfo.Mode().Perm()&0o100 == 0 {
		return fmt.Errorf("WatchKitSupport2/WK is missing the owner-execute permission")
	}
	if err := verifySigningResignPreservedExternalCode(ctx, filepath.Join(watchRoot, "WK")); err != nil {
		return fmt.Errorf("verify preserved WatchKitSupport2 code failed: %w", err)
	}
	return nil
}

func captureSigningResignWatchKitSupportInventory(ctx context.Context, treeRoot string) ([]signingResignSwiftSupportEntry, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	candidate := filepath.Join(treeRoot, "WatchKitSupport2", "WK")
	entryInfo, err := os.Lstat(candidate)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect WatchKitSupport2 entry: %w", err)
	}
	if entryInfo.Mode()&os.ModeSymlink != 0 || !entryInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("WatchKitSupport2 contains a non-regular entry")
	}
	if entryInfo.Size() > signingResignSwiftSupportMaxBytes {
		return nil, fmt.Errorf("WatchKitSupport2 entry exceeds %d bytes", signingResignSwiftSupportMaxBytes)
	}
	digest, err := hashSigningResignFile(ctx, candidate, entryInfo.Size())
	if err != nil {
		return nil, fmt.Errorf("hash WatchKitSupport2 entry: %w", err)
	}
	return []signingResignSwiftSupportEntry{{
		RelativePath: "WatchKitSupport2/WK",
		SizeBytes:    entryInfo.Size(),
		SHA256:       digest,
		Mode:         entryInfo.Mode().Perm(),
	}}, nil
}

// validateSigningResignPreservedExternalDirectories checks every supported
// distribution-side directory that is preserved instead of re-signed.
func validateSigningResignPreservedExternalDirectories(ctx context.Context, treeRoot string) error {
	if err := validateSigningResignSwiftSupport(ctx, treeRoot); err != nil {
		return err
	}
	return validateSigningResignWatchKitSupport(ctx, treeRoot)
}

// captureSigningResignPreservedInventory records the sorted path, size,
// digest, and mode of every preserved distribution-side runtime so repack can
// be held to byte-for-byte equality.
func captureSigningResignPreservedInventory(ctx context.Context, treeRoot string) ([]signingResignSwiftSupportEntry, error) {
	swift, err := captureSigningResignSwiftSupportInventory(ctx, treeRoot)
	if err != nil {
		return nil, err
	}
	watch, err := captureSigningResignWatchKitSupportInventory(ctx, treeRoot)
	if err != nil {
		return nil, err
	}
	combined := append(swift, watch...)
	sort.Slice(combined, func(left, right int) bool {
		return combined[left].RelativePath < combined[right].RelativePath
	})
	if len(combined) == 0 {
		return nil, nil
	}
	return combined, nil
}

func captureSigningResignSwiftSupportInventory(ctx context.Context, treeRoot string) ([]signingResignSwiftSupportEntry, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	swiftRoot := filepath.Join(treeRoot, "SwiftSupport", "iphoneos")
	info, err := os.Lstat(filepath.Join(treeRoot, "SwiftSupport"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect SwiftSupport directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("SwiftSupport is not a regular directory")
	}
	platformInfo, err := os.Lstat(swiftRoot)
	if err != nil {
		return nil, fmt.Errorf("inspect SwiftSupport/iphoneos directory: %w", err)
	}
	if platformInfo.Mode()&os.ModeSymlink != 0 || !platformInfo.IsDir() {
		return nil, fmt.Errorf("SwiftSupport/iphoneos is not a regular directory")
	}
	entries, err := os.ReadDir(swiftRoot)
	if err != nil {
		return nil, fmt.Errorf("read SwiftSupport/iphoneos directory: %w", err)
	}
	inventory := make([]signingResignSwiftSupportEntry, 0, len(entries))
	for _, entry := range entries {
		if err := contextError(ctx); err != nil {
			return nil, err
		}
		candidate := filepath.Join(swiftRoot, entry.Name())
		entryInfo, err := os.Lstat(candidate)
		if err != nil {
			return nil, fmt.Errorf("inspect SwiftSupport entry: %w", err)
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 || !entryInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("SwiftSupport/iphoneos contains a non-regular entry")
		}
		if entryInfo.Size() > signingResignSwiftSupportMaxBytes {
			return nil, fmt.Errorf("SwiftSupport entry exceeds %d bytes", signingResignSwiftSupportMaxBytes)
		}
		digest, err := hashSigningResignFile(ctx, candidate, entryInfo.Size())
		if err != nil {
			return nil, fmt.Errorf("hash SwiftSupport entry: %w", err)
		}
		inventory = append(inventory, signingResignSwiftSupportEntry{
			RelativePath: filepath.ToSlash(filepath.Join("SwiftSupport", "iphoneos", entry.Name())),
			SizeBytes:    entryInfo.Size(),
			SHA256:       digest,
			Mode:         entryInfo.Mode().Perm(),
		})
	}
	sort.Slice(inventory, func(left, right int) bool {
		return inventory[left].RelativePath < inventory[right].RelativePath
	})
	return inventory, nil
}

func signingResignTargetForCodePath(targets []signingResignTarget, treeRoot, codePath string) (signingResignTarget, bool) {
	var selected signingResignTarget
	selectedLength := -1
	for _, target := range targets {
		prefix := filepath.Join(treeRoot, filepath.FromSlash(target.RelativePath)) + string(filepath.Separator)
		if strings.HasPrefix(codePath, prefix) && len(prefix) > selectedLength {
			selected = target
			selectedLength = len(prefix)
		}
	}
	return selected, selectedLength >= 0
}

func validateSigningResignNestedEntitlements(entitlements, profile map[string]any) error {
	for key, value := range entitlements {
		if _, identityKey := signingResignIdentityEntitlementKeys[key]; identityKey {
			return fmt.Errorf("identity entitlement %s is not allowed on nested non-app code", key)
		}
		profileValue, exists := profile[key]
		if !exists || !signingResignEntitlementValuePermits(profileValue, value) {
			return fmt.Errorf("entitlement %s is not permitted by its target profile", key)
		}
	}
	return nil
}

func signSigningResignTree(ctx context.Context, treePath string, prepared signingResignPreparedTree, identitySHA1, keychainPath string) (resultErr error) {
	defer func() {
		resultErr = wrapSigningResignOperationalError(
			signingResignStageSigning,
			signingResignCodeSigning,
			resultErr,
		)
	}()
	plans := append([]signingResignCodePlan(nil), prepared.CodePlans...)
	sortSigningResignCodePlans(plans)
	targetExecutablePaths := make(map[string]struct{}, len(prepared.Archive.Targets))
	for _, target := range prepared.Archive.Targets {
		targetExecutablePaths[targetExecutablePath(treePath, target)] = struct{}{}
	}
	for _, plan := range plans {
		if _, targetExecutable := targetExecutablePaths[filepath.Clean(plan.Path)]; targetExecutable {
			continue
		}
		if err := signSigningResignObject(ctx, plan.Path, identitySHA1, keychainPath, plan.EntitlementsPath); err != nil {
			return fmt.Errorf("sign nested code %s: %w", signingResignDisplayPath(treePath, plan.Path), err)
		}
	}
	containers := signingResignFrameworkContainers(treePath, plans)
	for _, container := range containers {
		if err := signSigningResignObject(ctx, container, identitySHA1, keychainPath, ""); err != nil {
			return fmt.Errorf("sign code container %s: %w", signingResignDisplayPath(treePath, container), err)
		}
	}
	for _, target := range prepared.Archive.Targets {
		targetPath := filepath.Join(treePath, filepath.FromSlash(target.RelativePath))
		if err := signSigningResignObject(ctx, targetPath, identitySHA1, keychainPath, target.EntitlementsPath); err != nil {
			return fmt.Errorf("sign target %s: %w", target.BundleID, err)
		}
	}
	return nil
}

// isSigningResignCodeContainerName reports whether a directory name is a
// supported nested code container whose signature must be refreshed after the
// code inside it changes. App-like bundles (.app, .appex) are signed as
// discovered targets instead.
func isSigningResignCodeContainerName(name string) bool {
	for _, suffix := range []string{".framework", ".bundle", ".xpc"} {
		if name != suffix && strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func sortSigningResignCodePlans(plans []signingResignCodePlan) {
	sort.Slice(plans, func(left, right int) bool {
		leftDepth := strings.Count(plans[left].Path, string(filepath.Separator))
		rightDepth := strings.Count(plans[right].Path, string(filepath.Separator))
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		return plans[left].Path < plans[right].Path
	})
}

// signingResignFrameworkContainers returns every ancestor code container of a
// scheduled code plan, deepest first, so each container's signature and
// resource seal are refreshed after its contained code changes.
func signingResignFrameworkContainers(treePath string, plans []signingResignCodePlan) []string {
	seen := make(map[string]struct{})
	for _, plan := range plans {
		candidate := filepath.Dir(plan.Path)
		for candidate != treePath && strings.HasPrefix(candidate, treePath+string(filepath.Separator)) {
			if isSigningResignCodeContainerName(filepath.Base(candidate)) {
				seen[candidate] = struct{}{}
			}
			candidate = filepath.Dir(candidate)
		}
	}
	containers := make([]string, 0, len(seen))
	for candidate := range seen {
		containers = append(containers, candidate)
	}
	sort.Slice(containers, func(left, right int) bool {
		leftDepth := strings.Count(strings.TrimPrefix(containers[left], treePath), string(filepath.Separator))
		rightDepth := strings.Count(strings.TrimPrefix(containers[right], treePath), string(filepath.Separator))
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		return containers[left] < containers[right]
	})
	return containers
}

func verifySigningResignTree(ctx context.Context, treePath string, prepared signingResignPreparedTree, teamID, certificateSHA256 string) (resultErr error) {
	defer func() {
		resultErr = wrapSigningResignOperationalError(
			signingResignStageVerification,
			signingResignCodeVerification,
			resultErr,
		)
	}()
	plans := append([]signingResignCodePlan(nil), prepared.CodePlans...)
	for _, plan := range plans {
		if err := verifySigningResignObject(ctx, plan.Path, teamID, false); err != nil {
			return fmt.Errorf("verify nested code %s: %w", signingResignDisplayPath(treePath, plan.Path), err)
		}
		if err := verifySigningResignCertificate(ctx, plan.Path, certificateSHA256); err != nil {
			return fmt.Errorf("verify nested code certificate: %w", err)
		}
		if err := validateSigningResignCodeEntitlements(ctx, plan); err != nil {
			return fmt.Errorf("verify nested code entitlements: %w", err)
		}
	}
	for _, target := range prepared.Archive.Targets {
		targetPath := filepath.Join(treePath, filepath.FromSlash(target.RelativePath))
		if err := verifySigningResignObject(ctx, targetPath, teamID, false); err != nil {
			return fmt.Errorf("verify target %s: %w", target.BundleID, err)
		}
		if err := verifySigningResignCertificate(ctx, targetPath, certificateSHA256); err != nil {
			return fmt.Errorf("verify target %s certificate: %w", target.BundleID, err)
		}
		entitlements, err := readSigningResignEntitlements(ctx, targetExecutablePath(treePath, target))
		if err != nil {
			return fmt.Errorf("read verified target %s entitlements: %w", target.BundleID, err)
		}
		if strings.TrimSpace(target.EntitlementsPath) == "" {
			return fmt.Errorf("target %s generated entitlements document is missing", target.BundleID)
		}
		if err := validateSigningResignEntitlementsAgainstDocumentAtStage(entitlements, target.EntitlementsPath, fmt.Sprintf("target %s signed entitlements", target.BundleID), signingResignStageVerification); err != nil {
			return err
		}
		profileData, err := readRootedSigningResignFile(treePath, filepath.FromSlash(path.Join(target.RelativePath, "embedded.mobileprovision")), signingResignProfileMaxBytes)
		if err != nil {
			return fmt.Errorf("read verified target %s profile failed", target.BundleID)
		}
		if digest := signingResignSHA256(profileData); !strings.EqualFold(digest, target.Profile.SHA256) {
			return fmt.Errorf("verified target %s profile digest changed", target.BundleID)
		}
	}
	mainPath := filepath.Join(treePath, filepath.FromSlash(prepared.Archive.MainPath))
	if err := verifySigningResignObject(ctx, mainPath, teamID, true); err != nil {
		return fmt.Errorf("verify complete main app: %w", err)
	}
	return nil
}

func signingResignDisplayPath(rootPath, candidate string) string {
	relative, err := filepath.Rel(rootPath, candidate)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "staged-code"
	}
	return filepath.ToSlash(relative)
}

func validateSigningResignCodeEntitlements(ctx context.Context, plan signingResignCodePlan) error {
	actual, err := readSigningResignEntitlements(ctx, plan.Path)
	if err != nil {
		return err
	}
	return validateSigningResignEntitlementsAgainstDocumentAtStage(actual, plan.EntitlementsPath, "signed entitlements", signingResignStageVerification)
}

func validateSigningResignEntitlementsAgainstDocument(actual map[string]any, documentPath, subject string) error {
	return validateSigningResignEntitlementsAgainstDocumentAtStage(actual, documentPath, subject, signingResignStagePreparation)
}

func validateSigningResignEntitlementsAgainstDocumentAtStage(actual map[string]any, documentPath, subject string, stage signingResignOperationalStage) error {
	want, err := readSigningResignGeneratedEntitlementsAtStage(documentPath, stage)
	if err != nil {
		return err
	}
	if !signingResignEntitlementMapsEqual(actual, want) {
		return fmt.Errorf("%s do not exactly match the generated document", subject)
	}
	return nil
}

func readSigningResignGeneratedEntitlementsAtStage(documentPath string, stage signingResignOperationalStage) (map[string]any, error) {
	want := map[string]any{}
	if strings.TrimSpace(documentPath) == "" {
		return want, nil
	}
	data, err := readBoundedSigningRunFile(documentPath, infoplist.MaxBytes, false)
	if err != nil {
		return nil, wrapSigningResignOperationalError(
			stage,
			signingResignCodeGeneratedEntitlements,
			fmt.Errorf("read generated entitlements: %w", err),
		)
	}
	defer clear(data)
	if _, err := plist.Unmarshal(data, &want); err != nil {
		return nil, fmt.Errorf("decode generated entitlements: %w", err)
	}
	if want == nil {
		want = map[string]any{}
	}
	return want, nil
}

func signingResignEntitlementMapsEqual(left, right map[string]any) bool {
	if len(left) != len(right) {
		return false
	}
	for key, expected := range right {
		actual, exists := left[key]
		if !exists || !signingResignEntitlementValuesEqual(actual, expected) {
			return false
		}
	}
	return true
}

func signingResignEntitlementValuesEqual(left, right any) bool {
	leftList, leftIsList := signingResignEntitlementList(left)
	rightList, rightIsList := signingResignEntitlementList(right)
	if leftIsList || rightIsList {
		if !leftIsList || !rightIsList || len(leftList) != len(rightList) {
			return false
		}
		for index := range leftList {
			if !signingResignEntitlementValuesEqual(leftList[index], rightList[index]) {
				return false
			}
		}
		return true
	}
	leftMap, leftIsMap := left.(map[string]any)
	rightMap, rightIsMap := right.(map[string]any)
	if leftIsMap || rightIsMap {
		if !leftIsMap || !rightIsMap || len(leftMap) != len(rightMap) {
			return false
		}
		for key, expected := range rightMap {
			actual, exists := leftMap[key]
			if !exists || !signingResignEntitlementValuesEqual(actual, expected) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(left, right)
}

func readRootedSigningResignFile(rootPath, relativePath string, limit int64) ([]byte, error) {
	root, err := rootfs.New(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFileLimited(relativePath, limit)
}

func signingResignSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return strings.ToUpper(hex.EncodeToString(digest[:]))
}

func removeSigningResignStage(stagePath string) error {
	clean := filepath.Clean(stagePath)
	if filepath.Dir(clean) != filepath.Clean(os.TempDir()) || !strings.HasPrefix(filepath.Base(clean), "asc-signing-resign.") {
		return fmt.Errorf("refusing to remove unexpected re-signing directory %q", stagePath)
	}
	info, err := os.Lstat(clean)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("refusing to remove unsafe re-signing directory")
	}
	return os.RemoveAll(clean)
}

func signingResignRepackEntryLimitError(count int) error {
	if count > signingResignMaxArchiveEntries {
		return fmt.Errorf("repacked IPA would exceed the archive entry limit")
	}
	return nil
}

func repackSigningResignTree(ctx context.Context, stageRoot, treeRoot rootfs.Root) (packedPath string, packedSize int64, packedDigest string, resultErr error) {
	defer func() {
		resultErr = wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeFilesystem,
			resultErr,
		)
	}()
	if err := contextError(ctx); err != nil {
		return "", 0, "", err
	}
	type repackEntry struct {
		relative  string
		directory bool
		mode      os.FileMode
	}
	entries := make([]repackEntry, 0)
	fileCount := 0
	err := filepath.WalkDir(treeRoot.Path(), func(candidate string, entry os.DirEntry, walkErr error) error {
		if err := contextError(ctx); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("staging tree contains a symbolic link")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			relative, err := filepath.Rel(treeRoot.Path(), candidate)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return fmt.Errorf("staging tree contains an invalid relative path")
			}
			if relative == "." {
				return nil
			}
			// Directory entries carry validated modes and can be empty, so a
			// faithful repack must record them explicitly instead of relying
			// on ancestors implied by file paths.
			entries = append(entries, repackEntry{relative: relative, directory: true, mode: info.Mode().Perm()})
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("staging tree contains a non-regular file")
		}
		relative, err := filepath.Rel(treeRoot.Path(), candidate)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("staging tree contains an invalid relative path")
		}
		entries = append(entries, repackEntry{relative: relative})
		fileCount++
		return nil
	})
	if err != nil {
		return "", 0, "", err
	}
	if fileCount == 0 {
		return "", 0, "", fmt.Errorf("staging tree is empty")
	}
	if err := signingResignRepackEntryLimitError(len(entries)); err != nil {
		return "", 0, "", err
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].relative < entries[right].relative
	})
	stageOS, err := stageRoot.OpenRoot()
	if err != nil {
		return "", 0, "", err
	}
	defer stageOS.Close()
	packed, err := secureopen.OpenNewFileNoFollowInRoot(stageOS, "resigned.ipa", 0o600)
	if err != nil {
		return "", 0, "", fmt.Errorf("create re-signed IPA: %w", err)
	}
	zipWriter := zip.NewWriter(packed)
	var operationErr error
	for _, item := range entries {
		if err := contextError(ctx); err != nil {
			operationErr = err
			break
		}
		if item.directory {
			header := &zip.FileHeader{Name: filepath.ToSlash(item.relative) + "/", Method: zip.Store}
			header.Modified = time.Unix(0, 0)
			header.SetMode(os.ModeDir | item.mode)
			if _, err := zipWriter.CreateHeader(header); err != nil {
				operationErr = err
				break
			}
			continue
		}
		file, err := treeRoot.OpenFile(item.relative)
		if err != nil {
			operationErr = err
			break
		}
		info, err := file.Stat()
		if err != nil {
			_ = file.Close()
			operationErr = err
			break
		}
		header := &zip.FileHeader{Name: filepath.ToSlash(item.relative), Method: zip.Deflate}
		header.Modified = time.Unix(0, 0)
		header.SetMode(info.Mode().Perm())
		entry, err := zipWriter.CreateHeader(header)
		if err == nil {
			_, err = copySigningResignWithContext(ctx, entry, io.LimitReader(file, info.Size()+1), info.Size())
		}
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			operationErr = err
			break
		}
	}
	if operationErr == nil {
		operationErr = zipWriter.Close()
	} else {
		_ = zipWriter.Close()
	}
	if operationErr == nil {
		operationErr = packed.Sync()
	}
	closeErr := packed.Close()
	if operationErr != nil || closeErr != nil {
		_ = os.Remove(filepath.Join(stageRoot.Path(), "resigned.ipa"))
		return "", 0, "", errors.Join(operationErr, closeErr)
	}
	packedPath = filepath.Join(stageRoot.Path(), "resigned.ipa")
	info, err := os.Stat(packedPath)
	if err != nil {
		return "", 0, "", err
	}
	digest, err := hashSigningResignFile(ctx, packedPath, info.Size())
	if err != nil {
		return "", 0, "", err
	}
	return packedPath, info.Size(), digest, nil
}

func validatePackedSigningResignIPA(ctx context.Context, packedPath string, size int64) error {
	if err := contextError(ctx); err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageVerification,
			signingResignCodeVerification,
			err,
		)
	}
	file, err := os.Open(packedPath)
	if err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageVerification,
			signingResignCodeArtifactRead,
			fmt.Errorf("open re-signed IPA: %w", err),
		)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() != size {
		return fmt.Errorf("re-signed IPA changed before publication")
	}
	reader, err := zip.NewReader(file, size)
	if err != nil {
		return fmt.Errorf("read re-signed IPA: %w", err)
	}
	return validateSigningResignArchive(ctx, reader)
}

func verifyPackedSigningResignIPA(ctx context.Context, packedPath string, size int64, stageRoot rootfs.Root, originalTreePath string, original signingResignPreparedTree, teamID, certificateSHA256 string) error {
	if err := contextError(ctx); err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageVerification,
			signingResignCodeVerification,
			err,
		)
	}
	file, err := os.Open(packedPath)
	if err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageVerification,
			signingResignCodeArtifactRead,
			fmt.Errorf("open re-signed IPA for final verification: %w", err),
		)
	}
	defer file.Close()
	reader, err := zip.NewReader(file, size)
	if err != nil {
		return fmt.Errorf("read re-signed IPA for final verification: %w", err)
	}
	if err := validateSigningResignArchive(ctx, reader); err != nil {
		return err
	}
	if err := stageRoot.MkdirAll("packed-tree", 0o700); err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeFilesystem,
			fmt.Errorf("create final verification tree: %w", err),
		)
	}
	stageOS, err := stageRoot.OpenRoot()
	if err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeFilesystem,
			fmt.Errorf("open final verification root: %w", err),
		)
	}
	defer stageOS.Close()
	packedTreeOS, err := stageOS.OpenRoot("packed-tree")
	if err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeFilesystem,
			fmt.Errorf("open final verification tree: %w", err),
		)
	}
	defer packedTreeOS.Close()
	if err := materializeSigningResignArchive(ctx, reader, packedTreeOS); err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeFilesystem,
			fmt.Errorf("materialize final verification tree: %w", err),
		)
	}
	if err := validateSigningResignPreservedExternalDirectories(ctx, filepath.Join(stageRoot.Path(), "packed-tree")); err != nil {
		return fmt.Errorf("verify preserved SwiftSupport after repack: %w", err)
	}
	packedTreeRoot, err := rootfs.New(filepath.Join(stageRoot.Path(), "packed-tree"))
	if err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeFilesystem,
			fmt.Errorf("open final verification tree: %w", err),
		)
	}
	defer packedTreeRoot.Close()
	packedSwiftSupport, err := captureSigningResignPreservedInventory(ctx, packedTreeRoot.Path())
	if err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactRead,
			fmt.Errorf("capture packed preserved support inventory: %w", err),
		)
	}
	if err := validateSigningResignSwiftSupportInventory(packedSwiftSupport, original.SwiftSupport); err != nil {
		return wrapSigningResignOperationalError(
			signingResignStageVerification,
			signingResignCodeVerification,
			err,
		)
	}
	if err := validateSigningResignPackedCodeInventory(ctx, packedTreeRoot.Path(), originalTreePath, original); err != nil {
		return fmt.Errorf("verify packed Mach-O inventory: %w", err)
	}
	archive, err := discoverSigningResignArchive(ctx, reader, packedTreeRoot)
	if err != nil {
		return fmt.Errorf("inspect final verification targets: %w", err)
	}
	if archive.MainPath != original.Archive.MainPath || len(archive.Targets) != len(original.Archive.Targets) {
		return fmt.Errorf("re-signed IPA target inventory changed during repack")
	}
	for index, target := range archive.Targets {
		want := original.Archive.Targets[index]
		if target.Kind != want.Kind || target.RelativePath != want.RelativePath || target.BundleID != want.BundleID || target.Executable != want.Executable || target.ProfileMode.Perm() != want.ProfileMode.Perm() {
			return fmt.Errorf("re-signed IPA target inventory changed during repack")
		}
		profileData, err := readRootedSigningResignFile(packedTreeRoot.Path(), filepath.FromSlash(path.Join(target.RelativePath, "embedded.mobileprovision")), signingResignProfileMaxBytes)
		if err != nil || !strings.EqualFold(signingResignSHA256(profileData), want.Profile.SHA256) {
			return fmt.Errorf("re-signed IPA target profile changed during repack")
		}
	}
	finalPrepared, err := rebaseSigningResignPreparedTree(original, originalTreePath, packedTreeRoot.Path())
	if err != nil {
		return fmt.Errorf("rebase final verification targets: %w", err)
	}
	if err := verifySigningResignTree(ctx, packedTreeRoot.Path(), finalPrepared, teamID, certificateSHA256); err != nil {
		return fmt.Errorf("verify re-signed IPA after repack: %w", err)
	}
	return nil
}

func validateSigningResignSwiftSupportInventory(actual, expected []signingResignSwiftSupportEntry) error {
	if !slices.Equal(actual, expected) {
		return fmt.Errorf("re-signed IPA SwiftSupport inventory changed during repack")
	}
	return nil
}

func validateSigningResignPackedCodeInventory(ctx context.Context, packedTreePath, originalTreePath string, original signingResignPreparedTree) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	packedRoot, err := filepath.Abs(filepath.Clean(packedTreePath))
	if err != nil {
		return fmt.Errorf("resolve packed verification tree: %w", err)
	}
	originalRoot, err := filepath.Abs(filepath.Clean(originalTreePath))
	if err != nil {
		return fmt.Errorf("resolve original prepared tree: %w", err)
	}
	expected := make([]string, 0, len(original.Archive.Targets)+len(original.CodePlans))
	for _, target := range original.Archive.Targets {
		relative := filepath.Clean(filepath.FromSlash(path.Join(target.RelativePath, target.Executable)))
		expected = append(expected, filepath.ToSlash(relative))
	}
	for _, plan := range original.CodePlans {
		codePath := filepath.Clean(plan.Path)
		if !filepath.IsAbs(codePath) {
			return fmt.Errorf("original prepared code path is not absolute")
		}
		relative, err := filepath.Rel(originalRoot, codePath)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("original prepared code path is outside the staging tree")
		}
		expected = append(expected, filepath.ToSlash(filepath.Clean(relative)))
	}
	currentPaths, err := enumerateSigningResignMachOFiles(ctx, packedRoot)
	if err != nil {
		return fmt.Errorf("enumerate packed Mach-O files: %w", err)
	}
	current := make([]string, 0, len(currentPaths))
	for _, codePath := range currentPaths {
		if isSigningResignPreservedExternalCodePath(packedRoot, codePath) {
			continue
		}
		relative, err := filepath.Rel(packedRoot, codePath)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("packed code path is outside the staging tree")
		}
		current = append(current, filepath.ToSlash(filepath.Clean(relative)))
	}
	sort.Strings(expected)
	sort.Strings(current)
	if !slices.Equal(current, expected) {
		return fmt.Errorf("re-signed IPA Mach-O executable inventory changed during repack")
	}
	return nil
}

func rebaseSigningResignPreparedTree(original signingResignPreparedTree, originalTreePath, packedTreePath string) (signingResignPreparedTree, error) {
	if originalTreePath == "" || packedTreePath == "" {
		return signingResignPreparedTree{}, fmt.Errorf("prepared tree roots are missing")
	}
	originalRoot, err := filepath.Abs(filepath.Clean(originalTreePath))
	if err != nil {
		return signingResignPreparedTree{}, fmt.Errorf("resolve original prepared tree: %w", err)
	}
	packedRoot, err := filepath.Abs(filepath.Clean(packedTreePath))
	if err != nil {
		return signingResignPreparedTree{}, fmt.Errorf("resolve packed verification tree: %w", err)
	}
	rebased := original
	rebased.Archive.Targets = append([]signingResignTarget(nil), original.Archive.Targets...)
	rebased.CodePlans = append([]signingResignCodePlan(nil), original.CodePlans...)
	rebased.SwiftSupport = append([]signingResignSwiftSupportEntry(nil), original.SwiftSupport...)
	for index := range rebased.CodePlans {
		codePath := filepath.Clean(rebased.CodePlans[index].Path)
		if !filepath.IsAbs(codePath) {
			return signingResignPreparedTree{}, fmt.Errorf("prepared code path is not absolute")
		}
		relative, err := filepath.Rel(originalRoot, codePath)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return signingResignPreparedTree{}, fmt.Errorf("prepared code path is outside the original staging tree")
		}
		rebased.CodePlans[index].Path = filepath.Join(packedRoot, relative)
	}
	return rebased, nil
}

func publishSigningResignOutput(ctx context.Context, outputRoot rootfs.Root, name, packedPath string, packedSize int64, packedDigest string) (signingResignArtifactResult, error) {
	if err := contextError(ctx); err != nil {
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactPublish,
			err,
		)
	}
	file, err := os.Open(packedPath)
	if err != nil {
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactRead,
			fmt.Errorf("open staged re-signed IPA: %w", err),
		)
	}
	defer file.Close()
	written, err := outputRoot.CreateNewFrom(name, file, 0o600)
	if err != nil {
		// CreateNewFrom can report a durability/cleanup error after the
		// no-replace rename has already published the destination. If the
		// complete staged byte count was written and the destination is now
		// visible, preserve that uncertainty for the caller instead of
		// inviting a blind retry.
		if written == packedSize {
			if published, openErr := outputRoot.OpenFile(name); openErr == nil {
				_ = published.Close()
				return signingResignArtifactResult{}, wrapSigningResignOperationalError(
					signingResignStageArtifact,
					signingResignCodeArtifactPublish,
					signingResignPublicationAmbiguousError("publish re-signed IPA returned an uncertain result", err),
				)
			} else {
				err = errors.Join(err, openErr)
			}
		}
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactPublish,
			fmt.Errorf("publish re-signed IPA: %w", err),
		)
	}
	if err := contextError(ctx); err != nil {
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactPublish,
			signingResignPublicationAmbiguousError("publication completed but cancellation prevented validation", err),
		)
	}
	if written != packedSize {
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactPublish,
			signingResignPublicationAmbiguousError("published re-signed IPA size is inconsistent"),
		)
	}
	published, err := outputRoot.OpenFile(name)
	if err != nil {
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactPublish,
			signingResignPublicationAmbiguousError("reopen published re-signed IPA failed", err),
		)
	}
	defer func() {
		if published != nil {
			_ = published.Close()
		}
	}()
	info, err := published.Stat()
	if err != nil {
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactPublish,
			signingResignPublicationAmbiguousError("inspect published re-signed IPA failed", err),
		)
	}
	signingResignBeforePublishedHashFn()
	digest, err := hashSigningResignOpenFile(ctx, published, info.Size())
	if err != nil {
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactHash,
			signingResignPublicationAmbiguousError("hash published re-signed IPA failed", err),
		)
	}
	if !strings.EqualFold(digest, packedDigest) {
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactHash,
			signingResignPublicationAmbiguousError("published re-signed IPA digest is inconsistent"),
		)
	}
	if err := published.Close(); err != nil {
		published = nil
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactPublish,
			signingResignPublicationAmbiguousError("close published re-signed IPA failed", err),
		)
	}
	published = nil
	if err := contextError(ctx); err != nil {
		return signingResignArtifactResult{}, wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactPublish,
			signingResignPublicationAmbiguousError("publication completed but cancellation prevented success", err),
		)
	}
	return signingResignArtifactResult{Path: filepath.Join(outputRoot.Path(), name), SizeBytes: info.Size(), SHA256: digest}, nil
}

func hashSigningResignFile(ctx context.Context, pathValue string, size int64) (digest string, resultErr error) {
	defer func() {
		resultErr = wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactHash,
			resultErr,
		)
	}()
	if err := contextError(ctx); err != nil {
		return "", err
	}
	file, err := os.Open(pathValue)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return hashSigningResignOpenFile(ctx, file, size)
}

func hashSigningResignOpenFile(ctx context.Context, file *os.File, size int64) (digest string, resultErr error) {
	defer func() {
		resultErr = wrapSigningResignOperationalError(
			signingResignStageArtifact,
			signingResignCodeArtifactHash,
			resultErr,
		)
	}()
	if file == nil || size < 0 {
		return "", fmt.Errorf("hash input is invalid")
	}
	if err := contextError(ctx); err != nil {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hash := sha256.New()
	written, err := copySigningResignWithContext(ctx, hash, io.LimitReader(file, size+1), size)
	if err != nil {
		return "", err
	}
	if written != size {
		return "", fmt.Errorf("hash input size changed")
	}
	if err := contextError(ctx); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(hash.Sum(nil))), nil
}
