package itunes

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain keeps public client tests hermetic. Retry behavior resolves through
// asc config and environment, so pin a throwaway config path and disable
// retries by default; retry tests opt back in with t.Setenv.
func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "asc-itunes-*")
	if err != nil {
		panic(err)
	}

	_ = os.Setenv("ASC_CONFIG_PATH", filepath.Join(tempDir, "config.json"))
	_ = os.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	_ = os.Setenv("ASC_MAX_RETRIES", "0")

	code := m.Run()

	_ = os.RemoveAll(tempDir)
	os.Exit(code)
}
