//go:build !windows

package signing

import (
	"errors"
	"runtime/debug"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestSigningReconcileHelpersCloseRootDescriptors(t *testing.T) {
	previousGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previousGCPercent)

	t.Run("state writes", func(t *testing.T) {
		stateDir := t.TempDir()
		assertNoSigningDescriptorLeak(t, func() {
			if err := writeSigningStateJSON(stateDir, "state.json", map[string]string{"status": "running"}, true); err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("profile output preflight", func(t *testing.T) {
		stateDir := t.TempDir()
		assertNoSigningDescriptorLeak(t, func() {
			if err := prepareReconcileProfileOutput(stateDir); err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("profile materialization", func(t *testing.T) {
		certificate, key := newReconcileTestCertificate(t, "Descriptor Test")
		content := buildReconcileTestMobileProvision(t, map[string]any{
			"UUID":           "00000000-0000-0000-0000-0000000000FD",
			"ExpirationDate": time.Now().Add(30 * 24 * time.Hour),
			"Entitlements":   map[string]any{},
		}, certificate, key)
		stateDir := t.TempDir()
		assertNoSigningDescriptorLeak(t, func() {
			if _, err := writeVerifiedProfile(stateDir, content); err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("archive inspection", func(t *testing.T) {
		archive := t.TempDir()
		assertNoSigningDescriptorLeak(t, func() {
			if _, err := inspectSigningArchive(archive); err == nil {
				t.Fatal("empty archive inspection unexpectedly succeeded")
			}
		})
	})
}

func assertNoSigningDescriptorLeak(t *testing.T, operation func()) {
	t.Helper()
	operation()
	before := countOpenSigningDescriptors(t)
	for range 8 {
		operation()
	}
	after := countOpenSigningDescriptors(t)
	if leaked := after - before; leaked > 0 {
		t.Fatalf("operation leaked %d file descriptors", leaked)
	}
}

func countOpenSigningDescriptors(t *testing.T) int {
	t.Helper()
	var limit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		t.Fatal(err)
	}
	maximum := 4096
	if limit.Cur < uint64(maximum) {
		maximum = int(limit.Cur)
	}
	count := 0
	for descriptor := range maximum {
		if _, err := unix.FcntlInt(uintptr(descriptor), unix.F_GETFD, 0); err == nil {
			count++
		} else if !errors.Is(err, unix.EBADF) {
			t.Fatalf("inspect descriptor %d: %v", descriptor, err)
		}
	}
	return count
}
