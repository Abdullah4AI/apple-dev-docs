package asc

import (
	"strings"
	"unicode/utf8"
)

// SanitizeTerminalText removes characters that a terminal, pager, or CI log
// viewer interprets instead of displaying. Structured output (JSON) keeps the
// original values; only human-facing table, Markdown, stdout, and stderr text
// is sanitized.
//
// Removed or replaced characters:
//   - C0 controls (U+0000-U+001F) and DEL (U+007F), which include the ESC
//     introducer used by CSI/OSC sequences as well as BEL, CR, LF, and TAB
//   - C1 controls (U+0080-U+009F), which include the single-byte CSI (U+009B)
//     and OSC (U+009D) forms
//   - Bidirectional marks, embeddings, overrides, and isolates (U+061C, U+200E,
//     U+200F, U+202A-U+202E, U+2066-U+2069), which can visually reorder a
//     rendered cell
//   - Unicode line and paragraph separators (U+2028, U+2029), which
//     browser-backed log and Markdown viewers render as mandatory breaks
//
// Line breaks, tabs, and Unicode separators are replaced with ordinary spaces
// so a single value cannot forge extra rows or columns without concatenating
// the text on either side.
//
// Invalid UTF-8 bytes are replaced with U+FFFD. This preserves the fact that
// text was separated at that byte boundary while ensuring a raw 0x9B or 0x9D
// cannot act as a single-byte CSI or OSC introducer.
func SanitizeTerminalText(input string) string {
	if input == "" || !HasInterpretedTerminalSequence(input) {
		return input
	}

	var b strings.Builder
	b.Grow(len(input))
	for i := 0; i < len(input); {
		r, size := utf8.DecodeRuneInString(input[i:])
		if r == utf8.RuneError && size == 1 {
			b.WriteRune(utf8.RuneError)
			i += size
			continue
		}
		if isTerminalTextSeparator(r) {
			b.WriteByte(' ')
			i += size
			continue
		}
		if isInterpretedTerminalRune(r) {
			i += size
			continue
		}
		b.WriteString(input[i : i+size])
		i += size
	}
	return b.String()
}

// HasInterpretedTerminalSequence reports whether input contains any character
// that SanitizeTerminalText removes.
func HasInterpretedTerminalSequence(input string) bool {
	for i := 0; i < len(input); {
		r, size := utf8.DecodeRuneInString(input[i:])
		if (r == utf8.RuneError && size == 1) || isInterpretedTerminalRune(r) {
			return true
		}
		i += size
	}
	return false
}

func isTerminalTextSeparator(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', '\u0085', '\u2028', '\u2029':
		return true
	default:
		return false
	}
}

func isInterpretedTerminalRune(r rune) bool {
	switch {
	case r < 0x20, r == 0x7f:
		return true
	case r >= 0x80 && r <= 0x9f:
		return true
	case r == 0x061c:
		return true
	case r == 0x200e, r == 0x200f:
		return true
	case r >= 0x202a && r <= 0x202e:
		return true
	case r == 0x2028, r == 0x2029:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	default:
		return false
	}
}
