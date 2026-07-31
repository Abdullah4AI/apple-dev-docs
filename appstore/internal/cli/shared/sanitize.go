package shared

import "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"

// SanitizeTerminal removes characters interpreted by terminals and log viewers.
func SanitizeTerminal(input string) string {
	return asc.SanitizeTerminalText(input)
}
