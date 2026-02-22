//go:build integration

package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	originalBypass, hadBypass := os.LookupEnv("ASC_BYPASS_KEYCHAIN")
	originalConfig, hadConfig := os.LookupEnv("ASC_CONFIG_PATH")

	tempDir := ""
	if !hadConfig {
		dir, err := os.MkdirTemp("", "asc-auth-integration-*")
		if err != nil {
			panic(err)
		}
		tempDir = dir
		_ = os.Setenv("ASC_CONFIG_PATH", filepath.Join(dir, "config.json"))
	}
	_ = os.Setenv("ASC_BYPASS_KEYCHAIN", "1")

	code := m.Run()

	if hadBypass {
		_ = os.Setenv("ASC_BYPASS_KEYCHAIN", originalBypass)
	} else {
		_ = os.Unsetenv("ASC_BYPASS_KEYCHAIN")
	}
	if hadConfig {
		_ = os.Setenv("ASC_CONFIG_PATH", originalConfig)
	} else {
		_ = os.Unsetenv("ASC_CONFIG_PATH")
	}
	if tempDir != "" {
		_ = os.RemoveAll(tempDir)
	}

	os.Exit(code)
}
