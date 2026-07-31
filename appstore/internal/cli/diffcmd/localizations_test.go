package diffcmd

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestSanitizeDiffCellPreservesUTF8WhenUnderRuneLimit(t *testing.T) {
	input := strings.Repeat("本", 30)

	got := sanitizeDiffCell(input)

	if got != input {
		t.Fatalf("expected value to be unchanged when under rune limit, got %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("expected sanitized value to remain valid UTF-8")
	}
}

func TestSanitizeDiffCellTruncatesOnRuneBoundary(t *testing.T) {
	input := strings.Repeat("本", 100)

	got := sanitizeDiffCell(input)

	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated value to end with ellipsis, got %q", got)
	}
	if len([]rune(got)) != 80 {
		t.Fatalf("expected truncated value to be 80 runes including ellipsis, got %d", len([]rune(got)))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("expected truncated value to remain valid UTF-8")
	}
}

func TestSanitizeDiffCellTruncatesMixedASCIIAndUnicode(t *testing.T) {
	input := strings.Repeat("a", 70) + strings.Repeat("本", 20)

	got := sanitizeDiffCell(input)

	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected truncated value to end with ellipsis, got %q", got)
	}
	if len([]rune(got)) != 80 {
		t.Fatalf("expected truncated value to be 80 runes including ellipsis, got %d", len([]rune(got)))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("expected truncated value to remain valid UTF-8")
	}
}

func TestSanitizeDiffCellEscapesNewlinesBeforeTruncation(t *testing.T) {
	input := strings.Repeat("line\n", 30)

	got := sanitizeDiffCell(input)

	if !strings.Contains(got, "\\n") {
		t.Fatalf("expected newline characters to be escaped, got %q", got)
	}
	if strings.Contains(got, "\n") {
		t.Fatalf("expected no raw newline characters in output, got %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("expected sanitized value to remain valid UTF-8")
	}
}

func TestSanitizeDiffCellRemovesTerminalControls(t *testing.T) {
	for _, input := range []string{
		"promo\x1b[31mtext",
		"promo\x1b]0;pwned\x07text",
		"promo\u009b2Ktext",
		"promo\u202etxet",
		"promo\u2066text\u2069",
		"promo\rtext\u007f",
	} {
		got := sanitizeDiffCell(input)
		if asc.HasInterpretedTerminalSequence(got) {
			t.Fatalf("sanitizeDiffCell(%q) = %q, want no interpreted terminal sequences", input, got)
		}
	}
}

func TestSanitizeDiffCellRemovesControlsBeforeTruncation(t *testing.T) {
	// A control sequence at the truncation boundary must not survive as a
	// half-written escape.
	input := strings.Repeat("a", 78) + "\x1b[31m" + strings.Repeat("b", 20)

	got := sanitizeDiffCell(input)

	if asc.HasInterpretedTerminalSequence(got) {
		t.Fatalf("sanitizeDiffCell(...) = %q, want no interpreted terminal sequences", got)
	}
	if len([]rune(got)) != 80 {
		t.Fatalf("expected 80 runes after truncation, got %d (%q)", len([]rune(got)), got)
	}
}

func TestLocalizationDiffRowsRemoveTerminalControlsAndJSONKeepsOriginals(t *testing.T) {
	hostile := "New\x1b]0;pwned\x07 features\u202e"
	plan := buildLocalizationDiffPlan(
		"123456789",
		localizationDiffEndpoint{Kind: "local", Path: "./metadata"},
		localizationDiffEndpoint{Kind: "remote", VersionID: "VERSION_ID"},
		map[string]map[string]string{"en-US": {"whatsNew": "Bug fixes"}},
		map[string]map[string]string{"en-US": {"whatsNew": hostile}},
	)

	for _, row := range buildLocalizationDiffRows(plan) {
		for i, cell := range row {
			if asc.HasInterpretedTerminalSequence(cell) {
				t.Fatalf("diff row column %d = %q contains interpreted terminal sequences", i, cell)
			}
		}
	}

	payload, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	var decoded localizationDiffPlan
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal plan: %v", err)
	}
	if len(decoded.Updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(decoded.Updates))
	}
	if decoded.Updates[0].To != hostile {
		t.Fatalf("JSON update value = %q, want the original %q", decoded.Updates[0].To, hostile)
	}
}

func TestLocalizationDiffRowsSanitizeLocaleAndKey(t *testing.T) {
	hostileLocale := "en-US\x1b[31m\u202e"
	plan := buildLocalizationDiffPlan(
		"123456789",
		localizationDiffEndpoint{Kind: "local", Path: "./metadata"},
		localizationDiffEndpoint{Kind: "remote", VersionID: "VERSION_ID"},
		map[string]map[string]string{hostileLocale: {"whatsNew": "Bug fixes"}},
		map[string]map[string]string{hostileLocale: {"whatsNew": "New features"}},
	)

	rows := buildLocalizationDiffRows(plan)
	if len(rows) == 0 {
		t.Fatalf("expected at least one diff row")
	}
	for _, row := range rows {
		for i, cell := range row {
			if asc.HasInterpretedTerminalSequence(cell) {
				t.Fatalf("diff row column %d = %q contains interpreted terminal sequences", i, cell)
			}
		}
	}
}
