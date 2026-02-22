package diffcmd

import (
	"strings"
	"testing"
	"unicode/utf8"
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
