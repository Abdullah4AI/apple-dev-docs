package distribute

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
	"howett.net/plist"
)

const (
	archiveSnapshotMaxSizeBytes  = int64(32 << 30)
	archiveSnapshotMaxEntries    = 250_000
	archiveSnapshotMaxPathBytes  = 4_096
	archiveTreeDigestVersion     = "asc-xcarchive-tree-v1\x00"
	archiveSnapshotStagingPrefix = ".asc-xcarchive-"
	archiveIdentityPlistMaxBytes = int64(1 << 20)
)

// archiveTreeSnapshot is the bounded, content-addressed archive evidence kept
// by a distribution plan or run. RelativePath is empty for inspection-only
// evidence and is relative to the run root for a durable snapshot.
type archiveTreeSnapshot struct {
	RelativePath string             `json:"relativePath"`
	TreeSHA256   string             `json:"treeSha256"`
	SizeBytes    int64              `json:"sizeBytes"`
	EntryCount   int                `json:"entryCount"`
	App          archiveAppIdentity `json:"app,omitempty"`
}

type archiveAppIdentity struct {
	BundleID         string `json:"bundleId,omitempty"`
	Title            string `json:"title,omitempty"`
	Version          string `json:"version,omitempty"`
	BuildNumber      string `json:"buildNumber,omitempty"`
	MinimumOSVersion string `json:"minimumOSVersion,omitempty"`
}

// digestXCArchive inspects an archive without copying it or writing local
// state. The final input directory and every entry below it are opened without
// following symlinks.
func digestXCArchive(ctx context.Context, archivePath string) (archiveTreeSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return archiveTreeSnapshot{}, err
	}
	result, err := scanXCArchive(ctx, archivePath, nil)
	if err != nil {
		return archiveTreeSnapshot{}, err
	}
	return result, nil
}

// snapshotXCArchive copies an archive into a new owner-private directory below
// runRoot while computing the same evidence as digestXCArchive. Existing
// destinations are never replaced.
func snapshotXCArchive(
	ctx context.Context,
	archivePath string,
	runRoot rootfs.Root,
	destinationRelative string,
) (archiveTreeSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return archiveTreeSnapshot{}, err
	}
	destinationRelative = filepath.Clean(strings.TrimSpace(destinationRelative))
	if destinationRelative == "." || filepath.Ext(destinationRelative) != ".xcarchive" {
		return archiveTreeSnapshot{}, fmt.Errorf("archive snapshot destination must be a relative .xcarchive directory")
	}
	if err := rootfs.ValidateRelative(destinationRelative); err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("archive snapshot destination: %w", err)
	}
	parentRelative := filepath.Dir(destinationRelative)
	if parentRelative != "." {
		if err := runRoot.MkdirAll(parentRelative, 0o700); err != nil {
			return archiveTreeSnapshot{}, fmt.Errorf("create archive snapshot parent: %w", err)
		}
	}
	if err := runRoot.CheckCreateNewFile(destinationRelative); err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("create archive snapshot: %w", err)
	}
	parentPath, err := runRoot.Resolve(parentRelative)
	if err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("resolve archive snapshot parent: %w", err)
	}
	parentRoot, err := rootfs.New(parentPath)
	if err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("anchor archive snapshot parent: %w", err)
	}
	defer parentRoot.Close()
	openedParent, err := parentRoot.OpenRoot()
	if err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("open archive snapshot parent: %w", err)
	}
	defer openedParent.Close()

	stagingName, err := createArchiveSnapshotStaging(openedParent)
	if err != nil {
		return archiveTreeSnapshot{}, err
	}
	published := false
	defer func() {
		if !published {
			_ = openedParent.RemoveAll(stagingName)
		}
	}()
	stagingPath := filepath.Join(parentPath, stagingName)
	destinationRoot, err := rootfs.New(stagingPath)
	if err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("anchor staged archive snapshot: %w", err)
	}
	defer destinationRoot.Close()
	openedDestination, err := destinationRoot.OpenRoot()
	if err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("open staged archive snapshot: %w", err)
	}

	result, err := scanXCArchive(ctx, archivePath, &archiveSnapshotDestination{
		root:   destinationRoot,
		opened: openedDestination,
	})
	if err != nil {
		_ = openedDestination.Close()
		return archiveTreeSnapshot{}, err
	}
	if err := openedDestination.Chmod(".", 0o700); err != nil {
		_ = openedDestination.Close()
		return archiveTreeSnapshot{}, fmt.Errorf("make archive snapshot owner-private: %w", err)
	}
	if err := syncOpenedArchiveDirectory(openedDestination, "."); err != nil {
		_ = openedDestination.Close()
		return archiveTreeSnapshot{}, fmt.Errorf("sync staged archive snapshot: %w", err)
	}
	if err := openedDestination.Close(); err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("close staged archive snapshot: %w", err)
	}
	if err := syncOpenedArchiveDirectory(openedParent, "."); err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("sync archive snapshot staging parent: %w", err)
	}
	destinationBase := filepath.Base(destinationRelative)
	if err := secureopen.RenameNoReplaceInRoot(openedParent, stagingName, destinationBase); err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("publish archive snapshot: %w", err)
	}
	published = true
	if err := syncOpenedArchiveDirectory(openedParent, "."); err != nil {
		// The final name belongs exclusively to this invocation because the
		// publication used a no-replace rename. Remove it rather than leave a
		// final-looking path after reporting failure.
		if removeErr := openedParent.RemoveAll(destinationBase); removeErr == nil {
			published = false
			_ = syncOpenedArchiveDirectory(openedParent, ".")
		}
		return archiveTreeSnapshot{}, fmt.Errorf("sync published archive snapshot: %w", err)
	}
	result.RelativePath = destinationRelative
	return result, nil
}

// revalidateXCArchiveSnapshot confirms that a durable snapshot still matches
// the exact digest, byte count, and entry count recorded by the run.
func revalidateXCArchiveSnapshot(ctx context.Context, runRoot rootfs.Root, snapshot archiveTreeSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	relative := filepath.Clean(strings.TrimSpace(snapshot.RelativePath))
	if relative == "." || relative == "" {
		return fmt.Errorf("archive snapshot relative path is required")
	}
	if err := rootfs.ValidateRelative(relative); err != nil {
		return fmt.Errorf("archive snapshot path: %w", err)
	}
	if err := runRoot.CheckContained(relative); err != nil {
		return fmt.Errorf("archive snapshot path: %w", err)
	}
	absolute, err := runRoot.Resolve(relative)
	if err != nil {
		return fmt.Errorf("resolve archive snapshot: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return fmt.Errorf("inspect archive snapshot root: %w", err)
	}
	if info.Mode().Perm() != 0o700 {
		return fmt.Errorf("archive snapshot root mode %#o is not owner-private 0700", info.Mode().Perm())
	}
	actual, err := digestXCArchive(ctx, absolute)
	if err != nil {
		return fmt.Errorf("revalidate archive snapshot: %w", err)
	}
	if actual.TreeSHA256 != snapshot.TreeSHA256 || actual.SizeBytes != snapshot.SizeBytes || actual.EntryCount != snapshot.EntryCount || actual.App != snapshot.App {
		return fmt.Errorf(
			"archive snapshot does not match recorded tree (sha256=%s bytes=%d entries=%d)",
			snapshot.TreeSHA256,
			snapshot.SizeBytes,
			snapshot.EntryCount,
		)
	}
	return nil
}

type archiveSnapshotDestination struct {
	root   rootfs.Root
	opened *os.Root
}

type archiveTreeScanner struct {
	ctx         context.Context
	destination *archiveSnapshotDestination
	treeHash    hash.Hash
	sizeBytes   int64
	entryCount  int
	archiveInfo []byte
	appInfoPath string
	appInfo     []byte
}

func scanXCArchive(
	ctx context.Context,
	archivePath string,
	destination *archiveSnapshotDestination,
) (archiveTreeSnapshot, error) {
	archivePath = filepath.Clean(strings.TrimSpace(archivePath))
	if archivePath == "." || filepath.Ext(archivePath) != ".xcarchive" {
		return archiveTreeSnapshot{}, fmt.Errorf("archive path must name a .xcarchive directory")
	}
	before, err := os.Lstat(archivePath)
	if err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("inspect archive root: %w", err)
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return archiveTreeSnapshot{}, fmt.Errorf("archive root is a symlink; refusing to follow symlink")
	}
	if !before.IsDir() {
		return archiveTreeSnapshot{}, fmt.Errorf("archive path is not a directory")
	}
	opened, err := os.OpenRoot(archivePath)
	if err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("open archive root without following symlinks: %w", err)
	}
	defer opened.Close()
	openedInfo, err := opened.Stat(".")
	if err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("inspect opened archive root: %w", err)
	}
	if !os.SameFile(before, openedInfo) || !openedInfo.IsDir() {
		return archiveTreeSnapshot{}, fmt.Errorf("archive root changed while opening")
	}
	afterOpen, err := os.Lstat(archivePath)
	if err != nil || afterOpen.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, afterOpen) {
		if err != nil {
			return archiveTreeSnapshot{}, fmt.Errorf("reinspect archive root: %w", err)
		}
		return archiveTreeSnapshot{}, fmt.Errorf("archive root changed to a symlink while opening")
	}

	treeHash := sha256.New()
	_, _ = io.WriteString(treeHash, archiveTreeDigestVersion)
	scanner := &archiveTreeScanner{ctx: ctx, destination: destination, treeHash: treeHash}
	if err := scanner.scanDirectory(opened, "", openedInfo); err != nil {
		return archiveTreeSnapshot{}, err
	}
	finalInfo, err := os.Lstat(archivePath)
	if err != nil {
		return archiveTreeSnapshot{}, fmt.Errorf("reinspect archive root: %w", err)
	}
	if finalInfo.Mode()&os.ModeSymlink != 0 || !archiveInfoStable(before, finalInfo, false) {
		return archiveTreeSnapshot{}, fmt.Errorf("archive root changed during inspection")
	}
	app, err := scanner.appIdentity()
	if err != nil {
		return archiveTreeSnapshot{}, err
	}
	return archiveTreeSnapshot{
		TreeSHA256: hex.EncodeToString(treeHash.Sum(nil)),
		SizeBytes:  scanner.sizeBytes,
		EntryCount: scanner.entryCount,
		App:        app,
	}, nil
}

func (s *archiveTreeScanner) scanDirectory(directory *os.Root, relative string, initial os.FileInfo) error {
	if err := s.ctx.Err(); err != nil {
		return err
	}
	handle, err := directory.Open(".")
	if err != nil {
		return fmt.Errorf("open archive directory %q: %w", archiveDisplayPath(relative), err)
	}
	entries, readErr := handle.Readdir(-1)
	closeErr := handle.Close()
	if readErr != nil {
		return fmt.Errorf("read archive directory %q: %w", archiveDisplayPath(relative), readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close archive directory %q: %w", archiveDisplayPath(relative), closeErr)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, enumerated := range entries {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		name := enumerated.Name()
		if name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
			return fmt.Errorf("archive contains invalid entry name %q", name)
		}
		entryRelative := name
		if relative != "" {
			entryRelative = path.Join(relative, name)
		}
		if err := s.noteEntry(entryRelative); err != nil {
			return err
		}

		info, err := directory.Lstat(name)
		if err != nil {
			return fmt.Errorf("inspect archive entry %q: %w", entryRelative, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive entry %q is a symlink; refusing to follow symlink", entryRelative)
		}
		switch {
		case info.IsDir():
			if err := s.recordDirectory(directory, name, entryRelative, info); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			if err := s.recordFile(directory, name, entryRelative, info); err != nil {
				return err
			}
		default:
			return fmt.Errorf("archive entry %q is a special file (%s)", entryRelative, info.Mode().Type())
		}
	}

	final, err := directory.Stat(".")
	if err != nil {
		return fmt.Errorf("reinspect archive directory %q: %w", archiveDisplayPath(relative), err)
	}
	if !archiveInfoStable(initial, final, false) {
		return fmt.Errorf("archive directory %q changed during inspection", archiveDisplayPath(relative))
	}
	return nil
}

func (s *archiveTreeScanner) noteEntry(relative string) error {
	if len(relative) > archiveSnapshotMaxPathBytes {
		return fmt.Errorf("archive path length exceeds %d bytes: %q", archiveSnapshotMaxPathBytes, relative)
	}
	if s.entryCount >= archiveSnapshotMaxEntries {
		return fmt.Errorf("archive contains more than %d entries", archiveSnapshotMaxEntries)
	}
	s.entryCount++
	return nil
}

func (s *archiveTreeScanner) recordDirectory(
	parent *os.Root,
	name string,
	relative string,
	before os.FileInfo,
) error {
	opened, err := parent.OpenRoot(name)
	if err != nil {
		return fmt.Errorf("open archive directory %q without following symlinks: %w", relative, err)
	}
	defer opened.Close()
	openedInfo, err := opened.Stat(".")
	if err != nil {
		return fmt.Errorf("inspect opened archive directory %q: %w", relative, err)
	}
	afterOpen, err := parent.Lstat(name)
	if err != nil {
		return fmt.Errorf("reinspect archive directory %q: %w", relative, err)
	}
	if afterOpen.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, openedInfo) || !os.SameFile(before, afterOpen) {
		return fmt.Errorf("archive directory %q changed to a symlink while opening", relative)
	}
	archiveHashTreeEntry(s.treeHash, 'd', relative, archiveSnapshotDirectoryMode(), 0, nil)
	if s.destination != nil {
		if err := s.destination.opened.Mkdir(filepath.FromSlash(relative), 0o700); err != nil {
			return fmt.Errorf("create archive snapshot directory %q: %w", relative, err)
		}
	}
	if err := s.scanDirectory(opened, relative, openedInfo); err != nil {
		return err
	}
	finalParentInfo, err := parent.Lstat(name)
	if err != nil {
		return fmt.Errorf("reinspect archive directory %q: %w", relative, err)
	}
	if finalParentInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(openedInfo, finalParentInfo) {
		return fmt.Errorf("archive directory %q changed during inspection", relative)
	}
	if s.destination != nil {
		if err := chmodArchiveSnapshotEntry(s.destination.opened, filepath.FromSlash(relative), archiveSnapshotDirectoryMode()); err != nil {
			return fmt.Errorf("preserve archive directory mode %q: %w", relative, err)
		}
		if err := syncOpenedArchiveDirectory(s.destination.opened, filepath.FromSlash(relative)); err != nil {
			return fmt.Errorf("sync archive snapshot directory %q: %w", relative, err)
		}
	}
	return nil
}

func (s *archiveTreeScanner) recordFile(
	parent *os.Root,
	name string,
	relative string,
	before os.FileInfo,
) error {
	if hook := archiveSnapshotCopyHook(s.ctx); hook != nil {
		if err := hook(relative); err != nil {
			return err
		}
	}
	if err := s.ctx.Err(); err != nil {
		return err
	}
	multipleLinks, linkCountSupported := archiveHasMultipleHardLinks(before)
	if !linkCountSupported {
		return fmt.Errorf("archive entry %q hard-link validation is unsupported on this platform", relative)
	}
	if multipleLinks {
		return fmt.Errorf("archive entry %q has multiple hard links", relative)
	}
	if before.Size() < 0 || before.Size() > archiveSnapshotMaxSizeBytes-s.sizeBytes {
		return fmt.Errorf("archive content exceeds %d bytes", archiveSnapshotMaxSizeBytes)
	}
	file, err := secureopen.OpenExistingNoFollowInRoot(parent, name)
	if err != nil {
		return fmt.Errorf("open archive file %q without following symlinks: %w", relative, err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect opened archive file %q: %w", relative, err)
	}
	openedMultipleLinks, openedLinkCountSupported := archiveHasMultipleHardLinks(openedInfo)
	if !openedLinkCountSupported {
		return fmt.Errorf("archive file %q hard-link validation is unsupported on this platform", relative)
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(before, openedInfo) || openedMultipleLinks {
		return fmt.Errorf("archive file %q changed or gained a hard link while opening", relative)
	}
	if before.Size() != openedInfo.Size() || archiveSnapshotFileMode(before.Mode()) != archiveSnapshotFileMode(openedInfo.Mode()) {
		return fmt.Errorf("archive file %q changed while opening", relative)
	}

	contentHash := sha256.New()
	var identityBytes bytes.Buffer
	captureIdentity := relative == "Info.plist" || (s.appInfoPath != "" && relative == s.appInfoPath)
	if captureIdentity && openedInfo.Size() > archiveIdentityPlistMaxBytes {
		return fmt.Errorf("archive identity plist %q exceeds %d bytes", relative, archiveIdentityPlistMaxBytes)
	}
	contentWriter := io.Writer(contentHash)
	if captureIdentity {
		contentWriter = io.MultiWriter(contentHash, &identityBytes)
	}
	reader := io.TeeReader(&exactArchiveReader{
		ctx:       s.ctx,
		reader:    file,
		remaining: openedInfo.Size(),
	}, contentWriter)
	var written int64
	if s.destination == nil {
		written, err = io.Copy(io.Discard, reader)
	} else {
		written, err = s.destination.root.CreateNewFrom(
			filepath.FromSlash(relative),
			reader,
			archiveSnapshotFileMode(openedInfo.Mode()),
		)
	}
	if err != nil {
		return fmt.Errorf("read archive file %q: %w", relative, err)
	}
	if written != openedInfo.Size() {
		return fmt.Errorf("archive file %q size changed during inspection", relative)
	}
	finalInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("reinspect archive file %q: %w", relative, err)
	}
	finalParentInfo, parentErr := parent.Lstat(name)
	if parentErr != nil {
		return fmt.Errorf("reinspect archive file %q: %w", relative, parentErr)
	}
	if finalParentInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(finalInfo, finalParentInfo) ||
		!archiveInfoStable(openedInfo, finalInfo, true) {
		return fmt.Errorf("archive file %q changed during inspection", relative)
	}

	s.sizeBytes += written
	if captureIdentity {
		captured := append([]byte(nil), identityBytes.Bytes()...)
		if relative == "Info.plist" {
			s.archiveInfo = captured
			if err := s.bindArchiveApplicationPath(); err != nil {
				return err
			}
		} else {
			s.appInfo = captured
		}
	}
	archiveHashTreeEntry(
		s.treeHash,
		'f',
		relative,
		archiveSnapshotFileMode(openedInfo.Mode()),
		written,
		contentHash.Sum(nil),
	)
	return nil
}

type xcArchiveIdentityPlist struct {
	ApplicationProperties struct {
		ApplicationPath            string `plist:"ApplicationPath"`
		CFBundleIdentifier         string `plist:"CFBundleIdentifier"`
		CFBundleShortVersionString string `plist:"CFBundleShortVersionString"`
		CFBundleVersion            string `plist:"CFBundleVersion"`
	} `plist:"ApplicationProperties"`
}

type xcArchiveApplicationPlist struct {
	CFBundleIdentifier         string `plist:"CFBundleIdentifier"`
	CFBundleDisplayName        string `plist:"CFBundleDisplayName"`
	CFBundleName               string `plist:"CFBundleName"`
	CFBundleShortVersionString string `plist:"CFBundleShortVersionString"`
	CFBundleVersion            string `plist:"CFBundleVersion"`
	MinimumOSVersion           string `plist:"MinimumOSVersion"`
}

func (s *archiveTreeScanner) appIdentity() (archiveAppIdentity, error) {
	if s.appInfoPath == "" {
		return archiveAppIdentity{}, nil
	}
	var archiveInfo xcArchiveIdentityPlist
	if _, err := plist.Unmarshal(s.archiveInfo, &archiveInfo); err != nil {
		return archiveAppIdentity{}, fmt.Errorf("decode archive identity Info.plist: %w", err)
	}
	if len(s.appInfo) == 0 {
		return archiveAppIdentity{}, fmt.Errorf("archive ApplicationPath does not match a scanned main application")
	}
	var appInfo xcArchiveApplicationPlist
	if _, err := plist.Unmarshal(s.appInfo, &appInfo); err != nil {
		return archiveAppIdentity{}, fmt.Errorf("decode archived application Info.plist: %w", err)
	}
	title := strings.TrimSpace(appInfo.CFBundleDisplayName)
	if title == "" {
		title = strings.TrimSpace(appInfo.CFBundleName)
	}
	identity := archiveAppIdentity{
		BundleID: strings.TrimSpace(appInfo.CFBundleIdentifier), Title: title,
		Version: strings.TrimSpace(appInfo.CFBundleShortVersionString), BuildNumber: strings.TrimSpace(appInfo.CFBundleVersion),
		MinimumOSVersion: strings.TrimSpace(appInfo.MinimumOSVersion),
	}
	archiveProps := archiveInfo.ApplicationProperties
	if err := validateArchiveAppIdentity(identity); err != nil {
		return archiveAppIdentity{}, err
	}
	if strings.TrimSpace(archiveProps.CFBundleIdentifier) != identity.BundleID ||
		strings.TrimSpace(archiveProps.CFBundleShortVersionString) != identity.Version ||
		strings.TrimSpace(archiveProps.CFBundleVersion) != identity.BuildNumber {
		return archiveAppIdentity{}, fmt.Errorf("archive and application Info.plist identities differ")
	}
	return identity, nil
}

func (s *archiveTreeScanner) bindArchiveApplicationPath() error {
	var archiveInfo xcArchiveIdentityPlist
	if _, err := plist.Unmarshal(s.archiveInfo, &archiveInfo); err != nil {
		// Tree scanning is also used by low-level fixtures that are not complete
		// Xcode archives. Agent planning separately requires a non-empty app
		// identity, so malformed real archive metadata still fails closed there.
		return nil
	}
	applicationPath := strings.TrimSpace(archiveInfo.ApplicationProperties.ApplicationPath)
	if applicationPath == "" {
		// The lightweight archive scanner remains usable for non-Xcode fixtures,
		// but a real archive application must declare its exact path.
		return nil
	}
	components := strings.Split(applicationPath, "/")
	if strings.Contains(applicationPath, `\`) || path.Clean(applicationPath) != applicationPath || len(components) != 2 ||
		components[0] != "Applications" || !strings.HasSuffix(components[1], ".app") || components[1] == ".app" {
		return fmt.Errorf("archive ApplicationPath does not identify one main application")
	}
	s.appInfoPath = path.Join("Products", applicationPath, "Info.plist")
	return nil
}

func validateArchiveAppIdentity(identity archiveAppIdentity) error {
	if identity.BundleID == "" || identity.Title == "" || identity.Version == "" || identity.BuildNumber == "" {
		return fmt.Errorf("archived application identity requires bundle ID, title, version, and build number")
	}
	for name, item := range map[string]struct {
		value string
		limit int
	}{
		"bundle ID": {identity.BundleID, 256}, "title": {identity.Title, 512}, "version": {identity.Version, 256},
		"build number": {identity.BuildNumber, 256}, "minimum OS version": {identity.MinimumOSVersion, 256},
	} {
		if len(item.value) > item.limit || containsUnsafeArchiveIdentityText(item.value) {
			return fmt.Errorf("archived application %s is unsafe or too long", name)
		}
	}
	return nil
}

func containsUnsafeArchiveIdentityText(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) || unicode.In(character, unicode.Bidi_Control) || unicode.Is(unicode.Cf, character) {
			return true
		}
	}
	return false
}

type exactArchiveReader struct {
	ctx       context.Context
	reader    io.Reader
	remaining int64
	verified  bool
}

func (r *exactArchiveReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if r.remaining > 0 {
		if int64(len(buffer)) > r.remaining {
			buffer = buffer[:r.remaining]
		}
		n, err := r.reader.Read(buffer)
		r.remaining -= int64(n)
		if errors.Is(err, io.EOF) && r.remaining > 0 {
			return n, io.ErrUnexpectedEOF
		}
		return n, err
	}
	if r.verified {
		return 0, io.EOF
	}
	r.verified = true
	var probe [1]byte
	n, err := r.reader.Read(probe[:])
	if n != 0 || err == nil {
		return 0, fmt.Errorf("source grew while reading")
	}
	if !errors.Is(err, io.EOF) {
		return 0, err
	}
	return 0, io.EOF
}

func archiveHashTreeEntry(
	tree hash.Hash,
	kind byte,
	relative string,
	mode os.FileMode,
	size int64,
	contentDigest []byte,
) {
	_, _ = tree.Write([]byte{kind})
	var numeric [8]byte
	binary.BigEndian.PutUint32(numeric[:4], uint32(len(relative)))
	_, _ = tree.Write(numeric[:4])
	_, _ = io.WriteString(tree, relative)
	binary.BigEndian.PutUint32(numeric[:4], uint32(mode))
	_, _ = tree.Write(numeric[:4])
	binary.BigEndian.PutUint64(numeric[:], uint64(size))
	_, _ = tree.Write(numeric[:])
	if len(contentDigest) == sha256.Size {
		_, _ = tree.Write(contentDigest)
	} else {
		var empty [sha256.Size]byte
		_, _ = tree.Write(empty[:])
	}
}

func archiveInfoStable(before, after os.FileInfo, bindSize bool) bool {
	if before == nil || after == nil || !os.SameFile(before, after) {
		return false
	}
	if archiveSnapshotCanonicalMode(before.Mode()) != archiveSnapshotCanonicalMode(after.Mode()) || !before.ModTime().Equal(after.ModTime()) {
		return false
	}
	return !bindSize || before.Size() == after.Size()
}

func archiveSnapshotCanonicalMode(mode os.FileMode) os.FileMode {
	if mode.IsDir() {
		return archiveSnapshotDirectoryMode()
	}
	return archiveSnapshotFileMode(mode)
}

func archiveSnapshotDirectoryMode() os.FileMode {
	return 0o700
}

func archiveSnapshotFileMode(source os.FileMode) os.FileMode {
	// Contents stay private to the run owner. Retaining the source executable
	// bits is sufficient for Xcode tools while deliberately dropping setuid,
	// setgid, sticky, and group/other read-write privileges.
	return 0o600 | (source.Perm() & 0o111)
}

func chmodArchiveSnapshotEntry(root *os.Root, relative string, mode os.FileMode) error {
	file, err := secureopen.OpenExistingNoFollowInRoot(root, relative)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Chmod(mode)
}

func createArchiveSnapshotStaging(parent *os.Root) (string, error) {
	for attempt := 0; attempt < 16; attempt++ {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return "", fmt.Errorf("generate archive snapshot staging name: %w", err)
		}
		name := archiveSnapshotStagingPrefix + hex.EncodeToString(random[:]) + ".tmp"
		if err := parent.Mkdir(name, 0o700); err == nil {
			return name, nil
		} else if !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("create staged archive snapshot: %w", err)
		}
	}
	return "", fmt.Errorf("create staged archive snapshot: too many name collisions")
}

func syncOpenedArchiveDirectory(root *os.Root, relative string) error {
	directory, err := secureopen.OpenExistingNoFollowInRoot(root, relative)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

type archiveSnapshotCopyHookKey struct{}

type archiveSnapshotCopyHookFunc func(relative string) error

// archiveSnapshotCopyHook is an internal deterministic fault-injection seam.
// Only package tests put this private key into a context.
func archiveSnapshotCopyHook(ctx context.Context) archiveSnapshotCopyHookFunc {
	hook, _ := ctx.Value(archiveSnapshotCopyHookKey{}).(archiveSnapshotCopyHookFunc)
	return hook
}

func archiveDisplayPath(relative string) string {
	if relative == "" {
		return "."
	}
	return relative
}
