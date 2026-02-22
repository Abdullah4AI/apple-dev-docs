package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNotarizationValidationErrors(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "submit missing file",
			args:    []string{"notarization", "submit"},
			wantErr: "--file is required",
		},
		{
			name:    "status missing id",
			args:    []string{"notarization", "status"},
			wantErr: "--id is required",
		},
		{
			name:    "log missing id",
			args:    []string{"notarization", "log"},
			wantErr: "--id is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := root.Run(context.Background())
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected ErrHelp, got %v", err)
				}
			})

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected error %q in stderr, got %q", test.wantErr, stderr)
			}
		})
	}
}

func TestNotarizationSubmitInvalidPollInterval(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"notarization", "submit", "--file", "/tmp/test.zip", "--poll-interval", "invalid"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		t.Logf("got expected error: %v", err)
	})

	_ = stderr
}

func TestNotarizationSubmitInvalidTimeout(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"notarization", "submit", "--file", "/tmp/test.zip", "--timeout", "invalid"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		t.Logf("got expected error: %v", err)
	})

	_ = stderr
}

func TestNotarizationSubmitNonexistentFile(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	captureOutput(t, func() {
		if err := root.Parse([]string{"notarization", "submit", "--file", "/nonexistent/file.zip"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatal("expected error for nonexistent file, got nil")
		}
		t.Logf("got expected error: %v", err)
	})
}

func TestNotarizationSubmitDirectory(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	dir := t.TempDir()

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	captureOutput(t, func() {
		if err := root.Parse([]string{"notarization", "submit", "--file", dir}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatal("expected error for directory, got nil")
		}
		t.Logf("got expected error: %v", err)
	})
}

func TestNotarizationSubmitEmptyFile(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	dir := t.TempDir()
	emptyFile := filepath.Join(dir, "empty.zip")
	if err := os.WriteFile(emptyFile, []byte{}, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	captureOutput(t, func() {
		if err := root.Parse([]string{"notarization", "submit", "--file", emptyFile}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatal("expected error for empty file, got nil")
		}
		t.Logf("got expected error: %v", err)
	})
}

func TestNotarizationListNegativeLimit(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	captureOutput(t, func() {
		if err := root.Parse([]string{"notarization", "list", "--limit", "-1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatal("expected error for negative limit, got nil")
		}
		t.Logf("got expected error: %v", err)
	})
}

func TestNotarizationSubmitUnsupportedExtension(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	dir := t.TempDir()
	badFile := filepath.Join(dir, "app.txt")
	if err := os.WriteFile(badFile, []byte("not a real archive"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	captureOutput(t, func() {
		if err := root.Parse([]string{"notarization", "submit", "--file", badFile}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if err == nil {
			t.Fatal("expected error for unsupported file type, got nil")
		}
		t.Logf("got expected error: %v", err)
	})
}

func TestNotarizationHelpOutput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "notarization help",
			args: []string{"notarization"},
			want: "notarization",
		},
		{
			name: "notarization submit help",
			args: []string{"notarization", "submit"},
			want: "--file",
		},
		{
			name: "notarization status help",
			args: []string{"notarization", "status"},
			want: "--id",
		},
		{
			name: "notarization log help",
			args: []string{"notarization", "log"},
			want: "--id",
		},
		{
			name: "notarization group shows subcommands",
			args: []string{"notarization"},
			want: "submit",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			_, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := root.Run(context.Background())
				// Help output is expected
				if err != nil && !errors.Is(err, flag.ErrHelp) {
					t.Logf("run returned: %v", err)
				}
			})

			if !strings.Contains(stderr, test.want) {
				t.Fatalf("expected %q in help output, got %q", test.want, stderr)
			}
		})
	}
}
