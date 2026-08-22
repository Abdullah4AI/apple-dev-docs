package shared

import "strings"

// RejectPositionalArgs rejects operands for commands whose inputs are flags only.
func RejectPositionalArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}

	sanitized := make([]string, len(args))
	for i, arg := range args {
		sanitized[i] = SanitizeTerminal(arg)
	}
	return UsageErrorf("unexpected argument(s): %s", strings.Join(sanitized, " "))
}
