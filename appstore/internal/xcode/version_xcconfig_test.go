package xcode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestXCConfigRecursiveIncludesHandleCyclesOptionalFilesAndOrder(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "Root.xcconfig")
	sharedPath := filepath.Join(root, "Shared.xcconfig")
	if err := os.WriteFile(rootPath, []byte("#include? \"Missing\"\n#include \"Shared\"\nMARKETING_VERSION = 3.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sharedPath, []byte("#include \"Root.xcconfig\"\nMARKETING_VERSION = 2.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := collectXCConfigFiles(rootPath)
	if err != nil {
		t.Fatalf("collectXCConfigFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("files = %#v, want root and shared", files)
	}
	resolved, err := resolveXCConfigSetting(rootPath, marketingVersionSetting)
	if err != nil {
		t.Fatalf("resolveXCConfigSetting() error = %v", err)
	}
	if !resolved.found || resolved.value != "3.0.0" || resolved.path != rootPath {
		t.Fatalf("unexpected resolved value: %#v", resolved)
	}
}

func TestXCConfigEditorPreservesCommentsQuotesAndMissingFinalNewline(t *testing.T) {
	input := []byte("URL = \"https://example.com/path\" // URL comment\n/* MARKETING_VERSION = 9.9.9 */\nMARKETING_VERSION[sdk=iphoneos*] ?= 1.2.3 // keep me")
	updated, oldValues, changed, err := editXCConfig(input, marketingVersionSetting, "2.0.0")
	if err != nil {
		t.Fatalf("editXCConfig() error = %v", err)
	}
	if !changed || len(oldValues) != 1 || oldValues[0] != "1.2.3" {
		t.Fatalf("unexpected edit metadata changed=%v old=%#v", changed, oldValues)
	}
	got := string(updated)
	if !strings.Contains(got, "URL = \"https://example.com/path\" // URL comment") ||
		!strings.Contains(got, "/* MARKETING_VERSION = 9.9.9 */") ||
		!strings.HasSuffix(got, "MARKETING_VERSION[sdk=iphoneos*] = 2.0.0 // keep me") ||
		strings.HasSuffix(got, "\n") {
		t.Fatalf("lossless edit failed: %q", got)
	}
}

func TestXCConfigInheritedValueAndExactPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Values.xcconfig")
	contents := "OTHER = base\nOTHER = $(inherited)-child\nCURRENT_PROJECT_VERSION[sdk=iphoneos*] = 42\nCURRENT_PROJECT_VERSION = 42\nCURRENT_PROJECT_VERSION[sdk=macosx*] = 42\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	other, err := resolveXCConfigSetting(path, "OTHER")
	if err != nil || other.value != "base-child" {
		t.Fatalf("inherited resolution = %#v, err %v", other, err)
	}
	build, err := resolveXCConfigSetting(path, currentProjectSetting)
	if err != nil || build.value != "42" || !build.exact {
		t.Fatalf("exact resolution = %#v, err %v", build, err)
	}
}

func TestXCConfigResolverRejectsDivergentConditionalOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Values.xcconfig")
	contents := "CURRENT_PROJECT_VERSION = 42\nCURRENT_PROJECT_VERSION[sdk=iphoneos*] = 100\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := resolveXCConfigSetting(path, currentProjectSetting)
	if err == nil || !strings.Contains(err.Error(), "differing conditional") {
		t.Fatalf("expected divergent conditional error, got %v", err)
	}
}

func TestXCConfigOperatorsQuotesAndIncludeOrder(t *testing.T) {
	root := t.TempDir()
	rootPath := filepath.Join(root, "Root.xcconfig")
	childPath := filepath.Join(root, "Child.xcconfig")
	if err := os.WriteFile(rootPath, []byte("OTHER = base\n#include \"Child.xcconfig\"\nMARKETING_VERSION ?= \"1.2.3\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(childPath, []byte("OTHER += child\nOTHER ?= ignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	other, err := resolveXCConfigSetting(rootPath, "OTHER")
	if err != nil || other.value != "base child" {
		t.Fatalf("operator/include resolution = %#v, err %v", other, err)
	}
	version, err := resolveXCConfigSetting(rootPath, marketingVersionSetting)
	if err != nil || version.value != "1.2.3" {
		t.Fatalf("quoted/default resolution = %#v, err %v", version, err)
	}
}

func TestXCConfigEditorNormalizesOperatorsAndPreservesQuotes(t *testing.T) {
	for _, operator := range []string{"+=", "?="} {
		t.Run(operator, func(t *testing.T) {
			input := []byte("MARKETING_VERSION " + operator + " \"1.2.3\" // keep\n")
			updated, oldValues, changed, err := editXCConfig(input, marketingVersionSetting, "2.0.0")
			if err != nil {
				t.Fatalf("editXCConfig() error = %v", err)
			}
			if !changed || len(oldValues) != 1 || oldValues[0] != "1.2.3" {
				t.Fatalf("unexpected edit metadata changed=%v old=%#v", changed, oldValues)
			}
			if got := string(updated); got != "MARKETING_VERSION = \"2.0.0\" // keep\n" {
				t.Fatalf("operator/quote edit = %q", got)
			}
		})
	}
}

func TestXCConfigResolverRejectsConditionalOnlySetting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Conditional.xcconfig")
	contents := "CURRENT_PROJECT_VERSION[sdk=iphoneos*] = 41\nCURRENT_PROJECT_VERSION[sdk=macosx*] = 43\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := resolveXCConfigSetting(path, currentProjectSetting)
	if err == nil || !strings.Contains(err.Error(), "conditional") {
		t.Fatalf("expected conditional-only setting error, got %v", err)
	}
}

func TestXCConfigParserRejectsUnterminatedBlockComment(t *testing.T) {
	_, err := parseXCConfig([]byte("MARKETING_VERSION = 1.0\n/* never closed"))
	if err == nil {
		t.Fatal("expected unterminated-comment error")
	}
}

func TestXCConfigRequiredMissingIncludeFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Root.xcconfig")
	if err := os.WriteFile(path, []byte("#include \"Missing.xcconfig\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := collectXCConfigFiles(path)
	if err == nil {
		t.Fatal("expected required-include error")
	}
}
