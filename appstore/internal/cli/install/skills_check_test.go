package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/config"
)

func TestSkillsAutoCheckEnabled(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "default enabled", value: "", want: true},
		{name: "true", value: "true", want: true},
		{name: "yes", value: "yes", want: true},
		{name: "y", value: "y", want: true},
		{name: "on", value: "on", want: true},
		{name: "one", value: "1", want: true},
		{name: "false", value: "false", want: false},
		{name: "no", value: "no", want: false},
		{name: "n", value: "n", want: false},
		{name: "off", value: "off", want: false},
		{name: "zero", value: "0", want: false},
		{name: "invalid falls back to enabled", value: "maybe", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skillsAutoCheckEnabled(tt.value); got != tt.want {
				t.Fatalf("skillsAutoCheckEnabled(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestShouldRunSkillsCheck(t *testing.T) {
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	if !shouldRunSkillsCheck(now, "") {
		t.Fatal("expected empty timestamp to trigger check")
	}
	if !shouldRunSkillsCheck(now, "not-a-time") {
		t.Fatal("expected invalid timestamp to trigger check")
	}

	recent := now.Add(-2 * time.Hour).Format(skillsCheckedAtLayout)
	if shouldRunSkillsCheck(now, recent) {
		t.Fatal("expected recent timestamp to skip check")
	}

	old := now.Add(-26 * time.Hour).Format(skillsCheckedAtLayout)
	if !shouldRunSkillsCheck(now, old) {
		t.Fatal("expected old timestamp to trigger check")
	}
}

func TestSkillsOutputHasUpdates(t *testing.T) {
	if skillsOutputHasUpdates("all skills are up to date") {
		t.Fatal("expected up-to-date output to report no updates")
	}
	if skillsOutputHasUpdates("no update available") {
		t.Fatal("expected singular no-update output to report no updates")
	}
	if !skillsOutputHasUpdates("2 updates available") {
		t.Fatal("expected updates-available output to report updates")
	}
	if !skillsOutputHasUpdates("Update available for find-skills") {
		t.Fatal("expected singular update output to report updates")
	}
}

func TestSkillsCheckSchedulerSchedulesOnlyWhenDue(t *testing.T) {
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		legacyChecked string
		cache         skillsCheckCache
		cacheErr      error
		wantStarted   bool
	}{
		{
			name:        "missing cache and empty legacy timestamp is due",
			cacheErr:    os.ErrNotExist,
			wantStarted: true,
		},
		{
			name:          "missing cache honors recent legacy timestamp",
			legacyChecked: now.Add(-time.Hour).Format(skillsCheckedAtLayout),
			cacheErr:      os.ErrNotExist,
		},
		{
			name: "old cache is due",
			cache: skillsCheckCache{
				CheckedAt: now.Add(-25 * time.Hour).Format(skillsCheckedAtLayout),
			},
			wantStarted: true,
		},
		{
			name: "recent cache is not due",
			cache: skillsCheckCache{
				CheckedAt: now.Add(-time.Hour).Format(skillsCheckedAtLayout),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			startCalls := 0
			claimCalls := 0
			scheduler := skillsCheckScheduler{
				getenv: func(key string) string {
					if key == skillsAutoCheckEnvVar {
						return "true"
					}
					return ""
				},
				progressEnabled: func() bool { return true },
				loadConfig: func() (*config.Config, error) {
					return &config.Config{SkillsCheckedAt: tt.legacyChecked}, nil
				},
				statePaths: func() (skillsCheckStatePaths, error) {
					return skillsCheckStatePaths{
						cache: filepath.Join(tmpDir, skillsCheckCacheFilename),
						lock:  filepath.Join(tmpDir, skillsCheckLockFilename),
					}, nil
				},
				loadCache: func(string) (skillsCheckCache, error) {
					return tt.cache, tt.cacheErr
				},
				storeCache: func(string, skillsCheckCache) error {
					t.Fatal("storeCache should not run without a cached notice")
					return nil
				},
				claimWorker: func(string, time.Time) (string, bool, error) {
					claimCalls++
					return "worker-token", true, nil
				},
				releaseWorker: func(string, string) error { return nil },
				startWorker: func(spec skillsCheckWorkerSpec) error {
					startCalls++
					if spec.token != "worker-token" {
						t.Fatalf("worker token = %q, want worker-token", spec.token)
					}
					return nil
				},
				now:    func() time.Time { return now },
				stderr: io.Discard,
			}

			scheduler.run()
			if got := startCalls == 1; got != tt.wantStarted {
				t.Fatalf("worker started = %v, want %v", got, tt.wantStarted)
			}
			if tt.wantStarted && claimCalls != 1 {
				t.Fatalf("claim calls = %d, want 1", claimCalls)
			}
			if !tt.wantStarted && claimCalls != 0 {
				t.Fatalf("claim calls = %d, want 0", claimCalls)
			}
		})
	}
}

func TestSkillsCheckSchedulerSkipsDisabledCIAndNonTTY(t *testing.T) {
	tests := []struct {
		name     string
		auto     string
		ci       string
		progress bool
	}{
		{name: "disabled", auto: "false", progress: true},
		{name: "CI", auto: "true", ci: "1", progress: true},
		{name: "non TTY", auto: "true", progress: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loadCalled := false
			scheduler := skillsCheckScheduler{
				getenv: func(key string) string {
					switch key {
					case skillsAutoCheckEnvVar:
						return tt.auto
					case "CI":
						return tt.ci
					default:
						return ""
					}
				},
				progressEnabled: func() bool { return tt.progress },
				loadConfig: func() (*config.Config, error) {
					loadCalled = true
					return &config.Config{}, nil
				},
				claimWorker: func(string, time.Time) (string, bool, error) {
					t.Fatal("skipped scheduler must not claim a worker")
					return "", false, nil
				},
				startWorker: func(skillsCheckWorkerSpec) error {
					t.Fatal("skipped scheduler must not start a worker")
					return nil
				},
			}
			scheduler.run()
			if loadCalled {
				t.Fatal("config should not be loaded for skipped scheduler")
			}
		})
	}
}

func TestSkillsCheckSchedulerDisplaysAndConsumesCachedNotice(t *testing.T) {
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	cache := skillsCheckCache{
		CheckedAt:       now.Add(-time.Hour).Format(skillsCheckedAtLayout),
		UpdateAvailable: true,
	}
	var stderr bytes.Buffer
	stored := skillsCheckCache{}
	storeCalls := 0
	scheduler := skillsCheckScheduler{
		getenv:          func(string) string { return "" },
		progressEnabled: func() bool { return true },
		loadConfig:      func() (*config.Config, error) { return &config.Config{}, nil },
		statePaths: func() (skillsCheckStatePaths, error) {
			return skillsCheckStatePaths{cache: "/tmp/cache", lock: "/tmp/lock"}, nil
		},
		loadCache: func(string) (skillsCheckCache, error) { return cache, nil },
		storeCache: func(_ string, value skillsCheckCache) error {
			storeCalls++
			stored = value
			return nil
		},
		claimWorker: func(string, time.Time) (string, bool, error) {
			t.Fatal("recent cache should not claim a worker")
			return "", false, nil
		},
		now:    func() time.Time { return now },
		stderr: &stderr,
	}

	scheduler.run()
	if storeCalls != 1 {
		t.Fatalf("store calls = %d, want 1", storeCalls)
	}
	if stored.UpdateAvailable {
		t.Fatal("cached update notice should be consumed atomically")
	}
	if stored.CheckedAt != cache.CheckedAt {
		t.Fatalf("checked_at = %q, want %q", stored.CheckedAt, cache.CheckedAt)
	}
	if !strings.Contains(stderr.String(), "npx skills update") {
		t.Fatalf("expected stderr notice, got %q", stderr.String())
	}
}

func TestSkillsCheckSchedulerCacheFailuresAreSilent(t *testing.T) {
	tests := []struct {
		name       string
		loadCache  func(string) (skillsCheckCache, error)
		storeCache func(string, skillsCheckCache) error
	}{
		{
			name: "malformed cache",
			loadCache: func(string) (skillsCheckCache, error) {
				return skillsCheckCache{}, errors.New("malformed cache")
			},
			storeCache: func(string, skillsCheckCache) error { return nil },
		},
		{
			name: "cached notice write failure",
			loadCache: func(string) (skillsCheckCache, error) {
				return skillsCheckCache{
					CheckedAt:       time.Now().UTC().Format(skillsCheckedAtLayout),
					UpdateAvailable: true,
				}, nil
			},
			storeCache: func(string, skillsCheckCache) error {
				return errors.New("read-only cache")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			started := false
			scheduler := skillsCheckScheduler{
				getenv:          func(string) string { return "" },
				progressEnabled: func() bool { return true },
				loadConfig:      func() (*config.Config, error) { return &config.Config{}, nil },
				statePaths: func() (skillsCheckStatePaths, error) {
					return skillsCheckStatePaths{cache: "/tmp/cache", lock: "/tmp/lock"}, nil
				},
				loadCache:  tt.loadCache,
				storeCache: tt.storeCache,
				claimWorker: func(string, time.Time) (string, bool, error) {
					started = true
					return "token", true, nil
				},
				startWorker: func(skillsCheckWorkerSpec) error {
					started = true
					return nil
				},
				now:    time.Now,
				stderr: &stderr,
			}

			scheduler.run()
			if started {
				t.Fatal("cache failure should not schedule a worker")
			}
			if stderr.Len() != 0 {
				t.Fatalf("cache failure should stay silent, got %q", stderr.String())
			}
		})
	}
}

func TestSkillsCheckSchedulerReleasesLeaseWhenStartFails(t *testing.T) {
	releasedToken := ""
	scheduler := skillsCheckScheduler{
		getenv:          func(string) string { return "" },
		progressEnabled: func() bool { return true },
		loadConfig:      func() (*config.Config, error) { return &config.Config{}, nil },
		statePaths: func() (skillsCheckStatePaths, error) {
			return skillsCheckStatePaths{cache: "/tmp/cache", lock: "/tmp/lock"}, nil
		},
		loadCache: func(string) (skillsCheckCache, error) {
			return skillsCheckCache{}, os.ErrNotExist
		},
		storeCache: func(string, skillsCheckCache) error { return nil },
		claimWorker: func(string, time.Time) (string, bool, error) {
			return "claimed-token", true, nil
		},
		releaseWorker: func(_ string, token string) error {
			releasedToken = token
			return nil
		},
		startWorker: func(skillsCheckWorkerSpec) error {
			return errors.New("start failed")
		},
		now:    time.Now,
		stderr: io.Discard,
	}

	scheduler.run()
	if releasedToken != "claimed-token" {
		t.Fatalf("released token = %q, want claimed-token", releasedToken)
	}
}

func TestDefaultClaimSkillsCheckWorkerAllowsOnlyOneConcurrentWorker(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), skillsCheckLockFilename)
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	const contenders = 16
	start := make(chan struct{})
	var winners atomic.Int32
	var winnerToken string
	var winnerMu sync.Mutex
	var wg sync.WaitGroup

	for range contenders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			token, claimed, err := defaultClaimSkillsCheckWorker(lockPath, now)
			if err != nil {
				t.Errorf("defaultClaimSkillsCheckWorker() error: %v", err)
				return
			}
			if claimed {
				winners.Add(1)
				winnerMu.Lock()
				winnerToken = token
				winnerMu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := winners.Load(); got != 1 {
		t.Fatalf("worker claims = %d, want 1", got)
	}
	if err := defaultReleaseSkillsCheckWorker(lockPath, "not-the-owner"); err != nil {
		t.Fatalf("mismatched release error: %v", err)
	}
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("mismatched release removed lock: %v", err)
	}
	if err := defaultReleaseSkillsCheckWorker(lockPath, winnerToken); err != nil {
		t.Fatalf("owner release error: %v", err)
	}
}

func TestDefaultClaimSkillsCheckWorkerKeepsClaimWhenGuardReleaseReportsError(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), skillsCheckLockFilename)
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	originalRelease := releaseSkillsCheckClaimGuardFn
	t.Cleanup(func() {
		releaseSkillsCheckClaimGuardFn = originalRelease
	})
	releaseSkillsCheckClaimGuardFn = func(file *os.File) error {
		if err := originalRelease(file); err != nil {
			return err
		}
		return errors.New("simulated guard release failure")
	}

	token, claimed, err := defaultClaimSkillsCheckWorker(lockPath, now)
	if err != nil {
		t.Fatalf("defaultClaimSkillsCheckWorker() error: %v", err)
	}
	if !claimed || token == "" {
		t.Fatalf("claim after guard release error = (%q, %v), want owner", token, claimed)
	}
	if err := defaultReleaseSkillsCheckWorker(lockPath, token); err != nil {
		t.Fatalf("release claimed worker: %v", err)
	}
}

func TestDefaultClaimSkillsCheckWorkerReclaimsStaleLease(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), skillsCheckLockFilename)
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	if err := createSkillsCheckLock(lockPath, "stale-token", now.Add(-skillsWorkerRetryDelay-time.Second)); err != nil {
		t.Fatalf("createSkillsCheckLock() error: %v", err)
	}

	token, claimed, err := defaultClaimSkillsCheckWorker(lockPath, now)
	if err != nil {
		t.Fatalf("defaultClaimSkillsCheckWorker() error: %v", err)
	}
	if !claimed || token == "" || token == "stale-token" {
		t.Fatalf("stale lease claim = (%q, %v), want new owner", token, claimed)
	}
	quarantines, err := filepath.Glob(lockPath + ".stale-*")
	if err != nil {
		t.Fatalf("glob stale lock quarantines: %v", err)
	}
	if len(quarantines) != 0 {
		t.Fatalf("stale reclaim left quarantines: %v", quarantines)
	}
}

func TestDefaultClaimSkillsCheckWorkerConcurrentStaleReclaimKeepsWinner(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), skillsCheckLockFilename)
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	const contenders = 32

	for round := 0; round < 20; round++ {
		if err := createSkillsCheckLock(lockPath, "stale-token", now.Add(-skillsWorkerRetryDelay-time.Second)); err != nil {
			t.Fatalf("round %d: create stale lock: %v", round, err)
		}

		start := make(chan struct{})
		var winnersMu sync.Mutex
		winners := make([]string, 0, 1)
		errorsSeen := make(chan error, contenders)
		var wg sync.WaitGroup
		for range contenders {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				token, claimed, err := defaultClaimSkillsCheckWorker(lockPath, now)
				if err != nil {
					errorsSeen <- err
					return
				}
				if claimed {
					winnersMu.Lock()
					winners = append(winners, token)
					winnersMu.Unlock()
				}
			}()
		}
		close(start)
		wg.Wait()
		close(errorsSeen)

		for err := range errorsSeen {
			t.Fatalf("round %d: stale contender error: %v", round, err)
		}
		if len(winners) != 1 {
			t.Fatalf("round %d: worker claims = %d, want 1", round, len(winners))
		}

		data, err := os.ReadFile(lockPath)
		if err != nil {
			t.Fatalf("round %d: read winner lock: %v", round, err)
		}
		var lock skillsCheckLock
		if err := json.Unmarshal(data, &lock); err != nil {
			t.Fatalf("round %d: decode winner lock: %v", round, err)
		}
		if lock.Token != winners[0] {
			t.Fatalf("round %d: canonical token = %q, winner = %q", round, lock.Token, winners[0])
		}
		if err := defaultReleaseSkillsCheckWorker(lockPath, winners[0]); err != nil {
			t.Fatalf("round %d: release winner: %v", round, err)
		}
	}
}

func TestDefaultStoreSkillsCheckCacheWritesTimestampAndResultTogether(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), skillsCheckCacheFilename)
	want := skillsCheckCache{
		CheckedAt:       "2026-03-05T12:00:00Z",
		UpdateAvailable: true,
	}
	if err := defaultStoreSkillsCheckCache(cachePath, want); err != nil {
		t.Fatalf("defaultStoreSkillsCheckCache() error: %v", err)
	}

	got, err := defaultLoadSkillsCheckCache(cachePath)
	if err != nil {
		t.Fatalf("defaultLoadSkillsCheckCache() error: %v", err)
	}
	if got != want {
		t.Fatalf("cache = %#v, want %#v", got, want)
	}
	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("stat cache: %v", err)
	}
	if gotPerm := info.Mode().Perm(); gotPerm != 0o600 {
		t.Fatalf("cache mode = %o, want 600", gotPerm)
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(cachePath), ".skills-check-*"))
	if err != nil {
		t.Fatalf("glob cache temporary files: %v", err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("atomic cache write left temporary files: %v", leftovers)
	}
}

func TestDefaultStoreSkillsCheckCacheFailurePreservesSymlinkTarget(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "target.json")
	cachePath := filepath.Join(tmpDir, skillsCheckCacheFilename)
	wantTarget := []byte("do not replace")
	if err := os.WriteFile(targetPath, wantTarget, 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(targetPath, cachePath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	err := defaultStoreSkillsCheckCache(cachePath, skillsCheckCache{
		CheckedAt:       "2026-03-05T12:00:00Z",
		UpdateAvailable: true,
	})
	if err == nil {
		t.Fatal("expected symlink cache write to fail")
	}
	gotTarget, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("read target: %v", readErr)
	}
	if !bytes.Equal(gotTarget, wantTarget) {
		t.Fatalf("target changed to %q", gotTarget)
	}
}

func TestSkillsCheckWorkerCachesResultAndReleasesLease(t *testing.T) {
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	stored := skillsCheckCache{}
	released := false
	worker := skillsCheckWorker{
		runCheck: func(context.Context) (string, error) {
			return "2 updates available", nil
		},
		storeCache: func(path string, cache skillsCheckCache) error {
			if path != "/tmp/cache" {
				t.Fatalf("cache path = %q", path)
			}
			stored = cache
			return nil
		},
		releaseWorker: func(path, token string) error {
			released = path == "/tmp/lock" && token == "token"
			return nil
		},
		now:     func() time.Time { return now },
		timeout: time.Second,
	}
	spec := skillsCheckWorkerSpec{cachePath: "/tmp/cache", lockPath: "/tmp/lock", token: "token"}

	if err := worker.run(context.Background(), spec); err != nil {
		t.Fatalf("worker.run() error: %v", err)
	}
	if stored.CheckedAt != now.Format(skillsCheckedAtLayout) || !stored.UpdateAvailable {
		t.Fatalf("stored cache = %#v", stored)
	}
	if !released {
		t.Fatal("worker did not release successful lease")
	}
}

func TestSkillsCheckWorkerPersistsGenericFailureButKeepsLeaseOnMaintenanceFailure(t *testing.T) {
	errRun := errors.New("checker failed")
	tests := []struct {
		name        string
		runErr      error
		storeErr    error
		wantStored  bool
		wantRelease bool
	}{
		{
			name:        "generic checker failure records attempt",
			runErr:      errRun,
			wantStored:  true,
			wantRelease: true,
		},
		{
			name:   "unavailable checker keeps cooldown lease",
			runErr: errSkillsCheckUnavailable,
		},
		{
			name:       "cache failure keeps cooldown lease",
			storeErr:   errors.New("cache failed"),
			wantStored: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := false
			released := false
			worker := skillsCheckWorker{
				runCheck: func(context.Context) (string, error) {
					return "", tt.runErr
				},
				storeCache: func(_ string, cache skillsCheckCache) error {
					stored = true
					if cache.UpdateAvailable {
						t.Fatal("failed checker must not cache an update")
					}
					return tt.storeErr
				},
				releaseWorker: func(string, string) error {
					released = true
					return nil
				},
				now:     time.Now,
				timeout: time.Second,
			}
			spec := skillsCheckWorkerSpec{cachePath: "/tmp/cache", lockPath: "/tmp/lock", token: "token"}
			_ = worker.run(context.Background(), spec)
			if stored != tt.wantStored {
				t.Fatalf("cache stored = %v, want %v", stored, tt.wantStored)
			}
			if released != tt.wantRelease {
				t.Fatalf("lease released = %v, want %v", released, tt.wantRelease)
			}
		})
	}
}

func TestRunSkillsCheckWorkerIfRequestedRequiresPrivateMarker(t *testing.T) {
	t.Setenv(skillsWorkerEnvVar, "")
	if RunSkillsCheckWorkerIfRequested() {
		t.Fatal("ordinary process should not enter worker mode")
	}

	t.Setenv(skillsWorkerEnvVar, "1")
	t.Setenv(skillsWorkerCacheEnvVar, "")
	t.Setenv(skillsWorkerLockEnvVar, "")
	t.Setenv(skillsWorkerTokenEnvVar, "")
	if !RunSkillsCheckWorkerIfRequested() {
		t.Fatal("private marker should consume the worker process")
	}
}

func TestDefaultRunSkillsCheckCommandUsesSkillsBinaryCheckCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	restoreSkillsCheckLookups(t)
	mockSkills := writeExecutable(t, "#!/bin/sh\necho \"$@\"\n")
	lookupSkillsCheckCLI = func(file string) (string, error) {
		if file != "skills" {
			t.Fatalf("lookupSkillsCheckCLI called with %q, want skills", file)
		}
		return mockSkills, nil
	}
	lookupNpx = func(string) (string, error) {
		t.Fatal("lookupNpx should not run when skills is available")
		return "", errors.New("unexpected")
	}

	output, err := defaultRunSkillsCheckCommand(context.Background())
	if err != nil {
		t.Fatalf("defaultRunSkillsCheckCommand() error: %v", err)
	}
	if strings.TrimSpace(output) != "check" {
		t.Fatalf("checker args = %q, want check", output)
	}
}

func TestDefaultRunSkillsCheckCommandMissingCLIsIsUnavailable(t *testing.T) {
	restoreSkillsCheckLookups(t)
	lookupSkillsCheckCLI = func(string) (string, error) {
		return "", exec.ErrNotFound
	}
	lookupNpx = func(string) (string, error) {
		return "", exec.ErrNotFound
	}

	output, err := defaultRunSkillsCheckCommand(context.Background())
	if !errors.Is(err, errSkillsCheckUnavailable) {
		t.Fatalf("error = %v, want errSkillsCheckUnavailable", err)
	}
	if output != "" {
		t.Fatalf("output = %q, want empty", output)
	}
}

func TestDefaultRunSkillsCheckCommandFallsBackToNpxOffline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	restoreSkillsCheckLookups(t)
	mockNpx := writeExecutable(t, "#!/bin/sh\nprintf '%s|%s\\n' \"$*\" \"$npm_config_offline\"\n")
	lookupSkillsCheckCLI = func(string) (string, error) {
		return "", exec.ErrNotFound
	}
	lookupNpx = func(file string) (string, error) {
		if file != "npx" {
			t.Fatalf("lookupNpx called with %q, want npx", file)
		}
		return mockNpx, nil
	}

	output, err := defaultRunSkillsCheckCommand(context.Background())
	if err != nil {
		t.Fatalf("defaultRunSkillsCheckCommand() error: %v", err)
	}
	if !strings.Contains(output, "--offline --yes skills check|true") {
		t.Fatalf("offline npx invocation = %q", output)
	}
}

func TestDefaultRunSkillsCheckCommandOfflineCacheMissIsUnavailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	restoreSkillsCheckLookups(t)
	mockNpx := writeExecutable(t, "#!/bin/sh\necho ENOTCACHED >&2\nexit 1\n")
	lookupSkillsCheckCLI = func(string) (string, error) {
		return "", exec.ErrNotFound
	}
	lookupNpx = func(string) (string, error) {
		return mockNpx, nil
	}

	output, err := defaultRunSkillsCheckCommand(context.Background())
	if !errors.Is(err, errSkillsCheckUnavailable) {
		t.Fatalf("error = %v, want errSkillsCheckUnavailable", err)
	}
	if !strings.Contains(output, "ENOTCACHED") {
		t.Fatalf("output = %q, want ENOTCACHED", output)
	}
}

func TestDefaultRunSkillsCheckCommandDoesNotWaitForDescendantPipes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	restoreSkillsCheckLookups(t)
	mockSkills := writeExecutable(t, "#!/bin/sh\nsleep 10 &\nprintf '2 updates available\\n'\n")
	lookupSkillsCheckCLI = func(string) (string, error) {
		return mockSkills, nil
	}
	lookupNpx = func(string) (string, error) {
		t.Fatal("lookupNpx should not run")
		return "", errors.New("unexpected")
	}

	startedAt := time.Now()
	output, err := defaultRunSkillsCheckCommand(context.Background())
	if err != nil {
		t.Fatalf("defaultRunSkillsCheckCommand() error: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed >= 2*time.Second {
		t.Fatalf("checker waited %s for 10-second descendant", elapsed)
	}
	if !strings.Contains(output, "2 updates available") {
		t.Fatalf("output = %q, want update result", output)
	}
}

func TestCappedSkillsCheckOutputRetainsLimitWhileDraining(t *testing.T) {
	var output cappedSkillsCheckOutput
	payload := bytes.Repeat([]byte("x"), maxSkillsCheckOutputBytes+4096)

	written, err := output.Write(payload)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if written != len(payload) {
		t.Fatalf("Write() = %d, want %d", written, len(payload))
	}
	if got := len(output.String()); got != maxSkillsCheckOutputBytes {
		t.Fatalf("retained output = %d bytes, want %d", got, maxSkillsCheckOutputBytes)
	}

	written, err = output.Write([]byte("still drained"))
	if err != nil {
		t.Fatalf("second Write() error: %v", err)
	}
	if written != len("still drained") {
		t.Fatalf("second Write() = %d, want %d", written, len("still drained"))
	}
	if got := len(output.String()); got != maxSkillsCheckOutputBytes {
		t.Fatalf("retained output after cap = %d bytes, want %d", got, maxSkillsCheckOutputBytes)
	}
}

func TestRunSkillsCheckProcessReturnsContextDeadline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	mockSkills := writeExecutable(t, "#!/bin/sh\nsleep 10\n")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := runSkillsCheckProcess(ctx, mockSkills, nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("runSkillsCheckProcess() error = %v, want deadline exceeded", err)
	}
}

func TestDefaultRunSkillsCheckCommandUsesNonProjectWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	restoreSkillsCheckLookups(t)
	mockSkills := writeExecutable(t, "#!/bin/sh\nprintf '%s\\n' \"$PWD\"\n")
	lookupSkillsCheckCLI = func(string) (string, error) {
		return mockSkills, nil
	}

	output, err := defaultRunSkillsCheckCommand(context.Background())
	if err != nil {
		t.Fatalf("defaultRunSkillsCheckCommand() error: %v", err)
	}
	want := skillsCheckWorkingDirectory()
	if strings.TrimSpace(output) != want {
		t.Fatalf("working directory = %q, want %q", strings.TrimSpace(output), want)
	}
}

func TestShouldSkipProjectLocalSkillsBinary(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	projectDir := filepath.Join(repoRoot, "subdir")
	if err := os.Mkdir(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	t.Chdir(projectDir)

	localBinary := filepath.Join(repoRoot, "node_modules", ".bin", "skills")
	if err := os.MkdirAll(filepath.Dir(localBinary), 0o755); err != nil {
		t.Fatalf("mkdir local binary dir: %v", err)
	}
	if err := os.WriteFile(localBinary, []byte("fixture"), 0o755); err != nil {
		t.Fatalf("write local binary: %v", err)
	}
	if !shouldSkipProjectLocalSkillsBinary(localBinary) {
		t.Fatal("expected project-local skills binary to be skipped")
	}
	externalBinary := filepath.Join(t.TempDir(), "skills")
	if shouldSkipProjectLocalSkillsBinary(externalBinary) {
		t.Fatal("expected external skills binary to be allowed")
	}
}

func TestValidateSkillsCheckWorkerSpec(t *testing.T) {
	valid := skillsCheckWorkerSpec{
		cachePath: filepath.Join(t.TempDir(), "cache"),
		lockPath:  filepath.Join(t.TempDir(), "lock"),
		token:     "token",
	}
	if err := validateSkillsCheckWorkerSpec(valid); err != nil {
		t.Fatalf("valid spec error: %v", err)
	}

	invalid := []skillsCheckWorkerSpec{
		{cachePath: "relative", lockPath: valid.lockPath, token: valid.token},
		{cachePath: valid.cachePath, lockPath: "relative", token: valid.token},
		{cachePath: valid.cachePath, lockPath: valid.lockPath},
	}
	for _, spec := range invalid {
		if err := validateSkillsCheckWorkerSpec(spec); err == nil {
			t.Fatalf("expected invalid spec error for %#v", spec)
		}
	}
}

func restoreSkillsCheckLookups(t *testing.T) {
	t.Helper()
	origSkills := lookupSkillsCheckCLI
	origNpx := lookupNpx
	t.Cleanup(func() {
		lookupSkillsCheckCLI = origSkills
		lookupNpx = origNpx
	})
}

func writeExecutable(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "skills-check-fixture")
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	return path
}
