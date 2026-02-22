package shared

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestParseStringsContent(t *testing.T) {
	input := `
// Comment
"description" = "Hello\nWorld";
/* block comment */
"keywords" = "one, two";
`
	values, err := parseStringsContent(input)
	if err != nil {
		t.Fatalf("parseStringsContent() error: %v", err)
	}
	if values["description"] != "Hello\nWorld" {
		t.Fatalf("expected description to be parsed, got %q", values["description"])
	}
	if values["keywords"] != "one, two" {
		t.Fatalf("expected keywords to be parsed, got %q", values["keywords"])
	}
}

func TestWriteStringsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "en-US.strings")
	values := map[string]string{
		"description": "Hello",
		"keywords":    "one, two",
	}

	if err := writeStringsFile(path, values, []string{"description", "keywords"}); err != nil {
		t.Fatalf("writeStringsFile() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "\"description\" = \"Hello\";") {
		t.Fatalf("expected description line, got: %s", content)
	}
	if !strings.Contains(content, "\"keywords\" = \"one, two\";") {
		t.Fatalf("expected keywords line, got: %s", content)
	}
}

func TestReadLocalizationStrings_FileLocale(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "en-US.strings")
	if err := os.WriteFile(path, []byte("\"description\" = \"Hello\";\n"), 0o644); err != nil {
		t.Fatalf("write file error: %v", err)
	}

	values, err := ReadLocalizationStrings(path, nil)
	if err != nil {
		t.Fatalf("readLocalizationStrings() error: %v", err)
	}
	if values["en-US"]["description"] != "Hello" {
		t.Fatalf("expected description Hello, got %q", values["en-US"]["description"])
	}
}

func TestReadLocalizationStrings_RejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.strings")
	if err := os.WriteFile(target, []byte("\"description\" = \"Hello\";\n"), 0o644); err != nil {
		t.Fatalf("write file error: %v", err)
	}
	link := filepath.Join(dir, "en-US.strings")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	_, err := ReadLocalizationStrings(dir, nil)
	if err == nil {
		t.Fatal("expected error for symlinked strings file")
	}
}

func TestWriteVersionLocalizationStrings_Paginated(t *testing.T) {
	dir := t.TempDir()

	makePage := func(locale, next string) *asc.AppStoreVersionLocalizationsResponse {
		return &asc.AppStoreVersionLocalizationsResponse{
			Data: []asc.Resource[asc.AppStoreVersionLocalizationAttributes]{
				{
					ID: "loc-" + locale,
					Attributes: asc.AppStoreVersionLocalizationAttributes{
						Locale:      locale,
						Description: "Description " + locale,
						WhatsNew:    "Bug fixes",
					},
				},
			},
			Links: asc.Links{Next: next},
		}
	}

	firstPage := makePage("en-US", "page=2")
	response, err := asc.PaginateAll(context.Background(), firstPage, func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
		if nextURL != "page=2" {
			return nil, fmt.Errorf("unexpected next URL %q", nextURL)
		}
		return makePage("ja", ""), nil
	})
	if err != nil {
		t.Fatalf("PaginateAll() error: %v", err)
	}

	aggregated, ok := response.(*asc.AppStoreVersionLocalizationsResponse)
	if !ok {
		t.Fatalf("expected AppStoreVersionLocalizationsResponse, got %T", response)
	}
	if len(aggregated.Data) != 2 {
		t.Fatalf("expected 2 localizations, got %d", len(aggregated.Data))
	}

	files, err := WriteVersionLocalizationStrings(dir, aggregated.Data)
	if err != nil {
		t.Fatalf("writeVersionLocalizationStrings() error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	paths := map[string]string{}
	for _, file := range files {
		paths[file.Locale] = file.Path
	}
	for _, locale := range []string{"en-US", "ja"} {
		path, ok := paths[locale]
		if !ok {
			t.Fatalf("expected locale %q in results", locale)
		}
		expectedPath := filepath.Join(dir, locale+".strings")
		if path != expectedPath {
			t.Fatalf("expected path %q for %q, got %q", expectedPath, locale, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file error: %v", err)
		}
		content := string(data)
		if !strings.Contains(content, "\"description\" = \"Description "+locale+"\";") {
			t.Fatalf("expected description for %q, got %q", locale, content)
		}
		if !strings.Contains(content, "\"whatsNew\" = \"Bug fixes\";") {
			t.Fatalf("expected whatsNew for %q, got %q", locale, content)
		}
	}
}

func TestVersionLocalizationKeysReturnsDefensiveCopy(t *testing.T) {
	keysA := VersionLocalizationKeys()
	if len(keysA) == 0 {
		t.Fatal("expected non-empty localization keys")
	}

	originalFirst := keysA[0]
	keysA[0] = "mutated-key"

	keysB := VersionLocalizationKeys()
	if len(keysB) == 0 {
		t.Fatal("expected non-empty localization keys on second call")
	}
	if keysB[0] != originalFirst {
		t.Fatalf("expected defensive copy, got first key %q want %q", keysB[0], originalFirst)
	}
}

func TestMapVersionLocalizationStrings(t *testing.T) {
	attrs := asc.AppStoreVersionLocalizationAttributes{
		Locale:          "en-US",
		Description:     "Desc",
		Keywords:        "one,two",
		MarketingURL:    "https://example.com/marketing",
		PromotionalText: "Promo",
		SupportURL:      "https://example.com/support",
		WhatsNew:        "Bug fixes",
	}

	got := MapVersionLocalizationStrings(attrs)
	expected := map[string]string{
		"description":     "Desc",
		"keywords":        "one,two",
		"marketingUrl":    "https://example.com/marketing",
		"promotionalText": "Promo",
		"supportUrl":      "https://example.com/support",
		"whatsNew":        "Bug fixes",
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d keys, got %d: %#v", len(expected), len(got), got)
	}
	for key, want := range expected {
		if got[key] != want {
			t.Fatalf("expected key %q = %q, got %q", key, want, got[key])
		}
	}
}

func TestValidateVersionLocalizationKeys(t *testing.T) {
	t.Run("accepts known keys", func(t *testing.T) {
		err := ValidateVersionLocalizationKeys("en-US", map[string]string{
			"description": "Hello",
			"keywords":    "one,two",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("rejects unknown keys", func(t *testing.T) {
		err := ValidateVersionLocalizationKeys("en-US", map[string]string{
			"description": "Hello",
			"unknownKey":  "bad",
		})
		if err == nil {
			t.Fatal("expected unknown-key validation error")
		}
		if err.Error() != "unsupported keys for locale \"en-US\": unknownKey" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestValidateAppInfoLocalizationKeys(t *testing.T) {
	t.Run("accepts known keys", func(t *testing.T) {
		err := ValidateAppInfoLocalizationKeys("en-US", map[string]string{
			"name":     "My App",
			"subtitle": "Great app",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("rejects unknown keys", func(t *testing.T) {
		err := ValidateAppInfoLocalizationKeys("en-US", map[string]string{
			"name":       "My App",
			"unknownKey": "bad",
		})
		if err == nil {
			t.Fatal("expected unknown-key validation error")
		}
		if err.Error() != "unsupported keys for locale \"en-US\": unknownKey" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
