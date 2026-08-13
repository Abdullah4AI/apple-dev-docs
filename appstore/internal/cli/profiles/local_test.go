package profiles

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestResolveProfilesInstallDirVersionBoundary(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	originalHomeDir := profilesUserHomeDirFn
	originalActiveVersion := activeXcodeMajorVersionFn
	t.Cleanup(func() {
		profilesRuntimeGOOS = originalGOOS
		profilesUserHomeDirFn = originalHomeDir
		activeXcodeMajorVersionFn = originalActiveVersion
	})

	profilesRuntimeGOOS = "darwin"
	homeDir := t.TempDir()
	profilesUserHomeDirFn = func() (string, error) { return homeDir, nil }

	tests := []struct {
		major    int
		relative string
	}{
		{major: 15, relative: filepath.Join("Library", "MobileDevice", "Provisioning Profiles")},
		{major: 16, relative: filepath.Join("Library", "Developer", "Xcode", "UserData", "Provisioning Profiles")},
		{major: 27, relative: filepath.Join("Library", "Developer", "Xcode", "UserData", "Provisioning Profiles")},
	}

	for _, test := range tests {
		activeXcodeMajorVersionFn = func(context.Context) (int, error) { return test.major, nil }
		got, err := resolveProfilesInstallDir(context.Background(), "")
		if err != nil {
			t.Fatalf("resolveProfilesInstallDir() for Xcode %d error = %v", test.major, err)
		}
		want := filepath.Join(homeDir, test.relative)
		if got != want {
			t.Fatalf("resolveProfilesInstallDir() for Xcode %d = %q, want %q", test.major, got, want)
		}
	}
}

func TestResolveProfilesInstallDirExplicitOverrideSkipsDiscovery(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	originalHomeDir := profilesUserHomeDirFn
	originalActiveVersion := activeXcodeMajorVersionFn
	t.Cleanup(func() {
		profilesRuntimeGOOS = originalGOOS
		profilesUserHomeDirFn = originalHomeDir
		activeXcodeMajorVersionFn = originalActiveVersion
	})

	profilesRuntimeGOOS = "linux"
	profilesUserHomeDirFn = func() (string, error) { panic("home lookup must not run") }
	activeXcodeMajorVersionFn = func(context.Context) (int, error) { panic("Xcode discovery must not run") }

	input := filepath.Join(t.TempDir(), "nested", "..", "profiles")
	got, err := resolveProfilesInstallDir(context.Background(), input)
	if err != nil {
		t.Fatalf("resolveProfilesInstallDir() error = %v", err)
	}
	if want := filepath.Clean(input); got != want {
		t.Fatalf("resolveProfilesInstallDir() = %q, want %q", got, want)
	}
}

func TestResolveProfilesInstallDirFallsBackToLegacyDirectory(t *testing.T) {
	tests := []struct {
		name  string
		major int
		err   error
	}{
		{name: "discovery failure", err: errors.New("xcodebuild -version failed: command line tools instance")},
		{name: "unusable major version", major: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			originalGOOS := profilesRuntimeGOOS
			originalHomeDir := profilesUserHomeDirFn
			originalActiveVersion := activeXcodeMajorVersionFn
			t.Cleanup(func() {
				profilesRuntimeGOOS = originalGOOS
				profilesUserHomeDirFn = originalHomeDir
				activeXcodeMajorVersionFn = originalActiveVersion
			})

			profilesRuntimeGOOS = "darwin"
			homeDir := t.TempDir()
			profilesUserHomeDirFn = func() (string, error) { return homeDir, nil }
			activeXcodeMajorVersionFn = func(context.Context) (int, error) { return test.major, test.err }

			var got string
			var err error
			stderr := captureProfilesStderr(t, func() {
				got, err = resolveProfilesInstallDir(context.Background(), "")
			})
			if err != nil {
				t.Fatalf("resolveProfilesInstallDir() error = %v, want the legacy default", err)
			}
			want := filepath.Join(homeDir, "Library", "MobileDevice", "Provisioning Profiles")
			if got != want {
				t.Fatalf("resolveProfilesInstallDir() = %q, want %q", got, want)
			}
			if lines := strings.Count(strings.TrimSpace(stderr), "\n") + 1; lines != 1 {
				t.Fatalf("expected a single stderr notice, got %q", stderr)
			}
			for _, wantText := range []string{"active Xcode", want, "--install-dir"} {
				if !strings.Contains(stderr, wantText) {
					t.Fatalf("stderr notice %q missing %q", stderr, wantText)
				}
			}
		})
	}
}

func captureProfilesStderr(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	original := os.Stderr
	os.Stderr = writer
	cleanup := func() {
		os.Stderr = original
		_ = writer.Close()
	}
	defer cleanup()

	captured := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, reader)
		_ = reader.Close()
		captured <- buf.String()
	}()

	fn()

	cleanup()
	return <-captured
}

func TestCaptureProfilesStderrRestoresAfterGoexit(t *testing.T) {
	original := os.Stderr
	t.Cleanup(func() { os.Stderr = original })

	writerCh := make(chan *os.File, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		captureProfilesStderr(t, func() {
			writerCh <- os.Stderr
			// testing.T.Fatal and testing.T.FailNow both terminate their
			// goroutine with runtime.Goexit after running deferred cleanup.
			runtime.Goexit()
		})
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("captureProfilesStderr did not exit after runtime.Goexit")
	}
	if os.Stderr != original {
		t.Fatal("captureProfilesStderr did not restore os.Stderr after runtime.Goexit")
	}
	var writer *os.File
	select {
	case writer = <-writerCh:
	case <-time.After(time.Second):
		t.Fatal("captureProfilesStderr did not expose the redirected writer")
	}
	t.Cleanup(func() { _ = writer.Close() })
	if _, err := writer.Write([]byte("probe")); err == nil {
		t.Fatal("captureProfilesStderr did not close the pipe writer after runtime.Goexit")
	}
}

func TestResolveProfilesInstallDirPreservesCancellation(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	originalActiveVersion := activeXcodeMajorVersionFn
	t.Cleanup(func() {
		profilesRuntimeGOOS = originalGOOS
		activeXcodeMajorVersionFn = originalActiveVersion
	})

	profilesRuntimeGOOS = "darwin"
	activeXcodeMajorVersionFn = func(ctx context.Context) (int, error) { return 0, ctx.Err() }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := resolveProfilesInstallDir(ctx, "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("resolveProfilesInstallDir() error = %v, want context.Canceled", err)
	}
}

func TestResolveProfilesInstallDirHomeFailureIsRuntimeError(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	originalHomeDir := profilesUserHomeDirFn
	originalActiveVersion := activeXcodeMajorVersionFn
	t.Cleanup(func() {
		profilesRuntimeGOOS = originalGOOS
		profilesUserHomeDirFn = originalHomeDir
		activeXcodeMajorVersionFn = originalActiveVersion
	})

	profilesRuntimeGOOS = "darwin"
	activeXcodeMajorVersionFn = func(context.Context) (int, error) { return 16, nil }
	homeErr := errors.New("home unavailable")
	profilesUserHomeDirFn = func() (string, error) { return "", homeErr }

	_, err := resolveProfilesInstallDirForCommand(context.Background(), "", "profiles local list")
	if !errors.Is(err, homeErr) {
		t.Fatalf("resolveProfilesInstallDirForCommand() error = %v, want home error", err)
	}
	if errors.Is(err, flag.ErrHelp) {
		t.Fatalf("home failure must be a runtime error, got %v", err)
	}
	if !strings.HasPrefix(err.Error(), "profiles local list: resolve install directory:") {
		t.Fatalf("unexpected command context: %v", err)
	}
}

func TestResolveProfilesInstallDirNonMacRequiresOverride(t *testing.T) {
	originalGOOS := profilesRuntimeGOOS
	t.Cleanup(func() { profilesRuntimeGOOS = originalGOOS })
	profilesRuntimeGOOS = "linux"

	_, err := resolveProfilesInstallDir(context.Background(), "")
	if !errors.Is(err, errProfilesInstallDirRequired) {
		t.Fatalf("resolveProfilesInstallDir() error = %v, want --install-dir requirement", err)
	}
}

func TestProfilesLocalHelpDocumentsVersionedDefaults(t *testing.T) {
	commands := []struct {
		name string
		help string
	}{
		{name: "local", help: ProfilesLocalCommand().LongHelp},
		{name: "install", help: ProfilesLocalInstallCommand().LongHelp},
		{name: "list", help: ProfilesLocalListCommand().LongHelp},
		{name: "clean", help: ProfilesLocalCleanCommand().LongHelp},
	}

	for _, command := range commands {
		for _, want := range []string{
			"Xcode 16 or newer: ~/Library/Developer/Xcode/UserData/Provisioning Profiles",
			"Xcode 15 or older: ~/Library/MobileDevice/Provisioning Profiles",
			"Use --install-dir",
		} {
			if !strings.Contains(command.help, want) {
				t.Fatalf("%s help missing %q: %s", command.name, want, command.help)
			}
		}
	}
}

func TestIsExpired(t *testing.T) {
	t0 := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expiresAt time.Time
		now       time.Time
		want      bool
	}{
		{name: "zero", expiresAt: time.Time{}, now: t0, want: false},
		{name: "before", expiresAt: t0.Add(1 * time.Second), now: t0, want: false},
		{name: "equal", expiresAt: t0, now: t0, want: true},
		{name: "after", expiresAt: t0.Add(-1 * time.Second), now: t0, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExpired(tt.expiresAt, tt.now); got != tt.want {
				t.Fatalf("isExpired(expiresAt=%s, now=%s)=%t, want %t", tt.expiresAt.Format(time.RFC3339Nano), tt.now.Format(time.RFC3339Nano), got, tt.want)
			}
		})
	}
}
