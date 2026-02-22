package cmdtest

import (
	"os"
	"strings"
	"testing"
)

func TestCmdtestIsolationEnvSet(t *testing.T) {
	path := strings.TrimSpace(os.Getenv("ASC_CONFIG_PATH"))
	if path == "" {
		t.Fatal("ASC_CONFIG_PATH must be set for cmdtest")
	}
	if testConfigPath == "" {
		t.Fatal("testConfigPath must be set by TestMain")
	}
	if path != testConfigPath {
		t.Fatalf("expected ASC_CONFIG_PATH %q, got %q", testConfigPath, path)
	}
	if strings.TrimSpace(os.Getenv("ASC_BYPASS_KEYCHAIN")) != "1" {
		t.Fatal("ASC_BYPASS_KEYCHAIN must be set to 1 for cmdtest")
	}
}
