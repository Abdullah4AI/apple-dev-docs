package xcode

import (
	"archive/zip"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

var (
	afterExactIPACopyFn          = func(string) {}
	syncExactIPADestinationDirFn = syncExactIPADestinationDir
)

// finalizeExactExportedIPA treats xcodebuild output as untrusted. It pins the
// source without following links, copies and hashes it into a private
// same-directory staging file, proves the source stayed stable during that
// copy, validates the staged ZIP from its descriptor, then atomically publishes
// that already-validated inode without replacing an existing destination.
func finalizeExactExportedIPA(sourcePath, destinationPath string) (bundleInfo, error) {
	sourceRoot, err := rootfs.New(filepath.Dir(sourcePath))
	if err != nil {
		return bundleInfo{}, fmt.Errorf("open exported IPA root: %w", err)
	}
	defer sourceRoot.Close()
	source, err := sourceRoot.OpenFile(filepath.Base(sourcePath))
	if err != nil {
		return bundleInfo{}, fmt.Errorf("open exported IPA without following links: %w", err)
	}
	defer source.Close()
	sourceBefore, err := source.Stat()
	if err != nil {
		return bundleInfo{}, fmt.Errorf("inspect exported IPA: %w", err)
	}
	if err := validateExactIPAFileInfo(sourceBefore, "exported IPA"); err != nil {
		return bundleInfo{}, err
	}

	destinationRoot, err := rootfs.New(filepath.Dir(destinationPath))
	if err != nil {
		return bundleInfo{}, fmt.Errorf("open IPA destination root: %w", err)
	}
	defer destinationRoot.Close()
	openedDestinationRoot, err := destinationRoot.OpenRoot()
	if err != nil {
		return bundleInfo{}, fmt.Errorf("open IPA destination directory: %w", err)
	}
	defer openedDestinationRoot.Close()
	staged, stagedName, err := secureopen.CreateTempNoFollowInRoot(openedDestinationRoot, ".", ".asc-verified-ipa-*.ipa", 0o600)
	if err != nil {
		return bundleInfo{}, fmt.Errorf("create verified IPA staging file: %w", err)
	}
	published := false
	defer func() {
		_ = staged.Close()
		if !published {
			_ = openedDestinationRoot.Remove(stagedName)
		}
	}()

	copyHash := sha256.New()
	copied, err := io.Copy(io.MultiWriter(staged, copyHash), source)
	if err != nil {
		return bundleInfo{}, fmt.Errorf("copy exported IPA into verified staging: %w", err)
	}
	if copied != sourceBefore.Size() {
		return bundleInfo{}, fmt.Errorf("exported IPA size changed during copy")
	}
	if err := staged.Sync(); err != nil {
		return bundleInfo{}, fmt.Errorf("sync verified IPA staging file: %w", err)
	}
	afterExactIPACopyFn(sourcePath)
	sourceAfterCopy, err := source.Stat()
	if err != nil {
		return bundleInfo{}, fmt.Errorf("reinspect exported IPA after copy: %w", err)
	}
	if err := validateStableExactIPA(sourceBefore, sourceAfterCopy, "exported IPA"); err != nil {
		return bundleInfo{}, err
	}
	sourceHash, sourceBytes, err := hashExactIPAFile(source)
	if err != nil {
		return bundleInfo{}, fmt.Errorf("rehash exported IPA after copy: %w", err)
	}
	if sourceBytes != copied || !equalDigest(copyHash.Sum(nil), sourceHash) {
		return bundleInfo{}, fmt.Errorf("exported IPA contents changed during copy")
	}
	sourceAfterHash, err := source.Stat()
	if err != nil {
		return bundleInfo{}, fmt.Errorf("reinspect exported IPA after verification: %w", err)
	}
	if err := validateStableExactIPA(sourceBefore, sourceAfterHash, "exported IPA"); err != nil {
		return bundleInfo{}, err
	}
	if err := staged.Close(); err != nil {
		return bundleInfo{}, fmt.Errorf("close verified IPA staging file: %w", err)
	}

	pinned, err := secureopen.OpenExistingNoFollowInRoot(openedDestinationRoot, stagedName)
	if err != nil {
		return bundleInfo{}, fmt.Errorf("reopen verified IPA staging file: %w", err)
	}
	defer pinned.Close()
	stagedBefore, err := pinned.Stat()
	if err != nil {
		return bundleInfo{}, fmt.Errorf("inspect verified IPA staging file: %w", err)
	}
	if err := validateExactIPAFileInfo(stagedBefore, "verified IPA staging file"); err != nil {
		return bundleInfo{}, err
	}
	info, err := readIPABundleInfoFromReaderAt(pinned, stagedBefore.Size())
	if err != nil {
		return bundleInfo{}, fmt.Errorf("inspect exported IPA before installation: %w", err)
	}
	stagedHash, stagedBytes, err := hashExactIPAFile(pinned)
	if err != nil {
		return bundleInfo{}, fmt.Errorf("hash verified IPA staging file: %w", err)
	}
	if stagedBytes != copied || !equalDigest(copyHash.Sum(nil), stagedHash) {
		return bundleInfo{}, fmt.Errorf("verified IPA staging contents changed before publication")
	}
	stagedAfter, err := pinned.Stat()
	if err != nil {
		return bundleInfo{}, fmt.Errorf("reinspect verified IPA staging file: %w", err)
	}
	if err := validateStableExactIPA(stagedBefore, stagedAfter, "verified IPA staging file"); err != nil {
		return bundleInfo{}, err
	}
	if err := pinned.Close(); err != nil {
		return bundleInfo{}, fmt.Errorf("close verified IPA before publication: %w", err)
	}

	if err := secureopen.RenameNoReplaceInRoot(openedDestinationRoot, stagedName, filepath.Base(destinationPath)); err != nil {
		if errors.Is(err, os.ErrExist) {
			return bundleInfo{}, newDestinationExistsError(destinationPath, err)
		}
		return bundleInfo{}, fmt.Errorf("publish verified IPA: %w", err)
	}
	published = true
	if err := syncExactIPADestinationDirFn(openedDestinationRoot); err != nil {
		return bundleInfo{}, fmt.Errorf("sync published IPA directory: %w", err)
	}
	return info, nil
}

func syncExactIPADestinationDir(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	return errors.Join(syncErr, closeErr)
}

func validateExactIPAFileInfo(info os.FileInfo, description string) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", description)
	}
	uid, nlink, ok := exactIPAStatIdentity(info)
	currentUID, ownerAvailable := currentExactIPAOwner()
	if !ok || !ownerAvailable {
		return fmt.Errorf("%s ownership and hard-link validation is unsupported on this platform", description)
	}
	if uid != currentUID {
		return fmt.Errorf("%s must be owned by the current user", description)
	}
	if nlink != 1 {
		return fmt.Errorf("%s must not have multiple hard links", description)
	}
	return nil
}

func validateStableExactIPA(before, after os.FileInfo, description string) error {
	if !os.SameFile(before, after) || before.Mode() != after.Mode() || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return fmt.Errorf("%s changed during verification", description)
	}
	beforeUID, beforeLinks, beforeOK := exactIPAStatIdentity(before)
	afterUID, afterLinks, afterOK := exactIPAStatIdentity(after)
	if !beforeOK || !afterOK {
		return fmt.Errorf("%s identity validation is unsupported on this platform", description)
	}
	if beforeUID != afterUID || beforeLinks != afterLinks {
		return fmt.Errorf("%s identity changed during verification", description)
	}
	return validateExactIPAFileInfo(after, description)
}

func hashExactIPAFile(file *os.File) ([]byte, int64, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, 0, err
	}
	hash := sha256.New()
	written, err := io.Copy(hash, file)
	if err != nil {
		return nil, written, err
	}
	return hash.Sum(nil), written, nil
}

func equalDigest(left, right []byte) bool {
	return subtle.ConstantTimeCompare(left, right) == 1
}

func readIPABundleInfoFromReaderAt(reader io.ReaderAt, size int64) (bundleInfo, error) {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return bundleInfo{}, fmt.Errorf("open IPA: %w", err)
	}
	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if isTopLevelAppInfoPlist(file.Name) {
			return readBundleInfoFromZip(file)
		}
	}
	return bundleInfo{}, fmt.Errorf("missing Info.plist in IPA")
}
