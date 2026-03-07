package install

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/config"
)

const (
	skillsAutoCheckEnvVar  = "ASC_SKILLS_AUTO_CHECK"
	skillsCheckInterval    = 24 * time.Hour
	skillsCheckTimeout     = 8 * time.Second
	skillsCheckedAtLayout  = time.RFC3339
	skillsUpdateMessageFmt = "skills updates may be available. Run 'npx skills update' to refresh installed skills."
)

var (
	loadConfigForSkillsCheck       = config.Load
	persistSkillsCheckedAtForCheck = defaultPersistSkillsCheckedAt
	nowForSkillsCheck              = time.Now
	runSkillsCheckCommand          = defaultRunSkillsCheckCommand
	progressEnabledForCheck        = shared.ProgressEnabled
	lookupSkillsCheckCLI           = exec.LookPath
	errSkillsCheckUnavailable      = errors.New("skills check command unavailable")
)

// MaybeCheckForSkillUpdates checks for skill updates once per interval and prints
// a non-blocking stderr notice when updates appear available.
func MaybeCheckForSkillUpdates(ctx context.Context) {
	if !skillsAutoCheckEnabled(strings.TrimSpace(os.Getenv(skillsAutoCheckEnvVar))) {
		return
	}
	if os.Getenv("CI") != "" {
		return
	}
	if !progressEnabledForCheck() {
		return
	}

	cfg, err := loadConfigForSkillsCheck()
	if err != nil {
		// Keep command execution unaffected when config is absent or unreadable.
		return
	}
	if cfg == nil {
		return
	}

	now := nowForSkillsCheck().UTC()
	if !shouldRunSkillsCheck(now, cfg.SkillsCheckedAt) {
		return
	}

	checkCtx, cancel := context.WithTimeout(ctx, skillsCheckTimeout)
	defer cancel()

	output, runErr := runSkillsCheckCommand(checkCtx)
	if runErr != nil {
		// Avoid suppressing future checks when the command never actually ran due
		// to cancellation or timeout in the parent context.
		if !errors.Is(runErr, context.Canceled) &&
			!errors.Is(runErr, context.DeadlineExceeded) &&
			!errors.Is(runErr, errSkillsCheckUnavailable) {
			_ = persistSkillsCheckedAtForCheck(now.Format(skillsCheckedAtLayout))
		}
		return
	}

	_ = persistSkillsCheckedAtForCheck(now.Format(skillsCheckedAtLayout))
	if !skillsOutputHasUpdates(output) {
		return
	}

	fmt.Fprintln(os.Stderr, skillsUpdateMessageFmt)
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
		cmd := exec.CommandContext(ctx, skillsPath, "check")
		// Avoid resolving project-local node_modules in the current repository.
		cmd.Dir = skillsCheckWorkingDirectory()
		var combined bytes.Buffer
		cmd.Stdout = &combined
		cmd.Stderr = &combined

		if err := cmd.Run(); err != nil {
			return combined.String(), err
		}
		return combined.String(), nil
	}

	npxPath, err := lookupNpx("npx")
	if err != nil {
		return "", errSkillsCheckUnavailable
	}

	// Fall back to the install-skills execution path while forcing offline mode.
	cmd := exec.CommandContext(ctx, npxPath, "--offline", "--yes", "skills", "check")
	// Avoid resolving project-local node_modules in the current repository.
	cmd.Dir = skillsCheckWorkingDirectory()
	// Avoid contacting npm registries during passive background checks.
	cmd.Env = append(os.Environ(), "npm_config_offline=true")
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	if err := cmd.Run(); err != nil {
		if isUnavailableSkillsCheckOutput(combined.String()) {
			return combined.String(), errSkillsCheckUnavailable
		}
		return combined.String(), err
	}
	return combined.String(), nil
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

func defaultPersistSkillsCheckedAt(timestamp string) error {
	path, err := config.Path()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	order, doc, err := decodeConfigObjectPreservingOrder(data)
	if err != nil {
		return err
	}
	if doc == nil {
		doc = map[string]json.RawMessage{}
	}

	encoded, err := json.Marshal(strings.TrimSpace(timestamp))
	if err != nil {
		return err
	}
	doc["skills_checked_at"] = encoded
	if !containsKey(order, "skills_checked_at") {
		order = append(order, "skills_checked_at")
	}

	updated, err := marshalConfigObjectPreservingOrder(order, doc)
	if err != nil {
		return err
	}

	return os.WriteFile(path, updated, 0o600)
}

func decodeConfigObjectPreservingOrder(data []byte) ([]string, map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		return []string{}, map[string]json.RawMessage{}, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	firstToken, err := decoder.Token()
	if err != nil {
		return nil, nil, err
	}

	delim, ok := firstToken.(json.Delim)
	if !ok || delim != '{' {
		return nil, nil, fmt.Errorf("config must be a JSON object")
	}

	order := make([]string, 0)
	fields := make(map[string]json.RawMessage)
	seen := make(map[string]struct{})
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, nil, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, nil, fmt.Errorf("config key must be a string")
		}

		var rawValue json.RawMessage
		if err := decoder.Decode(&rawValue); err != nil {
			return nil, nil, err
		}

		if _, alreadySeen := seen[key]; !alreadySeen {
			order = append(order, key)
			seen[key] = struct{}{}
		}
		fields[key] = append(json.RawMessage(nil), rawValue...)
	}

	endToken, err := decoder.Token()
	if err != nil {
		return nil, nil, err
	}
	endDelim, ok := endToken.(json.Delim)
	if !ok || endDelim != '}' {
		return nil, nil, fmt.Errorf("config must end with object delimiter")
	}

	return order, fields, nil
}

func marshalConfigObjectPreservingOrder(order []string, fields map[string]json.RawMessage) ([]byte, error) {
	if fields == nil {
		fields = map[string]json.RawMessage{}
	}

	keys := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, key := range order {
		if _, ok := fields[key]; ok {
			keys = append(keys, key)
			seen[key] = struct{}{}
		}
	}

	extraKeys := make([]string, 0)
	for key := range fields {
		if _, ok := seen[key]; !ok {
			extraKeys = append(extraKeys, key)
		}
	}
	sort.Strings(extraKeys)
	keys = append(keys, extraKeys...)

	var buffer bytes.Buffer
	buffer.WriteString("{")
	if len(keys) > 0 {
		buffer.WriteString("\n")
	}

	for index, key := range keys {
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}

		buffer.WriteString("  ")
		buffer.Write(keyJSON)
		buffer.WriteString(": ")

		value := fields[key]
		if len(value) == 0 {
			value = []byte("null")
		}
		buffer.Write(value)

		if index < len(keys)-1 {
			buffer.WriteString(",")
		}
		buffer.WriteString("\n")
	}

	buffer.WriteString("}")
	return buffer.Bytes(), nil
}

func containsKey(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
