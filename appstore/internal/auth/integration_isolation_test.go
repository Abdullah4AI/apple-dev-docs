//go:build integration

package auth

import (
	"os"
	"strings"
	"testing"
)

func TestIntegrationBypassesKeychain(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ASC_BYPASS_KEYCHAIN")) != "1" {
		t.Fatal("ASC_BYPASS_KEYCHAIN must be set to 1 for integration tests")
	}
	if strings.TrimSpace(os.Getenv("ASC_CONFIG_PATH")) == "" {
		t.Fatal("ASC_CONFIG_PATH must be set for integration tests")
	}
}
