package docs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
)

func TestResolveOutputPath_RejectsNonASCMarkdownFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "README.md")
	if err := os.WriteFile(target, []byte("# Readme\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, _, err := resolveOutputPath(target)
	if !errors.Is(err, ErrInvalidASCReferencePath) {
		t.Fatalf("expected ErrInvalidASCReferencePath, got %v", err)
	}
}

func TestResolveOutputPath_RejectsFileLikeNonMarkdownPath(t *testing.T) {
	target := filepath.Join(t.TempDir(), "notes.txt")

	_, _, err := resolveOutputPath(target)
	if !errors.Is(err, ErrInvalidASCReferencePath) {
		t.Fatalf("expected ErrInvalidASCReferencePath, got %v", err)
	}
}

func TestResolveOutputPath_DirectoryPathResolvesASCFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "docs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create directory: %v", err)
	}

	path, linkRoot, err := resolveOutputPath(dir)
	if err != nil {
		t.Fatalf("resolveOutputPath error: %v", err)
	}

	expectedPath := filepath.Join(dir, ascReferenceFile)
	if path != expectedPath {
		t.Fatalf("expected path %q, got %q", expectedPath, path)
	}
	if linkRoot != dir {
		t.Fatalf("expected link root %q, got %q", dir, linkRoot)
	}
}

func TestInitReference_ReturnsTypedErrorWhenASCExists(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("create .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ascReferenceFile), []byte("# Existing\n"), 0o644); err != nil {
		t.Fatalf("write ASC.md: %v", err)
	}

	_, err := InitReference(InitOptions{Path: repo, Force: false, Link: false})
	if !errors.Is(err, ErrASCReferenceExists) {
		t.Fatalf("expected ErrASCReferenceExists, got %v", err)
	}
}

func TestResolveOutputPathPreservesWhitespaceInDirectoryName(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, " repo ")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("create whitespace-named repo: %v", err)
	}

	target, linkRoot, err := resolveOutputPath(repo)
	if err != nil {
		t.Fatalf("resolveOutputPath() error = %v", err)
	}
	if target != filepath.Join(repo, ascReferenceFile) {
		t.Fatalf("target = %q, want %q", target, filepath.Join(repo, ascReferenceFile))
	}
	if linkRoot != repo {
		t.Fatalf("link root = %q, want %q", linkRoot, repo)
	}
}

func TestPlanAgentsLink_RewritesLegacyReference(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	legacy := "# AGENTS\n\n## ASC CLI Reference\n\nSee `ASC.md` for the command catalog and workflows.\n"
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	root, err := rootfs.New(dir)
	if err != nil {
		t.Fatalf("rootfs.New error: %v", err)
	}

	content, changed, err := planAgentsLink(root, "AGENTS.md", "subdir/ASC.md")
	if err != nil {
		t.Fatalf("planAgentsLink error: %v", err)
	}
	if !changed {
		t.Fatal("expected planAgentsLink to update legacy reference")
	}
	if !strings.Contains(content, "See `subdir/ASC.md` for the command catalog and workflows.") {
		t.Fatalf("expected rewritten reference, got %q", content)
	}
	if strings.Contains(content, "See `ASC.md` for the command catalog and workflows.") {
		t.Fatalf("expected legacy reference removed, got %q", content)
	}

	// Planning must not touch the file; only the apply step writes.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(data) != legacy {
		t.Fatalf("AGENTS.md content = %q, want untouched during planning", string(data))
	}
}

func TestPlanClaudeLink_RewritesLegacyReference(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	legacy := "@Agents.md\n@ASC.md\n"
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	root, err := rootfs.New(dir)
	if err != nil {
		t.Fatalf("rootfs.New error: %v", err)
	}

	content, changed, err := planClaudeLink(root, "CLAUDE.md", "subdir/ASC.md")
	if err != nil {
		t.Fatalf("planClaudeLink error: %v", err)
	}
	if !changed {
		t.Fatal("expected planClaudeLink to update legacy reference")
	}
	if !strings.Contains(content, "@subdir/ASC.md") {
		t.Fatalf("expected rewritten directive, got %q", content)
	}
	if strings.Contains(content, "@ASC.md") {
		t.Fatalf("expected legacy directive removed, got %q", content)
	}

	// Planning must not touch the file; only the apply step writes.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	if string(data) != legacy {
		t.Fatalf("CLAUDE.md content = %q, want untouched during planning", string(data))
	}
}

func TestNewInitReferenceCommand_PrefixesErrors(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("create .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ascReferenceFile), []byte("# Existing\n"), 0o644); err != nil {
		t.Fatalf("write ASC.md: %v", err)
	}

	cmd := NewInitReferenceCommand(
		"docs init",
		"init",
		"asc docs init [flags]",
		"Create an ASC.md command reference for the asc cli in the current repo.",
		"Create an ASC.md command reference for the asc cli in the current repo.",
		"docs init",
	)
	if err := cmd.FlagSet.Parse([]string{"--path", repo}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	err := cmd.Exec(context.Background(), nil)
	if err == nil {
		t.Fatal("expected command to fail when ASC.md already exists")
	}
	if !errors.Is(err, ErrASCReferenceExists) {
		t.Fatalf("expected ErrASCReferenceExists, got %v", err)
	}
	if !strings.Contains(err.Error(), "docs init:") {
		t.Fatalf("expected prefixed error, got %v", err)
	}
}

func TestNewInitReferenceCommand_UsesDefaultUsageFunc(t *testing.T) {
	cmd := NewInitReferenceCommand(
		"docs init",
		"init",
		"asc docs init [flags]",
		"Create an ASC.md command reference for the asc cli in the current repo.",
		"Create an ASC.md command reference for the asc cli in the current repo.",
		"docs init",
	)
	if cmd.UsageFunc == nil {
		t.Fatal("expected usage function to be set")
	}
}
