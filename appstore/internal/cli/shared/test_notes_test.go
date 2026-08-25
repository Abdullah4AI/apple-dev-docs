package shared

import (
	"errors"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestNewTestNotesRecoveryErrorSeparatesHumanAndMachineRecovery(t *testing.T) {
	cause := errors.New("server rejected notes\x1b[31m\nforged line")
	buildID := `build 'quoted' $(touch build)`
	locale := `en-US; touch locale`
	notes := "First line; $(touch notes)\nIt's still quoted\x1b[0m"

	err := NewTestNotesRecoveryError(buildID, locale, notes, cause)
	if !errors.Is(err, cause) {
		t.Fatalf("expected recovery error to wrap cause, got %v", err)
	}
	if asc.HasInterpretedTerminalSequence(err.Error()) {
		t.Fatalf("human recovery error contains terminal controls: %q", err)
	}
	if strings.Contains(err.Error(), "First line") {
		t.Fatalf("human recovery error must not embed notes: %q", err)
	}
	wantHumanParts := []string{
		"retry without uploading the build again",
		"reuse the original notes",
		"asc builds test-notes create --build-id BUILD_ID --locale LOCALE --whats-new NOTES",
	}
	for _, want := range wantHumanParts {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("human recovery error missing %q: %q", want, err)
		}
	}

	recovery := err.Recovery()
	if recovery.BuildID != buildID || recovery.Locale != locale || recovery.SubmittedNotes != notes {
		t.Fatalf("recovery fields lost exact values: %#v", recovery)
	}
	if recovery.Command != "asc" {
		t.Fatalf("recovery command = %q, want asc", recovery.Command)
	}
	want := []string{
		"builds", "test-notes", "create",
		"--build-id", buildID,
		"--locale", locale,
		"--whats-new", notes,
	}
	if len(recovery.Arguments) != len(want) {
		t.Fatalf("retry args = %#v, want %#v", recovery.Arguments, want)
	}
	for i := range want {
		if recovery.Arguments[i] != want[i] {
			t.Fatalf("retry arg %d = %q, want %q; all args=%#v", i, recovery.Arguments[i], want[i], recovery.Arguments)
		}
	}
}

func TestNewTestNotesRecoveryErrorDoesNotEchoNotesFromServerDetail(t *testing.T) {
	notes := "First line\nSecond line\x1b[31m"
	cause := errors.New("server rejected value: " + notes)

	err := NewTestNotesRecoveryError("build-1", "en-US", notes, cause)
	human := err.Error()
	if !strings.Contains(human, "server rejected value: (original notes omitted)") {
		t.Fatalf("human recovery error lost the non-sensitive server diagnostic: %q", human)
	}
	if strings.Contains(human, "First line") || strings.Contains(human, "Second line") || strings.Contains(human, "[31m") {
		t.Fatalf("human recovery error must not echo submitted notes from server detail: %q", human)
	}
}

func TestNewTestNotesRecoveryErrorRedactsExactEchoesWithoutCorruptingDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		notes  string
		cause  string
		redact bool
	}{
		{
			name:   "short exact echo",
			notes:  "invalid",
			cause:  "invalid",
			redact: true,
		},
		{
			name:   "short quoted echo",
			notes:  "invalid",
			cause:  `value "invalid" is not accepted`,
			redact: true,
		},
		{
			name:  "short diagnostic word",
			notes: "invalid",
			cause: "invalid attribute",
		},
		{
			name:  "single-letter substring collision",
			notes: "a",
			cause: "request failed while validating an attribute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTestNotesRecoveryError("build-1", "en-US", tt.notes, errors.New(tt.cause))
			human := err.Error()
			gotRedaction := strings.Contains(human, "(original notes omitted)")
			if gotRedaction != tt.redact {
				t.Fatalf("redaction = %t, want %t: %q", gotRedaction, tt.redact, human)
			}
			if !tt.redact && !strings.Contains(human, tt.cause) {
				t.Fatalf("normal diagnostic was rewritten: %q", human)
			}
		})
	}
}
