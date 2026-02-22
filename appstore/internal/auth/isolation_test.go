//go:build !integration

package auth

import (
	"os"
	"strings"
	"testing"
)

func TestAuthTestIsolationConfigPathSet(t *testing.T) {
	path := strings.TrimSpace(os.Getenv("ASC_CONFIG_PATH"))
	if path == "" {
		t.Fatal("ASC_CONFIG_PATH must be set for auth tests")
	}
	if testConfigPath == "" {
		t.Fatal("testConfigPath must be set by TestMain")
	}
	if path != testConfigPath {
		t.Fatalf("expected ASC_CONFIG_PATH %q, got %q", testConfigPath, path)
	}
}
