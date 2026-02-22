package shared

import (
	"context"
	"fmt"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

// UpsertBetaBuildLocalization creates or updates a beta build localization.
func UpsertBetaBuildLocalization(ctx context.Context, client *asc.Client, buildID, locale, notes string) (*asc.BetaBuildLocalizationResponse, error) {
	localeValue := strings.TrimSpace(locale)
	notesValue := strings.TrimSpace(notes)
	if localeValue == "" || notesValue == "" {
		return nil, fmt.Errorf("locale and notes are required")
	}

	resp, err := client.GetBetaBuildLocalizations(ctx, buildID,
		asc.WithBetaBuildLocalizationLocales([]string{localeValue}),
		asc.WithBetaBuildLocalizationsLimit(200),
	)
	if err != nil {
		return nil, err
	}

	if resp != nil && len(resp.Data) > 0 {
		localizationID := strings.TrimSpace(resp.Data[0].ID)
		if localizationID == "" {
			return nil, fmt.Errorf("missing localization ID for locale %q", localeValue)
		}
		attrs := asc.BetaBuildLocalizationAttributes{
			WhatsNew: notesValue,
		}
		return client.UpdateBetaBuildLocalization(ctx, localizationID, attrs)
	}

	attrs := asc.BetaBuildLocalizationAttributes{
		Locale:   localeValue,
		WhatsNew: notesValue,
	}
	return client.CreateBetaBuildLocalization(ctx, buildID, attrs)
}
