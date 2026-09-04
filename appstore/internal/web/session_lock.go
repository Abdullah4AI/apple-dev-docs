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
// The lock is advisory and best effort by design. Acquisition is bounded, a
// lock left behind by a killed process remains harmless because advisory locks
// are released when descriptors close, and any failure to lock falls through
// to the unlocked operation: refusing to persist or discard a session because
// a lock file cannot be created would turn a cache
// optimization into an auth outage, and refusing to discard one would leave a
// proven-stale jar to burn another 2FA code.
var (
	errSessionLockHeld      = errors.New("session lock is held")
	sessionLockPollInterval = 2 * time.Millisecond
	sessionLockWaitTimeout  = 2 * time.Second
	sessionSharedLockRoot   = platformSessionLockRoot
)

// withSessionEntryLock runs fn while holding the advisory lock for one cached
// session entry.
func withSessionEntryLock(key string, fn func() error) error {
	release := acquireSessionEntryLock(key)
	defer release()
	return fn()
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
