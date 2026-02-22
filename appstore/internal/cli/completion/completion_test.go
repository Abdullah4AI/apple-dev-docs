package completion

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestRootCommandNamesSortedAndDeduplicated(t *testing.T) {
	names := rootCommandNames([]*ffcli.Command{
		{Name: "apps"},
		{Name: "builds"},
		{Name: "apps"},
		nil,
		{Name: "   "},
	})

	expected := []string{"apps", "builds", "completion"}
	if len(names) != len(expected) {
		t.Fatalf("unexpected names length: got %d want %d (%v)", len(names), len(expected), names)
	}
	for i := range expected {
		if names[i] != expected[i] {
			t.Fatalf("unexpected names[%d]: got %q want %q", i, names[i], expected[i])
		}
	}
}

func TestCompletionCommandValidationAndOutput(t *testing.T) {
	cmd := CompletionCommand([]*ffcli.Command{
		{Name: "apps"},
		{Name: "builds"},
	})

	// Missing shell should fail with ErrHelp.
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp for missing shell, got %v", err)
	}

	// Unsupported shell should fail with ErrHelp.
	cmd = CompletionCommand([]*ffcli.Command{{Name: "apps"}})
	if err := cmd.FlagSet.Parse([]string{"--shell", "tcsh"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp for unsupported shell, got %v", err)
	}

	// Supported shell should print script and succeed.
	cmd = CompletionCommand([]*ffcli.Command{{Name: "apps"}, {Name: "builds"}})
	if err := cmd.FlagSet.Parse([]string{"--shell", "bash"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	stdout := captureStdout(t, func() error {
		return cmd.Exec(context.Background(), nil)
	})
	if !strings.Contains(stdout, "complete -F _asc_completions asc") {
		t.Fatalf("expected bash completion script output, got %q", stdout)
	}
}

func TestCompletionScriptHelpers(t *testing.T) {
	if !strings.Contains(bashScript([]string{"apps"}), "apps") {
		t.Fatalf("bash script missing command names")
	}
	if !strings.Contains(zshScript([]string{"apps"}), "#compdef asc") {
		t.Fatalf("zsh script missing compdef header")
	}
	if !strings.Contains(fishScript([]string{"apps"}), "complete -c asc") {
		t.Fatalf("fish script missing completion command")
	}
}

func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	os.Stdout = w

	var runErr error
	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	runErr = fn()
	_ = w.Close()
	<-done
	os.Stdout = orig
	_ = r.Close()

	if runErr != nil {
		t.Fatalf("unexpected command error: %v", runErr)
	}
	return buf.String()
}
