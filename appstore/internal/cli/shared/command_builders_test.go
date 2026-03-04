package shared

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

type testPaginatedResponse struct{}

func (r *testPaginatedResponse) GetLinks() *asc.Links {
	return &asc.Links{}
}

func (r *testPaginatedResponse) GetData() any {
	return nil
}

func TestBuildIDGetCommand_MissingIDReturnsUsageError(t *testing.T) {
	cmd := BuildIDGetCommand(IDGetCommandConfig{
		FlagSetName: "test-id-get",
		Name:        "get",
		ShortUsage:  "test get",
		ShortHelp:   "test",
		ErrorPrefix: "test get",
		Fetch:       func(context.Context, *asc.Client, string) (any, error) { return nil, nil },
		ContextTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return context.WithCancel(ctx)
		},
	})

	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = cmd.Exec(context.Background(), nil)
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "Error: --id is required") {
		t.Fatalf("expected missing id usage error, got %q", stderr)
	}
}

func TestBuildPaginatedListCommand_MissingParentIDReturnsUsageError(t *testing.T) {
	cmd := BuildPaginatedListCommand(PaginatedListCommandConfig{
		FlagSetName: "test-list",
		Name:        "list",
		ShortUsage:  "test list",
		ShortHelp:   "test",
		ParentFlag:  "app-id",
		ErrorPrefix: "test list",
		FetchPage: func(context.Context, *asc.Client, string, int, string) (asc.PaginatedResponse, error) {
			return &testPaginatedResponse{}, nil
		},
		ContextTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return context.WithCancel(ctx)
		},
	})

	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = cmd.Exec(context.Background(), nil)
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "Error: --app-id is required") {
		t.Fatalf("expected missing app-id usage error, got %q", stderr)
	}
}

func TestBuildConfirmDeleteCommand_MissingConfirmReturnsUsageError(t *testing.T) {
	cmd := BuildConfirmDeleteCommand(ConfirmDeleteCommandConfig{
		FlagSetName: "test-delete",
		Name:        "delete",
		ShortUsage:  "test delete",
		ShortHelp:   "test",
		ErrorPrefix: "test delete",
		Delete:      func(context.Context, *asc.Client, string) error { return nil },
		Result:      func(string) any { return map[string]string{"status": "ok"} },
		ContextTimeout: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return context.WithCancel(ctx)
		},
	})

	if err := cmd.FlagSet.Parse([]string{"--id", "abc"}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = cmd.Exec(context.Background(), nil)
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "Error: --confirm is required") {
		t.Fatalf("expected missing confirm usage error, got %q", stderr)
	}
}
