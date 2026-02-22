package crashes

import (
	"context"
	"errors"
	"flag"
	"testing"
)

func TestCrashesCommand_MissingApp(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	cmd := CrashesCommand()

	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

func TestCrashesCommand_InvalidLimit(t *testing.T) {
	cmd := CrashesCommand()

	if err := cmd.FlagSet.Parse([]string{"--limit", "201", "--app", "123"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected validation error for --limit, got %v", err)
	}
}

func TestCrashesCommand_InvalidSort(t *testing.T) {
	cmd := CrashesCommand()

	if err := cmd.FlagSet.Parse([]string{"--sort", "invalid", "--app", "123"}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}
	err := cmd.Exec(context.Background(), nil)
	if err == nil || errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected sort validation error, got %v", err)
	}
}

func TestCommandWrapper(t *testing.T) {
	if got := CrashesCommand(); got == nil {
		t.Fatal("expected Command wrapper to return a command")
	}
}
