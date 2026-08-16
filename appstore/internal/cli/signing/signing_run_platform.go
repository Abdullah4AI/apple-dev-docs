//go:build darwin

package signing

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/secureopen"
)

const (
	signingRunDiagnosticLimit  = 64 << 10
	signingRunChildWaitDelay   = 5 * time.Second
	signingRunLockPollInterval = 50 * time.Millisecond
)

var (
	signingRunActiveXcodeMajorVersion     = activeSigningRunXcodeMajorVersion
	signingRunCommandContext              = exec.CommandContext
	signingRunProfileInstallDirFn         = signingRunProfileInstallDir
	signingRunStateDirFn                  = signingRunStateDir
	signingRunRecoveryRemoveSearchEntryFn = removeKeychainSearchEntry
	signingRunRecoveryDeleteKeychainFn    = deleteSigningRunKeychain
	signingRunKillProcessGroupFn          = func(pid int, signal syscall.Signal) error {
		return syscall.Kill(-pid, signal)
	}
)

func platformSigningRunDeps() signingRunDeps {
	return signingRunDeps{
		GOOS:                      "darwin",
		Stderr:                    os.Stderr,
		RandomBytes:               signingRunRandomBytes,
		TempDir:                   createSigningRunTempDir,
		RemoveTempDir:             removeSigningRunTempDir,
		AcquireLock:               acquireSigningRunLock,
		Recover:                   recoverSigningRunJournal,
		WriteJournal:              writeSigningRunJournal,
		RemoveJournal:             removeSigningRunJournal,
		KeychainSearchList:        keychainSearchList,
		CreateKeychain:            createSigningRunKeychain,
		ImportIdentity:            importSigningRunIdentity,
		SetKeychainSearchList:     setKeychainSearchList,
		RemoveKeychainSearchEntry: removeKeychainSearchEntry,
		DeleteKeychain:            deleteSigningRunKeychain,
		InstallProfile:            installSigningRunProfile,
		RemoveProfile:             removeSigningRunProfile,
		RunChild:                  runSigningRunChild,
	}
}

func signingRunRandomBytes(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("random byte count must be positive")
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		return nil, err
	}
	return data, nil
}

func createSigningRunTempDir() (string, error) {
	path, err := os.MkdirTemp("", "asc-signing-run.")
	if err != nil {
		return "", err
	}
	if err := os.Chmod(path, 0o700); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func removeSigningRunTempDir(path string) error {
	cleanPath := filepath.Clean(path)
	if filepath.Dir(cleanPath) != filepath.Clean(os.TempDir()) ||
		!strings.HasPrefix(filepath.Base(cleanPath), "asc-signing-run.") {
		return fmt.Errorf("refusing to remove unexpected signing directory %q", path)
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("refusing to remove unsafe signing directory %q", path)
	}
	rooted, err := os.OpenRoot(cleanPath)
	if err != nil {
		return err
	}
	directory, err := rooted.Open(".")
	if err != nil {
		_ = rooted.Close()
		return err
	}
	entries, err := directory.ReadDir(-1)
	_ = directory.Close()
	if err != nil {
		_ = rooted.Close()
		return err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			_ = rooted.Close()
			return err
		}
		if !info.Mode().IsRegular() {
			_ = rooted.Close()
			return fmt.Errorf("refusing to remove unexpected non-regular entry %q", entry.Name())
		}
	}
	for _, entry := range entries {
		if err := rooted.Remove(entry.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
			_ = rooted.Close()
			return err
		}
	}
	if err := rooted.Close(); err != nil {
		return err
	}
	return os.Remove(cleanPath)
}

func signingRunStateDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	cacheRoot, err := rootfs.New(cacheDir)
	if err != nil {
		return "", err
	}
	const relative = "asc/signing-run/v1"
	if err := cacheRoot.MkdirAll(relative, 0o700); err != nil {
		return "", err
	}
	dir, err := cacheRoot.Resolve(relative)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func signingRunJournalPath() (string, error) {
	dir, err := signingRunStateDirFn()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "journal.json"), nil
}

func writeSigningRunJournal(journal signingRunJournal, overwrite bool) error {
	data, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path, err := signingRunJournalPath()
	if err != nil {
		return err
	}
	_, err = shared.SafeWriteFileNoSymlink(
		path,
		0o600,
		overwrite,
		".journal-*",
		".journal-backup-*",
		func(file *os.File) (int64, error) {
			written, writeErr := file.Write(data)
			return int64(written), writeErr
		},
	)
	return err
}

func removeSigningRunJournal() error {
	path, err := signingRunJournalPath()
	if err != nil {
		return err
	}
	rooted, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer rooted.Close()
	err = rooted.Remove(filepath.Base(path))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func recoverSigningRunJournal(ctx context.Context) error {
	path, err := signingRunJournalPath()
	if err != nil {
		return err
	}
	stateRoot, err := rootfs.New(filepath.Dir(path))
	if err != nil {
		return err
	}
	file, err := stateRoot.OpenFile(filepath.Base(path))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	info, statErr := file.Stat()
	if statErr != nil {
		_ = file.Close()
		return statErr
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() || stat.Nlink != 1 || info.Mode().Perm() != 0o600 {
		_ = file.Close()
		return fmt.Errorf("%w: ownership or permissions", ErrEphemeralRecoveryJournalInvalid)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, signingRunDiagnosticLimit+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return errors.Join(readErr, closeErr)
	}
	if len(data) > signingRunDiagnosticLimit {
		return fmt.Errorf("%w: size limit exceeded", ErrEphemeralRecoveryJournalInvalid)
	}
	if err := rejectDuplicateSigningRunJSONKeys(data); err != nil {
		return fmt.Errorf("%w: %w", ErrEphemeralRecoveryJournalInvalid, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var journal signingRunJournal
	if err := decoder.Decode(&journal); err != nil {
		return fmt.Errorf("%w: %w", ErrEphemeralRecoveryJournalInvalid, err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return fmt.Errorf("%w: %w", ErrEphemeralRecoveryJournalInvalid, err)
	}
	if err := validateSigningRunJournal(journal); err != nil {
		return fmt.Errorf("%w: %w", ErrEphemeralRecoveryJournalInvalid, err)
	}
	var recoveryErr error
	if journal.ProfileCreated {
		recoveryErr = errors.Join(recoveryErr, removeSigningRunStagedProfile(
			journal.StagedProfilePath, journal.ProfileDevice, journal.ProfileInode,
		))
		recoveryErr = errors.Join(recoveryErr, removeSigningRunProfile(signingRunProfileInstall{
			Path: journal.ProfilePath, Created: true, Digest: journal.ProfileDigest,
			Device: journal.ProfileDevice, Inode: journal.ProfileInode,
		}))
	}
	recoveryErr = errors.Join(recoveryErr, signingRunRecoveryRemoveSearchEntryFn(ctx, journal.KeychainPath))
	recoveryErr = errors.Join(recoveryErr, signingRunRecoveryDeleteKeychainFn(ctx, journal.KeychainPath))
	if recoveryErr != nil {
		return recoveryErr
	}
	if err := removeSigningRunTempDir(journal.TempDir); err != nil {
		return err
	}
	return removeSigningRunJournal()
}

type signingRunJSONFrame struct {
	object    bool
	expectKey bool
	keys      map[string]struct{}
}

func rejectDuplicateSigningRunJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var stack []signingRunJSONFrame
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		switch value := token.(type) {
		case json.Delim:
			switch value {
			case '{':
				stack = append(stack, signingRunJSONFrame{object: true, expectKey: true, keys: make(map[string]struct{})})
			case '[':
				stack = append(stack, signingRunJSONFrame{})
			case '}', ']':
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				markSigningRunJSONValueConsumed(stack)
			}
		case string:
			if len(stack) > 0 && stack[len(stack)-1].object && stack[len(stack)-1].expectKey {
				current := &stack[len(stack)-1]
				if _, exists := current.keys[value]; exists {
					return fmt.Errorf("duplicate JSON field %q", value)
				}
				current.keys[value] = struct{}{}
				current.expectKey = false
			} else {
				markSigningRunJSONValueConsumed(stack)
			}
		default:
			markSigningRunJSONValueConsumed(stack)
		}
	}
}

func markSigningRunJSONValueConsumed(stack []signingRunJSONFrame) {
	if len(stack) > 0 && stack[len(stack)-1].object && !stack[len(stack)-1].expectKey {
		stack[len(stack)-1].expectKey = true
	}
}

func validateSigningRunJournal(journal signingRunJournal) error {
	if journal.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema version %d", journal.SchemaVersion)
	}
	tempDir := filepath.Clean(journal.TempDir)
	if filepath.Dir(tempDir) != filepath.Clean(os.TempDir()) ||
		!strings.HasPrefix(filepath.Base(tempDir), "asc-signing-run.") {
		return fmt.Errorf("temporary directory is outside the signing runtime root")
	}
	if filepath.Clean(journal.KeychainPath) != filepath.Join(tempDir, "signing.keychain-db") {
		return fmt.Errorf("keychain path does not match the temporary directory")
	}
	if !journal.ProfileCreated {
		if journal.ProfilePath != "" || journal.StagedProfilePath != "" || journal.ProfileDigest != "" ||
			journal.ProfileDevice != 0 || journal.ProfileInode != 0 {
			return fmt.Errorf("profile recovery fields are inconsistent")
		}
		return nil
	}
	if len(journal.ProfileDigest) != sha256.Size*2 {
		return fmt.Errorf("profile digest is invalid")
	}
	if _, err := hex.DecodeString(journal.ProfileDigest); err != nil {
		return fmt.Errorf("profile digest is invalid")
	}
	if journal.ProfileDevice == 0 || journal.ProfileInode == 0 {
		return fmt.Errorf("profile file identity is missing")
	}
	base := filepath.Base(journal.ProfilePath)
	uuid := strings.TrimSuffix(base, ".mobileprovision")
	if base == uuid || !signingRunUUIDPattern.MatchString(uuid) {
		return fmt.Errorf("profile path has an invalid UUID")
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	allowedDirs := []string{
		filepath.Join(homeDir, "Library", "Developer", "Xcode", "UserData", "Provisioning Profiles"),
		filepath.Join(homeDir, "Library", "MobileDevice", "Provisioning Profiles"),
	}
	for _, dir := range allowedDirs {
		stagedBase := filepath.Base(journal.StagedProfilePath)
		if filepath.Clean(journal.ProfilePath) == filepath.Join(dir, base) &&
			filepath.Clean(journal.StagedProfilePath) == filepath.Join(dir, stagedBase) &&
			strings.HasPrefix(stagedBase, ".asc-signing-run-profile-") {
			return nil
		}
	}
	return fmt.Errorf("profile path is outside an Xcode provisioning profile directory")
}

func acquireSigningRunLock(ctx context.Context) (func() error, error) {
	dir, err := signingRunStateDirFn()
	if err != nil {
		return nil, err
	}
	lockPath := filepath.Join(dir, "lock")
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return func() error { return errors.Join(unix.Flock(int(file.Fd()), unix.LOCK_UN), file.Close()) }, nil
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			_ = file.Close()
			return nil, err
		}
		timer := time.NewTimer(signingRunLockPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			_ = file.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func keychainSearchList(ctx context.Context) ([]string, error) {
	stdout, stderr, err := runSigningUtility(ctx, nil, "list-keychains", "-d", "user")
	if err != nil {
		return nil, utilityFailure("read keychain search list", stderr, err)
	}
	return parseKeychainSearchList(stdout)
}

func parseKeychainSearchList(data []byte) ([]string, error) {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		value, err := strconv.Unquote(line)
		if err != nil {
			return nil, fmt.Errorf("parse keychain search list entry: %w", err)
		}
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("parse keychain search list entry: empty path")
		}
		paths = append(paths, value)
	}
	return paths, nil
}

func createSigningRunKeychain(ctx context.Context, keychainPath string, password []byte) error {
	passwordHex := []byte(hex.EncodeToString(password))
	defer clear(passwordHex)
	if err := createKeychainWithSecurityFramework(keychainPath, passwordHex); err != nil {
		return fmt.Errorf("create keychain: %w", err)
	}
	_, stderr, err := runSigningUtility(ctx, nil, "set-keychain-settings", "-l", keychainPath)
	if err != nil {
		return utilityFailure("configure keychain", stderr, err)
	}
	return nil
}

func importSigningRunIdentity(ctx context.Context, keychainPath string, keychainPassword, identityData, importPassword []byte, expectedSHA1 string) error {
	if err := importPKCS12WithSecurityFramework(keychainPath, identityData, importPassword); err != nil {
		return fmt.Errorf("import identity: %w", err)
	}
	if err := withSigningRunPartitionPasswordInput(keychainPassword, func(stdin []byte) error {
		_, stderr, err := runSigningUtility(ctx, stdin, "set-key-partition-list", "-S", "apple-tool:,apple:", "-s", "-t", "private", keychainPath)
		if err != nil {
			return utilityFailure("restrict key partition list", stderr, err)
		}
		return nil
	}); err != nil {
		return err
	}
	stdout, stderr, err := runSigningUtility(ctx, nil, "find-certificate", "-a", "-Z", keychainPath)
	if err != nil {
		return utilityFailure("verify imported certificate", stderr, err)
	}
	certificates := parseSigningRunCertificateFingerprints(stdout)
	if len(certificates) != 1 || !strings.EqualFold(certificates[0], expectedSHA1) {
		return fmt.Errorf("verify imported certificate: expected only certificate %s, found %v", expectedSHA1, certificates)
	}
	_, stderr, err = runSigningUtility(ctx, nil, "find-key", "-s", "-t", "private", keychainPath)
	if err != nil {
		return utilityFailure("verify imported private key", stderr, err)
	}
	return verifySigningRunIdentityUsable(ctx, filepath.Dir(keychainPath), keychainPath, expectedSHA1)
}

func withSigningRunPartitionPasswordInput(keychainPassword []byte, operation func([]byte) error) error {
	stdin := make([]byte, hex.EncodedLen(len(keychainPassword))+1)
	hex.Encode(stdin[:len(stdin)-1], keychainPassword)
	stdin[len(stdin)-1] = '\n'
	defer clear(stdin)
	return operation(stdin)
}

func verifySigningRunIdentityUsable(ctx context.Context, tempDir, keychainPath, expectedSHA1 string) error {
	source, err := os.Open("/usr/bin/true")
	if err != nil {
		return fmt.Errorf("open codesign probe: %w", err)
	}
	defer source.Close()
	tempRoot, err := rootfs.New(tempDir)
	if err != nil {
		return err
	}
	const probeName = "codesign-probe"
	if _, err := tempRoot.WriteFrom(probeName, io.LimitReader(source, signingRunInputLimit), 0o700); err != nil {
		return fmt.Errorf("create codesign probe: %w", err)
	}
	probePath := filepath.Join(tempDir, probeName)
	defer func() {
		if rooted, openErr := os.OpenRoot(tempDir); openErr == nil {
			_ = rooted.Remove(probeName)
			_ = rooted.Close()
		}
	}()
	cmd := exec.CommandContext(ctx, "/usr/bin/codesign", "--force", "--sign", expectedSHA1, "--keychain", keychainPath, probePath)
	stdout := &limitedSigningBuffer{limit: signingRunDiagnosticLimit}
	stderr := &limitedSigningBuffer{limit: signingRunDiagnosticLimit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("verify imported signing identity: %w", err)
	}
	return nil
}

func parseSigningRunCertificateFingerprints(data []byte) []string {
	var fingerprints []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		const prefix = "SHA-1 hash: "
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		fingerprint := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if len(fingerprint) != 40 {
			continue
		}
		if _, err := hex.DecodeString(fingerprint); err == nil {
			fingerprints = append(fingerprints, strings.ToUpper(fingerprint))
		}
	}
	return fingerprints
}

func setKeychainSearchList(ctx context.Context, paths []string) error {
	args := []string{"list-keychains", "-d", "user", "-s"}
	args = append(args, paths...)
	_, stderr, err := runSigningUtility(ctx, nil, args...)
	if err != nil {
		return utilityFailure("set keychain search list", stderr, err)
	}
	return nil
}

func removeKeychainSearchEntry(ctx context.Context, keychainPath string) error {
	paths, err := keychainSearchList(ctx)
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		if path != keychainPath {
			filtered = append(filtered, path)
		}
	}
	if len(filtered) == len(paths) {
		return nil
	}
	return setKeychainSearchList(ctx, filtered)
}

func deleteSigningRunKeychain(ctx context.Context, keychainPath string) error {
	_, stderr, err := runSigningUtility(ctx, nil, "delete-keychain", keychainPath)
	if err != nil && !strings.Contains(string(stderr), "could not be found") {
		return utilityFailure("delete keychain", stderr, err)
	}
	return nil
}

func installSigningRunProfile(uuid string, data []byte, digest string, beforeCreate func(signingRunProfileInstall) error) (signingRunProfileInstall, error) {
	installDir, err := signingRunProfileInstallDirFn(context.Background())
	if err != nil {
		return signingRunProfileInstall{}, err
	}
	installRoot, err := rootfs.New(installDir)
	if err != nil {
		return signingRunProfileInstall{}, err
	}
	if err := installRoot.MkdirAll(".", 0o755); err != nil {
		return signingRunProfileInstall{}, err
	}
	name := strings.ToLower(uuid) + ".mobileprovision"
	path := filepath.Join(installDir, name)
	existingFile, err := installRoot.OpenFile(name)
	if err == nil {
		info, statErr := existingFile.Stat()
		if statErr != nil {
			_ = existingFile.Close()
			return signingRunProfileInstall{}, statErr
		}
		if info.Size() > signingRunInputLimit {
			_ = existingFile.Close()
			return signingRunProfileInstall{}, fmt.Errorf("profile destination exceeds the size limit")
		}
		existing, readErr := io.ReadAll(io.LimitReader(existingFile, signingRunInputLimit+1))
		closeErr := existingFile.Close()
		if readErr != nil || closeErr != nil {
			return signingRunProfileInstall{}, errors.Join(readErr, closeErr)
		}
		if len(existing) > signingRunInputLimit {
			return signingRunProfileInstall{}, fmt.Errorf("profile destination exceeds the size limit")
		}
		existingDigest := sha256.Sum256(existing)
		if !strings.EqualFold(hex.EncodeToString(existingDigest[:]), digest) {
			return signingRunProfileInstall{}, fmt.Errorf("profile destination already contains different content")
		}
		return signingRunProfileInstall{Path: path, Digest: digest}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return signingRunProfileInstall{}, err
	}
	rooted, err := os.OpenRoot(installDir)
	if err != nil {
		return signingRunProfileInstall{}, err
	}
	defer rooted.Close()
	file, stagedName, err := secureopen.CreateTempNoFollowInRoot(rooted, ".", ".asc-signing-run-profile-*", 0o600)
	if err != nil {
		return signingRunProfileInstall{}, err
	}
	stagedPath := filepath.Join(installDir, stagedName)
	stagedExists := true
	defer func() {
		if stagedExists {
			_ = rooted.Remove(stagedName)
		}
	}()
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return signingRunProfileInstall{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		_ = file.Close()
		return signingRunProfileInstall{}, fmt.Errorf("inspect installed provisioning profile identity")
	}
	planned := signingRunProfileInstall{
		Path: path, StagedPath: stagedPath, Created: true, Digest: digest,
		Device: uint64(stat.Dev), Inode: stat.Ino,
	}
	if err := beforeCreate(planned); err != nil {
		_ = file.Close()
		return signingRunProfileInstall{}, fmt.Errorf("journal profile installation: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return signingRunProfileInstall{}, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return signingRunProfileInstall{}, err
	}
	if err := file.Close(); err != nil {
		return signingRunProfileInstall{}, err
	}
	if err := secureopen.RenameNoReplaceInRoot(rooted, stagedName, name); err != nil {
		return signingRunProfileInstall{}, err
	}
	stagedExists = false
	planned.StagedPath = ""
	return planned, nil
}

func removeSigningRunProfile(install signingRunProfileInstall) error {
	return removeSigningRunProfileWithHook(install, nil)
}

func removeSigningRunProfileWithHook(install signingRunProfileInstall, afterVerify func() error) error {
	if !install.Created {
		return nil
	}
	rooted, err := os.OpenRoot(filepath.Dir(install.Path))
	if err != nil {
		return err
	}
	defer rooted.Close()
	name := filepath.Base(install.Path)
	quarantineName := ".asc-signing-run-profile-remove-" + name

	err = verifySigningRunProfileEntry(rooted, name, install)
	if errors.Is(err, os.ErrNotExist) {
		err = verifySigningRunProfileEntry(rooted, quarantineName, install)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("refusing to remove quarantined profile: %w", err)
		}
		return rooted.Remove(quarantineName)
	}
	if err != nil {
		return err
	}
	if afterVerify != nil {
		if err := afterVerify(); err != nil {
			return err
		}
	}
	if err := secureopen.RenameNoReplaceInRoot(rooted, name, quarantineName); err != nil {
		return fmt.Errorf("quarantine installed profile: %w", err)
	}
	if err := verifySigningRunProfileEntry(rooted, quarantineName, install); err != nil {
		verificationErr := fmt.Errorf("refusing to remove profile because it changed during cleanup: %w", err)
		if restoreErr := secureopen.RenameNoReplaceInRoot(rooted, quarantineName, name); restoreErr != nil {
			return errors.Join(verificationErr, fmt.Errorf("restore changed profile: %w", restoreErr))
		}
		return verificationErr
	}
	return rooted.Remove(quarantineName)
}

func verifySigningRunProfileEntry(rooted *os.Root, name string, install signingRunProfileInstall) error {
	file, err := secureopen.OpenExistingNoFollowInRoot(rooted, name)
	if err != nil {
		return err
	}
	info, statErr := file.Stat()
	data, readErr := io.ReadAll(io.LimitReader(file, signingRunInputLimit+1))
	closeErr := file.Close()
	if statErr != nil || readErr != nil || closeErr != nil {
		return errors.Join(statErr, readErr, closeErr)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to remove profile because it is not a regular file")
	}
	if len(data) > signingRunInputLimit {
		return fmt.Errorf("refusing to remove profile because it exceeds the size limit")
	}
	digest := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), install.Digest) {
		return fmt.Errorf("refusing to remove profile because its content changed")
	}
	if install.Device != 0 || install.Inode != 0 {
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || uint64(stat.Dev) != install.Device || stat.Ino != install.Inode {
			return fmt.Errorf("refusing to remove profile because its file identity changed")
		}
	}
	return nil
}

func removeSigningRunStagedProfile(path string, device, inode uint64) error {
	if path == "" {
		return nil
	}
	installRoot, err := rootfs.New(filepath.Dir(path))
	if err != nil {
		return err
	}
	file, err := installRoot.OpenFile(filepath.Base(path))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil || closeErr != nil {
		return errors.Join(statErr, closeErr)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || uint64(stat.Dev) != device || stat.Ino != inode {
		return fmt.Errorf("refusing to remove staged profile because its file identity changed")
	}
	rooted, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer rooted.Close()
	return rooted.Remove(filepath.Base(path))
}

var signingRunXcodeVersionPattern = regexp.MustCompile(`(?m)^Xcode[\t ]+([0-9]+)(?:[.][0-9]+)*(?:[\t ].*)?$`)

func activeSigningRunXcodeMajorVersion(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	cmd := signingRunCommandContext(ctx, "xcodebuild", "-version")
	cmd.Env = SanitizedChildEnvironment(os.Environ())
	stdout := &limitedSigningBuffer{limit: signingRunDiagnosticLimit}
	stderr := &limitedSigningBuffer{limit: signingRunDiagnosticLimit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, ctxErr
		}
		return 0, utilityFailure("inspect active Xcode version", stderr.Bytes(), err)
	}
	match := signingRunXcodeVersionPattern.FindSubmatch(stdout.Bytes())
	if len(match) != 2 {
		return 0, fmt.Errorf("inspect active Xcode version: unexpected xcodebuild output")
	}
	major, err := strconv.Atoi(string(match[1]))
	if err != nil || major < 1 {
		return 0, fmt.Errorf("inspect active Xcode version: invalid major version")
	}
	return major, nil
}

func signingRunProfileInstallDir(ctx context.Context) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	major, err := signingRunActiveXcodeMajorVersion(ctx)
	if err == nil && major >= 16 {
		return filepath.Join(homeDir, "Library", "Developer", "Xcode", "UserData", "Provisioning Profiles"), nil
	}
	return filepath.Join(homeDir, "Library", "MobileDevice", "Provisioning Profiles"), nil
}

func runSigningRunChild(ctx context.Context, argv []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = signingRunKillProcessGroupFn(pid, signingRunCancellationSignal(ctx))
		timer := time.NewTimer(signingRunChildWaitDelay)
		select {
		case err = <-done:
			timer.Stop()
		case <-timer.C:
			_ = signingRunKillProcessGroupFn(pid, syscall.SIGKILL)
			err = <-done
		}
	}
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if code := exitErr.ExitCode(); code >= 0 {
			return shared.NewProcessExitError(code)
		}
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			return shared.NewProcessExitError(128 + int(status.Signal()))
		}
	}
	return err
}

func runSigningUtility(ctx context.Context, stdin []byte, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/security", args...)
	cmd.Stdin = bytes.NewReader(stdin)
	stdout := &limitedSigningBuffer{limit: signingRunDiagnosticLimit}
	stderr := &limitedSigningBuffer{limit: signingRunDiagnosticLimit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

type limitedSigningBuffer struct {
	data  []byte
	limit int
}

func (b *limitedSigningBuffer) Write(data []byte) (int, error) {
	if len(b.data) < b.limit {
		remaining := b.limit - len(b.data)
		b.data = append(b.data, data[:min(len(data), remaining)]...)
	}
	return len(data), nil
}

func (b *limitedSigningBuffer) Bytes() []byte { return append([]byte(nil), b.data...) }

func utilityFailure(operation string, stderr []byte, err error) error {
	_ = stderr // Captured and bounded, but intentionally excluded from diagnostics.
	return fmt.Errorf("%s: %w", operation, err)
}

func systemSigningRunRoots() (*x509.CertPool, error) { return x509.SystemCertPool() }

func validateSigningRunInputPermissions(path string, info os.FileInfo, private bool) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Geteuid() {
		return fmt.Errorf("%q must be owned by the current user", path)
	}
	if stat.Nlink != 1 {
		return fmt.Errorf("%q must not have multiple hard links", path)
	}
	if private && info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%q must not be accessible by group or other users", path)
	}
	if !private && info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("%q must not be writable by group or other users", path)
	}
	return nil
}

func platformSigningRunContext(ctx context.Context) (context.Context, func()) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	return contextWithSigningRunSignals(ctx, signals, func() {
		signal.Stop(signals)
	})
}

type signingRunSignalCause struct {
	signal syscall.Signal
}

func (cause *signingRunSignalCause) Error() string {
	return fmt.Sprintf("received signal %s", cause.signal)
}

func contextWithSigningRunSignals(
	parent context.Context,
	signals <-chan os.Signal,
	stopSignals func(),
) (context.Context, func()) {
	ctx, cancel := context.WithCancelCause(parent)
	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			stopSignals()
			cancel(context.Canceled)
		})
	}
	go func() {
		select {
		case received := <-signals:
			sig, ok := received.(syscall.Signal)
			if !ok {
				cancel(fmt.Errorf("received unsupported signal %v", received))
				return
			}
			cancel(&signingRunSignalCause{signal: sig})
		case <-ctx.Done():
		}
	}()
	return ctx, stop
}

func signingRunCancellationSignal(ctx context.Context) syscall.Signal {
	var cause *signingRunSignalCause
	if errors.As(context.Cause(ctx), &cause) {
		return cause.signal
	}
	return syscall.SIGINT
}
