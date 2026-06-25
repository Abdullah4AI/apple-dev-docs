package shared

import (
	"fmt"
	"net/url"
	"strings"
)

// MergeNextURLQuery reapplies required query parameters to a validated next URL.
func MergeNextURLQuery(next string, additions url.Values) (string, error) {
	next = strings.TrimSpace(next)
	if next == "" {
		return "", nil
	}

	parsed, err := url.Parse(next)
	if err != nil {
		return "", err
	}
	if parsed.IsAbs() {
		if err := validateNextURL(next); err != nil {
			return "", err
		}
	} else if parsed.Host != "" || strings.HasPrefix(next, "//") {
		return "", fmt.Errorf("relative next URL must not specify a host")
	}

	query := parsed.Query()
	for key, values := range additions {
		query.Del(key)
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			query.Add(key, value)
		}
	}

	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
