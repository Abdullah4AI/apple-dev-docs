package install

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	detachedProcessHelperEnv      = "ASC_TEST_SKILLS_DETACHED_HELPER"
	detachedProcessStartedPathEnv = "ASC_TEST_SKILLS_DETACHED_STARTED"
	detachedProcessDonePathEnv    = "ASC_TEST_SKILLS_DETACHED_DONE"
	claimGuardHelperEnv           = "ASC_TEST_SKILLS_CLAIM_GUARD_HELPER"
	claimGuardLockPathEnv         = "ASC_TEST_SKILLS_CLAIM_GUARD_LOCK"
	claimGuardStartedPathEnv      = "ASC_TEST_SKILLS_CLAIM_GUARD_STARTED"
	claimGuardReleasePathEnv      = "ASC_TEST_SKILLS_CLAIM_GUARD_RELEASE"
)

func TestStartDetachedSkillsCheckProcessReturnsWithoutWaiting(t *testing.T) {
	if os.Getenv(detachedProcessHelperEnv) == "1" {
		startedPath := os.Getenv(detachedProcessStartedPathEnv)
		donePath := os.Getenv(detachedProcessDonePathEnv)
		if err := os.WriteFile(startedPath, []byte("started"), 0o600); err != nil {
			os.Exit(2)
		}
		time.Sleep(time.Second)
		if err := os.WriteFile(donePath, []byte("done"), 0o600); err != nil {
			os.Exit(3)
		}
		return
	}

	tmpDir := t.TempDir()
	startedPath := filepath.Join(tmpDir, "started")
	donePath := filepath.Join(tmpDir, "done")
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error: %v", err)
	}
	env := append(
		os.Environ(),
		detachedProcessHelperEnv+"=1",
		detachedProcessStartedPathEnv+"="+startedPath,
		detachedProcessDonePathEnv+"="+donePath,
	)

	startedAt := time.Now()
	err = startDetachedSkillsCheckProcess(
		executable,
		[]string{"-test.run=^TestStartDetachedSkillsCheckProcessReturnsWithoutWaiting$"},
		env,
	)
	if err != nil {
		t.Fatalf("startDetachedSkillsCheckProcess() error: %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed >= 500*time.Millisecond {
		t.Fatalf("detached launcher waited %s, want under 500ms", elapsed)
	}

	waitForTestFile(t, startedPath)
	if _, err := os.Stat(donePath); err == nil {
		t.Fatal("helper completed before detached launcher returned")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat helper completion marker: %v", err)
	}
	waitForTestFile(t, donePath)
}

func TestSkillsCheckWorkerEnvironmentReplacesPrivateValues(t *testing.T) {
	spec := skillsCheckWorkerSpec{
		cachePath: "/new/cache",
		lockPath:  "/new/lock",
		token:     "new-token",
	}
	env := skillsCheckWorkerEnvironment([]string{
		"PATH=/bin",
		skillsWorkerEnvVar + "=old",
		skillsWorkerCacheEnvVar + "=/old/cache",
		skillsWorkerLockEnvVar + "=/old/lock",
		skillsWorkerTokenEnvVar + "=old-token",
	}, spec)

	want := map[string]string{
		skillsWorkerEnvVar:      "1",
		skillsWorkerCacheEnvVar: spec.cachePath,
		skillsWorkerLockEnvVar:  spec.lockPath,
		skillsWorkerTokenEnvVar: spec.token,
	}
	counts := make(map[string]int)
	for _, entry := range env {
		key, value, _ := strings.Cut(entry, "=")
		if wantValue, ok := want[key]; ok {
			counts[key]++
			if value != wantValue {
				t.Fatalf("%s = %q, want %q", key, value, wantValue)
			}
		}
	}
	for key := range want {
		if counts[key] != 1 {
			t.Fatalf("%s occurrences = %d, want 1", key, counts[key])
		}
	}
}

func TestSkillsCheckClaimGuardExcludesOtherProcess(t *testing.T) {
	if os.Getenv(claimGuardHelperEnv) == "1" {
		lockPath := os.Getenv(claimGuardLockPathEnv)
		guard, acquired, err := acquireSkillsCheckClaimGuard(lockPath+skillsCheckClaimSuffix, true)
		if err != nil || !acquired {
			t.Fatalf("helper acquire guard = (%v, %v)", acquired, err)
		}
		if err := os.WriteFile(os.Getenv(claimGuardStartedPathEnv), []byte("started"), 0o600); err != nil {
			t.Fatalf("helper write started marker: %v", err)
		}
		waitForTestFile(t, os.Getenv(claimGuardReleasePathEnv))
		if err := releaseSkillsCheckClaimGuard(guard); err != nil {
			t.Fatalf("helper release guard: %v", err)
		}
		return
	}

	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, skillsCheckLockFilename)
	startedPath := filepath.Join(tmpDir, "guard-started")
	releasePath := filepath.Join(tmpDir, "guard-release")
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	if err := createSkillsCheckLock(lockPath, "stale-token", now.Add(-skillsWorkerRetryDelay-time.Second)); err != nil {
		t.Fatalf("create stale lock: %v", err)
	}

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error: %v", err)
	}
	cmd := exec.Command(executable, "-test.run=^TestSkillsCheckClaimGuardExcludesOtherProcess$")
	cmd.Env = append(
		os.Environ(),
		claimGuardHelperEnv+"=1",
		claimGuardLockPathEnv+"="+lockPath,
		claimGuardStartedPathEnv+"="+startedPath,
		claimGuardReleasePathEnv+"="+releasePath,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start guard helper: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	waitForTestFile(t, startedPath)

	startedAt := time.Now()
	token, claimed, err := defaultClaimSkillsCheckWorker(lockPath, now)
	if err != nil {
		t.Fatalf("claim while guard busy: %v", err)
	}
	if claimed || token != "" {
		t.Fatalf("claim while guard busy = (%q, %v), want no claim", token, claimed)
	}
	if elapsed := time.Since(startedAt); elapsed >= 500*time.Millisecond {
		t.Fatalf("busy guard delayed foreground claim by %s", elapsed)
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read guarded stale lock: %v", err)
	}
	if !strings.Contains(string(data), "stale-token") {
		t.Fatalf("busy contender changed stale lock: %s", data)
	}

	if err := os.WriteFile(releasePath, []byte("release"), 0o600); err != nil {
		t.Fatalf("write guard release marker: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("guard helper failed: %v", err)
	}

	token, claimed, err = defaultClaimSkillsCheckWorker(lockPath, now)
	if err != nil {
		t.Fatalf("claim after guard release: %v", err)
	}
	if !claimed || token == "" {
		t.Fatalf("claim after guard release = (%q, %v), want winner", token, claimed)
	}
}

func waitForTestFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
