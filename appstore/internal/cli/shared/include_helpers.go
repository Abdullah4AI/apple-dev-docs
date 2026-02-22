package shared

import (
	"fmt"
	"slices"
	"strings"
)

// HasInclude returns true when include is present in values.
func HasInclude(values []string, include string) bool {
	return slices.Contains(values, include)
}

// NormalizeSelection validates comma-separated values against an allow-list.
func NormalizeSelection(value string, allowed []string, flagName string) ([]string, error) {
	values := SplitCSV(value)
	if len(values) == 0 {
		return nil, nil
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, item := range allowed {
		allowedSet[item] = struct{}{}
	}
	for _, item := range values {
		if _, ok := allowedSet[item]; !ok {
			return nil, fmt.Errorf("%s must be one of: %s", flagName, strings.Join(allowed, ", "))
		}
	}

	return values, nil
}
