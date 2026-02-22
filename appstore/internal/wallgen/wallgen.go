package wallgen

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	startMarker         = "<!-- WALL-OF-APPS:START -->"
	endMarker           = "<!-- WALL-OF-APPS:END -->"
	wallPullRequestsURL = "https://github.com/Abdullah4AI/apple-developer-toolkit/appstore/pulls"
	wallWebsiteURL      = "https://asccli.sh/#wall-of-apps"
)

var platformAliases = map[string]string{
	"ios":       "IOS",
	"macos":     "MAC_OS",
	"mac_os":    "MAC_OS",
	"watchos":   "WATCH_OS",
	"watch_os":  "WATCH_OS",
	"tvos":      "TV_OS",
	"tv_os":     "TV_OS",
	"visionos":  "VISION_OS",
	"vision_os": "VISION_OS",
}

// WallEntry defines a single docs/wall-of-apps.json entry.
type WallEntry struct {
	App      string   `json:"app"`
	Link     string   `json:"link"`
	Creator  string   `json:"creator"`
	Icon     string   `json:"icon,omitempty"`
	Platform []string `json:"platform"`
}

// Result contains generated output paths.
type Result struct {
	ReadmePath string
}

// Generate reads docs/wall-of-apps.json and updates the README wall snippet.
func Generate(repoRoot string) (Result, error) {
	sourcePath := filepath.Join(repoRoot, "docs", "wall-of-apps.json")
	readmePath := filepath.Join(repoRoot, "README.md")

	entries, err := readEntries(sourcePath)
	if err != nil {
		return Result{}, err
	}
	snippet := buildSnippet(entries)

	if err := syncReadme(snippet, readmePath); err != nil {
		return Result{}, err
	}

	return Result{ReadmePath: readmePath}, nil
}

func readEntries(sourcePath string) ([]WallEntry, error) {
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("missing source file: %s", sourcePath)
	}
	if strings.TrimSpace(string(raw)) == "" {
		return nil, fmt.Errorf("source file is empty: %s", sourcePath)
	}

	var parsed []WallEntry
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", sourcePath, err)
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("source file has no entries: %s", sourcePath)
	}

	normalized := make([]WallEntry, 0, len(parsed))
	for idx, entry := range parsed {
		item, err := normalizeEntry(entry, idx+1)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, item)
	}

	SortEntriesByApp(normalized)

	return normalized, nil
}

// SortEntriesByApp sorts entries by app name (case-insensitive), then link.
func SortEntriesByApp(entries []WallEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		leftApp := strings.ToLower(strings.TrimSpace(entries[i].App))
		rightApp := strings.ToLower(strings.TrimSpace(entries[j].App))
		if leftApp != rightApp {
			return leftApp < rightApp
		}
		return strings.ToLower(strings.TrimSpace(entries[i].Link)) < strings.ToLower(strings.TrimSpace(entries[j].Link))
	})
}

func normalizeEntry(entry WallEntry, index int) (WallEntry, error) {
	entry.App = strings.TrimSpace(entry.App)
	entry.Link = strings.TrimSpace(entry.Link)
	entry.Creator = strings.TrimSpace(entry.Creator)
	entry.Icon = strings.TrimSpace(entry.Icon)
	if entry.App == "" {
		return WallEntry{}, fmt.Errorf("entry #%d: 'app' is required", index)
	}
	if entry.Link == "" {
		return WallEntry{}, fmt.Errorf("entry #%d: 'link' is required", index)
	}
	if entry.Creator == "" {
		return WallEntry{}, fmt.Errorf("entry #%d: 'creator' is required", index)
	}
	parsedURL, err := url.Parse(entry.Link)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return WallEntry{}, fmt.Errorf("entry #%d: 'link' must be a valid http/https URL", index)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return WallEntry{}, fmt.Errorf("entry #%d: 'link' must be a valid http/https URL", index)
	}
	if entry.Icon != "" {
		parsedIconURL, iconErr := url.Parse(entry.Icon)
		if iconErr != nil || parsedIconURL.Scheme == "" || parsedIconURL.Host == "" {
			return WallEntry{}, fmt.Errorf("entry #%d: 'icon' must be a valid http/https URL", index)
		}
		if parsedIconURL.Scheme != "http" && parsedIconURL.Scheme != "https" {
			return WallEntry{}, fmt.Errorf("entry #%d: 'icon' must be a valid http/https URL", index)
		}
	}
	if len(entry.Platform) == 0 {
		return WallEntry{}, fmt.Errorf("entry #%d: 'platform' must be a non-empty array", index)
	}

	platforms := make([]string, 0, len(entry.Platform))
	for _, value := range entry.Platform {
		token := strings.TrimSpace(value)
		if token == "" {
			return WallEntry{}, fmt.Errorf("entry #%d: 'platform' entries must be non-empty strings", index)
		}
		normalized := normalizePlatform(token)
		if !containsFold(platforms, normalized) {
			platforms = append(platforms, normalized)
		}
	}
	entry.Platform = platforms
	return entry, nil
}

func normalizePlatform(value string) string {
	key := strings.ToLower(value)
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ReplaceAll(key, " ", "")
	if normalized, ok := platformAliases[key]; ok {
		return normalized
	}
	return value
}

func containsFold(values []string, needle string) bool {
	for _, value := range values {
		if strings.EqualFold(value, needle) {
			return true
		}
	}
	return false
}

func buildSnippet(entries []WallEntry) string {
	lines := []string{
		"## Wall of Apps",
		"",
		fmt.Sprintf("**%d apps ship with asc.** [See the Wall of Apps →](%s)", len(entries), wallWebsiteURL),
		"",
		fmt.Sprintf("Want to add yours? [Open a PR](%s).", wallPullRequestsURL),
	}

	return strings.Join(lines, "\n") + "\n"
}

func syncReadme(snippet string, readmePath string) error {
	contentBytes, err := os.ReadFile(readmePath)
	if err != nil {
		return fmt.Errorf("missing README file: %s", readmePath)
	}
	content := string(contentBytes)
	start := strings.Index(content, startMarker)
	end := strings.Index(content, endMarker)
	if start == -1 || end == -1 || end < start {
		return fmt.Errorf("README markers not found. Expected WALL-OF-APPS markers in README.md")
	}

	before := content[:start]
	after := content[end+len(endMarker):]
	updated := before + startMarker + "\n" + snippet + endMarker + after

	if err := os.WriteFile(readmePath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write README: %w", err)
	}
	return nil
}
