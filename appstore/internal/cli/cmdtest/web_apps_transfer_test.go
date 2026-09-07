package cmdtest

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestWebAppsTransferStatusRequiresApp(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	root := RootCommand("test")
	if findSubcommand(root, "web", "apps", "transfer", "status") == nil {
		t.Fatal("transfer status is not registered")
	}
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"web", "apps", "transfer", "status"}); err != nil {
			t.Fatalf("parse: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if !errors.Is(runErr, flag.ErrHelp) || !strings.Contains(stderr, "--app is required") || stdout != "" {
		t.Fatalf("err=%v stdout=%q stderr=%q", runErr, stdout, stderr)
	}
}
