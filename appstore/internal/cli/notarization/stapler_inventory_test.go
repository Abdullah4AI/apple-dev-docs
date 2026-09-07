package notarization

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStaplerDirectoryInventoryBindsNestedBytesAndMode(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	nestedPath := filepath.Join(targetPath, "Contents", "Info.plist")
	if err := os.MkdirAll(filepath.Dir(nestedPath), 0o755); err != nil {
		t.Fatalf("create bundle contents: %v", err)
	}
	if err := os.WriteFile(nestedPath, []byte("original"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	baseline, err := target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err != nil {
		t.Fatalf("capture baseline: %v", err)
	}
	if baseline.entryCount != 3 {
		t.Fatalf("baseline entry count = %d, want one root, directory, and file", baseline.entryCount)
	}
	// Keep the size and mode unchanged: the content digest must still bind the
	// nested file, rather than relying on a directory or outer-target identity.
	if err := os.WriteFile(nestedPath, []byte("changed!"), 0o600); err != nil {
		t.Fatalf("replace nested file: %v", err)
	}
	err = target.verifyDirectoryInventory(context.Background(), baseline, "before validation")
	if err == nil {
		t.Fatal("verifyDirectoryInventory() = nil, want nested-byte mismatch")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("verifyDirectoryInventory() error = %T %v, want identity error", err, err)
	}
	if strings.Contains(err.Error(), targetPath) || strings.Contains(err.Error(), "Info.plist") {
		t.Fatalf("verification error = %q, must not expose nested path", err.Error())
	}

	if err := os.WriteFile(nestedPath, []byte("original"), 0o640); err != nil {
		t.Fatalf("restore nested file with changed mode: %v", err)
	}
	if err := os.Chmod(nestedPath, 0o640); err != nil {
		t.Fatalf("change nested mode: %v", err)
	}
	err = target.verifyDirectoryInventory(context.Background(), baseline, "before validation")
	if err == nil {
		t.Fatal("verifyDirectoryInventory() = nil, want nested-mode mismatch")
	}
	if !errors.As(err, &identityErr) {
		t.Fatalf("verifyDirectoryInventory() error = %T %v, want identity error", err, err)
	}
}

func TestStaplerDirectoryInventoryRejectsSymlinkAndSpecialEntries(t *testing.T) {
	tests := []struct {
		name string
		make func(t *testing.T, directory string)
	}{
		{
			name: "escaping symlink",
			make: func(t *testing.T, directory string) {
				if err := os.Symlink(filepath.Dir(directory), filepath.Join(directory, "Contents-link")); err != nil {
					if runtime.GOOS == "windows" {
						t.Skipf("symlink creation unavailable: %v", err)
					}
					t.Fatalf("create symlink: %v", err)
				}
			},
		},
		{
			name: "special file",
			make: func(t *testing.T, directory string) {
				makeStaplerSpecialEntry(t, directory)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			targetPath := filepath.Join(t.TempDir(), "MyApp.app")
			if err := os.Mkdir(targetPath, 0o755); err != nil {
				t.Fatalf("create bundle: %v", err)
			}
			test.make(t, targetPath)
			target, err := validateStaplerTargetDetails(targetPath)
			if err != nil {
				t.Fatalf("validate target: %v", err)
			}
			t.Cleanup(target.close)

			_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
			if err == nil {
				t.Fatal("captureDirectoryInventoryAtStage() = nil, want fail-closed entry rejection")
			}
			var verifyErr *staplerTargetVerifyError
			if !errors.As(err, &verifyErr) {
				t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want verify error", err, err)
			}
			if strings.Contains(err.Error(), targetPath) || strings.Contains(err.Error(), "Contents-link") || strings.Contains(err.Error(), "named-pipe") {
				t.Fatalf("inventory error = %q, must not expose entry path", err.Error())
			}
		})
	}
}

func TestStaplerDirectoryInventoryAllowsContainedSymlinkWithoutFollowing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixtures require platform support")
	}
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	versioned := filepath.Join(targetPath, "Versions", "1")
	if err := os.MkdirAll(versioned, 0o755); err != nil {
		t.Fatalf("create versioned bundle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(versioned, "Info.plist"), []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write versioned bundle file: %v", err)
	}
	linkPath := filepath.Join(targetPath, "Versions", "Current")
	if err := os.Symlink("1", linkPath); err != nil {
		t.Fatalf("create contained symlink: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	inventory, err := target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err != nil {
		t.Fatalf("capture contained-symlink inventory: %v", err)
	}
	if inventory.entryCount != 5 {
		t.Fatalf("inventory entry count = %d, want root, Versions, 1, Info.plist, and Current", inventory.entryCount)
	}

	if err := os.Remove(linkPath); err != nil {
		t.Fatalf("remove contained symlink: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatalf("write outside target: %v", err)
	}
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatalf("create escaping symlink: %v", err)
	}
	if _, err := target.captureDirectoryInventoryAtStage(context.Background(), "before validation"); err == nil {
		t.Fatal("capture escaping-symlink inventory = nil, want fail-closed rejection")
	}
}

func TestStaplerDirectoryInventoryRejectsSymlinkReplacedBeforeReadlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixtures require platform support")
	}
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	versionedPath := filepath.Join(targetPath, "Versions", "1")
	if err := os.MkdirAll(versionedPath, 0o755); err != nil {
		t.Fatalf("create versioned bundle: %v", err)
	}
	linkPath := filepath.Join(targetPath, "Versions", "Current")
	if err := os.Symlink("1", linkPath); err != nil {
		t.Fatalf("create contained symlink: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := afterStaplerInventoryEntryLstatFn
	replaced := false
	afterStaplerInventoryEntryLstatFn = func(relative string) {
		if replaced || relative != "Versions/Current" {
			return
		}
		replaced = true
		if err := os.Remove(linkPath); err != nil {
			t.Fatalf("remove retained symlink: %v", err)
		}
		if err := os.WriteFile(linkPath, []byte("replacement"), 0o600); err != nil {
			t.Fatalf("write replacement entry: %v", err)
		}
	}
	t.Cleanup(func() { afterStaplerInventoryEntryLstatFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want symlink replacement rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
	var verifyErr *staplerTargetVerifyError
	if errors.As(err, &verifyErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity rather than verification error", err, err)
	}
}

func TestStaplerDirectoryInventoryHonorsCanceledContext(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	if err := os.Mkdir(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	baseline, err := target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err != nil {
		t.Fatalf("capture baseline: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = target.captureDirectoryInventoryAtStage(ctx, "before validation")
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %v, want context cancellation", err)
	}
	var verifyErr *staplerTargetVerifyError
	if errors.As(err, &verifyErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, cancellation must not be classified as verify error", err, err)
	}
	if strings.Contains(err.Error(), targetPath) {
		t.Fatalf("inventory error = %q, must not expose target path", err.Error())
	}
	err = target.verifyDirectoryInventory(ctx, baseline, "before validation")
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("verifyDirectoryInventory() error = %v, want context cancellation", err)
	}
	if errors.As(err, &verifyErr) {
		t.Fatalf("verifyDirectoryInventory() error = %T %v, cancellation must not be classified as verify error", err, err)
	}
}

func TestStaplerDirectoryInventoryStopsWhenContextCancelsDuringScan(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	nestedPath := filepath.Join(targetPath, "Contents", "Info.plist")
	if err := os.MkdirAll(filepath.Dir(nestedPath), 0o755); err != nil {
		t.Fatalf("create bundle contents: %v", err)
	}
	if err := os.WriteFile(nestedPath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	ctx := &cancelDuringStaplerInventoryContext{cancelAfterChecks: 4}
	_, err = target.captureDirectoryInventoryAtStage(ctx, "before validation")
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %v, want cancellation during scan", err)
	}
	var verifyErr *staplerTargetVerifyError
	if errors.As(err, &verifyErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, cancellation must not be classified as verify error", err, err)
	}
	if strings.Contains(err.Error(), targetPath) || strings.Contains(err.Error(), "Info.plist") {
		t.Fatalf("inventory error = %q, must not expose path", err.Error())
	}
}

func TestStaplerRegularFileFingerprintPreservesContextCancellation(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.dmg")
	if err := os.WriteFile(targetPath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	expected, err := target.captureRegularFileFingerprintAtStage(context.Background(), "before validation")
	if err != nil {
		t.Fatalf("capture baseline fingerprint: %v", err)
	}
	tests := []struct {
		name string
		ctx  context.Context
		want error
	}{
		{name: "canceled", ctx: canceledContextForStaplerTest(), want: context.Canceled},
		{name: "deadline exceeded", ctx: expiredContextForStaplerTest(), want: context.DeadlineExceeded},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := target.captureRegularFileFingerprintAtStage(test.ctx, "before validation")
			if err == nil || !errors.Is(err, test.want) {
				t.Fatalf("captureRegularFileFingerprintAtStage() error = %v, want %v", err, test.want)
			}
			var verifyErr *staplerTargetVerifyError
			if errors.As(err, &verifyErr) {
				t.Fatalf("captureRegularFileFingerprintAtStage() error = %T %v, context failure must not be classified as verify error", err, err)
			}
			err = target.verifyRegularFileFingerprint(test.ctx, expected, "after validation")
			if err == nil || !errors.Is(err, test.want) {
				t.Fatalf("verifyRegularFileFingerprint() error = %v, want %v", err, test.want)
			}
			if errors.As(err, &verifyErr) {
				t.Fatalf("verifyRegularFileFingerprint() error = %T %v, context failure must not be classified as verify error", err, err)
			}
		})
	}
}

func TestStaplerRegularFileFingerprintWrapsOperationalFailure(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.dmg")
	if err := os.WriteFile(targetPath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)
	if err := target.handle.Close(); err != nil {
		t.Fatalf("close retained target handle: %v", err)
	}

	_, err = target.captureRegularFileFingerprintAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureRegularFileFingerprintAtStage() = nil, want operational failure")
	}
	var verifyErr *staplerTargetVerifyError
	if !errors.As(err, &verifyErr) {
		t.Fatalf("captureRegularFileFingerprintAtStage() error = %T %v, want verify error", err, err)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("captureRegularFileFingerprintAtStage() error = %v, unexpected context failure", err)
	}
}

func canceledContextForStaplerTest() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func expiredContextForStaplerTest() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	cancel()
	return ctx
}

type cancelDuringStaplerInventoryContext struct {
	checks            int
	cancelAfterChecks int
}

func (ctx *cancelDuringStaplerInventoryContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (*cancelDuringStaplerInventoryContext) Done() <-chan struct{} {
	return nil
}

func (ctx *cancelDuringStaplerInventoryContext) Err() error {
	ctx.checks++
	if ctx.checks > ctx.cancelAfterChecks {
		return context.Canceled
	}
	return nil
}

func (*cancelDuringStaplerInventoryContext) Value(any) any {
	return nil
}

func TestStaplerDirectoryInventoryEnforcesEntryBounds(t *testing.T) {
	scanner := staplerInventoryScanner{entryCount: staplerInventoryMaxEntries}
	if err := scanner.noteEntry("entry"); err == nil {
		t.Fatal("noteEntry() = nil at entry cap, want rejection")
	}
	scanner = staplerInventoryScanner{}
	if err := scanner.noteEntry(strings.Repeat("x", staplerInventoryMaxPath+1)); err == nil {
		t.Fatal("noteEntry() = nil over path cap, want rejection")
	}
}

func TestStaplerDirectoryInventoryClassifiesKindSwapDuringOpenRootAsChange(t *testing.T) {
	for _, replacementKind := range []string{"regular file", "symlink"} {
		t.Run(replacementKind, func(t *testing.T) {
			rootPath := t.TempDir()
			entryPath := filepath.Join(rootPath, "Contents")
			preservedPath := filepath.Join(rootPath, "Contents.original")
			if err := os.Mkdir(entryPath, 0o700); err != nil {
				t.Fatalf("create original directory: %v", err)
			}
			root, err := os.OpenRoot(rootPath)
			if err != nil {
				t.Fatalf("open inventory root: %v", err)
			}
			t.Cleanup(func() { _ = root.Close() })
			before, err := root.Lstat("Contents")
			if err != nil {
				t.Fatalf("lstat original directory: %v", err)
			}
			if err := os.Rename(entryPath, preservedPath); err != nil {
				t.Fatalf("preserve original directory: %v", err)
			}
			switch replacementKind {
			case "regular file":
				if err := os.WriteFile(entryPath, []byte("replacement"), 0o600); err != nil {
					t.Fatalf("create regular replacement: %v", err)
				}
			case "symlink":
				if err := os.Symlink("Contents.original", entryPath); err != nil {
					t.Fatalf("create symlink replacement: %v", err)
				}
			}
			t.Cleanup(func() {
				_ = os.Remove(entryPath)
				_ = os.Rename(preservedPath, entryPath)
			})

			scanner := staplerInventoryScanner{ctx: context.Background()}
			err = scanner.recordDirectory(root, "Contents", "Contents", before)
			if !errors.Is(err, errStaplerInventoryChanged) {
				t.Fatalf("recordDirectory() error = %v, want inventory-change classification", err)
			}
		})
	}
}

func TestStaplerDirectoryInventoryReadsLargeDirectoriesInBoundedBatches(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	if err := os.Mkdir(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := readdirStaplerInventoryNamesFn
	var calls, requested int
	readdirStaplerInventoryNamesFn = func(_ *os.File, count int) ([]string, error) {
		calls++
		requested = count
		// Simulate a directory larger than the hard cap without creating
		// hundreds of thousands of filesystem entries. The scanner must reject
		// the batch before retaining or inspecting its names.
		return make([]string, staplerInventoryMaxEntries), io.EOF
	}
	t.Cleanup(func() { readdirStaplerInventoryNamesFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want entry-cap rejection")
	}
	if calls != 1 {
		t.Fatalf("directory read calls = %d, want one bounded-cap read", calls)
	}
	if requested != staplerInventoryReadBatchSize {
		t.Fatalf("directory read request = %d, want bounded batch size %d", requested, staplerInventoryReadBatchSize)
	}
	if strings.Contains(err.Error(), targetPath) {
		t.Fatalf("inventory error = %q, must not expose target path", err.Error())
	}
	var identityErr *staplerTargetIdentityError
	if errors.As(err, &identityErr) {
		t.Fatalf("inventory error = %v, initial entry overflow must not be inventory change", err)
	}
	var verifyErr *staplerTargetVerifyError
	if !errors.As(err, &verifyErr) {
		t.Fatalf("inventory error = %T %v, want bounded inspection failure", err, err)
	}
}

func TestStaplerDirectoryInventoryIgnoresDirectoryMtimeOnlyChanges(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	nestedPath := filepath.Join(targetPath, "Contents", "Info.plist")
	if err := os.MkdirAll(filepath.Dir(nestedPath), 0o755); err != nil {
		t.Fatalf("create bundle contents: %v", err)
	}
	if err := os.WriteFile(nestedPath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := readdirStaplerInventoryNamesFn
	changed := false
	readdirStaplerInventoryNamesFn = func(file *os.File, count int) ([]string, error) {
		batch, readErr := file.Readdirnames(count)
		if !changed {
			changed = true
			mtime := time.Now().Add(2 * time.Second)
			if err := os.Chtimes(targetPath, mtime, mtime); err != nil {
				t.Fatalf("change directory mtime: %v", err)
			}
		}
		return batch, readErr
	}
	t.Cleanup(func() { readdirStaplerInventoryNamesFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err != nil {
		t.Fatalf("captureDirectoryInventoryAtStage() = %v, want content-only inventory to ignore mtime-only metadata change", err)
	}
}

func TestStaplerDirectoryInventoryIgnoresUnrelatedSiblingMetadataChurn(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "MyApp.app")
	if err := os.MkdirAll(filepath.Join(targetPath, "Contents"), 0o755); err != nil {
		t.Fatalf("create bundle contents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetPath, "Contents", "Info.plist"), []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	before, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("stat bundle: %v", err)
	}

	// A sibling create can perturb a directory's reported size even when the
	// recursively inventoried entries are otherwise stable. Keep the siblings
	// in this isolated fixture; this test exercises the identity check only,
	// while the recursive inventory remains responsible for binding contents.
	const siblingCount = 4096
	var after os.FileInfo
	for index := 0; index < siblingCount; index++ {
		path := filepath.Join(targetPath, "sibling-"+strconv.Itoa(index))
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatalf("create sibling %d: %v", index, err)
		}
		after, err = os.Stat(targetPath)
		if err != nil {
			t.Fatalf("stat bundle after sibling %d: %v", index, err)
		}
		if before.Size() != after.Size() {
			break
		}
	}
	if before.Size() == after.Size() {
		// Some filesystems do not expose directory-size churn. Force the other
		// directory-only metadata field to change so the test remains portable.
		mtime := before.ModTime().Add(2 * time.Second)
		if err := os.Chtimes(targetPath, mtime, mtime); err != nil {
			t.Fatalf("change directory metadata: %v", err)
		}
		after, err = os.Stat(targetPath)
		if err != nil {
			t.Fatalf("restat bundle: %v", err)
		}
	}
	if before.Size() == after.Size() {
		if before.ModTime().Equal(after.ModTime()) {
			t.Fatal("sibling metadata churn did not change directory metadata")
		}
	}
	if !staplerInventoryInfoStable(before, after) {
		t.Fatal("directory metadata churn must not invalidate a stable directory identity")
	}
}

func TestStaplerInventoryRetainsRegularFileMetadataBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Info.plist")
	if err := os.WriteFile(path, []byte("fixture"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if err := os.WriteFile(path, []byte("fixture!"), 0o644); err != nil {
		t.Fatalf("change file size: %v", err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("restat file: %v", err)
	}
	if staplerInventoryInfoStable(before, after) {
		t.Fatal("regular-file size changes must invalidate the bound metadata")
	}
}

func TestStaplerDirectoryInventoryRejectsEntryAddedAfterEnumeration(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	nestedPath := filepath.Join(targetPath, "Contents", "Info.plist")
	if err := os.MkdirAll(filepath.Dir(nestedPath), 0o755); err != nil {
		t.Fatalf("create bundle contents: %v", err)
	}
	if err := os.WriteFile(nestedPath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := readdirStaplerInventoryNamesFn
	injected := false
	readdirStaplerInventoryNamesFn = func(file *os.File, count int) ([]string, error) {
		batch, readErr := file.Readdirnames(count)
		if !injected && errors.Is(readErr, io.EOF) {
			injected = true
			if err := os.WriteFile(filepath.Join(targetPath, "added-after-enumeration"), []byte("late"), 0o600); err != nil {
				t.Fatalf("add late bundle entry: %v", err)
			}
		}
		return batch, readErr
	}
	t.Cleanup(func() { readdirStaplerInventoryNamesFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want entry-addition race rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
}

func TestStaplerDirectoryInventoryRejectsSameNameReplacementAfterEnumeration(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	entryPath := filepath.Join(targetPath, "Info.plist")
	if err := os.Mkdir(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := os.WriteFile(entryPath, []byte("original"), 0o600); err != nil {
		t.Fatalf("write bundle entry: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := afterStaplerInventoryNamesFn
	afterStaplerInventoryNamesFn = func() {
		originalPath := entryPath + ".original"
		if err := os.Rename(entryPath, originalPath); err != nil {
			t.Fatalf("move original entry: %v", err)
		}
		if err := os.WriteFile(entryPath, []byte("replaced"), 0o600); err != nil {
			t.Fatalf("replace bundle entry: %v", err)
		}
		if err := os.Remove(originalPath); err != nil {
			t.Fatalf("remove original entry: %v", err)
		}
	}
	t.Cleanup(func() { afterStaplerInventoryNamesFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want same-name replacement rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
}

func TestStaplerDirectoryInventoryRejectsDirectChildAddedAfterFinalEnumeration(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	entryPath := filepath.Join(targetPath, "Info.plist")
	latePath := filepath.Join(targetPath, "Late.plist")
	if err := os.Mkdir(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := os.WriteFile(entryPath, []byte("original"), 0o600); err != nil {
		t.Fatalf("write bundle entry: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := afterStaplerInventoryNamesFn
	afterStaplerInventoryNamesFn = func() {
		if err := os.WriteFile(latePath, []byte("late"), 0o600); err != nil {
			t.Fatalf("add late bundle entry: %v", err)
		}
	}
	t.Cleanup(func() { afterStaplerInventoryNamesFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want late direct-child rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
	if strings.Contains(err.Error(), targetPath) || strings.Contains(err.Error(), "Late.plist") {
		t.Fatalf("inventory error = %q, must not expose entry path", err.Error())
	}
}

func TestStaplerDirectoryInventoryRejectsSameNameReplacementAfterRetainedChecks(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	entryPath := filepath.Join(targetPath, "Info.plist")
	originalPath := entryPath + ".original"
	if err := os.Mkdir(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := os.WriteFile(entryPath, []byte("original"), 0o600); err != nil {
		t.Fatalf("write bundle entry: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := afterStaplerInventoryEntriesFn
	afterStaplerInventoryEntriesFn = func() {
		if err := os.Rename(entryPath, originalPath); err != nil {
			t.Fatalf("move original entry: %v", err)
		}
		if err := os.WriteFile(entryPath, []byte("replaced"), 0o600); err != nil {
			t.Fatalf("replace bundle entry: %v", err)
		}
		if err := os.Remove(originalPath); err != nil {
			t.Fatalf("remove original entry: %v", err)
		}
	}
	t.Cleanup(func() { afterStaplerInventoryEntriesFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want late same-name replacement rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
	if strings.Contains(err.Error(), targetPath) || strings.Contains(err.Error(), "Info.plist") {
		t.Fatalf("inventory error = %q, must not expose entry path", err.Error())
	}
}

func TestStaplerDirectoryInventoryRejectsEntryRemovedAfterEnumeration(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	if err := os.Mkdir(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	keptPath := filepath.Join(targetPath, "Info.plist")
	if err := os.WriteFile(keptPath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write kept file: %v", err)
	}
	removedPath := filepath.Join(targetPath, "Removed.bin")
	if err := os.WriteFile(removedPath, []byte("vanishing"), 0o600); err != nil {
		t.Fatalf("write removed file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := readdirStaplerInventoryNamesFn
	injected := false
	readdirStaplerInventoryNamesFn = func(file *os.File, count int) ([]string, error) {
		batch, readErr := file.Readdirnames(count)
		if !injected && errors.Is(readErr, io.EOF) {
			injected = true
			if err := os.Remove(removedPath); err != nil {
				t.Fatalf("remove enumerated bundle entry: %v", err)
			}
		}
		return batch, readErr
	}
	t.Cleanup(func() { readdirStaplerInventoryNamesFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want entry-removal race rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
}

func TestStaplerDirectoryInventoryRejectsNestedDirectoryRemovedBeforeOpen(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	nestedPath := filepath.Join(targetPath, "Contents")
	if err := os.MkdirAll(nestedPath, 0o755); err != nil {
		t.Fatalf("create bundle contents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedPath, "Info.plist"), []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := afterStaplerInventoryEntryLstatFn
	removed := false
	afterStaplerInventoryEntryLstatFn = func(relative string) {
		if removed || relative != "Contents" {
			return
		}
		removed = true
		if err := os.RemoveAll(nestedPath); err != nil {
			t.Fatalf("remove enumerated bundle directory: %v", err)
		}
	}
	t.Cleanup(func() { afterStaplerInventoryEntryLstatFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want directory-removal race rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
}

func TestStaplerDirectoryInventoryRejectsNestedFileRemovedBeforeOpen(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	if err := os.Mkdir(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	entryPath := filepath.Join(targetPath, "Info.plist")
	if err := os.WriteFile(entryPath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := afterStaplerInventoryEntryLstatFn
	removed := false
	afterStaplerInventoryEntryLstatFn = func(relative string) {
		if removed || relative != "Info.plist" {
			return
		}
		removed = true
		if err := os.Remove(entryPath); err != nil {
			t.Fatalf("remove enumerated bundle file: %v", err)
		}
	}
	t.Cleanup(func() { afterStaplerInventoryEntryLstatFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want file-removal race rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
}

func TestStaplerDirectoryInventoryRejectsNestedFileReplacedBySymlinkBeforeOpen(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixtures require platform support")
	}
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	if err := os.Mkdir(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	entryPath := filepath.Join(targetPath, "Info.plist")
	if err := os.WriteFile(entryPath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := afterStaplerInventoryEntryLstatFn
	replaced := false
	afterStaplerInventoryEntryLstatFn = func(relative string) {
		if replaced || relative != "Info.plist" {
			return
		}
		replaced = true
		if err := os.Remove(entryPath); err != nil {
			t.Fatalf("remove enumerated bundle file: %v", err)
		}
		if err := os.Symlink("replacement", entryPath); err != nil {
			t.Fatalf("replace enumerated file with symlink: %v", err)
		}
	}
	t.Cleanup(func() { afterStaplerInventoryEntryLstatFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want regular-file-to-symlink race rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
	var verifyErr *staplerTargetVerifyError
	if errors.As(err, &verifyErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity rather than verification error", err, err)
	}
}

func TestStaplerDirectoryInventoryRejectsBundleRemovedBeforeFinalRebind(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	if err := os.Mkdir(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetPath, "Info.plist"), []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := afterStaplerDirectoryInventoryScanFn
	removed := false
	afterStaplerDirectoryInventoryScanFn = func() {
		if removed {
			return
		}
		removed = true
		if err := os.RemoveAll(targetPath); err != nil {
			t.Fatalf("remove scanned bundle: %v", err)
		}
	}
	t.Cleanup(func() { afterStaplerDirectoryInventoryScanFn = previous })

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want bundle-removal rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
}

func TestStaplerDirectoryInventoryRejectsParentReplacedByRegularFileBeforeFinalRebind(t *testing.T) {
	root := t.TempDir()
	parentPath := filepath.Join(root, "parent")
	originalParentPath := filepath.Join(root, "parent.original")
	targetPath := filepath.Join(parentPath, "MyApp.app")
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetPath, "Info.plist"), []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := afterStaplerDirectoryInventoryScanFn
	replaced := false
	afterStaplerDirectoryInventoryScanFn = func() {
		if replaced {
			return
		}
		replaced = true
		if err := os.Rename(parentPath, originalParentPath); err != nil {
			t.Fatalf("preserve scanned parent: %v", err)
		}
		if err := os.WriteFile(parentPath, []byte("replacement"), 0o600); err != nil {
			t.Fatalf("replace parent with regular file: %v", err)
		}
	}
	t.Cleanup(func() { afterStaplerDirectoryInventoryScanFn = previous })
	t.Cleanup(func() {
		if err := os.Remove(parentPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Errorf("remove replacement parent: %v", err)
		}
		if err := os.Rename(originalParentPath, parentPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Errorf("restore original parent: %v", err)
		}
	})

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want parent replacement rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
	var verifyErr *staplerTargetVerifyError
	if errors.As(err, &verifyErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity rather than verification error", err, err)
	}
}

func TestStaplerRegularFileFingerprintRejectsFileRemovedBeforeFinalRebind(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.dmg")
	if err := os.WriteFile(targetPath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)

	previous := afterStaplerRegularFileFingerprintFn
	removed := false
	afterStaplerRegularFileFingerprintFn = func() {
		if removed {
			return
		}
		removed = true
		if err := os.Remove(targetPath); err != nil {
			t.Fatalf("remove hashed target: %v", err)
		}
	}
	t.Cleanup(func() { afterStaplerRegularFileFingerprintFn = previous })

	_, err = target.captureRegularFileFingerprintAtStage(context.Background(), "before validation")
	if err == nil {
		t.Fatal("captureRegularFileFingerprintAtStage() = nil, want target-removal rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureRegularFileFingerprintAtStage() error = %T %v, want identity error", err, err)
	}
}

func TestStaplerRegularFileFingerprintRejectsFileRemovedBeforeStageOpen(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.dmg")
	if err := os.WriteFile(targetPath, []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)
	if err := target.verifyIdentity("before stapling"); err != nil {
		t.Fatalf("verify retained identity: %v", err)
	}
	// The retained descriptor keeps the inode alive, so the pinned identity
	// still matches while the pathname no longer resolves.
	if err := os.Remove(targetPath); err != nil {
		t.Fatalf("remove target: %v", err)
	}

	_, err = target.captureRegularFileFingerprintAtStage(context.Background(), "before stapling")
	if err == nil {
		t.Fatal("captureRegularFileFingerprintAtStage() = nil, want stage-open removal rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureRegularFileFingerprintAtStage() error = %T %v, want identity error", err, err)
	}
	var verifyErr *staplerTargetVerifyError
	if errors.As(err, &verifyErr) {
		t.Fatalf("captureRegularFileFingerprintAtStage() error = %T %v, a vanished pathname must not read as a filesystem-inspection failure", err, err)
	}
}

func TestStaplerDirectoryInventoryRejectsBundleRemovedBeforeStageOpen(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "MyApp.app")
	if err := os.Mkdir(targetPath, 0o755); err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetPath, "Info.plist"), []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write bundle file: %v", err)
	}
	target, err := validateStaplerTargetDetails(targetPath)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	t.Cleanup(target.close)
	if err := target.verifyIdentity("before stapling"); err != nil {
		t.Fatalf("verify retained identity: %v", err)
	}
	if err := os.RemoveAll(targetPath); err != nil {
		t.Fatalf("remove bundle: %v", err)
	}

	_, err = target.captureDirectoryInventoryAtStage(context.Background(), "before stapling")
	if err == nil {
		t.Fatal("captureDirectoryInventoryAtStage() = nil, want stage-open removal rejection")
	}
	var identityErr *staplerTargetIdentityError
	if !errors.As(err, &identityErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, want identity error", err, err)
	}
	var verifyErr *staplerTargetVerifyError
	if errors.As(err, &verifyErr) {
		t.Fatalf("captureDirectoryInventoryAtStage() error = %T %v, a vanished pathname must not read as a filesystem-inspection failure", err, err)
	}
}

func TestStaplerInventoryPathVanishedCoversMissingAndNonDirectoryComponents(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "missing component", err: &fs.PathError{Op: "openat", Path: "MyApp.dmg", Err: syscall.ENOENT}, want: true},
		{name: "parent is not a directory", err: &fs.PathError{Op: "openat", Path: "MyApp.dmg", Err: syscall.ENOTDIR}, want: true},
		{name: "permission revoked", err: &fs.PathError{Op: "openat", Path: "MyApp.dmg", Err: syscall.EACCES}, want: false},
		{name: "descriptor limit", err: &fs.PathError{Op: "openat", Path: "MyApp.dmg", Err: syscall.EMFILE}, want: false},
		{name: "no error", err: nil, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := staplerInventoryPathVanished(test.err); got != test.want {
				t.Fatalf("staplerInventoryPathVanished(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

func TestStaplerInventoryEntryVanishedDoesNotClassifyNonDirectoryParents(t *testing.T) {
	err := &fs.PathError{Op: "lstat", Path: "MyApp.dmg", Err: syscall.ENOTDIR}
	if staplerInventoryEntryVanished(err) {
		t.Fatal("staplerInventoryEntryVanished() classified ENOTDIR as a missing entry")
	}
	if !staplerInventoryPathVanished(err) {
		t.Fatal("staplerInventoryPathVanished() did not classify ENOTDIR as a changed path")
	}
}
