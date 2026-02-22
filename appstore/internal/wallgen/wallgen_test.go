package wallgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func TestGenerateWritesReadmeSnippet(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Zulu App",
    "link": "https://example.com/zulu",
    "creator": "Zulu Creator",
    "platform": ["iOS"]
  },
  {
    "app": "Alpha App",
    "link": "https://example.com/alpha",
    "creator": "Alpha Creator",
    "platform": ["macOS", "iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo

Before.
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
After.
`)

	result, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	readmeBytes, err := os.ReadFile(result.ReadmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	readme := string(readmeBytes)
	if strings.Contains(readme, "Old content.") {
		t.Fatalf("expected README snippet to be replaced, got:\n%s", readme)
	}
	if !strings.Contains(readme, "**2 apps ship with asc.**") {
		t.Fatalf("expected app count teaser in README, got:\n%s", readme)
	}
	if !strings.Contains(readme, "https://asccli.sh/#wall-of-apps") {
		t.Fatalf("expected website link in README, got:\n%s", readme)
	}
	if !strings.Contains(readme, "Want to add yours?") {
		t.Fatalf("expected PR link in README, got:\n%s", readme)
	}
	if _, err := os.Stat(filepath.Join(tmpRepo, "docs", "generated", "app-wall.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no generated wall file, stat error: %v", err)
	}
}

func TestGenerateCountsEntriesCorrectly(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Alpha App",
    "link": "https://example.com/alpha",
    "creator": "Alpha Creator",
    "icon": "https://example.com/alpha-icon.png",
    "platform": ["iOS"]
  },
  {
    "app": "Beta App",
    "link": "https://example.com/beta",
    "creator": "Beta Creator",
    "platform": ["iOS"]
  },
  {
    "app": "Zulu App",
    "link": "https://example.com/zulu",
    "creator": "Zulu Creator",
    "platform": ["iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	result, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	readmeBytes, err := os.ReadFile(result.ReadmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	readme := string(readmeBytes)

	if !strings.Contains(readme, "**3 apps ship with asc.**") {
		t.Fatalf("expected 3-app count in README, got:\n%s", readme)
	}
}

func TestGenerateDoesNotIncludeIconGrid(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Alpha App",
    "link": "https://example.com/alpha",
    "creator": "Alpha Creator",
    "icon": "https://example.com/alpha-icon.png",
    "platform": ["iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	result, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	readmeBytes, err := os.ReadFile(result.ReadmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	readme := string(readmeBytes)

	if strings.Contains(readme, "### App Icons") {
		t.Fatalf("expected no icon grid in slim snippet, got:\n%s", readme)
	}
	if strings.Contains(readme, "### Details") {
		t.Fatalf("expected no details table in slim snippet, got:\n%s", readme)
	}
	if strings.Contains(readme, "<img src=") {
		t.Fatalf("expected no img tags in slim snippet, got:\n%s", readme)
	}
}

func TestGenerateFailsWhenCreatorMissing(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "No Creator App",
    "link": "https://example.com/no-creator",
    "platform": ["iOS"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail for missing creator")
	}
	if !strings.Contains(err.Error(), "'creator' is required") {
		t.Fatalf("expected missing creator error, got %v", err)
	}
}

func TestGenerateAcceptsCustomPlatformLabels(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Platform App",
    "link": "https://example.com/platform",
    "creator": "Platform Creator",
    "platform": ["Android", "WATCH_OS", "watchOS", "TV_OS", "tvos", "IOS", "ios", "Vision OS", "visionos"]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	_, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
}

func TestGenerateFailsWhenPlatformEntryEmpty(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Bad Platform App",
    "link": "https://example.com/bad-platform",
    "creator": "Platform Creator",
    "platform": ["iOS", "   "]
  }
]`)

	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail for empty platform value")
	}
	if !strings.Contains(err.Error(), "'platform' entries must be non-empty strings") {
		t.Fatalf("expected empty platform entry error, got %v", err)
	}
}

func TestGenerateFailsWhenReadmeMarkersMissing(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Marker App",
    "link": "https://example.com/marker",
    "creator": "Marker Creator",
    "platform": ["iOS"]
  }
]`)
	writeFile(t, filepath.Join(tmpRepo, "README.md"), "# Demo\nNo wall markers here.\n")

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when README markers are missing")
	}
	if !strings.Contains(err.Error(), "README markers not found") {
		t.Fatalf("expected README marker error, got %v", err)
	}
}

func TestGenerateFailsWhenReadmeMarkersOutOfOrder(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Marker App",
    "link": "https://example.com/marker",
    "creator": "Marker Creator",
    "platform": ["iOS"]
  }
]`)
	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:END -->
Old content.
<!-- WALL-OF-APPS:START -->
`)

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when README markers are out of order")
	}
	if !strings.Contains(err.Error(), "README markers not found") {
		t.Fatalf("expected README marker error, got %v", err)
	}
}

func TestGenerateFailsWhenSourceFileMissing(t *testing.T) {
	tmpRepo := t.TempDir()

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when source file is missing")
	}
	if !strings.Contains(err.Error(), "missing source file") {
		t.Fatalf("expected missing source error, got %v", err)
	}
}

func TestGenerateFailsWhenSourceFileEmpty(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), "   \n")

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when source file is empty")
	}
	if !strings.Contains(err.Error(), "source file is empty") {
		t.Fatalf("expected empty source error, got %v", err)
	}
}

func TestGenerateFailsWhenSourceFileInvalidJSON(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), "{invalid")

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when source JSON is invalid")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("expected invalid JSON error, got %v", err)
	}
}

func TestGenerateFailsWhenSourceFileHasNoEntries(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), "[]")

	_, err := Generate(tmpRepo)
	if err == nil {
		t.Fatal("expected generate to fail when source has no entries")
	}
	if !strings.Contains(err.Error(), "source file has no entries") {
		t.Fatalf("expected no entries error, got %v", err)
	}
}

func TestGenerateValidatesLinkURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		link string
	}{
		{name: "missing scheme", link: "example.com/app"},
		{name: "unsupported scheme", link: "ftp://example.com/app"},
		{name: "missing host", link: "https:///path-only"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmpRepo := t.TempDir()

			writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "URL App",
    "link": "`+tc.link+`",
    "creator": "URL Creator",
    "platform": ["iOS"]
  }
]`)
			writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
<!-- WALL-OF-APPS:START -->
Old content.
<!-- WALL-OF-APPS:END -->
`)

			_, err := Generate(tmpRepo)
			if err == nil {
				t.Fatal("expected generate to fail for invalid link")
			}
			if !strings.Contains(err.Error(), "'link' must be a valid http/https URL") {
				t.Fatalf("expected link validation error, got %v", err)
			}
		})
	}
}

func TestGeneratePreservesReadmeContentOutsideMarkers(t *testing.T) {
	tmpRepo := t.TempDir()

	writeFile(t, filepath.Join(tmpRepo, "docs", "wall-of-apps.json"), `[
  {
    "app": "Preserve App",
    "link": "https://example.com/preserve",
    "creator": "Preserve Creator",
    "platform": ["iOS"]
  }
]`)
	writeFile(t, filepath.Join(tmpRepo, "README.md"), `# Demo
Top section.
<!-- WALL-OF-APPS:START -->
Replace me.
<!-- WALL-OF-APPS:END -->
Bottom section.
`)

	result, err := Generate(tmpRepo)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	readmeBytes, err := os.ReadFile(result.ReadmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	readme := string(readmeBytes)

	if !strings.Contains(readme, "# Demo\nTop section.\n<!-- WALL-OF-APPS:START -->") {
		t.Fatalf("expected top section preserved, got:\n%s", readme)
	}
	if !strings.Contains(readme, "<!-- WALL-OF-APPS:END -->\nBottom section.\n") {
		t.Fatalf("expected bottom section preserved, got:\n%s", readme)
	}
}
