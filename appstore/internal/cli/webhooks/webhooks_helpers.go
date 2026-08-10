package webhooks

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func normalizeWebhookEvents(value string) ([]asc.WebhookEventType, error) {
	values := shared.SplitCSV(value)
	if len(values) == 0 {
		return nil, fmt.Errorf("--events must include at least one value")
	}

	normalized := make([]asc.WebhookEventType, 0, len(values))
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		normalizedValue := strings.ToUpper(trimmed)
		if !slices.Contains(asc.ValidWebhookEventTypes, normalizedValue) {
			return nil, fmt.Errorf("invalid --events value %q; must be one of: %s", trimmed, strings.Join(asc.ValidWebhookEventTypes, ", "))
		}
		normalized = append(normalized, asc.WebhookEventType(normalizedValue))
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf("--events must include at least one value")
	}

	return normalized, nil
}

func extractWebhookIDFromNextURL(nextURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(nextURL))
	if err != nil {
		return "", fmt.Errorf("invalid --next URL")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 5 || parts[0] != "v1" || parts[1] != "webhooks" || parts[3] != "relationships" || parts[4] != "deliveries" {
		return "", fmt.Errorf("invalid --next URL")
	}
	if strings.TrimSpace(parts[2]) == "" {
		return "", fmt.Errorf("invalid --next URL")
	}
	return parts[2], nil
}
