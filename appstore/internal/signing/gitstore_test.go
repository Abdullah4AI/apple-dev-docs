package signing

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestEnsureInsideDir(t *testing.T) {
	baseDir := t.TempDir()

	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{
			name:   "allows base directory itself",
			target: baseDir,
		},
		{
			name:   "allows child path",
			target: filepath.Join(baseDir, "nested", "file.txt"),
		},
		{
			name:    "rejects parent directory escape",
			target:  filepath.Join(baseDir, "..", "escaped.txt"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureInsideDir(baseDir, tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("EnsureInsideDir(%q, %q) expected error, got nil", baseDir, tt.target)
				}
				return
			}
			if err != nil {
				t.Fatalf("EnsureInsideDir(%q, %q) unexpected error: %v", baseDir, tt.target, err)
			}
		})
	}
}

func TestGitStoreWriteAndReadEncryptedFileRoundTrip(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}
	relPath := filepath.Join("profiles", "development", "com.example.app.mobileprovision")
	plaintext := []byte("profile-data")
	password := "test-password"

	if err := store.WriteEncryptedFile(relPath, plaintext, password); err != nil {
		t.Fatalf("WriteEncryptedFile: %v", err)
	}

	encryptedPath := filepath.Join(store.LocalDir, relPath+".enc")
	encrypted, err := os.ReadFile(encryptedPath)
	if err != nil {
		t.Fatalf("read encrypted output: %v", err)
	}
	if bytes.Equal(encrypted, plaintext) {
		t.Fatal("encrypted file should not match plaintext bytes")
	}

	got, err := store.ReadEncryptedFile(relPath, password)
	if err != nil {
		t.Fatalf("ReadEncryptedFile: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("decrypted output mismatch: got %q, want %q", got, plaintext)
	}
}

func TestGitStoreWriteEncryptedFileRejectsPathEscape(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}

	err := store.WriteEncryptedFile(filepath.Join("..", "escaped"), []byte("secret"), "test-password")
	if err == nil {
		t.Fatal("expected path escape error, got nil")
	}
	if !strings.Contains(err.Error(), "escapes base directory") {
		t.Fatalf("expected path escape error, got %v", err)
	}
}

func TestGitStoreWriteEncryptedFileRejectsSymlinkedParentDirectory(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}
	outsideDir := t.TempDir()

	if err := os.Symlink(outsideDir, filepath.Join(store.LocalDir, "linked")); err != nil {
		t.Fatalf("create parent symlink: %v", err)
	}

	err := store.WriteEncryptedFile(filepath.Join("linked", "secret"), []byte("secret"), "test-password")
	if err == nil {
		t.Fatal("expected symlink rejection error, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection error, got %v", err)
	}

	_, statErr := os.Stat(filepath.Join(outsideDir, "secret.enc"))
	if statErr == nil {
		t.Fatal("did not expect write through symlinked parent directory")
	}
	if !os.IsNotExist(statErr) {
		t.Fatalf("stat outside write target: %v", statErr)
	}
}

func TestGitStoreWriteEncryptedFileRejectsSymlinkTarget(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "secret.enc")

	if err := os.WriteFile(outsidePath, []byte("original"), 0o600); err != nil {
		t.Fatalf("write outside target: %v", err)
	}
	if err := os.Symlink(outsidePath, filepath.Join(store.LocalDir, "secret.enc")); err != nil {
		t.Fatalf("create file symlink: %v", err)
	}

	err := store.WriteEncryptedFile("secret", []byte("secret"), "test-password")
	if err == nil {
		t.Fatal("expected symlink rejection error, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection error, got %v", err)
	}

	got, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside target: %v", readErr)
	}
	if string(got) != "original" {
		t.Fatalf("did not expect write through symlink target, got %q", got)
	}
}

func TestGitStoreReadEncryptedFileRejectsSymlink(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}
	targetDir := t.TempDir()
	password := "test-password"

	encrypted, err := Encrypt([]byte("secret"), password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	targetPath := filepath.Join(targetDir, "secret.enc")
	if err := os.WriteFile(targetPath, encrypted, 0o600); err != nil {
		t.Fatalf("write target encrypted file: %v", err)
	}

	linkPath := filepath.Join(store.LocalDir, "secret.enc")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err = store.ReadEncryptedFile("secret", password)
	if err == nil {
		t.Fatal("expected symlink rejection error, got nil")
	}
	if !strings.Contains(err.Error(), "refusing to read symlink") {
		t.Fatalf("expected symlink rejection error, got %v", err)
	}
}

func TestGitStoreReadEncryptedFileRejectsSymlinkedParentDirectory(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}
	targetDir := t.TempDir()
	password := "test-password"

	encrypted, err := Encrypt([]byte("secret"), password)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	targetPath := filepath.Join(targetDir, "secret.enc")
	if err := os.WriteFile(targetPath, encrypted, 0o600); err != nil {
		t.Fatalf("write target encrypted file: %v", err)
	}

	if err := os.Symlink(targetDir, filepath.Join(store.LocalDir, "linked")); err != nil {
		t.Fatalf("create parent symlink: %v", err)
	}

	_, err = store.ReadEncryptedFile(filepath.Join("linked", "secret"), password)
	if err == nil {
		t.Fatal("expected symlink rejection error, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection error, got %v", err)
	}
}

func TestGitStoreListEncryptedFilesSkipsGitDirAndSymlinks(t *testing.T) {
	store := &GitStore{LocalDir: t.TempDir()}

	if err := os.MkdirAll(filepath.Join(store.LocalDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(store.LocalDir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	write := func(rel string) {
		t.Helper()
		path := filepath.Join(store.LocalDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	write("root.enc")
	write(filepath.Join("nested", "child.enc"))
	write(filepath.Join(".git", "ignored.enc"))

	if err := os.Symlink(filepath.Join(store.LocalDir, "root.enc"), filepath.Join(store.LocalDir, "symlink.enc")); err != nil {
		t.Fatalf("create file symlink: %v", err)
	}
	if err := os.Symlink(filepath.Join(store.LocalDir, "nested"), filepath.Join(store.LocalDir, "linked-dir")); err != nil {
		t.Fatalf("create dir symlink: %v", err)
	}

	got, err := store.ListEncryptedFiles()
	if err != nil {
		t.Fatalf("ListEncryptedFiles: %v", err)
	}

	gotSet := map[string]bool{}
	for _, rel := range got {
		gotSet[filepath.ToSlash(rel)] = true
	}

	if !gotSet["root"] {
		t.Fatalf("expected root file in list, got %v", got)
	}
	if !gotSet["nested/child"] {
		t.Fatalf("expected nested file in list, got %v", got)
	}
	if gotSet[".git/ignored"] {
		t.Fatalf("did not expect .git file in list, got %v", got)
	}
	if gotSet["symlink"] {
		t.Fatalf("did not expect symlink file in list, got %v", got)
	}
	if gotSet["linked-dir/child"] {
		t.Fatalf("did not expect symlinked directory contents in list, got %v", got)
	}
}

func TestRedactRepoURLRemovesCredentials(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "token in password position",
			raw:  "https://x-access-token:ghp_SUPERSECRET@github.com/team/certs.git",
			want: "https://%5BREDACTED%5D@github.com/team/certs.git",
		},
		{
			name: "token in user position",
			raw:  "https://ghp_SUPERSECRET@github.com/team/certs.git",
			want: "https://%5BREDACTED%5D@github.com/team/certs.git",
		},
		{
			name: "unparseable userinfo",
			raw:  "https://user:sec ret@github.com/team/certs.git",
			want: "https://[REDACTED]@github.com/team/certs.git",
		},
		{
			name: "scp style remote keeps its user",
			raw:  "git@github.com:team/certs.git",
			want: "git@github.com:team/certs.git",
		},
		{
			name: "scp style remote with credentials",
			raw:  "user:secret@github.com:team/certs.git",
			want: "[REDACTED]@github.com:team/certs.git",
		},
		{
			name: "no credentials",
			raw:  "https://github.com/team/certs.git",
			want: "https://github.com/team/certs.git",
		},
		{
			name: "local path",
			raw:  "/srv/git/certs.git",
			want: "/srv/git/certs.git",
		},
		{
			name: "empty",
			raw:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RedactRepoURL(tt.raw); got != tt.want {
				t.Fatalf("RedactRepoURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestGitStoreCloneErrorRedactsRepositoryCredentials(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable scripts require a POSIX shell")
	}

	binDir := t.TempDir()
	writeTestExecutable(t, filepath.Join(binDir, "git"), "#!/bin/sh\nexit 1\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", "")

	store := &GitStore{
		RepoURL:  "https://x-access-token:ghp_SUPERSECRET@github.com/team/certs.git",
		LocalDir: filepath.Join(t.TempDir(), "clone"),
		Branch:   "signing",
	}

	err := store.Clone(context.Background(), false)
	if err == nil {
		t.Fatal("expected clone failure for a missing branch")
	}
	for _, secret := range []string{"ghp_SUPERSECRET", "x-access-token"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("clone error leaks repository credentials: %v", err)
		}
	}
	if !strings.Contains(err.Error(), "github.com/team/certs.git") {
		t.Fatalf("clone error should still name the repository host: %v", err)
	}
	if !strings.Contains(err.Error(), `branch "signing" not found`) {
		t.Fatalf("clone error should still report the missing branch: %v", err)
	}
}

func TestGitStoreCloneReportsGitConfigProbeFailures(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable scripts require a POSIX shell")
	}

	tests := []struct {
		name        string
		allowCreate bool
	}{
		{name: "pull mode"},
		{name: "push mode", allowCreate: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			binDir := t.TempDir()
			callCapture := filepath.Join(t.TempDir(), "git-calls.txt")
			writeTestExecutable(t, filepath.Join(binDir, "git"), `#!/bin/sh
set -eu
printf '%s\n' "$1" >> "$ASC_FAKE_GIT_CALLS"
if [ "$1" = "config" ]; then
  printf 'fatal: bad config line 1 in file .gitconfig\n' >&2
  exit 128
fi
exit 0
`)
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("ASC_FAKE_GIT_CALLS", callCapture)
			t.Setenv("GIT_SSH_COMMAND", "")
			t.Setenv("GIT_SSH", "")

			store := &GitStore{
				RepoURL:  "git@github.com:team/certs.git",
				LocalDir: filepath.Join(t.TempDir(), "clone"),
				Branch:   "main",
			}

			err := store.Clone(context.Background(), test.allowCreate)
			if err == nil {
				t.Fatal("expected the Git configuration probe failure to surface")
			}
			if !strings.Contains(err.Error(), "core.sshCommand") {
				t.Fatalf("error = %v, want the Git configuration failure", err)
			}
			if strings.Contains(err.Error(), "not found") {
				t.Fatalf("local Git configuration failure reported as a missing branch: %v", err)
			}

			calls, readErr := os.ReadFile(callCapture)
			if readErr != nil {
				t.Fatalf("read fake git calls: %v", readErr)
			}
			if got := strings.Count(string(calls), "config"); got != 1 {
				t.Fatalf("Git configuration probe ran %d times, want 1: %q", got, calls)
			}
			if strings.Contains(string(calls), "clone") {
				t.Fatalf("clone ran despite an unusable Git configuration: %q", calls)
			}
		})
	}
}

func TestGitStoreGitHelpersUseNonInteractiveExecutables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable scripts require a POSIX shell")
	}

	binDir := t.TempDir()
	gitCapture := filepath.Join(t.TempDir(), "git-env.txt")
	sshCapture := filepath.Join(t.TempDir(), "ssh-args.txt")

	writeTestExecutable(t, filepath.Join(binDir, "git"), `#!/bin/sh
set -eu
if [ "${1-}" = "config" ]; then
  exit 1
fi
printf '%s|%s\n' "${GIT_TERMINAL_PROMPT-}" "${GIT_SSH_COMMAND-}" >> "$ASC_FAKE_GIT_CAPTURE"
sh -c "$GIT_SSH_COMMAND \"\$@\"" asc-fake-git "$@"
if [ "${1-}" = "status" ]; then
  printf 'clean\n'
fi
`)
	writeTestExecutable(t, filepath.Join(binDir, "ssh"), `#!/bin/sh
set -eu
{
  printf '%s\n' '--call--'
  for arg in "$@"; do
    printf '%s\n' "$arg"
  done
} >> "$ASC_FAKE_SSH_CAPTURE"
`)

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ASC_FAKE_GIT_CAPTURE", gitCapture)
	t.Setenv("ASC_FAKE_SSH_CAPTURE", sshCapture)
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	t.Setenv("GIT_SSH_COMMAND", "")

	store := &GitStore{}
	if err := store.gitRun(context.Background(), "", "clone", "source", "destination"); err != nil {
		t.Fatalf("gitRun: %v", err)
	}
	output, err := store.gitOutput(context.Background(), "", "status", "--porcelain")
	if err != nil {
		t.Fatalf("gitOutput: %v", err)
	}
	if output != "clean\n" {
		t.Fatalf("gitOutput() = %q, want %q", output, "clean\n")
	}

	gitEnvironment, err := os.ReadFile(gitCapture)
	if err != nil {
		t.Fatalf("read fake git environment: %v", err)
	}
	if got, want := string(gitEnvironment), "0|ssh -o BatchMode=yes\n0|ssh -o BatchMode=yes\n"; got != want {
		t.Fatalf("fake git environment = %q, want %q", got, want)
	}

	sshArguments, err := os.ReadFile(sshCapture)
	if err != nil {
		t.Fatalf("read fake ssh arguments: %v", err)
	}
	wantCalls := "--call--\n-o\nBatchMode=yes\nclone\nsource\ndestination\n" +
		"--call--\n-o\nBatchMode=yes\nstatus\n--porcelain\n"
	if got := string(sshArguments); got != wantCalls {
		t.Fatalf("fake ssh arguments = %q, want %q", got, wantCalls)
	}
}

func TestNewGitCommandSSHSelection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable scripts require a POSIX shell")
	}

	tests := []struct {
		name              string
		setup             string
		clone             bool
		wantDefault       bool
		wantLowerPriority bool
	}{
		{name: "GIT_SSH_COMMAND overrides GIT_SSH and core config", setup: "command-precedence", clone: true},
		{name: "GIT_SSH and core config suppress default injection", setup: "ssh-and-core", clone: true, wantLowerPriority: true},
		{name: "global config for clone", setup: "global", clone: true},
		{name: "unconditional global include for clone", setup: "include", clone: true},
		{name: "command config for clone", setup: "command", clone: true},
		{name: "repository config for ls-remote", setup: "repository"},
		{name: "clone ignores local repository config", setup: "local", clone: true, wantDefault: true},
		{name: "clone ignores gitdir conditional config", setup: "conditional", clone: true, wantDefault: true},
		{name: "blank command config overrides global config", setup: "blank-command", clone: true, wantDefault: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callerRepository := t.TempDir()
			runTestGit(t, "init", "--quiet", callerRepository)

			configuredCapture := filepath.Join(t.TempDir(), "configured-transport.txt")
			configuredTransport := filepath.Join(t.TempDir(), "configured-ssh")
			writeTestExecutable(t, configuredTransport, `#!/bin/sh
set -eu
printf '%s\n' "$@" > "$ASC_CONFIGURED_SSH_CAPTURE"
exit 17
`)
			lowerPriorityCapture := filepath.Join(t.TempDir(), "lower-priority-transport.txt")
			lowerPriorityTransport := filepath.Join(t.TempDir(), "lower-priority-ssh")
			writeTestExecutable(t, lowerPriorityTransport, `#!/bin/sh
set -eu
printf '%s\n' "$@" > "$ASC_LOWER_PRIORITY_SSH_CAPTURE"
exit 17
`)
			binDir := t.TempDir()
			defaultCapture := filepath.Join(t.TempDir(), "default-transport.txt")
			writeTestExecutable(t, filepath.Join(binDir, "ssh"), `#!/bin/sh
set -eu
printf '%s\n' "$@" > "$ASC_DEFAULT_SSH_CAPTURE"
exit 17
`)

			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("HOME", t.TempDir())
			t.Setenv("ASC_CONFIGURED_SSH_CAPTURE", configuredCapture)
			t.Setenv("ASC_LOWER_PRIORITY_SSH_CAPTURE", lowerPriorityCapture)
			t.Setenv("ASC_DEFAULT_SSH_CAPTURE", defaultCapture)
			t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
			t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "missing-global-config"))
			t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(t.TempDir(), "missing-system-config"))
			t.Setenv("GIT_CONFIG_COUNT", "0")
			t.Setenv("GIT_SSH_COMMAND", "")
			t.Setenv("GIT_SSH", "")
			t.Setenv("GIT_SSH_VARIANT", "ssh")

			switch tt.setup {
			case "command-precedence":
				globalConfig := filepath.Join(t.TempDir(), "global-gitconfig")
				runTestGit(t, "config", "--file", globalConfig, "core.sshCommand", lowerPriorityTransport)
				t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
				t.Setenv("GIT_SSH", lowerPriorityTransport)
				t.Setenv("GIT_SSH_COMMAND", configuredTransport+` --identity 'release key'`)
			case "ssh-and-core":
				globalConfig := filepath.Join(t.TempDir(), "global-gitconfig")
				runTestGit(t, "config", "--file", globalConfig, "core.sshCommand", lowerPriorityTransport)
				t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
				t.Setenv("GIT_SSH", configuredTransport)
			case "global":
				globalConfig := filepath.Join(t.TempDir(), "global-gitconfig")
				runTestGit(t, "config", "--file", globalConfig, "core.sshCommand", configuredTransport)
				t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
			case "include":
				includedConfig := filepath.Join(t.TempDir(), "included-gitconfig")
				runTestGit(t, "config", "--file", includedConfig, "core.sshCommand", configuredTransport)
				globalConfig := filepath.Join(t.TempDir(), "global-gitconfig")
				runTestGit(t, "config", "--file", globalConfig, "include.path", includedConfig)
				t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
			case "command":
				t.Setenv("GIT_CONFIG_COUNT", "1")
				t.Setenv("GIT_CONFIG_KEY_0", "core.sshCommand")
				t.Setenv("GIT_CONFIG_VALUE_0", configuredTransport)
			case "repository", "local":
				runTestGit(t, "-C", callerRepository, "config", "core.sshCommand", configuredTransport)
			case "conditional":
				includedConfig := filepath.Join(t.TempDir(), "included-gitconfig")
				runTestGit(t, "config", "--file", includedConfig, "core.sshCommand", configuredTransport)
				globalConfig := filepath.Join(t.TempDir(), "global-gitconfig")
				resolvedCallerRepository, err := filepath.EvalSymlinks(callerRepository)
				if err != nil {
					t.Fatalf("resolve caller repository: %v", err)
				}
				conditionKey := "includeIf.gitdir:" + filepath.ToSlash(resolvedCallerRepository) + "/.path"
				runTestGit(t, "config", "--file", globalConfig, conditionKey, includedConfig)
				preconditionEnvironment := replaceCommandEnvironmentValue(
					standaloneTestGitEnvironment(t), "GIT_CONFIG_GLOBAL", globalConfig, false,
				)
				got := strings.TrimSpace(runTestGitOutput(
					t, preconditionEnvironment, callerRepository, "config", "--get", "core.sshCommand",
				))
				if got != configuredTransport {
					t.Fatalf("conditional core.sshCommand = %q, want %q", got, configuredTransport)
				}
				t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
			case "blank-command":
				globalConfig := filepath.Join(t.TempDir(), "global-gitconfig")
				runTestGit(t, "config", "--file", globalConfig, "core.sshCommand", configuredTransport)
				t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
				t.Setenv("GIT_CONFIG_COUNT", "1")
				t.Setenv("GIT_CONFIG_KEY_0", "core.sshCommand")
				t.Setenv("GIT_CONFIG_VALUE_0", "")
			default:
				t.Fatalf("unknown test setup %q", tt.setup)
			}

			args := []string{"ls-remote", "ssh://git@127.0.0.1:1/repository"}
			if tt.clone {
				args = []string{"clone", "ssh://git@127.0.0.1:1/repository", filepath.Join(t.TempDir(), "clone")}
			}
			cmd, err := newGitCommand(context.Background(), callerRepository, args...)
			if err != nil {
				t.Fatalf("newGitCommand: %v", err)
			}
			if err := cmd.Run(); err == nil {
				t.Fatal("expected fake SSH transport failure")
			}

			selectedCapture := configuredCapture
			if tt.wantDefault {
				selectedCapture = defaultCapture
			} else if tt.wantLowerPriority {
				selectedCapture = lowerPriorityCapture
			}
			selectedArguments, err := os.ReadFile(selectedCapture)
			if err != nil {
				t.Fatalf("read selected SSH transport arguments: %v", err)
			}
			if tt.wantDefault && !strings.Contains(string(selectedArguments), "-o\nBatchMode=yes\n") {
				t.Fatalf("default SSH transport arguments = %q, want BatchMode=yes", selectedArguments)
			}
			for _, capture := range []string{configuredCapture, lowerPriorityCapture, defaultCapture} {
				if capture == selectedCapture {
					continue
				}
				if _, err := os.Stat(capture); err == nil {
					t.Fatalf("unselected SSH transport %q unexpectedly ran", capture)
				} else if !os.IsNotExist(err) {
					t.Fatalf("stat unselected SSH transport %q: %v", capture, err)
				}
			}
		})
	}
}

func TestNewGitCommandFailsClosedWhenSSHConfigLookupFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable scripts require a POSIX shell")
	}

	binDir := t.TempDir()
	mainCommandCapture := filepath.Join(t.TempDir(), "main-command.txt")
	writeTestExecutable(t, filepath.Join(binDir, "git"), `#!/bin/sh
set -eu
if [ "${1-}" = "config" ]; then
  printf 'invalid git configuration\n' >&2
  exit 2
fi
printf 'invoked\n' > "$ASC_FAKE_GIT_CAPTURE"
`)

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ASC_FAKE_GIT_CAPTURE", mainCommandCapture)
	t.Setenv("GIT_SSH_COMMAND", "")
	t.Setenv("GIT_SSH", "")

	cmd, err := newGitCommand(context.Background(), "", "clone", "source", "destination")
	if err == nil {
		t.Fatalf("newGitCommand() = %v, want SSH config lookup error", cmd)
	}
	if !strings.Contains(err.Error(), "core.sshCommand") {
		t.Fatalf("newGitCommand() error = %q, want core.sshCommand context", err)
	}
	if _, statErr := os.Stat(mainCommandCapture); !os.IsNotExist(statErr) {
		t.Fatalf("network-capable Git command ran after lookup failure: %v", statErr)
	}
}

func TestNewGitCommandIgnoresInheritedRepositorySelectors(t *testing.T) {
	sentinelRepository := filepath.Join(t.TempDir(), "sentinel")
	targetRepository := filepath.Join(t.TempDir(), "target")
	standaloneEnvironment := standaloneTestGitEnvironment(t)
	runTestGitWithEnvironment(t, standaloneEnvironment, "init", "--quiet", sentinelRepository)
	runTestGitWithEnvironment(t, standaloneEnvironment, "init", "--quiet", targetRepository)

	globalConfig := filepath.Join(t.TempDir(), "global-config")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(t.TempDir(), "missing-system-config"))
	t.Setenv("GIT_DIR", filepath.Join(sentinelRepository, ".git"))
	t.Setenv("GIT_WORK_TREE", sentinelRepository)
	t.Setenv("GIT_COMMON_DIR", filepath.Join(sentinelRepository, ".git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(sentinelRepository, ".git", "index"))
	t.Setenv("GIT_OBJECT_DIRECTORY", filepath.Join(sentinelRepository, ".git", "objects"))
	t.Setenv("GIT_SSH", "/opt/team/bin/ssh-wrapper")
	t.Setenv("GIT_TERMINAL_PROMPT", "1")

	cmd, err := newGitCommand(context.Background(), targetRepository, "config", "core.testMarker", "target-command")
	if err != nil {
		t.Fatalf("newGitCommand: %v", err)
	}
	if err := cmd.Run(); err != nil {
		t.Fatalf("run Git command: %v", err)
	}

	targetValue, targetConfigured := standaloneTestGitConfigValue(t, standaloneEnvironment, targetRepository, "core.testMarker")
	if !targetConfigured || targetValue != "target-command" {
		t.Errorf("target core.testMarker = %q, configured = %t; want target-command in target repository", targetValue, targetConfigured)
	}
	sentinelValue, sentinelConfigured := standaloneTestGitConfigValue(t, standaloneEnvironment, sentinelRepository, "core.testMarker")
	if sentinelConfigured {
		t.Errorf("sentinel core.testMarker = %q; inherited repository selectors redirected Git", sentinelValue)
	}
	if _, ok := commandEnvironmentValue(cmd.Env, "GIT_DIR", false); ok {
		t.Error("GIT_DIR unexpectedly remained in command environment")
	}
}

func TestRunTestGitIsolatesRepositoryEnvironment(t *testing.T) {
	sentinelRepository := filepath.Join(t.TempDir(), "sentinel")
	targetRepository := filepath.Join(t.TempDir(), "target")
	standaloneEnvironment := standaloneTestGitEnvironment(t)
	runTestGitWithEnvironment(t, standaloneEnvironment, "init", "--quiet", sentinelRepository)
	runTestGitWithEnvironment(t, standaloneEnvironment, "init", "--quiet", targetRepository)

	t.Setenv("GIT_DIR", filepath.Join(sentinelRepository, ".git"))
	t.Setenv("GIT_WORK_TREE", sentinelRepository)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "missing-global-config"))
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(t.TempDir(), "missing-system-config"))
	t.Setenv("HOME", t.TempDir())
	runTestGit(t, "-C", targetRepository, "config", "core.sshCommand", "target-command")

	targetValue, targetConfigured := standaloneTestGitConfigValue(t, standaloneEnvironment, targetRepository, "core.sshCommand")
	if !targetConfigured || targetValue != "target-command" {
		t.Errorf("target core.sshCommand = %q, configured = %t; want target-command in target repository", targetValue, targetConfigured)
	}
	sentinelValue, sentinelConfigured := standaloneTestGitConfigValue(t, standaloneEnvironment, sentinelRepository, "core.sshCommand")
	if sentinelConfigured {
		t.Fatalf("sentinel core.sshCommand = %q; inherited repository selectors escaped test isolation", sentinelValue)
	}
}

func TestGitCommandEnvironmentSelection(t *testing.T) {
	customCommand := `team-ssh-wrapper --identity '/tmp/release key' -o BatchMode=no`
	tests := []struct {
		name                     string
		goos                     string
		environment              []string
		coreSSHCommandConfigured bool
		want                     []string
	}{
		{
			name: "preserves explicit command byte for byte",
			goos: "linux",
			environment: []string{
				"PATH=/usr/bin",
				"GIT_TERMINAL_PROMPT=1",
				"GIT_TERMINAL_PROMPT=true",
				"GIT_SSH_COMMAND=" + customCommand,
			},
			want: []string{"PATH=/usr/bin", "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=" + customCommand},
		},
		{
			name: "preserves GIT_SSH after blank command",
			goos: "linux",
			environment: []string{
				"PATH=/usr/bin",
				"GIT_TERMINAL_PROMPT=1",
				"GIT_SSH_COMMAND= \t",
				"GIT_SSH=/opt/team/bin/ssh-wrapper",
			},
			want: []string{"PATH=/usr/bin", "GIT_SSH=/opt/team/bin/ssh-wrapper", "GIT_TERMINAL_PROMPT=0"},
		},
		{
			name:        "defaults missing SSH settings",
			goos:        "linux",
			environment: []string{"PATH=/usr/bin"},
			want:        []string{"PATH=/usr/bin", "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=ssh -o BatchMode=yes"},
		},
		{
			name:        "defaults blank SSH settings",
			goos:        "linux",
			environment: []string{"PATH=/usr/bin", "GIT_SSH_COMMAND= \t", "GIT_SSH= "},
			want:        []string{"PATH=/usr/bin", "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=ssh -o BatchMode=yes"},
		},
		{
			name:                     "configured core command suppresses default",
			goos:                     "linux",
			environment:              []string{"PATH=/usr/bin"},
			coreSSHCommandConfigured: true,
			want:                     []string{"PATH=/usr/bin", "GIT_TERMINAL_PROMPT=0"},
		},
		{
			name: "Windows matches command keys case insensitively",
			goos: "windows",
			environment: []string{
				"PATH=C:\\Windows\\System32",
				"git_terminal_prompt=1",
				`git_ssh_command=team-ssh-wrapper --identity 'release key'`,
			},
			want: []string{
				"PATH=C:\\Windows\\System32",
				"GIT_TERMINAL_PROMPT=0",
				`GIT_SSH_COMMAND=team-ssh-wrapper --identity 'release key'`,
			},
		},
		{
			name: "Windows preserves mixed-case GIT_SSH",
			goos: "windows",
			environment: []string{
				"PATH=C:\\Windows\\System32",
				"git_terminal_prompt=1",
				"git_ssh_command= ",
				`git_ssh=C:\tools\team-ssh.exe`,
			},
			want: []string{
				"PATH=C:\\Windows\\System32",
				`git_ssh=C:\tools\team-ssh.exe`,
				"GIT_TERMINAL_PROMPT=0",
			},
		},
		{
			name: "POSIX keeps differently cased keys",
			goos: "linux",
			environment: []string{
				"PATH=/usr/bin",
				"git_terminal_prompt=1",
				"git_ssh_command=team-ssh-wrapper",
			},
			want: []string{
				"PATH=/usr/bin",
				"git_terminal_prompt=1",
				"git_ssh_command=team-ssh-wrapper",
				"GIT_TERMINAL_PROMPT=0",
				"GIT_SSH_COMMAND=ssh -o BatchMode=yes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gitCommandEnvironmentWithConfig(tt.environment, tt.goos, tt.coreSSHCommandConfigured)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("gitCommandEnvironmentWithConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitEnvironmentWithoutRepositorySelectorsMatchesWindowsKeysCaseInsensitively(t *testing.T) {
	environment := gitEnvironmentWithoutRepositorySelectors([]string{
		"PATH=C:\\Windows\\System32",
		`git_dir=C:\sensitive\repo\.git`,
		`Git_Work_Tree=C:\sensitive\repo`,
		`GIT_COMMON_DIR=C:\sensitive\repo\.git`,
		`GIT_CONFIG_GLOBAL=C:\config\gitconfig`,
		`GIT_SSH=C:\tools\team-ssh.exe`,
	}, "windows")
	want := []string{
		"PATH=C:\\Windows\\System32",
		`GIT_CONFIG_GLOBAL=C:\config\gitconfig`,
		`GIT_SSH=C:\tools\team-ssh.exe`,
	}
	if !slices.Equal(environment, want) {
		t.Fatalf("gitEnvironmentWithoutRepositorySelectors() = %v, want %v", environment, want)
	}
}

func writeTestExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

func runTestGit(t *testing.T, args ...string) {
	t.Helper()
	runTestGitWithEnvironment(t, standaloneTestGitEnvironment(t), args...)
}

func standaloneTestGitEnvironment(t *testing.T) []string {
	t.Helper()
	return []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + t.TempDir(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + filepath.Join(t.TempDir(), "missing-global-config"),
		"GIT_CONFIG_SYSTEM=" + filepath.Join(t.TempDir(), "missing-system-config"),
	}
}

func runTestGitWithEnvironment(t *testing.T, environment []string, args ...string) {
	t.Helper()
	runTestGitOutput(t, environment, t.TempDir(), args...)
}

func TestRunTestGitOutputSeparatesStandardError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake executable scripts require a POSIX shell")
	}

	binDir := t.TempDir()
	writeTestExecutable(t, filepath.Join(binDir, "git"), `#!/bin/sh
printf 'configured transport\n'
printf 'warning: diagnostic only\n' >&2
`)
	t.Setenv("PATH", binDir)
	environment := replaceCommandEnvironmentValue(standaloneTestGitEnvironment(t), "PATH", binDir, false)
	if got := runTestGitOutput(t, environment, t.TempDir(), "config", "--get", "core.sshCommand"); got != "configured transport\n" {
		t.Fatalf("runTestGitOutput() = %q, want stdout only", got)
	}
}

func runTestGitOutput(t *testing.T, environment []string, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = environment
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

func standaloneTestGitConfigValue(t *testing.T, environment []string, repository, key string) (string, bool) {
	t.Helper()
	cmd := exec.Command("git", "--git-dir", filepath.Join(repository, ".git"), "config", "--get", key)
	cmd.Dir = t.TempDir()
	cmd.Env = environment
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), true
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 1 {
		return "", false
	}
	t.Fatalf("standalone git config --get %s: %v", key, err)
	return "", false
}
