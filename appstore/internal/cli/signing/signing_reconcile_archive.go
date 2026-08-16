package signing

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/infoplist"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"howett.net/plist"
)

type signingArchiveRequirements struct {
	TeamID  string          `json:"teamId"`
	Targets []signingTarget `json:"targets"`
}

var (
	readSigningArchiveRequirements = inspectSigningArchive
	signingReconcilePlatformCheck  = requireSigningReconcilePlatform
)

func inspectSigningArchive(archivePath string) (signingArchiveRequirements, error) {
	archivePath = filepath.Clean(strings.TrimSpace(archivePath))
	info, err := os.Lstat(archivePath)
	if err != nil {
		return signingArchiveRequirements{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return signingArchiveRequirements{}, fmt.Errorf("archive path must be a non-symlinked directory")
	}
	root, err := rootfs.New(archivePath)
	if err != nil {
		return signingArchiveRequirements{}, fmt.Errorf("open archive: %w", err)
	}
	defer root.Close()

	archiveInfo, err := readSigningPlist(root, "Info.plist")
	if err != nil {
		return signingArchiveRequirements{}, fmt.Errorf("read archive Info.plist: %w", err)
	}
	properties, ok := archiveInfo["ApplicationProperties"].(map[string]any)
	if !ok {
		return signingArchiveRequirements{}, fmt.Errorf("archive Info.plist missing ApplicationProperties")
	}
	applicationPath := strings.TrimSpace(plistString(properties["ApplicationPath"]))
	if err := validateSigningArchiveRelativePath(applicationPath); err != nil {
		return signingArchiveRequirements{}, fmt.Errorf("unsafe ApplicationPath: %w", err)
	}
	teamID := strings.TrimSpace(plistString(properties["Team"]))
	if teamID == "" {
		return signingArchiveRequirements{}, fmt.Errorf("archive Info.plist missing signing team")
	}

	mainPath := filepath.ToSlash(filepath.Join("Products", filepath.FromSlash(applicationPath)))
	mainInfo, err := readSigningPlist(root, filepath.ToSlash(filepath.Join(mainPath, "Info.plist")))
	if err != nil {
		return signingArchiveRequirements{}, fmt.Errorf("read application Info.plist: %w", err)
	}
	if err := validateSigningArchivePlatform(mainInfo); err != nil {
		return signingArchiveRequirements{}, err
	}
	targetPaths := []archiveTargetPath{{Kind: "application", RelativePath: mainPath}}
	embedded, err := discoverEmbeddedSigningTargets(root, mainPath)
	if err != nil {
		return signingArchiveRequirements{}, err
	}
	targetPaths = append(targetPaths, embedded...)
	sort.Slice(targetPaths[1:], func(i, j int) bool {
		return targetPaths[i+1].RelativePath < targetPaths[j+1].RelativePath
	})

	result := signingArchiveRequirements{TeamID: teamID}
	seenBundleIDs := make(map[string]string)
	for _, path := range targetPaths {
		target, err := inspectSigningTarget(root, path)
		if err != nil {
			return signingArchiveRequirements{}, fmt.Errorf("inspect %s: %w", path.RelativePath, err)
		}
		if previous, ok := seenBundleIDs[target.BundleID]; ok {
			return signingArchiveRequirements{}, fmt.Errorf("duplicate bundle identifier %s in %s and %s", target.BundleID, previous, target.RelativePath)
		}
		seenBundleIDs[target.BundleID] = target.RelativePath
		entitlementTeam := strings.TrimSpace(plistString(target.Entitlements["com.apple.developer.team-identifier"]))
		if entitlementTeam == "" {
			return signingArchiveRequirements{}, fmt.Errorf("target %s signed entitlements are missing com.apple.developer.team-identifier", target.BundleID)
		}
		if entitlementTeam != teamID {
			return signingArchiveRequirements{}, fmt.Errorf("target %s uses team %s, archive uses %s", target.BundleID, entitlementTeam, teamID)
		}
		if err := validateTargetApplicationIdentifier(target); err != nil {
			return signingArchiveRequirements{}, err
		}
		target.AppIDPrefix = targetApplicationIdentifierPrefix(target)
		result.Targets = append(result.Targets, target)
	}
	return result, nil
}

func targetApplicationIdentifierPrefix(target signingTarget) string {
	identifier := plistString(target.Entitlements["application-identifier"])
	if identifier == "" {
		identifier = plistString(target.Entitlements["com.apple.application-identifier"])
	}
	return strings.TrimSuffix(identifier, "."+target.BundleID)
}

func validateTargetApplicationIdentifier(target signingTarget) error {
	values := uniqueSortedStrings([]string{
		plistString(target.Entitlements["application-identifier"]),
		plistString(target.Entitlements["com.apple.application-identifier"]),
	})
	if len(values) != 1 {
		return fmt.Errorf("target %s signed entitlements must contain one coherent application identifier", target.BundleID)
	}
	suffix := "." + target.BundleID
	prefix := strings.TrimSuffix(values[0], suffix)
	if prefix == values[0] || strings.TrimSpace(prefix) == "" {
		return fmt.Errorf("target %s signed application identifier %s does not match its bundle identifier", target.BundleID, values[0])
	}
	return nil
}

func validateSigningArchivePlatform(info map[string]any) error {
	supported := uniqueSortedStrings(plistStrings(info["CFBundleSupportedPlatforms"]))
	platformName := strings.ToLower(plistString(info["DTPlatformName"]))
	hasPlatformMetadata := len(supported) > 0 || platformName != ""

	if len(supported) != 0 {
		if len(supported) != 1 || !strings.EqualFold(supported[0], "iPhoneOS") {
			return fmt.Errorf("archive application supports platform %s; signing reconcile supports only iOS archives", strings.Join(supported, ","))
		}
	}
	if platformName != "" && platformName != "iphoneos" {
		return fmt.Errorf("archive application uses platform %s; signing reconcile supports only iOS archives", platformName)
	}
	if !hasPlatformMetadata {
		return fmt.Errorf("archive application platform cannot be verified; signing reconcile supports only iOS archives")
	}
	return nil
}

func plistStrings(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if stringValue := plistString(item); stringValue != "" {
				result = append(result, stringValue)
			}
		}
		return result
	default:
		return nil
	}
}

type archiveTargetPath struct {
	Kind         string
	RelativePath string
}

func discoverEmbeddedSigningTargets(root rootfs.Root, mainPath string) ([]archiveTargetPath, error) {
	var result []archiveTargetPath
	plugins, err := listBundleDirectories(root, filepath.ToSlash(filepath.Join(mainPath, "PlugIns")), ".appex")
	if err != nil {
		return nil, err
	}
	for _, path := range plugins {
		result = append(result, archiveTargetPath{Kind: "app-extension", RelativePath: path})
	}

	watchApps, err := listBundleDirectories(root, filepath.ToSlash(filepath.Join(mainPath, "Watch")), ".app")
	if err != nil {
		return nil, err
	}
	for _, watchPath := range watchApps {
		result = append(result, archiveTargetPath{Kind: "watch-application", RelativePath: watchPath})
		extensions, err := listBundleDirectories(root, filepath.ToSlash(filepath.Join(watchPath, "PlugIns")), ".appex")
		if err != nil {
			return nil, err
		}
		for _, path := range extensions {
			result = append(result, archiveTargetPath{Kind: "watch-extension", RelativePath: path})
		}
	}

	clips, err := listBundleDirectories(root, filepath.ToSlash(filepath.Join(mainPath, "AppClips")), ".app")
	if err != nil {
		return nil, err
	}
	for _, path := range clips {
		result = append(result, archiveTargetPath{Kind: "app-clip", RelativePath: path})
	}
	return result, nil
}

func listBundleDirectories(root rootfs.Root, directory, suffix string) ([]string, error) {
	file, err := root.OpenDir(filepath.FromSlash(directory))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open archive directory %s: %w", directory, err)
	}
	defer file.Close()
	entries, err := file.ReadDir(-1)
	if err != nil {
		return nil, fmt.Errorf("read archive directory %s: %w", directory, err)
	}
	var result []string
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("refusing symlinked archive entry %s", filepath.ToSlash(filepath.Join(directory, entry.Name())))
		}
		if entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), suffix) {
			result = append(result, filepath.ToSlash(filepath.Join(directory, entry.Name())))
		}
	}
	sort.Strings(result)
	return result, nil
}

func inspectSigningTarget(root rootfs.Root, targetPath archiveTargetPath) (signingTarget, error) {
	infoPath := filepath.ToSlash(filepath.Join(targetPath.RelativePath, "Info.plist"))
	info, err := readSigningPlist(root, infoPath)
	if err != nil {
		return signingTarget{}, err
	}
	bundleID := strings.TrimSpace(plistString(info["CFBundleIdentifier"]))
	if bundleID == "" {
		return signingTarget{}, fmt.Errorf("info.plist: missing CFBundleIdentifier")
	}
	if targetPath.Kind == "application" {
		if err := validateSigningArchivePlatform(info); err != nil {
			return signingTarget{}, fmt.Errorf("info.plist: %w", err)
		}
	}
	executable := strings.TrimSpace(plistString(info["CFBundleExecutable"]))
	if executable == "" || executable == "." || executable == ".." || filepath.Base(executable) != executable || strings.ContainsAny(executable, `/\\`) {
		return signingTarget{}, fmt.Errorf("info.plist: missing or unsafe CFBundleExecutable")
	}
	relativeExecutable := filepath.ToSlash(filepath.Join(targetPath.RelativePath, executable))
	handle, err := root.OpenFile(filepath.FromSlash(relativeExecutable))
	if err != nil {
		return signingTarget{}, fmt.Errorf("open signed executable: %w", err)
	}
	defer handle.Close()
	stat, err := handle.Stat()
	if err != nil || !stat.Mode().IsRegular() {
		return signingTarget{}, fmt.Errorf("signed executable is not a regular file")
	}
	entitlements, err := readCodesignEntitlements(handle)
	if err != nil {
		return signingTarget{}, err
	}
	return signingTarget{
		Kind: targetPath.Kind, RelativePath: targetPath.RelativePath, BundleID: bundleID,
		Executable: executable, Entitlements: entitlements,
	}, nil
}

func readCodesignEntitlements(executable *os.File) (map[string]any, error) {
	if _, err := executable.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek signed executable: %w", err)
	}
	if err := signingReconcilePlatformCheck(); err != nil {
		return nil, err
	}
	// codesign refuses /dev/fd code objects. Copy the already-open no-follow
	// handle into a private directory instead of reconstructing and reopening the
	// archive path after validation.
	temporaryDir, err := os.MkdirTemp("", "asc-signing-executable-*")
	if err != nil {
		return nil, fmt.Errorf("create private codesign directory: %w", err)
	}
	defer os.RemoveAll(temporaryDir)
	if err := os.Chmod(temporaryDir, 0o700); err != nil {
		return nil, fmt.Errorf("secure private codesign directory: %w", err)
	}
	temporaryPath := filepath.Join(temporaryDir, "Executable")
	temporary, err := os.OpenFile(temporaryPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
	if err != nil {
		return nil, fmt.Errorf("create private executable copy: %w", err)
	}
	if _, err := io.Copy(temporary, executable); err != nil {
		_ = temporary.Close()
		return nil, fmt.Errorf("copy signed executable: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("close signed executable copy: %w", err)
	}

	cmd := exec.Command("/usr/bin/codesign", "-d", "--entitlements", ":-", temporaryPath)
	stdout := &boundedCapture{limit: infoplist.MaxBytes + 1}
	stderr := &boundedCapture{limit: infoplist.MaxBytes + 64*1024}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("read signed entitlements with codesign: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if stdout.overflow || stderr.overflow {
		return nil, fmt.Errorf("codesign output exceeded the bounded entitlement limit")
	}
	data := bytes.TrimSpace(stdout.Bytes())
	if len(data) == 0 {
		data = extractPlistDocument(stderr.Bytes())
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	if len(data) > infoplist.MaxBytes {
		return nil, fmt.Errorf("signed entitlements exceed %d bytes", infoplist.MaxBytes)
	}
	if err := infoplist.ValidateStructure(data); err != nil {
		return nil, fmt.Errorf("validate signed entitlements: %w", err)
	}
	var entitlements map[string]any
	if _, err := plist.Unmarshal(data, &entitlements); err != nil {
		return nil, fmt.Errorf("decode signed entitlements: %w", err)
	}
	if entitlements == nil {
		entitlements = map[string]any{}
	}
	return entitlements, nil
}

type boundedCapture struct {
	bytes.Buffer
	limit    int
	overflow bool
}

func (capture *boundedCapture) Write(data []byte) (int, error) {
	original := len(data)
	remaining := capture.limit - capture.Len()
	if remaining <= 0 {
		capture.overflow = true
		return original, nil
	}
	if len(data) > remaining {
		data = data[:remaining]
		capture.overflow = true
	}
	_, _ = capture.Buffer.Write(data)
	return original, nil
}

func extractPlistDocument(data []byte) []byte {
	start := bytes.Index(data, []byte("<?xml"))
	if start < 0 {
		start = bytes.Index(data, []byte("<plist"))
	}
	endMarker := []byte("</plist>")
	end := bytes.LastIndex(data, endMarker)
	if start < 0 || end < start {
		return nil
	}
	return bytes.TrimSpace(data[start : end+len(endMarker)])
}

func readSigningPlist(root rootfs.Root, path string) (map[string]any, error) {
	file, err := root.OpenFile(filepath.FromSlash(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if err := infoplist.CheckDeclaredSize(uint64(stat.Size())); err != nil {
		return nil, err
	}
	data, err := infoplist.ReadBounded(io.LimitReader(file, infoplist.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if err := infoplist.ValidateStructure(data); err != nil {
		return nil, err
	}
	var payload map[string]any
	if _, err := plist.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode plist: %w", err)
	}
	return payload, nil
}

func validateSigningArchiveRelativePath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	converted := filepath.Clean(filepath.FromSlash(path))
	if filepath.IsAbs(converted) || converted == "." || converted == ".." || strings.HasPrefix(converted, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes archive")
	}
	return nil
}

func plistString(value any) string {
	if stringValue, ok := value.(string); ok {
		return strings.TrimSpace(stringValue)
	}
	return ""
}
