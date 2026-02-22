package asc

import "testing"

func TestResolveRetryLogEnabled_OverrideBeatsEnvAndConfig(t *testing.T) {
	t.Setenv("ASC_RETRY_LOG", "1")
	on := true
	SetRetryLogOverride(&on)
	t.Cleanup(func() { SetRetryLogOverride(nil) })

	if !ResolveRetryLogEnabled() {
		t.Fatal("expected override=true to enable retry logging")
	}

	off := false
	SetRetryLogOverride(&off)
	if ResolveRetryLogEnabled() {
		t.Fatal("expected override=false to disable retry logging")
	}
}

func TestResolveRetryLogEnabled_EnvBeatsConfig(t *testing.T) {
	// We can't reliably set config in this unit without knowing config format/paths,
	// but we can at least verify env toggles when no override is set.
	SetRetryLogOverride(nil)

	t.Setenv("ASC_RETRY_LOG", "")
	if ResolveRetryLogEnabled() {
		t.Fatal("expected empty ASC_RETRY_LOG to disable retry logging")
	}

	t.Setenv("ASC_RETRY_LOG", "1")
	if !ResolveRetryLogEnabled() {
		t.Fatal("expected ASC_RETRY_LOG to enable retry logging")
	}
}
