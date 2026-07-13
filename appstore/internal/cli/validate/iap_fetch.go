package validate

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/validation"
)

var fetchIAPsFn = fetchIAPs

func fetchIAPs(ctx context.Context, client *asc.Client, appID string) ([]validation.IAP, error) {
	ctx = withReadinessRequestGate(ctx)
	firstPage, err := doReadinessRequest(ctx, func(requestCtx context.Context) (*asc.InAppPurchasesV2Response, error) {
		return client.GetInAppPurchasesV2(requestCtx, strings.TrimSpace(appID), asc.WithIAPLimit(200))
	})
	if err != nil {
		return nil, err
	}

	paginated, err := asc.PaginateAll(ctx, firstPage, func(_ context.Context, nextURL string) (asc.PaginatedResponse, error) {
		return doReadinessRequest(ctx, func(requestCtx context.Context) (asc.PaginatedResponse, error) {
			return client.GetInAppPurchasesV2(requestCtx, strings.TrimSpace(appID), asc.WithIAPNextURL(nextURL))
		})
	})
	if err != nil {
		return nil, err
	}

	return mapIAPsResponse(paginated)
}

func mapIAPsResponse(paginated asc.PaginatedResponse) ([]validation.IAP, error) {
	if paginated == nil {
		return nil, fmt.Errorf("unexpected nil in-app purchases pagination response")
	}

	resp, ok := paginated.(*asc.InAppPurchasesV2Response)
	if !ok {
		return nil, fmt.Errorf("unexpected in-app purchases pagination response type: %T", paginated)
	}

	iaps := make([]validation.IAP, 0, len(resp.Data))
	for _, item := range resp.Data {
		attrs := item.Attributes
		iaps = append(iaps, validation.IAP{
			ID:        item.ID,
			Name:      attrs.Name,
			ProductID: attrs.ProductID,
			Type:      attrs.InAppPurchaseType,
			State:     attrs.State,
		})
	}
	sort.SliceStable(iaps, func(i, j int) bool {
		if iaps[i].ProductID != iaps[j].ProductID {
			return iaps[i].ProductID < iaps[j].ProductID
		}
		return iaps[i].ID < iaps[j].ID
	})

	return iaps, nil
}
