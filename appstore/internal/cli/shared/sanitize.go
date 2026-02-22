package shared

import "strings"

// SanitizeTerminal strips ASCII control characters to prevent terminal escape injection.
func SanitizeTerminal(input string) string {
	if input == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(input))
	for _, r := range input {
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
