package validate

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/validation"
)

type subscriptionImageStatus struct {
	HasImage   bool
	Verified   bool
	SkipReason string
}

var fetchSubscriptionsFn = fetchSubscriptions

func fetchSubscriptions(ctx context.Context, client *asc.Client, appID string) ([]validation.Subscription, error) {
	groupsCtx, groupsCancel := shared.ContextWithTimeout(ctx)
	groupsResp, err := client.GetSubscriptionGroups(groupsCtx, appID, asc.WithSubscriptionGroupsLimit(200))
	groupsCancel()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscription groups: %w", err)
	}

	paginatedGroups, err := asc.PaginateAll(ctx, groupsResp, func(_ context.Context, nextURL string) (asc.PaginatedResponse, error) {
		pageCtx, pageCancel := shared.ContextWithTimeout(ctx)
		defer pageCancel()
		return client.GetSubscriptionGroups(pageCtx, appID, asc.WithSubscriptionGroupsNextURL(nextURL))
	})
	if err != nil {
		return nil, fmt.Errorf("paginate subscription groups: %w", err)
	}

	groups, ok := paginatedGroups.(*asc.SubscriptionGroupsResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected subscription groups response type %T", paginatedGroups)
	}

	subscriptions := make([]validation.Subscription, 0)
	for _, group := range groups.Data {
		groupID := strings.TrimSpace(group.ID)
		if groupID == "" {
			continue
		}

		subsCtx, subsCancel := shared.ContextWithTimeout(ctx)
		subsResp, err := client.GetSubscriptions(subsCtx, groupID, asc.WithSubscriptionsLimit(200))
		subsCancel()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch subscriptions for group %s: %w", groupID, err)
		}

		paginatedSubs, err := asc.PaginateAll(ctx, subsResp, func(_ context.Context, nextURL string) (asc.PaginatedResponse, error) {
			pageCtx, pageCancel := shared.ContextWithTimeout(ctx)
			defer pageCancel()
			return client.GetSubscriptions(pageCtx, groupID, asc.WithSubscriptionsNextURL(nextURL))
		})
		if err != nil {
			return nil, fmt.Errorf("paginate subscriptions: %w", err)
		}

		subsResult, ok := paginatedSubs.(*asc.SubscriptionsResponse)
		if !ok {
			return nil, fmt.Errorf("unexpected subscriptions response type %T", paginatedSubs)
		}

		for _, sub := range subsResult.Data {
			imageStatus, err := subscriptionHasImage(ctx, client, sub.ID)
			if err != nil {
				return nil, fmt.Errorf("fetch subscription images for %s: %w", strings.TrimSpace(sub.ID), err)
			}

			attrs := sub.Attributes
			subscriptions = append(subscriptions, validation.Subscription{
				ID:                   sub.ID,
				Name:                 attrs.Name,
				ProductID:            attrs.ProductID,
				State:                attrs.State,
				GroupID:              groupID,
				HasImage:             imageStatus.HasImage,
				ImageCheckSkipped:    !imageStatus.Verified,
				ImageCheckSkipReason: imageStatus.SkipReason,
			})
		}
	}

	return subscriptions, nil
}

func subscriptionHasImage(ctx context.Context, client *asc.Client, subscriptionID string) (subscriptionImageStatus, error) {
	requestCtx, cancel := shared.ContextWithTimeout(ctx)
	defer cancel()

	resp, err := client.GetSubscriptionImages(requestCtx, strings.TrimSpace(subscriptionID), asc.WithSubscriptionImagesLimit(1))
	if err != nil {
		if asc.IsNotFound(err) {
			return subscriptionImageStatus{Verified: true}, nil
		}
		if errors.Is(err, context.Canceled) {
			return subscriptionImageStatus{}, err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return subscriptionImageStatus{
				Verified:   false,
				SkipReason: "Image verification was skipped because the App Store Connect image endpoint timed out",
			}, nil
		}
		if errors.Is(err, asc.ErrForbidden) || asc.IsUnauthorized(err) {
			return subscriptionImageStatus{
				Verified:   false,
				SkipReason: "Image verification was skipped because this App Store Connect account cannot read subscription image assets",
			}, nil
		}
		if asc.IsRetryable(err) {
			return subscriptionImageStatus{
				Verified:   false,
				SkipReason: "Image verification was skipped because the App Store Connect image endpoint was temporarily unavailable or rate limited",
			}, nil
		}
		var netErr net.Error
		if errors.As(err, &netErr) {
			return subscriptionImageStatus{
				Verified:   false,
				SkipReason: "Image verification was skipped because the App Store Connect image endpoint could not be reached",
			}, nil
		}
		return subscriptionImageStatus{}, err
	}

	return subscriptionImageStatus{
		HasImage: resp != nil && len(resp.Data) > 0,
		Verified: true,
	}, nil
}
