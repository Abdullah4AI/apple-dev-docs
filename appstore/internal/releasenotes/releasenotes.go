package releasenotes

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Commit is a minimal commit representation used to generate release notes.
type Commit struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
}

// FormatNotes renders commits into a single notes string.
//
// Supported formats:
//   - plain: "- <subject>" bullet list
//   - markdown: currently the same as plain (safe to paste into Markdown fields)
func FormatNotes(commits []Commit, format string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "", "plain":
		// ok
	case "markdown":
		// ok
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	var b strings.Builder
	first := true
	for _, c := range commits {
		subject := strings.TrimSpace(c.Subject)
		if subject == "" {
			continue
		}
		if !first {
			b.WriteByte('\n')
		}
		first = false
		b.WriteString("- ")
		b.WriteString(subject)
	}
	return b.String(), nil
}

// TruncateNotes truncates notes to maxChars (in runes), attempting to keep whole lines.
// It returns the truncated notes and whether truncation occurred.
func TruncateNotes(notes string, maxChars int) (string, bool) {
	if maxChars < 0 {
		return notes, false
	}
	if maxChars == 0 {
		if notes == "" {
			return "", false
		}
		return "", true
	}
	if utf8.RuneCountInString(notes) <= maxChars {
		return notes, false
	}

	lines := strings.Split(notes, "\n")
	var b strings.Builder
	used := 0
	truncated := false

	for i, line := range lines {
		sep := ""
		if i > 0 {
			sep = "\n"
		}
		segment := sep + line
		segRunes := utf8.RuneCountInString(segment)

		if used+segRunes <= maxChars {
			b.WriteString(segment)
			used += segRunes
			continue
		}

		remaining := maxChars - used
		if remaining > 0 {
			b.WriteString(truncateRunes(segment, remaining))
		}
		truncated = true
		break
	}

	out := b.String()
	if utf8.RuneCountInString(out) > maxChars {
		out = truncateRunes(out, maxChars)
		truncated = true
	}
	return out, truncated
}

func truncateRunes(s string, n int) string {
	if n <= 0 || s == "" {
		return ""
	}
	// Walk rune boundaries and slice at the nth rune.
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}
