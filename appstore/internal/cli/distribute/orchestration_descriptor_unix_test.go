//go:build !windows

package distribute

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"golang.org/x/sys/unix"
)

func TestHashDistributionFileClosesRootDescriptor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.ipa")
	if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	previousGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previousGCPercent)
	before := countOpenDistributionDescriptors(t)
	for range 32 {
		if _, err := hashDistributionFile(path); err != nil {
			t.Fatal(err)
		}
	}
	after := countOpenDistributionDescriptors(t)
	if leaked := after - before; leaked > 1 {
		t.Fatalf("hashDistributionFile leaked %d file descriptors", leaked)
	}
}

func TestSnapshotXCArchiveClosesRootDescriptors(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "App.xcarchive")
	writeIdentityArchiveFixture(t, archive, "Descriptor Leak", "9.8.7", "654", "16.0")
	runRoot, err := rootfs.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer runRoot.Close()

	previousGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(previousGCPercent)
	before := countOpenDistributionDescriptors(t)
	for index := range 8 {
		relative := filepath.Join("inputs", fmt.Sprintf("App-%d.xcarchive", index))
		if _, err := snapshotXCArchive(context.Background(), archive, runRoot, relative); err != nil {
			t.Fatal(err)
		}
	}
	after := countOpenDistributionDescriptors(t)
	if leaked := after - before; leaked > 1 {
		t.Fatalf("snapshotXCArchive leaked %d file descriptors", leaked)
	}
}

func countOpenDistributionDescriptors(t *testing.T) int {
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
