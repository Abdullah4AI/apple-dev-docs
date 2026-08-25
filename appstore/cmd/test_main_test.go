package cmd

import (
	"os"
	"testing"
)

var testTempDir string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "asc-cmd-*")
	if err != nil {
		panic(err)
	}
	testTempDir = tempDir

	_ = os.Setenv("ASC_TELEMETRY_DISABLED", "1")
	// Classification tests assert on terminal HTTP failures, so keep clients on
	// the single-attempt path instead of waiting out retry backoff.
	_ = os.Setenv("ASC_MAX_RETRIES", "0")
	code := m.Run()

	_ = os.RemoveAll(tempDir)
	os.Exit(code)
}
