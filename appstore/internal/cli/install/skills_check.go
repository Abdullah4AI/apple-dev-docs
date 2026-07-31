package install

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/config"
)

const (
	skillsAutoCheckEnvVar     = "ASC_SKILLS_AUTO_CHECK"
	skillsWorkerEnvVar        = "ASC_INTERNAL_SKILLS_CHECK_WORKER"
	skillsWorkerCacheEnvVar   = "ASC_INTERNAL_SKILLS_CHECK_CACHE"
	skillsWorkerLockEnvVar    = "ASC_INTERNAL_SKILLS_CHECK_LOCK"
	skillsWorkerTokenEnvVar   = "ASC_INTERNAL_SKILLS_CHECK_TOKEN"
	skillsCheckCacheFilename  = "skills-check.json"
	skillsCheckLockFilename   = "skills-check.lock"
	skillsCheckClaimSuffix    = ".claim"
	skillsCheckInterval       = 24 * time.Hour
	skillsCheckTimeout        = 8 * time.Second
	skillsCheckPipeWaitDelay  = 250 * time.Millisecond
	skillsWorkerRetryDelay    = 10 * time.Minute
	skillsCheckedAtLayout     = time.RFC3339
	skillsUpdateMessageFmt    = "skills updates may be available. Run 'npx skills update' to refresh installed skills."
	maxSkillsCheckOutputBytes = 1 << 20
)

var (
	lookupSkillsCheckCLI           = exec.LookPath
	errSkillsCheckUnavailable      = errors.New("skills check command unavailable")
	releaseSkillsCheckClaimGuardFn = releaseSkillsCheckClaimGuard
	skillsCheckClaimMu             sync.Mutex
)

type skillsCheckCache struct {
	CheckedAt       string `json:"checked_at"`
	UpdateAvailable bool   `json:"update_available"`
}

type skillsCheckStatePaths struct {
	cache string
	lock  string
}

type skillsCheckWorkerSpec struct {
	cachePath string
	lockPath  string
	token     string
}

type skillsCheckLock struct {
	Token     string `json:"token"`
	StartedAt string `json:"started_at"`
}

type cappedSkillsCheckOutput struct {
	data []byte
}

type skillsCheckScheduler struct {
	getenv          func(string) string
	progressEnabled func() bool
	loadConfig      func() (*config.Config, error)
	statePaths      func() (skillsCheckStatePaths, error)
	loadCache       func(string) (skillsCheckCache, error)
	storeCache      func(string, skillsCheckCache) error
	claimWorker     func(string, time.Time) (string, bool, error)
	releaseWorker   func(string, string) error
	startWorker     func(skillsCheckWorkerSpec) error
	now             func() time.Time
	stderr          io.Writer
}

type skillsCheckWorker struct {
	runCheck      func(context.Context) (string, error)
	storeCache    func(string, skillsCheckCache) error
	releaseWorker func(string, string) error
	now           func() time.Time
	timeout       time.Duration
}

// MaybeScheduleSkillsUpdateCheck performs only bounded local work in the
// foreground. The actual checker runs in a detached maintenance process.
func MaybeScheduleSkillsUpdateCheck() {
	defaultSkillsCheckScheduler().run()
}

// RunSkillsCheckWorkerIfRequested handles the private detached-worker entry
// point before the public command tree is constructed.
func RunSkillsCheckWorkerIfRequested() bool {
	if strings.TrimSpace(os.Getenv(skillsWorkerEnvVar)) != "1" {
		return false
	}

	spec, err := skillsCheckWorkerSpecFromEnvironment()
	if err == nil {
		_ = defaultSkillsCheckWorker().run(context.Background(), spec)
	}
	return true
}

func defaultSkillsCheckScheduler() skillsCheckScheduler {
	return skillsCheckScheduler{
		getenv:          os.Getenv,
		progressEnabled: shared.ProgressEnabled,
		loadConfig:      config.Load,
		statePaths:      defaultSkillsCheckStatePaths,
		loadCache:       defaultLoadSkillsCheckCache,
		storeCache:      defaultStoreSkillsCheckCache,
		claimWorker:     defaultClaimSkillsCheckWorker,
		releaseWorker:   defaultReleaseSkillsCheckWorker,
		startWorker:     defaultStartSkillsCheckWorker,
		now:             time.Now,
		stderr:          os.Stderr,
	}
}

func defaultSkillsCheckWorker() skillsCheckWorker {
	return skillsCheckWorker{
		runCheck:      defaultRunSkillsCheckCommand,
		storeCache:    defaultStoreSkillsCheckCache,
		releaseWorker: defaultReleaseSkillsCheckWorker,
		now:           time.Now,
		timeout:       skillsCheckTimeout,
	}
}

func (s skillsCheckScheduler) run() {
	if !skillsAutoCheckEnabled(strings.TrimSpace(s.getenv(skillsAutoCheckEnvVar))) {
		return
	}
	if s.getenv("CI") != "" || !s.progressEnabled() {
		return
	}

	cfg, err := s.loadConfig()
	if err != nil || cfg == nil {
		return
	}
	paths, err := s.statePaths()
	if err != nil {
		return
	}

	cache, err := s.loadCache(paths.cache)
	if errors.Is(err, os.ErrNotExist) {
		cache.CheckedAt = cfg.SkillsCheckedAt
	} else if err != nil {
		// A malformed or unreadable cache must not turn every command into a
		// worker launch attempt.
		return
	}

	if cache.UpdateAvailable {
		cache.UpdateAvailable = false
		if err := s.storeCache(paths.cache, cache); err != nil {
			return
		}
		fmt.Fprintln(s.stderr, skillsUpdateMessageFmt)
	}

	now := s.now().UTC()
	if !shouldRunSkillsCheck(now, cache.CheckedAt) {
		return
	}

	token, claimed, err := s.claimWorker(paths.lock, now)
	if err != nil || !claimed {
		return
	}

	spec := skillsCheckWorkerSpec{
		cachePath: paths.cache,
		lockPath:  paths.lock,
		token:     token,
	}
	if err := s.startWorker(spec); err != nil {
		_ = s.releaseWorker(paths.lock, token)
	}
}

func (w skillsCheckWorker) run(ctx context.Context, spec skillsCheckWorkerSpec) error {
	if err := validateSkillsCheckWorkerSpec(spec); err != nil {
		return err
	}

	checkCtx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	now := w.now().UTC()
	output, runErr := w.runCheck(checkCtx)
	if errors.Is(runErr, context.Canceled) ||
		errors.Is(runErr, context.DeadlineExceeded) ||
		errors.Is(runErr, errSkillsCheckUnavailable) {
		// Keep the lease as a short retry cooldown. It becomes reclaimable if
		// the worker is canceled, crashes, or cannot find the checker.
		return runErr
	}

	cache := skillsCheckCache{
		CheckedAt:       now.Format(skillsCheckedAtLayout),
		UpdateAvailable: runErr == nil && skillsOutputHasUpdates(output),
	}
	if err := w.storeCache(spec.cachePath, cache); err != nil {
		// Preserve the lease on cache failure to avoid a worker storm.
		return err
	}
	if err := w.releaseWorker(spec.lockPath, spec.token); err != nil {
		return err
	}
	return runErr
}

func skillsCheckWorkerSpecFromEnvironment() (skillsCheckWorkerSpec, error) {
	spec := skillsCheckWorkerSpec{
		cachePath: strings.TrimSpace(os.Getenv(skillsWorkerCacheEnvVar)),
		lockPath:  strings.TrimSpace(os.Getenv(skillsWorkerLockEnvVar)),
		token:     strings.TrimSpace(os.Getenv(skillsWorkerTokenEnvVar)),
	}
	return spec, validateSkillsCheckWorkerSpec(spec)
}

func validateSkillsCheckWorkerSpec(spec skillsCheckWorkerSpec) error {
	if spec.cachePath == "" || !filepath.IsAbs(spec.cachePath) {
		return fmt.Errorf("invalid skills check cache path")
	}
	if spec.lockPath == "" || !filepath.IsAbs(spec.lockPath) {
		return fmt.Errorf("invalid skills check lock path")
	}
	if spec.token == "" {
		return fmt.Errorf("invalid skills check worker token")
	}
	return nil
}

func defaultSkillsCheckStatePaths() (skillsCheckStatePaths, error) {
	configPath, err := config.Path()
	if err != nil {
		return skillsCheckStatePaths{}, err
	}
	dir := filepath.Dir(configPath)
	return skillsCheckStatePaths{
		cache: filepath.Join(dir, skillsCheckCacheFilename),
		lock:  filepath.Join(dir, skillsCheckLockFilename),
	}, nil
}

func defaultLoadSkillsCheckCache(path string) (skillsCheckCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return skillsCheckCache{}, err
	}

	var cache skillsCheckCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return skillsCheckCache{}, err
	}
	return cache, nil
}

func defaultStoreSkillsCheckCache(path string, cache skillsCheckCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = shared.WriteFileNoSymlinkOverwrite(
		path,
		bytes.NewReader(data),
		0o600,
		".skills-check-*.tmp",
		".skills-check-*.bak",
	)
	return err
}

func defaultClaimSkillsCheckWorker(path string, now time.Time) (string, bool, error) {
	skillsCheckClaimMu.Lock()
	defer skillsCheckClaimMu.Unlock()

	guard, acquired, err := acquireSkillsCheckClaimGuard(path+skillsCheckClaimSuffix, false)
	if err != nil || !acquired {
		return "", false, err
	}

	token, claimed, claimErr := claimSkillsCheckWorkerUnderGuard(path, now)
	guardErr := releaseSkillsCheckClaimGuardFn(guard)
	if claimErr != nil {
		return "", false, claimErr
	}
	if claimed {
		return token, true, nil
	}
	if guardErr != nil {
		return "", false, guardErr
	}
	return "", false, nil
}

func claimSkillsCheckWorkerUnderGuard(path string, now time.Time) (string, bool, error) {
	token, err := newSkillsCheckWorkerToken()
	if err != nil {
		return "", false, err
	}

	for attempt := 0; attempt < 2; attempt++ {
		if err := createSkillsCheckLock(path, token, now); err == nil {
			return token, true, nil
		} else if !errors.Is(err, os.ErrExist) {
			return "", false, err
		}

		reclaimed, err := quarantineStaleSkillsCheckLock(path, token, now)
		if err != nil || !reclaimed {
			return "", false, err
		}
	}

	return "", false, nil
}

func quarantineStaleSkillsCheckLock(path, token string, now time.Time) (bool, error) {
	stale, err := skillsCheckLockIsStale(path, now)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil || !stale {
		return false, err
	}

	quarantinePath := path + ".stale-" + token
	if err := os.Rename(path, quarantinePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}
	if err := os.Remove(quarantinePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	return true, nil
}

func createSkillsCheckLock(path, token string, now time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	removeOnFailure := true
	defer func() {
		_ = file.Close()
		if removeOnFailure {
			_ = os.Remove(path)
		}
	}()

	lock := skillsCheckLock{
		Token:     token,
		StartedAt: now.UTC().Format(skillsCheckedAtLayout),
	}
	if err := json.NewEncoder(file).Encode(lock); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	removeOnFailure = false
	return nil
}

func skillsCheckLockIsStale(path string, now time.Time) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}

	startedAt := info.ModTime().UTC()
	if data, readErr := os.ReadFile(path); readErr == nil {
		var lock skillsCheckLock
		if json.Unmarshal(data, &lock) == nil {
			if parsed, parseErr := time.Parse(skillsCheckedAtLayout, lock.StartedAt); parseErr == nil {
				startedAt = parsed.UTC()
			}
		}
	}
	return now.UTC().Sub(startedAt) >= skillsWorkerRetryDelay, nil
}

func defaultReleaseSkillsCheckWorker(path, token string) error {
	skillsCheckClaimMu.Lock()
	defer skillsCheckClaimMu.Unlock()

	guard, acquired, err := acquireSkillsCheckClaimGuard(path+skillsCheckClaimSuffix, true)
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}

	releaseErr := releaseSkillsCheckWorkerUnderGuard(path, token)
	guardErr := releaseSkillsCheckClaimGuard(guard)
	if releaseErr != nil {
		return releaseErr
	}
	return guardErr
}

func releaseSkillsCheckWorkerUnderGuard(path, token string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var lock skillsCheckLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return err
	}
	if lock.Token != token {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func acquireSkillsCheckClaimGuard(path string, wait bool) (*os.File, bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, false, err
	}

	created, err := shared.OpenNewFileNoFollow(path, 0o600)
	if err == nil {
		err = created.Close()
	} else if errors.Is(err, os.ErrExist) {
		err = nil
	}
	if err != nil {
		return nil, false, err
	}

	file, err := openSkillsCheckClaimGuard(path)
	if err != nil {
		return nil, false, err
	}
	acquired, err := lockSkillsCheckClaimFile(file, wait)
	if err != nil || !acquired {
		_ = file.Close()
		return nil, false, err
	}
	return file, true, nil
}

func openSkillsCheckClaimGuard(path string) (*os.File, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, fmt.Errorf("skills check claim guard must be a regular file")
	}

	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	closeOnFailure := true
	defer func() {
		if closeOnFailure {
			_ = file.Close()
		}
	}()

	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	after, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, after) || !os.SameFile(opened, after) {
		return nil, fmt.Errorf("skills check claim guard changed while opening")
	}

	closeOnFailure = false
	return file, nil
}

func releaseSkillsCheckClaimGuard(file *os.File) error {
	unlockErr := unlockSkillsCheckClaimFile(file)
	closeErr := file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

func newSkillsCheckWorkerToken() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func shouldRunSkillsCheck(now time.Time, lastCheckedAt string) bool {
	lastCheckedAt = strings.TrimSpace(lastCheckedAt)
	if lastCheckedAt == "" {
		return true
	}

	lastChecked, err := time.Parse(skillsCheckedAtLayout, lastCheckedAt)
	if err != nil {
		return true
	}
	return now.Sub(lastChecked.UTC()) >= skillsCheckInterval
}

func skillsAutoCheckEnabled(value string) bool {
	if value == "" {
		return true
	}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return true
	}
}

func skillsOutputHasUpdates(output string) bool {
	normalized := strings.ToLower(strings.TrimSpace(output))
	if normalized == "" {
		return false
	}

	switch {
	case strings.Contains(normalized, "all skills are up to date"):
		return false
	case strings.Contains(normalized, "no updates available"), strings.Contains(normalized, "no update available"):
		return false
	case strings.Contains(normalized, "update available"):
		return true
	case strings.Contains(normalized, "updates available"):
		return true
	default:
		return false
	}
}

func defaultRunSkillsCheckCommand(ctx context.Context) (string, error) {
	skillsPath, err := lookupSkillsCheckCLI("skills")
	if err == nil && !shouldSkipProjectLocalSkillsBinary(skillsPath) {
		return runSkillsCheckProcess(ctx, skillsPath, []string{"check"}, skillsCheckHelperEnvironment(os.Environ()))
	}

	npxPath, err := lookupExecutable("npx")
	if err != nil {
		return "", errSkillsCheckUnavailable
	}

	output, runErr := runSkillsCheckProcess(
		ctx,
		npxPath,
		[]string{"--offline", "--yes", skillsInstallerPackage, "check"},
		append(skillsCheckHelperEnvironment(os.Environ()), "npm_config_offline=true"),
	)
	if runErr != nil && isUnavailableSkillsCheckOutput(output) {
		return output, errSkillsCheckUnavailable
	}
	return output, runErr
}

func runSkillsCheckProcess(ctx context.Context, name string, args []string, env []string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = skillsCheckWorkingDirectory()
	if env != nil {
		cmd.Env = env
	}
	cmd.WaitDelay = skillsCheckPipeWaitDelay

	var combined cappedSkillsCheckOutput
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	err := cmd.Run()
	if errors.Is(err, exec.ErrWaitDelay) && cmd.ProcessState != nil && cmd.ProcessState.Success() {
		err = nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		err = ctxErr
	}

	return combined.String(), err
}

func (output *cappedSkillsCheckOutput) Write(p []byte) (int, error) {
	remaining := maxSkillsCheckOutputBytes - len(output.data)
	if remaining > 0 {
		if len(p) < remaining {
			remaining = len(p)
		}
		output.data = append(output.data, p[:remaining]...)
	}
	return len(p), nil
}

func (output *cappedSkillsCheckOutput) String() string {
	return string(output.data)
}

func isUnavailableSkillsCheckOutput(output string) bool {
	normalized := strings.ToLower(output)
	return strings.Contains(normalized, "enotcached") ||
		strings.Contains(normalized, "could not determine executable to run") ||
		strings.Contains(normalized, "command not found")
}

func skillsCheckWorkingDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(homeDir) != "" {
		return homeDir
	}
	return os.TempDir()
}

func shouldSkipProjectLocalSkillsBinary(binaryPath string) bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}

	repoRoot, ok := detectRepoRoot(cwd)
	if !ok {
		return false
	}

	resolvedBinary := resolvePathForComparison(binaryPath)
	resolvedRoot := resolvePathForComparison(repoRoot)
	return isPathWithin(resolvedBinary, resolvedRoot)
}

func detectRepoRoot(start string) (string, bool) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func resolvePathForComparison(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		return resolved
	}
	return absPath
}

func isPathWithin(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
