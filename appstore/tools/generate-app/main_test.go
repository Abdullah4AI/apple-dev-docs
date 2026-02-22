package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/wallgen"
)

func TestMain(m *testing.M) {
	lookupAppStoreArtworkURLs = func(ids []string) (map[string]string, error) {
		return map[string]string{}, nil
	}
	os.Exit(m.Run())
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func withWorkingDirectory(t *testing.T, path string) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get current working directory: %v", err)
	}
	if err := os.Chdir(path); err != nil {
		t.Fatalf("chdir to %s: %v", path, err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(cwd); chdirErr != nil {
			t.Fatalf("restore working directory: %v", chdirErr)
		}
	})
}

func readJSONEntries(t *testing.T, path string) []wallgen.WallEntry {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read JSON file %s: %v", path, err)
	}
	var entries []wallgen.WallEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("unmarshal JSON file %s: %v", path, err)
	}
	return entries
}

func TestRunAddsAppAndSyncsReadme(t *testing.T) {
	tmpRepo := t.TempDir()
	withWorkingDirectory(t, tmpRepo)

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "CodexMonitor",
    "link": "https://github.com/Dimillian/CodexMonitor",
    "creator": "Dimillian",
    "platform": ["macOS", "iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old wall content.
<!-- WALL-OF-APPS:END -->
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"--app", "Dandelion",
		"--link", "https://apps.apple.com/us/app/dandelion-write-and-let-go/id6757363901",
		"--creator", "joeycast",
		"--platform", "iOS, macOS",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v (stderr: %s)", err, stderr.String())
	}

	entries := readJSONEntries(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"))
	if len(entries) != 2 {
		t.Fatalf("expected 2 JSON entries, got %d", len(entries))
	}
	added := entries[1]
	if added.App != "Dandelion" || added.Creator != "joeycast" {
		t.Fatalf("unexpected added entry: %+v", added)
	}
	if len(added.Platform) != 2 || added.Platform[0] != "iOS" || added.Platform[1] != "macOS" {
		t.Fatalf("unexpected platform values in added entry: %+v", added.Platform)
	}

	readmeBytes, err := os.ReadFile(filepath.Join(tmpRepo, "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	readme := string(readmeBytes)
	if !strings.Contains(readme, "**2 apps ship with asc.**") {
		t.Fatalf("expected app count teaser in README, got:\n%s", readme)
	}
	if !strings.Contains(readme, "https://asccli.sh/#wall-of-apps") {
		t.Fatalf("expected website link in README, got:\n%s", readme)
	}
	if !strings.Contains(stdout.String(), "Added app entry in") {
		t.Fatalf("expected add confirmation in stdout, got: %s", stdout.String())
	}
}

func TestRunSortsJSONEntriesAlphabeticallyByApp(t *testing.T) {
	tmpRepo := t.TempDir()
	withWorkingDirectory(t, tmpRepo)

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Zulu",
    "link": "https://apps.apple.com/app/id1000000001",
    "creator": "creator-zulu",
    "platform": ["iOS"]
  },
  {
    "app": "alpha",
    "link": "https://apps.apple.com/app/id1000000002",
    "creator": "creator-alpha",
    "platform": ["iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old wall content.
<!-- WALL-OF-APPS:END -->
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"--app", "Beta",
		"--link", "https://apps.apple.com/app/id1000000003",
		"--creator", "creator-beta",
		"--platform", "iOS",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v (stderr: %s)", err, stderr.String())
	}

	entries := readJSONEntries(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"))
	if len(entries) != 3 {
		t.Fatalf("expected 3 JSON entries, got %d", len(entries))
	}

	orderedApps := []string{entries[0].App, entries[1].App, entries[2].App}
	expectedApps := []string{"alpha", "Beta", "Zulu"}
	if strings.Join(orderedApps, ",") != strings.Join(expectedApps, ",") {
		t.Fatalf("expected JSON apps sorted alphabetically, got %v", orderedApps)
	}
}

func TestRunAddsIconFromAppStoreLookup(t *testing.T) {
	tmpRepo := t.TempDir()
	withWorkingDirectory(t, tmpRepo)

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "CodexMonitor",
    "link": "https://github.com/Dimillian/CodexMonitor",
    "creator": "Dimillian",
    "platform": ["macOS", "iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old wall content.
<!-- WALL-OF-APPS:END -->
`)

	previousLookup := lookupAppStoreArtworkURLs
	t.Cleanup(func() { lookupAppStoreArtworkURLs = previousLookup })
	lookupAppStoreArtworkURLs = func(ids []string) (map[string]string, error) {
		return map[string]string{
			"1000000003": "https://example.com/beta-icon.png",
		}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"--app", "Beta",
		"--link", "https://apps.apple.com/app/id1000000003",
		"--creator", "creator-beta",
		"--platform", "iOS",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v (stderr: %s)", err, stderr.String())
	}

	entries := readJSONEntries(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"))
	var betaEntry wallgen.WallEntry
	found := false
	for _, entry := range entries {
		if entry.App == "Beta" {
			betaEntry = entry
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Beta entry in source JSON, got %+v", entries)
	}
	if betaEntry.Icon != "https://example.com/beta-icon.png" {
		t.Fatalf("expected icon URL on Beta entry, got %q", betaEntry.Icon)
	}

	readmeBytes, err := os.ReadFile(filepath.Join(tmpRepo, "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	readme := string(readmeBytes)
	if !strings.Contains(readme, "**2 apps ship with asc.**") {
		t.Fatalf("expected app count teaser in README, got:\n%s", readme)
	}
}

func TestRunUpdatesExistingEntryByApp(t *testing.T) {
	tmpRepo := t.TempDir()
	withWorkingDirectory(t, tmpRepo)

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Dandelion",
    "link": "https://apps.apple.com/us/app/dandelion-write-and-let-go/id6757363901",
    "creator": "old-creator",
    "platform": ["iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old wall content.
<!-- WALL-OF-APPS:END -->
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"--app", "Dandelion",
		"--link", "https://apps.apple.com/us/app/dandelion-write-and-let-go/id6757363901",
		"--creator", "joeycast",
		"--platform", "iOS, macOS",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v (stderr: %s)", err, stderr.String())
	}

	entries := readJSONEntries(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"))
	if len(entries) != 1 {
		t.Fatalf("expected single JSON entry after update, got %d", len(entries))
	}
	if entries[0].Creator != "joeycast" {
		t.Fatalf("expected updated creator, got %q", entries[0].Creator)
	}
	if len(entries[0].Platform) != 2 {
		t.Fatalf("expected updated platforms, got %+v", entries[0].Platform)
	}
	if !strings.Contains(stdout.String(), "Updated app entry in") {
		t.Fatalf("expected update confirmation in stdout, got: %s", stdout.String())
	}
}

func TestRunUpdatePreservesExistingIconWhenLookupFails(t *testing.T) {
	tmpRepo := t.TempDir()
	withWorkingDirectory(t, tmpRepo)

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Dandelion",
    "link": "https://apps.apple.com/us/app/dandelion-write-and-let-go/id6757363901",
    "creator": "old-creator",
    "icon": "https://example.com/existing-icon.png",
    "platform": ["iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old wall content.
<!-- WALL-OF-APPS:END -->
`)

	previousLookup := lookupAppStoreArtworkURLs
	t.Cleanup(func() { lookupAppStoreArtworkURLs = previousLookup })
	lookupAppStoreArtworkURLs = func(ids []string) (map[string]string, error) {
		return nil, fmt.Errorf("temporary app store outage")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"--app", "Dandelion",
		"--link", "https://apps.apple.com/us/app/dandelion-write-and-let-go/id6757363901",
		"--creator", "joeycast",
		"--platform", "iOS, macOS",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v (stderr: %s)", err, stderr.String())
	}

	entries := readJSONEntries(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"))
	if len(entries) != 1 {
		t.Fatalf("expected single JSON entry after update, got %d", len(entries))
	}
	if entries[0].Icon != "https://example.com/existing-icon.png" {
		t.Fatalf("expected existing icon to be preserved, got %q", entries[0].Icon)
	}
	if !strings.Contains(stderr.String(), "Warning: unable to refresh App Store icons:") {
		t.Fatalf("expected icon refresh warning in stderr, got: %s", stderr.String())
	}
}

func TestRunFailsWhenPlatformMissing(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"--app", "Dandelion",
		"--link", "https://apps.apple.com/us/app/dandelion-write-and-let-go/id6757363901",
		"--creator", "joeycast",
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error when platform is missing")
	}
	if !strings.Contains(err.Error(), "--platform is required") {
		t.Fatalf("expected missing platform error, got %v", err)
	}
}

func TestRunRestoresJSONWhenReadmeSyncFails(t *testing.T) {
	tmpRepo := t.TempDir()
	withWorkingDirectory(t, tmpRepo)

	originalJSON := `[
  {
    "app": "CodexMonitor",
    "link": "https://github.com/Dimillian/CodexMonitor",
    "creator": "Dimillian",
    "platform": ["macOS", "iOS"]
  }
]`
	originalReadme := "# Demo\nNo wall markers.\n"

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), originalJSON)
	writeFile(t, filepath.Join(tmpRepo, "README.md"), originalReadme)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"--app", "Dandelion",
		"--link", "https://apps.apple.com/us/app/dandelion-write-and-let-go/id6757363901",
		"--creator", "joeycast",
		"--platform", "iOS, macOS",
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected run to fail when README markers are missing")
	}
	if !strings.Contains(err.Error(), "README markers not found") {
		t.Fatalf("expected README marker error, got %v", err)
	}

	rawJSON, err := os.ReadFile(filepath.Join(tmpRepo, "docs", "wall-of-apps.json"))
	if err != nil {
		t.Fatalf("read JSON: %v", err)
	}
	if string(rawJSON) != originalJSON {
		t.Fatalf("expected JSON to be restored after failure, got:\n%s", string(rawJSON))
	}

	readmeBytes, err := os.ReadFile(filepath.Join(tmpRepo, "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	if string(readmeBytes) != originalReadme {
		t.Fatalf("expected README to remain unchanged, got:\n%s", string(readmeBytes))
	}
}
