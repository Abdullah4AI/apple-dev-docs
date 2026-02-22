package shared

import (
	"fmt"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

var platformValues = map[string]asc.Platform{
	"IOS":       asc.PlatformIOS,
	"MAC_OS":    asc.PlatformMacOS,
	"TV_OS":     asc.PlatformTVOS,
	"VISION_OS": asc.PlatformVisionOS,
}

// NormalizePlatform validates and normalizes a platform string.
func NormalizePlatform(value string) (asc.Platform, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", fmt.Errorf("--platform is required")
	}
	platform, ok := platformValues[normalized]
	if !ok {
		return "", fmt.Errorf("--platform must be one of: %s", strings.Join(platformList(), ", "))
	}
	return platform, nil
}

// NormalizePlatforms validates and normalizes multiple platform strings.
func NormalizePlatforms(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.ToUpper(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}
		if _, ok := platformValues[trimmed]; !ok {
			return nil, fmt.Errorf("--platform must be one of: %s", strings.Join(platformList(), ", "))
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	return normalized, nil
}

// PlatformList returns the allowed platform values.
func PlatformList() []string {
	return platformList()
}

func platformList() []string {
	return []string{"IOS", "MAC_OS", "TV_OS", "VISION_OS"}
}
