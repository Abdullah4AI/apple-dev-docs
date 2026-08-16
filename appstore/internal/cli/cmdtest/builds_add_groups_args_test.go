package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildsAddGroupsRejectsUnexpectedArgsBeforeNetwork(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s %s", req.Method, req.URL.String())
		return nil, nil
	})

	root := RootCommand("dev")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{
		"builds", "add-groups",
		"--build-id", "build-1",
		"--group", "group-1",
		"unexpected",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var runErr error
	_, stderr := captureOutput(t, func() {
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) ||
		runErr.Error() != "unexpected argument(s): unexpected" ||
		!strings.Contains(stderr, "unexpected argument(s): unexpected") {
		t.Fatalf("run error = %v stderr = %q, want unexpected argument usage error", runErr, stderr)
	}
}
