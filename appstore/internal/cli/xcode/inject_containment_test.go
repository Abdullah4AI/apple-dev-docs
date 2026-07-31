package xcode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestXcodeInjectReportsManifestRelativePathsForRelativeManifest(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".asc", 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(".asc", "icon.txt"), []byte("icon\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	writeXcodeInjectTestManifest(t, filepath.Join(".asc", "deployment.json"), `{
		"outputs": [
			{"type": "text", "path": "../Generated/version.txt", "contents": "1.2.3\n"},
			{"type": "copy", "source": "icon.txt", "path": "../Generated/icon.txt"}
		]
	}`)

	result, err := runXcodeInject(xcodeInjectOptions{ManifestPath: filepath.Join(".asc", "deployment.json"), Overwrite: true})
	if err != nil {
		t.Fatalf("runXcodeInject() error = %v", err)
	}
	if len(result.Outputs) != 2 {
		t.Fatalf("outputs = %#v, want 2 entries", result.Outputs)
	}
	// A relative --manifest must keep manifest-relative result paths, matching
	// the pre-rooting output contract.
	if want := filepath.Join("Generated", "version.txt"); result.Outputs[0].Path != want {
		t.Fatalf("outputs[0].path = %q, want %q", result.Outputs[0].Path, want)
	}
	if want := filepath.Join("Generated", "icon.txt"); result.Outputs[1].Path != want {
		t.Fatalf("outputs[1].path = %q, want %q", result.Outputs[1].Path, want)
	}
	if want := filepath.Join(".asc", "icon.txt"); result.Outputs[1].Source != want {
		t.Fatalf("outputs[1].source = %q, want %q", result.Outputs[1].Source, want)
	}
}

func TestXcodeInjectRejectsParentTraversingOutputPath(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "text", "path": "../escaped.txt", "contents": "attacker\n"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want traversal rejection")
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(dir), "escaped.txt")); statErr == nil {
		t.Fatal("output escaped the manifest root")
	}
}

func TestXcodeInjectRejectsAbsoluteOutputPath(t *testing.T) {
	dir := t.TempDir()
	externalPath := filepath.Join(t.TempDir(), "escaped.txt")
	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "text", "path": "`+filepath.ToSlash(externalPath)+`", "contents": "attacker\n"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want absolute path rejection")
	}
	if _, statErr := os.Stat(externalPath); statErr == nil {
		t.Fatal("output escaped the manifest root")
	}
}

func TestXcodeInjectRejectsParentTraversingCopySource(t *testing.T) {
	dir := t.TempDir()
	secretDir := t.TempDir()
	secretPath := filepath.Join(secretDir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("local secret"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	// A sibling directory of the manifest root reached with "..".
	nested := filepath.Join(dir, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "outside.txt"), []byte("root level"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifestPath := filepath.Join(nested, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "copy", "source": "../outside.txt", "path": "Generated/Copied.txt"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want copy source traversal rejection")
	}
	if _, statErr := os.Stat(filepath.Join(nested, "Generated", "Copied.txt")); statErr == nil {
		t.Fatal("copy produced an artifact from a source outside the manifest root")
	}
}

func TestXcodeInjectRejectsAbsoluteCopySource(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secretPath, []byte("local secret"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "copy", "source": "`+filepath.ToSlash(secretPath)+`", "path": "Generated/Copied.txt"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want copy source rejection")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Generated", "Copied.txt")); statErr == nil {
		t.Fatal("copy produced an artifact from an absolute source")
	}
}

func TestXcodeInjectRejectsEscapingSymlinkedOutputParent(t *testing.T) {
	dir := t.TempDir()
	external := t.TempDir()
	if err := os.Symlink(external, filepath.Join(dir, "Generated")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "text", "path": "Generated/Info.plist", "contents": "attacker\n"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("runXcodeInject() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(external, "Info.plist")); statErr == nil {
		t.Fatal("output escaped through a symlinked parent")
	}
}

func TestXcodeInjectRejectsEscapingSymlinkedCopySource(t *testing.T) {
	dir := t.TempDir()
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "secret.txt"), []byte("local secret"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Symlink(external, filepath.Join(dir, "Assets")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "copy", "source": "Assets/secret.txt", "path": "Generated/Copied.txt"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("runXcodeInject() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Generated", "Copied.txt")); statErr == nil {
		t.Fatal("copy produced an artifact from a symlinked external source")
	}
}

func TestXcodeInjectRejectsSymlinkedCopySourceFile(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secretPath, []byte("local secret"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Symlink(secretPath, filepath.Join(dir, "Contents.json")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "copy", "source": "Contents.json", "path": "Generated/Copied.json"}
		]
	}`)

	_, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath, Overwrite: true})
	if err == nil {
		t.Fatal("runXcodeInject() error = nil, want symlink rejection")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("runXcodeInject() error = %v, want symlink rejection", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "Generated", "Copied.json")); statErr == nil {
		t.Fatal("copy produced an artifact from a symlinked source file")
	}
}

func TestXcodeInjectPreservesWhitespaceInCopySourcePath(t *testing.T) {
	dir := t.TempDir()
	sourceName := " source.txt "
	if err := os.WriteFile(filepath.Join(dir, sourceName), []byte("exact source\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "copy", "source": " source.txt ", "path": "copied.txt"}
		]
	}`)

	result, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath})
	if err != nil {
		t.Fatalf("runXcodeInject() error = %v", err)
	}
	if want := filepath.Join(dir, sourceName); result.Outputs[0].Source != want {
		t.Fatalf("reported source = %q, want exact path %q", result.Outputs[0].Source, want)
	}
	got, err := os.ReadFile(filepath.Join(dir, "copied.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if want := "exact source\n"; string(got) != want {
		t.Fatalf("copied contents = %q, want %q", got, want)
	}
}

func TestXcodeInjectPreservesWhitespaceInOutputPath(t *testing.T) {
	dir := t.TempDir()
	outputName := " generated.txt "
	manifestPath := filepath.Join(dir, "deployment.json")
	writeXcodeInjectTestManifest(t, manifestPath, `{
		"outputs": [
			{"type": "text", "path": " generated.txt ", "contents": "exact output\n"}
		]
	}`)

	result, err := runXcodeInject(xcodeInjectOptions{ManifestPath: manifestPath})
	if err != nil {
		t.Fatalf("runXcodeInject() error = %v", err)
	}
	if want := filepath.Join(dir, outputName); result.Outputs[0].Path != want {
		t.Fatalf("reported output = %q, want exact path %q", result.Outputs[0].Path, want)
	}
	got, err := os.ReadFile(filepath.Join(dir, outputName))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if want := "exact output\n"; string(got) != want {
		t.Fatalf("output contents = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "generated.txt")); !os.IsNotExist(err) {
		t.Fatalf("trimmed output path exists or returned unexpected error: %v", err)
	}
}
