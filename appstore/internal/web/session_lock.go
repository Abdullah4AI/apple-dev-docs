package web

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Cached web sessions are shared mutable state across concurrent `asc`
// processes: one can persist a refreshed session while another is deciding
// whether the entry it loaded is still the one to delete. A per-entry advisory
// lock file serializes those transactions so a compare and the delete it
// authorizes cannot straddle a persist.
//
// The lock is taken at two anchors, in a fixed order so concurrent holders
// cannot deadlock against each other. The session cache directory covers
// file-backed entries, which live there anyway. A stable per-user OS directory
// covers the keychain store, which is global to the user: two processes that
// select the keychain backend under different ASC_WEB_SESSION_CACHE_DIR or HOME
// values share no cache directory, yet still read and write the same keychain
// item. Holding whichever anchors can be created makes those processes exclude
// each other as long as either anchor is shared.
//
// The lock is advisory and acquisition is bounded. A lock left behind by a
// killed process remains harmless because advisory locks are released when
// descriptors close. Store mutations fail closed when the shared lock cannot
// be acquired, preventing an aggregate read-modify-write from racing another
// process and silently dropping a different Apple ID's session.
var (
	errSessionLockHeld             = errors.New("session lock is held")
	errSessionStoreLockUnavailable = errors.New("shared session-store lock unavailable")
	sessionLockPollInterval        = 2 * time.Millisecond
	sessionLockWaitTimeout         = 2 * time.Second
	sessionSharedLockRoot          = platformSessionLockRoot
)

// withSessionEntryLock runs fn while holding the cache-local global lock and
// the advisory locks for one cached session entry. The cache-local lock is
// what lets DeleteAllSessions exclude file mutations; keychain mutations take
// the stable store lock below while they are inside this critical section.
func withSessionEntryLock(key string, fn func() error) error {
	releaseGlobal := acquireSessionCacheGlobalLock()
	defer releaseGlobal()

	release := acquireSessionEntryLock(key)
	defer release()
	return fn()
}

// withSessionDeleteAllLock serializes a delete-all transaction with every
// backend mutation that can touch the selected cache. The cache-local anchor
// is attempted first, followed by the stable aggregate-keychain anchor, which
// is the same order used by per-entry persistence and deletion paths. A cache
// directory that cannot hold a local lock follows the existing fail-open file
// mutation behavior, so the underlying file operation still reports its own
// error. The callback must use unlocked keychain helpers because the store
// lock is held for the whole cross-backend transaction.
func withSessionDeleteAllLock(selection backendSelection, fn func() error) error {
	needsFileLock := selection.backend == sessionBackendFile || selection.fallbackFile
	needsStoreLock := selection.backend == sessionBackendKeychain || selection.fallbackKeychain

	if needsFileLock {
		release := acquireSessionCacheGlobalLock()
		defer release()
	}

	if needsStoreLock {
		return withSessionStoreLock(fn)
	}
	return fn()
}

// withSessionStoreLock serializes read-modify-write operations on the single
// aggregate keychain item shared by all Apple IDs.
func withSessionStoreLock(fn func() error) error {
	dir := strings.TrimSpace(sessionSharedLockRoot())
	if dir == "" {
		return errSessionStoreLockUnavailable
	}
	path := filepath.Join(dir, sessionSharedLockDirName(), "store.lock")
	if release, ok := acquireSharedSessionLockFile(path); ok {
		defer release()
	} else {
		return errSessionStoreLockUnavailable
	}
	return fn()
}

// acquireSessionCacheGlobalLock is best effort, matching the existing
// per-entry file mutation behavior when the selected cache directory cannot
// host a lock file. The required form below remains available to tests and to
// callers that need to distinguish an unavailable local barrier.
func acquireSessionCacheGlobalLock() func() {
	release, ok := acquireRequiredSessionCacheGlobalLock()
	if !ok || release == nil {
		return func() {}
	}
	return release
}

// acquireRequiredSessionCacheGlobalLock acquires the store anchor belonging
// to the selected file cache and reports whether it was acquired. A missing
// cache anchor is acceptable when the cache directory itself cannot be
// resolved; the file operation will report that error. Callers that require
// the barrier can fail closed, while normal file mutations use the best-effort
// wrapper above for compatibility with existing behavior.
func acquireRequiredSessionCacheGlobalLock() (func(), bool) {
	paths := sessionGlobalLockPaths()
	if len(paths) == 0 {
		return func() {}, true
	}
	if paths[0] == sessionSharedGlobalLockPath() {
		return acquireSharedSessionLockFile(paths[0])
	}
	return acquireLockFile(paths[0])
}

// acquireSessionEntryLock takes the entry lock at every anchor that can be
// locked and returns a release func for them. Anchors that cannot be taken are
// skipped rather than reported: see the fail-open rationale above.
func acquireSessionEntryLock(key string) func() {
	if strings.TrimSpace(key) == "" {
		return func() {}
	}
	paths := sessionEntryLockPaths(key)
	sharedPath := sessionSharedEntryLockPath(key)
	releases := make([]func(), 0, len(paths))
	for _, path := range paths {
		acquire := acquireLockFile
		if path == sharedPath {
			acquire = acquireSharedSessionLockFile
		}
		if release, ok := acquire(path); ok {
			releases = append(releases, release)
		}
	}
	return func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}
}

// sessionEntryLockPaths returns the lock anchors for one entry, in the fixed
// order every caller acquires them.
func sessionEntryLockPaths(key string) []string {
	name := "session-" + key + ".lock"
	paths := make([]string, 0, 2)
	if dir, err := webSessionCacheDir(); err == nil && strings.TrimSpace(dir) != "" {
		paths = append(paths, filepath.Join(dir, name))
	}
	if path := sessionSharedEntryLockPath(key); path != "" {
		paths = append(paths, path)
	}
	return paths
}

// sessionGlobalLockPaths describes the cache-local and stable shared store
// anchors used by lock-observation tests and the delete-all transaction.
// Per-account keychain mutations use withSessionStoreLock for only the stable
// shared anchor; file mutations use the cache-local anchor through
// withSessionEntryLock.
func sessionGlobalLockPaths() []string {
	paths := make([]string, 0, 2)
	if dir, err := webSessionCacheDir(); err == nil && strings.TrimSpace(dir) != "" {
		paths = append(paths, filepath.Join(dir, "store.lock"))
	}
	if path := sessionSharedGlobalLockPath(); path != "" {
		if len(paths) == 0 || paths[0] != path {
			paths = append(paths, path)
		}
	}
	return paths
}

func sessionSharedGlobalLockPath() string {
	dir := strings.TrimSpace(sessionSharedLockRoot())
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, sessionSharedLockDirName(), "store.lock")
}

// acquireSessionGlobalLock is retained as a lock-test seam. It follows the
// same cache-directory-then-stable-anchor order as sessionEntryLockPaths but
// is not used by production store mutations.
func acquireSessionGlobalLock() func() {
	paths := sessionGlobalLockPaths()
	releases := make([]func(), 0, len(paths))
	sharedPath := sessionSharedGlobalLockPath()
	for _, path := range paths {
		acquire := acquireLockFile
		if path == sharedPath {
			acquire = acquireSharedSessionLockFile
		}
		if release, ok := acquire(path); ok {
			releases = append(releases, release)
		}
	}
	return func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}
}

func sessionSharedEntryLockPath(key string) string {
	dir := strings.TrimSpace(sessionSharedLockRoot())
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, sessionSharedLockDirName(), "session-"+key+".lock")
}

// sessionSharedLockDirName keeps the persistent shared anchor per OS user. Its
// parent is stable across process-local cache, HOME, and temporary-directory
// settings so processes that reach the same keychain derive the same path.
func sessionSharedLockDirName() string { return platformSessionLockDirName() }

func acquireLockFile(path string) (func(), bool) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, false
	}
	return acquirePreparedLockFile(path, func(path string) (*os.File, error) {
		return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	})
}

func acquirePreparedLockFile(path string, openFile func(string) (*os.File, error)) (func(), bool) {
	deadline := time.Now().Add(sessionLockWaitTimeout)
	for {
		file, err := openFile(path)
		if err == nil {
			if err := lockSessionFile(file); err == nil {
				return func() {
					_ = unlockSessionFile(file)
					_ = file.Close()
				}, true
			} else if !isSessionLockHeld(err) {
				_ = file.Close()
				return nil, false
			}
			_ = file.Close()
			if !time.Now().Before(deadline) {
				return nil, false
			}
			time.Sleep(sessionLockPollInterval)
			continue
		}
		if !os.IsNotExist(err) && !os.IsPermission(err) {
			// A read-only or otherwise unusable directory cannot be locked.
			return nil, false
		}
		if !time.Now().Before(deadline) {
			return nil, false
		}
		time.Sleep(sessionLockPollInterval)
	}
}

func isSessionLockHeld(err error) bool { return errors.Is(err, errSessionLockHeld) }
