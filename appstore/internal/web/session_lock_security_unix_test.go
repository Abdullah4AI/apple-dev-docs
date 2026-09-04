//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package web

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionEntryLockRejectsInsecureSharedDirectory(t *testing.T) {
	root := t.TempDir()
	withStubbedSessionSharedLockRoot(t, root)
	t.Setenv(webSessionCacheDirEnv, filepath.Join(t.TempDir(), "web-cache"))

	sharedDir := filepath.Join(root, sessionSharedLockDirName())
	if err := os.Mkdir(sharedDir, 0o755); err != nil {
		t.Fatalf("Mkdir error: %v", err)
	}
	if err := os.Chmod(sharedDir, 0o755); err != nil {
		t.Fatalf("Chmod error: %v", err)
	}

	key := webSessionCacheKey("user@example.com")
	acquireSessionEntryLock(key)()
	sharedPath := sessionEntryLockPaths(key)[1]
	if _, err := os.Stat(sharedPath); !os.IsNotExist(err) {
		t.Fatalf("expected insecure shared directory to be rejected, stat error: %v", err)
	}
}

func TestSessionEntryLockRejectsSharedLockSymlink(t *testing.T) {
	root := t.TempDir()
	withStubbedSessionSharedLockRoot(t, root)
	t.Setenv(webSessionCacheDirEnv, filepath.Join(t.TempDir(), "web-cache"))
	withShortSessionLockWait(t, 10*time.Millisecond)

	key := webSessionCacheKey("user@example.com")
	sharedPath := sessionEntryLockPaths(key)[1]
	if err := os.Mkdir(filepath.Dir(sharedPath), 0o700); err != nil {
		t.Fatalf("Mkdir error: %v", err)
	}
	target := filepath.Join(t.TempDir(), "target.lock")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	if err := os.Symlink(target, sharedPath); err != nil {
		t.Fatalf("Symlink error: %v", err)
	}

	release := acquireSessionEntryLock(key)
	defer release()
	targetRelease, ok := acquireLockFile(target)
	if !ok {
		t.Fatal("expected shared symlink to be rejected instead of locking its target")
	}
	targetRelease()
}
