package validate

import (
	"context"
	"errors"
	"fmt"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

var fetchAppBuildCountFn = fetchAppBuildCount

func fetchAppBuildCount(ctx context.Context, client *asc.Client, appID string) (int, metadataCheckStatus, error) {
	ctx = withReadinessRequestGate(ctx)
	resp, err := doReadinessRequest(ctx, func(requestCtx context.Context) (*asc.BuildsResponse, error) {
		return client.GetBuilds(requestCtx, appID)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return 0, metadataCheckStatus{}, err
		}
		if reason, ok := metadataCheckSkipReason(err, "app builds"); ok {
			return 0, metadataCheckStatus{SkipReason: reason}, nil
		}
		return 0, metadataCheckStatus{}, fmt.Errorf("failed to fetch app builds: %w", err)
	}

	if total, ok := asc.ParsePagingTotalOK(resp.Meta); ok {
		return total, metadataCheckStatus{Verified: true}, nil
	}

	return len(resp.Data), metadataCheckStatus{Verified: true}, nil
}
