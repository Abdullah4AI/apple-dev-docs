package shared

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

// UpsertBetaBuildLocalization creates or updates a beta build localization.
func UpsertBetaBuildLocalization(ctx context.Context, client *asc.Client, buildID, locale, notes string) (*asc.BetaBuildLocalizationResponse, error) {
	localeValue := strings.TrimSpace(locale)
	notesValue := strings.TrimSpace(notes)
	if localeValue == "" || notesValue == "" {
		return nil, fmt.Errorf("locale and notes are required")
	}

	resp, err := client.GetBetaBuildLocalizations(
		ctx, buildID,
		asc.WithBetaBuildLocalizationsLimit(200),
	)
	if err != nil {
		return nil, err
	}

	localizationID := ""
	foundLocale := false
	if resp != nil {
		for _, localization := range resp.Data {
			if !strings.EqualFold(strings.TrimSpace(localization.Attributes.Locale), localeValue) {
				continue
			}
			foundLocale = true
			localizationID = strings.TrimSpace(localization.ID)
			break
		}
	}
	if foundLocale {
		if localizationID == "" {
			return nil, fmt.Errorf("missing localization ID for locale %q", localeValue)
		}
		attrs := asc.BetaBuildLocalizationAttributes{
			WhatsNew: notesValue,
		}
		return client.UpdateBetaBuildLocalization(ctx, localizationID, attrs)
	}

	attrs := asc.BetaBuildLocalizationAttributes{
		Locale:   localeValue,
		WhatsNew: notesValue,
	}
	return client.CreateBetaBuildLocalization(ctx, buildID, attrs)
}

// TestNotesRecoveryError preserves a discovered build and the exact retry
// arguments while keeping its human-facing diagnostic terminal-safe.
type TestNotesRecoveryError struct {
	buildID string
	locale  string
	notes   string
	cause   error
}

// NewTestNotesRecoveryError returns recovery context for a failed post-upload
// What to Test request.
func NewTestNotesRecoveryError(buildID, locale, notes string, cause error) *TestNotesRecoveryError {
	return &TestNotesRecoveryError{
		buildID: buildID,
		locale:  locale,
		notes:   notes,
		cause:   cause,
	}
}

func (e *TestNotesRecoveryError) Error() string {
	buildID := asc.SanitizeTerminalText(e.buildID)
	locale := asc.SanitizeTerminalText(e.locale)
	cause := "unknown error"
	if e.cause != nil {
		cause = redactTestNotes(asc.SanitizeTerminalText(e.cause.Error()), e.notes)
	}
	return fmt.Sprintf(
		"build %q is available, but setting What to Test notes for locale %q failed: %s; retry without uploading the build again and reuse the original notes: asc builds test-notes create --build-id BUILD_ID --locale LOCALE --whats-new NOTES",
		buildID,
		locale,
		cause,
	)
}

func redactTestNotes(message, notes string) string {
	const redacted = "(original notes omitted)"

	sanitizedNotes := asc.SanitizeTerminalText(notes)
	candidate := strings.TrimSpace(sanitizedNotes)
	if candidate == "" {
		return message
	}

	// A cause that consists of the submitted value is unambiguous, even for
	// short notes such as "a" or "invalid".
	if strings.TrimSpace(message) == candidate {
		return redacted
	}

	// Quoted values are also unambiguous. Keep the surrounding quotes so the
	// shape of Apple's diagnostic remains useful to the operator.
	for _, quote := range []string{`"`, `'`, "`"} {
		if strings.Contains(candidate, quote) {
			continue
		}
		message = strings.ReplaceAll(message, quote+candidate+quote, quote+redacted+quote)
	}

	// Terminal-sensitive notes can be echoed after sanitization turns their
	// separators into spaces. Redact only a whole phrase in that case; plain
	// short notes are intentionally not treated as secrets inside arbitrary
	// diagnostic text because there is no reliable way to distinguish an echo
	// from normal prose.
	if asc.HasInterpretedTerminalSequence(notes) && hasTestNotesBoundary(candidate) {
		message = replaceWholeTestNotes(message, candidate, redacted)
	}
	return message
}

func hasTestNotesBoundary(candidate string) bool {
	for _, r := range candidate {
		if unicode.IsSpace(r) || !isTestNotesWordRune(r) {
			return true
		}
	}
	return false
}

func isTestNotesWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r) || r == '_'
}

func replaceWholeTestNotes(message, candidate, replacement string) string {
	for offset := 0; offset < len(message); {
		relativeStart := strings.Index(message[offset:], candidate)
		if relativeStart < 0 {
			break
		}
		start := offset + relativeStart
		end := start + len(candidate)
		if testNotesBoundaryAt(message, start, end) {
			message = message[:start] + replacement + message[end:]
			offset = start + len(replacement)
			continue
		}
		offset = end
	}
	return message
}

func testNotesBoundaryAt(message string, start, end int) bool {
	if start > 0 {
		before, _ := utf8.DecodeLastRuneInString(message[:start])
		if isTestNotesWordRune(before) {
			return false
		}
	}
	if end < len(message) {
		after, _ := utf8.DecodeRuneInString(message[end:])
		if isTestNotesWordRune(after) {
			return false
		}
	}
	return true
}

// Unwrap preserves API error status and exit classification.
func (e *TestNotesRecoveryError) Unwrap() error {
	return e.cause
}

// Recovery returns exact, shell-neutral retry data for structured output.
func (e *TestNotesRecoveryError) Recovery() *asc.TestNotesRecovery {
	return &asc.TestNotesRecovery{
		BuildID:        e.buildID,
		Locale:         e.locale,
		SubmittedNotes: e.notes,
		Command:        "asc",
		Arguments: []string{
			"builds", "test-notes", "create",
			"--build-id", e.buildID,
			"--locale", e.locale,
			"--whats-new", e.notes,
		},
	}
}
