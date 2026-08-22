package productpages

import (
	"context"
	"fmt"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func screenshotSetListResult(ctx context.Context, client *asc.Client, localizationID string, response *asc.AppScreenshotSetsResponse) (*asc.AppScreenshotSetListResult, error) {
	result := &asc.AppScreenshotSetListResult{
		LocalizationID: localizationID,
		Sets:           make([]asc.AppScreenshotSetWithScreenshots, 0, len(response.Data)),
	}
	for _, set := range response.Data {
		requestCtx, cancel := shared.ContextWithTimeout(ctx)
		screenshots, err := client.GetAppScreenshots(requestCtx, set.ID)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch screenshots for set %s: %w", set.ID, err)
		}
		result.Sets = append(result.Sets, asc.AppScreenshotSetWithScreenshots{
			Set:         set,
			Screenshots: screenshots.Data,
		})
	}
	return result, nil
}
