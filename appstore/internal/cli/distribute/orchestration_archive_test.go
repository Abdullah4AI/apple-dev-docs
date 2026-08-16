package distribute

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	"howett.net/plist"
)

func TestDigestXCArchiveBindsMainApplicationIdentityFromScannedPlists(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "App.xcarchive")
	writeIdentityArchiveFixture(t, archive, "Preview", "1.2.3", "42", "17.0")

	digest, err := digestXCArchive(context.Background(), archive)
	if err != nil {
		t.Fatalf("digestXCArchive() error = %v", err)
	}
	want := archiveAppIdentity{
		BundleID: "com.example.preview", Title: "Preview", Version: "1.2.3", BuildNumber: "42", MinimumOSVersion: "17.0",
	}
	if digest.App != want {
		t.Fatalf("archive app identity = %#v, want %#v", digest.App, want)
	}

	runPath := t.TempDir()
	runRoot, err := rootfs.New(runPath)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := snapshotXCArchive(context.Background(), archive, runRoot, "inputs/App.xcarchive")
	if err != nil {
		t.Fatalf("snapshotXCArchive() error = %v", err)
	}
	if snapshot.App != want {
		t.Fatalf("snapshot app identity = %#v, want %#v", snapshot.App, want)
	}
	if err := revalidateXCArchiveSnapshot(context.Background(), runRoot, snapshot); err != nil {
		t.Fatalf("revalidateXCArchiveSnapshot() error = %v", err)
	}
}

func TestDigestXCArchiveRejectsArchiveAndApplicationIdentityMismatch(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "App.xcarchive")
	writeIdentityArchiveFixture(t, archive, "Preview", "1.2.3", "42", "17.0")
	rootPath := filepath.Join(archive, "Info.plist")
	rootData, err := os.ReadFile(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if _, err := plist.Unmarshal(rootData, &root); err != nil {
		t.Fatal(err)
	}
	properties := root["ApplicationProperties"].(map[string]any)
	properties["CFBundleVersion"] = "43"
	rootData, err = plist.Marshal(root, plist.XMLFormat)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootPath, rootData, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := digestXCArchive(context.Background(), archive); err == nil || !strings.Contains(err.Error(), "identities differ") {
		t.Fatalf("digestXCArchive() error = %v, want identity mismatch", err)
	}
}

func TestDigestXCArchiveRetainsOnlyDeclaredMainApplicationIdentity(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "App.xcarchive")
	writeIdentityArchiveFixture(t, archive, "Preview", "1.2.3", "42", "17.0")
	applications := filepath.Join(archive, "Products", "Applications")
	decoyData, err := plist.Marshal(map[string]any{
		"CFBundleIdentifier": "com.example.decoy", "CFBundleDisplayName": "Decoy",
		"CFBundleShortVersionString": "9.9", "CFBundleVersion": "999",
	}, plist.XMLFormat)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 512; index++ {
		decoy := filepath.Join(applications, fmt.Sprintf("Decoy-%04d.app", index))
		if err := os.Mkdir(decoy, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(decoy, "Info.plist"), decoyData, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	digest, err := digestXCArchive(context.Background(), archive)
	if err != nil {
		t.Fatalf("digestXCArchive() error = %v", err)
	}
	if digest.App.BundleID != "com.example.preview" || digest.App.Title != "Preview" || digest.App.BuildNumber != "42" {
		t.Fatalf("declared main application identity changed by decoys: %#v", digest.App)
	}
}

func TestDigestXCArchiveIsDeterministicAndBindsModeAndContent(t *testing.T) {
	first := filepath.Join(t.TempDir(), "App.xcarchive")
	second := filepath.Join(t.TempDir(), "App.xcarchive")
	writeArchiveFixture(t, first, false)
	writeArchiveFixture(t, second, true)

	firstDigest, err := digestXCArchive(context.Background(), first)
	if err != nil {
		t.Fatalf("digestXCArchive(first) error = %v", err)
	}
	secondDigest, err := digestXCArchive(context.Background(), second)
	if err != nil {
		t.Fatalf("digestXCArchive(second) error = %v", err)
	}
	if firstDigest.RelativePath != "" || secondDigest.RelativePath != "" {
		t.Fatalf("inspection relative paths = %q, %q, want empty", firstDigest.RelativePath, secondDigest.RelativePath)
	}
	if firstDigest.TreeSHA256 != secondDigest.TreeSHA256 {
		t.Fatalf("tree digests differ by creation order: %q != %q", firstDigest.TreeSHA256, secondDigest.TreeSHA256)
	}
	if len(firstDigest.TreeSHA256) != 64 {
		t.Fatalf("tree digest length = %d, want 64", len(firstDigest.TreeSHA256))
	}
	if firstDigest.SizeBytes != int64(len("plist")+len("binary")) || firstDigest.EntryCount != 3 {
		t.Fatalf("archive facts = %+v, want 1 directory, 2 files, and 11 bytes", firstDigest)
	}

	modeBound := filepath.Join(t.TempDir(), "App.xcarchive")
	writeArchiveFixture(t, modeBound, false)
	if err := os.Chmod(filepath.Join(modeBound, "Products", "App"), 0o644); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	changed, err := digestXCArchive(context.Background(), modeBound)
	if err != nil {
		t.Fatalf("digestXCArchive(mode changed) error = %v", err)
	}
	if changed.TreeSHA256 == firstDigest.TreeSHA256 {
		t.Fatal("tree digest did not bind executable mode")
	}
}

func TestSnapshotXCArchiveCopiesPrivatelyAndRevalidates(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "App.xcarchive")
	writeArchiveFixture(t, archive, false)
	planned, err := digestXCArchive(context.Background(), archive)
	if err != nil {
		t.Fatalf("digestXCArchive() error = %v", err)
	}
	runPath := t.TempDir()
	runRoot, err := rootfs.New(runPath)
	if err != nil {
		t.Fatalf("rootfs.New() error = %v", err)
	}

	result, err := snapshotXCArchive(context.Background(), archive, runRoot, filepath.Join("inputs", "App.xcarchive"))
	if err != nil {
		t.Fatalf("snapshotXCArchive() error = %v", err)
	}
	if result.RelativePath != filepath.Join("inputs", "App.xcarchive") {
		t.Fatalf("relative path = %q", result.RelativePath)
	}
	if result.TreeSHA256 != planned.TreeSHA256 || result.SizeBytes != planned.SizeBytes || result.EntryCount != planned.EntryCount {
		t.Fatalf("snapshot evidence = %+v, want planned evidence %+v", result, planned)
	}
	info, err := os.Stat(filepath.Join(runPath, result.RelativePath))
	if err != nil {
		t.Fatalf("Stat(snapshot) error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("snapshot root mode = %#o, want 0700", got)
	}
	executable, err := os.Stat(filepath.Join(runPath, result.RelativePath, "Products", "App"))
	if err != nil {
		t.Fatalf("Stat(executable) error = %v", err)
	}
	if executable.Mode().Perm() != 0o711 {
		t.Fatalf("executable mode = %#o, want private content with source executable bits (0711)", executable.Mode().Perm())
	}
	if err := revalidateXCArchiveSnapshot(context.Background(), runRoot, result); err != nil {
		t.Fatalf("revalidateXCArchiveSnapshot() error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(runPath, result.RelativePath, "Info.plist"), []byte("changed"), 0o644); err != nil {
		t.Fatalf("mutate snapshot: %v", err)
	}
	if err := revalidateXCArchiveSnapshot(context.Background(), runRoot, result); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("revalidate after mutation error = %v, want mismatch", err)
	}
}

func TestSnapshotXCArchiveDropsPrivilegedAndNonPrivateModes(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "App.xcarchive")
	writeArchiveFixture(t, archive, false)
	sourceExecutable := filepath.Join(archive, "Products", "App")
	if err := os.Chmod(sourceExecutable, 0o755|os.ModeSetuid|os.ModeSetgid|os.ModeSticky); err != nil {
		t.Fatalf("Chmod(source executable) error = %v", err)
	}
	if err := os.Chmod(filepath.Join(archive, "Products"), 0o777|os.ModeSetgid|os.ModeSticky); err != nil {
		t.Fatalf("Chmod(source directory) error = %v", err)
	}
	runPath := t.TempDir()
	runRoot, err := rootfs.New(runPath)
	if err != nil {
		t.Fatalf("rootfs.New() error = %v", err)
	}
	result, err := snapshotXCArchive(context.Background(), archive, runRoot, "App.xcarchive")
	if err != nil {
		t.Fatalf("snapshotXCArchive() error = %v", err)
	}

	fileInfo, err := os.Stat(filepath.Join(runPath, result.RelativePath, "Products", "App"))
	if err != nil {
		t.Fatalf("Stat(snapshot executable) error = %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o711 {
		t.Fatalf("snapshot executable permissions = %#o, want 0711", got)
	}
	if got := fileInfo.Mode() & (os.ModeSetuid | os.ModeSetgid | os.ModeSticky); got != 0 {
		t.Fatalf("snapshot executable retained privileged mode bits: %s", got)
	}
	directoryInfo, err := os.Stat(filepath.Join(runPath, result.RelativePath, "Products"))
	if err != nil {
		t.Fatalf("Stat(snapshot directory) error = %v", err)
	}
	if got := directoryInfo.Mode(); got.Perm() != 0o700 || got&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		t.Fatalf("snapshot directory mode = %s, want owner-private 0700", got)
	}
	if err := revalidateXCArchiveSnapshot(context.Background(), runRoot, result); err != nil {
		t.Fatalf("revalidateXCArchiveSnapshot() error = %v", err)
	}
}

func TestXCArchiveSnapshotRejectsLinksAndSpecialFiles(t *testing.T) {
	t.Run("final input symlink", func(t *testing.T) {
		realArchive := filepath.Join(t.TempDir(), "Real.xcarchive")
		writeArchiveFixture(t, realArchive, false)
		link := filepath.Join(t.TempDir(), "App.xcarchive")
		if err := os.Symlink(realArchive, link); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}
		if _, err := digestXCArchive(context.Background(), link); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("digestXCArchive(symlink) error = %v, want symlink refusal", err)
		}
	})

	t.Run("inner symlink", func(t *testing.T) {
		archive := filepath.Join(t.TempDir(), "App.xcarchive")
		writeArchiveFixture(t, archive, false)
		if err := os.Symlink("Info.plist", filepath.Join(archive, "linked")); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}
		if _, err := digestXCArchive(context.Background(), archive); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("digestXCArchive(inner symlink) error = %v, want symlink refusal", err)
		}
	})

	t.Run("hard link", func(t *testing.T) {
		archive := filepath.Join(t.TempDir(), "App.xcarchive")
		writeArchiveFixture(t, archive, false)
		if err := os.Link(filepath.Join(archive, "Info.plist"), filepath.Join(archive, "Info-copy.plist")); err != nil {
			t.Skipf("hard links unavailable: %v", err)
		}
		if _, err := digestXCArchive(context.Background(), archive); err == nil || !strings.Contains(err.Error(), "hard link") {
			t.Fatalf("digestXCArchive(hard link) error = %v, want hard-link refusal", err)
		}
	})

	if runtime.GOOS != "windows" {
		t.Run("named pipe", func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "App.xcarchive")
			writeArchiveFixture(t, archive, false)
			pipe := filepath.Join(archive, "pipe")
			if err := makeArchiveFIFO(pipe); err != nil {
				t.Skipf("named pipes unavailable: %v", err)
			}
			if _, err := digestXCArchive(context.Background(), archive); err == nil || !strings.Contains(err.Error(), "special file") {
				t.Fatalf("digestXCArchive(pipe) error = %v, want special-file refusal", err)
			}
		})
	}
}

func TestSnapshotXCArchiveIsCreateOnlyAndCancelable(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "App.xcarchive")
	writeArchiveFixture(t, archive, false)
	runPath := t.TempDir()
	runRoot, err := rootfs.New(runPath)
	if err != nil {
		t.Fatalf("rootfs.New() error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(runPath, "existing.xcarchive"), 0o700); err != nil {
		t.Fatalf("Mkdir(existing) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(runPath, "existing.xcarchive", "sentinel"), []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile(sentinel) error = %v", err)
	}
	if _, err := snapshotXCArchive(context.Background(), archive, runRoot, "existing.xcarchive"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("snapshotXCArchive(existing) error = %v, want os.ErrExist", err)
	}
	if got, err := os.ReadFile(filepath.Join(runPath, "existing.xcarchive", "sentinel")); err != nil || string(got) != "keep" {
		t.Fatalf("existing destination sentinel = %q, %v; want keep", got, err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := snapshotXCArchive(canceled, archive, runRoot, "canceled.xcarchive"); !errors.Is(err, context.Canceled) {
		t.Fatalf("snapshotXCArchive(canceled) error = %v, want context.Canceled", err)
	}
	if _, err := os.Lstat(filepath.Join(runPath, "canceled.xcarchive")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled destination Lstat() error = %v, want not exist", err)
	}
	if _, err := digestXCArchive(canceled, archive); !errors.Is(err, context.Canceled) {
		t.Fatalf("digestXCArchive(canceled) error = %v, want context.Canceled", err)
	}
}

func TestSnapshotXCArchiveCancellationLeavesNoFinalOrStagingResidue(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "App.xcarchive")
	writeArchiveFixture(t, archive, false)
	runPath := t.TempDir()
	runRoot, err := rootfs.New(runPath)
	if err != nil {
		t.Fatalf("rootfs.New() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	copies := 0
	ctx = context.WithValue(ctx, archiveSnapshotCopyHookKey{}, archiveSnapshotCopyHookFunc(func(string) error {
		copies++
		if copies == 2 {
			cancel()
			return ctx.Err()
		}
		return nil
	}))

	if _, err := snapshotXCArchive(ctx, archive, runRoot, filepath.Join("inputs", "App.xcarchive")); !errors.Is(err, context.Canceled) {
		t.Fatalf("snapshotXCArchive(mid-copy canceled) error = %v, want context.Canceled", err)
	}
	if copies != 2 {
		t.Fatalf("copy hook calls = %d, want 2", copies)
	}
	if _, err := os.Lstat(filepath.Join(runPath, "inputs", "App.xcarchive")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("final snapshot Lstat() error = %v, want not exist", err)
	}
	entries, err := os.ReadDir(filepath.Join(runPath, "inputs"))
	if err != nil {
		t.Fatalf("ReadDir(inputs) error = %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), archiveSnapshotStagingPrefix) {
			t.Fatalf("staging residue remains after cancellation: %q", entry.Name())
		}
	}
}

func TestDigestXCArchiveRejectsWrongShape(t *testing.T) {
	file := filepath.Join(t.TempDir(), "App.xcarchive")
	if err := os.WriteFile(file, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := digestXCArchive(context.Background(), file); err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("digestXCArchive(file) error = %v, want directory error", err)
	}

	notArchive := filepath.Join(t.TempDir(), "App")
	if err := os.Mkdir(notArchive, 0o700); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if _, err := digestXCArchive(context.Background(), notArchive); err == nil || !strings.Contains(err.Error(), ".xcarchive") {
		t.Fatalf("digestXCArchive(non-archive) error = %v, want extension error", err)
	}
}

func TestArchiveTreeScannerEnforcesEntryAndPathBounds(t *testing.T) {
	scanner := &archiveTreeScanner{entryCount: archiveSnapshotMaxEntries - 1}
	if err := scanner.noteEntry("last"); err != nil {
		t.Fatalf("noteEntry(last) error = %v", err)
	}
	if err := scanner.noteEntry("one-too-many"); err == nil || !strings.Contains(err.Error(), "entries") {
		t.Fatalf("noteEntry(over limit) error = %v, want entry-limit refusal", err)
	}

	overlong := strings.Repeat("a", archiveSnapshotMaxPathBytes+1)
	if err := (&archiveTreeScanner{}).noteEntry(overlong); err == nil || !strings.Contains(err.Error(), "path length") {
		t.Fatalf("noteEntry(overlong path) error = %v, want path-length refusal", err)
	}
}

func TestDigestXCArchiveRejectsOversizeTreeBeforeReading(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "App.xcarchive")
	if err := os.Mkdir(archive, 0o700); err != nil {
		t.Fatalf("Mkdir(archive) error = %v", err)
	}
	large := filepath.Join(archive, "large")
	file, err := os.OpenFile(large, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	truncateErr := file.Truncate(archiveSnapshotMaxSizeBytes + 1)
	closeErr := file.Close()
	if truncateErr != nil {
		t.Skipf("sparse files unavailable: %v", truncateErr)
	}
	if closeErr != nil {
		t.Fatalf("Close() error = %v", closeErr)
	}
	if _, err := digestXCArchive(context.Background(), archive); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("digestXCArchive(oversize) error = %v, want size-limit refusal", err)
	}
}

func writeArchiveFixture(t *testing.T, archive string, reverse bool) {
	t.Helper()
	if err := os.Mkdir(archive, 0o700); err != nil {
		t.Fatalf("Mkdir(archive) error = %v", err)
	}
	products := filepath.Join(archive, "Products")
	if !reverse {
		if err := os.Mkdir(products, 0o750); err != nil {
			t.Fatalf("Mkdir(Products) error = %v", err)
		}
	}
	writes := []struct {
		path string
		data string
		mode os.FileMode
	}{
		{filepath.Join(archive, "Info.plist"), "plist", 0o640},
		{filepath.Join(products, "App"), "binary", 0o755},
	}
	if reverse {
		if err := os.Mkdir(products, 0o750); err != nil {
			t.Fatalf("Mkdir(Products) error = %v", err)
		}
		writes[0], writes[1] = writes[1], writes[0]
	}
	for _, write := range writes {
		if err := os.WriteFile(write.path, []byte(write.data), write.mode); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", write.path, err)
		}
		if err := os.Chmod(write.path, write.mode); err != nil {
			t.Fatalf("Chmod(%s) error = %v", write.path, err)
		}
	}
}

func writeIdentityArchiveFixture(t *testing.T, archive, title, version, build, minimumOS string) {
	t.Helper()
	applicationPath := filepath.Join("Applications", "Preview.app")
	appRelative := filepath.Join("Products", applicationPath)
	appPath := filepath.Join(archive, appRelative)
	if err := os.MkdirAll(appPath, 0o700); err != nil {
		t.Fatal(err)
	}
	root := map[string]any{
		"ApplicationProperties": map[string]any{
			"ApplicationPath": filepath.ToSlash(applicationPath), "CFBundleIdentifier": "com.example.preview",
			"CFBundleShortVersionString": version, "CFBundleVersion": build,
		},
	}
	app := map[string]any{
		"CFBundleIdentifier": "com.example.preview", "CFBundleDisplayName": title,
		"CFBundleShortVersionString": version, "CFBundleVersion": build, "MinimumOSVersion": minimumOS,
	}
	for path, value := range map[string]any{filepath.Join(archive, "Info.plist"): root, filepath.Join(appPath, "Info.plist"): app} {
		data, err := plist.Marshal(value, plist.XMLFormat)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(appPath, "Preview"), []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
}

func makeArchiveFIFO(path string) error {
	return exec.Command("mkfifo", path).Run()
}
