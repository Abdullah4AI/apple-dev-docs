package migrate

import (
	"fmt"
	"regexp"
	"strings"
)

var localeValidationRegex = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]+)*$`)

func normalizeLocale(locale string) (string, error) {
	trimmed := strings.TrimSpace(locale)
	if trimmed == "" {
		return "", fmt.Errorf("locale is empty")
	}
	normalized := strings.ReplaceAll(trimmed, "_", "-")
	parts := strings.Split(normalized, "-")
	for i, part := range parts {
		if part == "" {
			return "", fmt.Errorf("locale %q is invalid", locale)
		}
		switch {
		case i == 0:
			parts[i] = strings.ToLower(part)
		case len(part) == 2 || len(part) == 3:
			parts[i] = strings.ToUpper(part)
		case len(part) == 4:
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		default:
			parts[i] = strings.ToLower(part)
		}
	}
	normalized = strings.Join(parts, "-")
	if !localeValidationRegex.MatchString(normalized) {
		return "", fmt.Errorf("locale %q must match pattern like 'en', 'en-US', or 'zh-Hans'", locale)
	}
	return normalized, nil
}
