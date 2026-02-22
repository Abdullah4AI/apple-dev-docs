package testutil

import (
	"testing"
)

func TestSkipUnlessIntegration_Skips(t *testing.T) {
	t.Setenv("ASC_INTEGRATION_TEST", "")
	if isIntegrationEnabled() {
		t.Fatal("expected integration to be disabled when env is empty")
	}
}

func TestSkipUnlessIntegration_EnabledTrue(t *testing.T) {
	t.Setenv("ASC_INTEGRATION_TEST", "true")
	if !isIntegrationEnabled() {
		t.Fatal("expected integration to be enabled when env is 'true'")
	}
}

func TestSkipUnlessIntegration_Enabled1(t *testing.T) {
	t.Setenv("ASC_INTEGRATION_TEST", "1")
	if !isIntegrationEnabled() {
		t.Fatal("expected integration to be enabled when env is '1'")
	}
}

func TestSkipUnlessIntegration_EnabledYes(t *testing.T) {
	t.Setenv("ASC_INTEGRATION_TEST", "yes")
	if !isIntegrationEnabled() {
		t.Fatal("expected integration to be enabled when env is 'yes'")
	}
}

func TestSkipUnlessIntegration_DisabledFalse(t *testing.T) {
	t.Setenv("ASC_INTEGRATION_TEST", "false")
	if isIntegrationEnabled() {
		t.Fatal("expected integration to be disabled when env is 'false'")
	}
}

func TestSkipUnlessIntegration_CaseInsensitive(t *testing.T) {
	t.Setenv("ASC_INTEGRATION_TEST", "TRUE")
	if !isIntegrationEnabled() {
		t.Fatal("expected integration to be enabled when env is 'TRUE'")
	}
}
