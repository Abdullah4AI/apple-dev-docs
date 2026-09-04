package web

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// The lock is what makes a compare-and-delete and a persist mutually
// exclusive, so overlapping holders must be impossible.
func TestWithSessionEntryLockExcludesConcurrentHolders(t *testing.T) {
	t.Setenv(webSessionCacheDirEnv, filepath.Join(t.TempDir(), "web-cache"))

	key := webSessionCacheKey("user@example.com")
	var mu sync.Mutex
	holders, maxHolders := 0, 0
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = withSessionEntryLock(key, func() error {
				mu.Lock()
				holders++
				if holders > maxHolders {
					maxHolders = holders
				}
				mu.Unlock()
				time.Sleep(time.Millisecond)
				mu.Lock()
				holders--
				mu.Unlock()
				return nil
			})
		}()
	}
	wg.Wait()

	if maxHolders != 1 {
		t.Fatalf("expected the entry lock to admit one holder at a time, got %d", maxHolders)
	}
	for _, path := range sessionEntryLockPaths(key) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected the persistent lock file at %q, stat error: %v", path, err)
		}
	}
}

// Two processes on the keychain backend can be configured with different cache
// directories and still share one global keychain store, so at least one anchor
// must not depend on the cache directory.
func TestSessionEntryLockSharesAnAnchorAcrossCacheDirs(t *testing.T) {
	shared := t.TempDir()
	withStubbedSessionSharedLockRoot(t, shared)

	key := webSessionCacheKey("user@example.com")
	t.Setenv(webSessionCacheDirEnv, filepath.Join(t.TempDir(), "cache-a"))
	first := sessionEntryLockPaths(key)
	t.Setenv(webSessionCacheDirEnv, filepath.Join(t.TempDir(), "cache-b"))
	second := sessionEntryLockPaths(key)

	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("expected two anchors per configuration, got %v and %v", first, second)
	}
	if first[0] == second[0] {
		t.Fatalf("expected the cache directory anchor to differ, got %q twice", first[0])
	}
	if first[1] != second[1] {
		t.Fatalf("expected a cache-directory-independent anchor, got %q and %q", first[1], second[1])
	}
	if !strings.HasPrefix(first[1], shared) {
		t.Fatalf("expected the shared anchor under %q, got %q", shared, first[1])
	}
}

func TestSessionEntryLockSharedAnchorIgnoresEnvironmentOverrides(t *testing.T) {
	key := webSessionCacheKey("user@example.com")
	t.Setenv(webSessionCacheDirEnv, filepath.Join(t.TempDir(), "cache-a"))
	first := sessionEntryLockPaths(key)[1]
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "different"))
	t.Setenv("HOME", filepath.Join(t.TempDir(), "different-home"))
	t.Setenv(webSessionCacheDirEnv, filepath.Join(t.TempDir(), "cache-b"))
	second := sessionEntryLockPaths(key)[1]
	if first != second {
		t.Fatalf("shared anchor changed with environment overrides: %q -> %q", first, second)
	}
}

// The shared anchor has to actually exclude: a holder configured with one cache
// directory must block a second one configured with another.
func TestSessionEntryLockExcludesHoldersWithDifferentCacheDirs(t *testing.T) {
	withStubbedSessionSharedLockRoot(t, t.TempDir())
	withShortSessionLockWait(t, 100*time.Millisecond)

	key := webSessionCacheKey("user@example.com")
	t.Setenv(webSessionCacheDirEnv, filepath.Join(t.TempDir(), "cache-a"))
	release := acquireSessionEntryLock(key)

	t.Setenv(webSessionCacheDirEnv, filepath.Join(t.TempDir(), "cache-b"))
	done := make(chan time.Duration, 1)
	go func() {
		start := time.Now()
		acquireSessionEntryLock(key)()
		done <- time.Since(start)
	}()

	select {
	case waited := <-done:
		if waited < sessionLockWaitTimeout {
			t.Fatalf("expected the shared anchor to hold off the second acquisition, it returned after %s", waited)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the second acquisition never returned")
	}
	sharedLock := sessionEntryLockPaths(key)[1]
	if _, err := os.Stat(sharedLock); err != nil {
		t.Fatalf("expected the holder to keep the shared lock the blocked caller gave up on: %v", err)
	}
	release()
	if _, err := os.Stat(sharedLock); err != nil {
		t.Fatalf("expected the persistent shared lock, stat error: %v", err)
	}
}

// Releasing one descriptor must not remove or damage the persistent anchor.
func TestSessionEntryLockReleaseKeepsPersistentAnchor(t *testing.T) {
	t.Setenv(webSessionCacheDirEnv, filepath.Join(t.TempDir(), "web-cache"))
	withStubbedSessionSharedLockRoot(t, t.TempDir())

	key := webSessionCacheKey("user@example.com")
	release := acquireSessionEntryLock(key)
	release()

	for _, path := range sessionEntryLockPaths(key) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected persistent anchor at %q: %v", path, err)
		}
		if reacquire, ok := acquireLockFile(path); !ok {
			t.Fatalf("expected anchor %q to be acquirable after release", path)
		} else {
			reacquire()
		}
	}
}

func withStubbedSessionSharedLockRoot(t *testing.T, dir string) {
	t.Helper()
	prev := sessionSharedLockRoot
	sessionSharedLockRoot = func() string { return dir }
	t.Cleanup(func() { sessionSharedLockRoot = prev })
}

func withShortSessionLockWait(t *testing.T, wait time.Duration) {
	t.Helper()
	prev := sessionLockWaitTimeout
	sessionLockWaitTimeout = wait
	t.Cleanup(func() { sessionLockWaitTimeout = prev })
}

// A process killed mid-transaction leaves its persistent lock file behind.
// Descriptor release makes it harmless and enables the next acquisition.
func TestWithSessionEntryLockWaitsForDescriptorRelease(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "web-cache")
	t.Setenv(webSessionCacheDirEnv, dir)

	key := webSessionCacheKey("user@example.com")
	lockPath := sessionEntryLockPaths(key)[0]
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	holder, ok := acquireLockFile(lockPath)
	if !ok {
		t.Fatal("expected first descriptor acquisition")
	}
	defer holder()
	withShortSessionLockWait(t, 20*time.Millisecond)

	ran := false
	start := time.Now()
	if err := withSessionEntryLock(key, func() error {
		ran = true
		return nil
	}); err != nil {
		t.Fatalf("withSessionEntryLock error: %v", err)
	}
	if !ran {
		t.Fatal("expected the locked operation to run")
	}
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Fatalf("expected bounded wait while descriptor is held, took %s", elapsed)
	}
}

// A cache directory that cannot hold a lock file must not block the operation:
// an unlocked persist or delete is the pre-existing behavior, an aborted login
// is not.
func TestWithSessionEntryLockFallsThroughWhenTheDirectoryIsUnusable(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	t.Setenv(webSessionCacheDirEnv, filepath.Join(blocker, "web-cache"))

	ran := false
	if err := withSessionEntryLock(webSessionCacheKey("user@example.com"), func() error {
		ran = true
		return nil
	}); err != nil {
		t.Fatalf("withSessionEntryLock error: %v", err)
	}
	if !ran {
		t.Fatal("expected the operation to run unlocked when no lock file can be created")
	}
}
