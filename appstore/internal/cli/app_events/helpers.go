package app_events

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

const appEventAssetUploadDefaultTimeout = 10 * time.Minute

func normalizeAppEventBadge(value string, required bool) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		if required {
			return "", fmt.Errorf("--event-type is required")
		}
		return "", nil
	}
	if slices.Contains(asc.ValidAppEventBadges, normalized) {
		return normalized, nil
	}
	return "", fmt.Errorf("--event-type must be one of: %s", strings.Join(asc.ValidAppEventBadges, ", "))
}

func normalizeAppEventPriority(value string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", nil
	}
	if slices.Contains(asc.ValidAppEventPriorities, normalized) {
		return normalized, nil
	}
	return "", fmt.Errorf("--priority must be one of: %s", strings.Join(asc.ValidAppEventPriorities, ", "))
}

func normalizeAppEventPurpose(value string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", nil
	}
	if slices.Contains(asc.ValidAppEventPurposes, normalized) {
		return normalized, nil
	}
	return "", fmt.Errorf("--purpose must be one of: %s", strings.Join(asc.ValidAppEventPurposes, ", "))
}

func normalizeAppEventAssetType(value string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", fmt.Errorf("--asset-type is required")
	}
	if slices.Contains(asc.ValidAppEventAssetTypes, normalized) {
		return normalized, nil
	}
	return "", fmt.Errorf("--asset-type must be one of: %s", strings.Join(asc.ValidAppEventAssetTypes, ", "))
}

func normalizeRFC3339(value, flagName string, required bool) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if required {
			return "", fmt.Errorf("%s is required", flagName)
		}
		return "", nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return "", fmt.Errorf("%s must be in RFC3339 format", flagName)
	}
	return parsed.Format(time.RFC3339), nil
}

func buildAppEventTerritorySchedule(territories []string, publishStart, start, end string) asc.AppEventTerritorySchedule {
	schedule := asc.AppEventTerritorySchedule{
		EventStart: start,
		EventEnd:   end,
	}
	if len(territories) > 0 {
		schedule.Territories = territories
	}
	if strings.TrimSpace(publishStart) != "" {
		schedule.PublishStart = publishStart
	}
	return schedule
}

func resolveAppEventLocalizationID(ctx context.Context, client *asc.Client, eventID, localizationID, locale string) (string, error) {
	localizationID = strings.TrimSpace(localizationID)
	if localizationID != "" {
		return localizationID, nil
	}
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return "", fmt.Errorf("--event-id is required")
	}
	locale = strings.TrimSpace(locale)
	if locale == "" {
		event, err := client.GetAppEvent(ctx, eventID)
		if err != nil {
			return "", err
		}
		locale = strings.TrimSpace(event.Data.Attributes.PrimaryLocale)
	}
	if locale == "" {
		return "", fmt.Errorf("no locale resolved for app event %q (use --locale or --localization-id)", eventID)
	}

	resp, err := client.GetAppEventLocalizations(ctx, eventID, asc.WithAppEventLocalizationsLimit(200))
	if err != nil {
		return "", err
	}
	for _, localization := range resp.Data {
		if strings.EqualFold(localization.Attributes.Locale, locale) {
			return localization.ID, nil
		}
	}
	return "", fmt.Errorf("no localization found for locale %q (use --localization-id to specify)", locale)
}

func openAssetFile(path string) (*os.File, os.FileInfo, error) {
	if err := asc.ValidateAssetFile(path); err != nil {
		return nil, nil, err
	}
	file, err := shared.OpenExistingNoFollow(path)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	return file, info, nil
}

func contextWithAssetUploadTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, asc.ResolveTimeoutWithDefault(appEventAssetUploadDefaultTimeout))
}
