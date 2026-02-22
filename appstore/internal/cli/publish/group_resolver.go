package publish

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

type publishBetaGroupsClient interface {
	GetBetaGroups(ctx context.Context, appID string, opts ...asc.BetaGroupsOption) (*asc.BetaGroupsResponse, error)
}

func resolvePublishBetaGroupIDs(ctx context.Context, client publishBetaGroupsClient, appID string, groups []string) ([]string, error) {
	allGroups, err := listAllPublishBetaGroups(ctx, client, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to list beta groups: %w", err)
	}
	return resolvePublishBetaGroupIDsFromList(groups, allGroups)
}

func listAllPublishBetaGroups(ctx context.Context, client publishBetaGroupsClient, appID string) (*asc.BetaGroupsResponse, error) {
	firstPage, err := client.GetBetaGroups(ctx, appID, asc.WithBetaGroupsLimit(200))
	if err != nil {
		return nil, err
	}
	if firstPage == nil || firstPage.Links.Next == "" {
		return firstPage, nil
	}

	paginated, err := asc.PaginateAll(ctx, firstPage, func(ctx context.Context, nextURL string) (asc.PaginatedResponse, error) {
		return client.GetBetaGroups(ctx, appID, asc.WithBetaGroupsNextURL(nextURL))
	})
	if err != nil {
		return nil, err
	}

	allGroups, ok := paginated.(*asc.BetaGroupsResponse)
	if !ok {
		return nil, fmt.Errorf("unexpected beta groups pagination type %T", paginated)
	}
	return allGroups, nil
}

func resolvePublishBetaGroupIDsFromList(inputGroups []string, groups *asc.BetaGroupsResponse) ([]string, error) {
	if groups == nil {
		return nil, fmt.Errorf("no beta groups returned for app")
	}

	groupIDs := make(map[string]struct{}, len(groups.Data))
	groupNameToIDs := make(map[string][]string)
	for _, item := range groups.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		groupIDs[id] = struct{}{}

		name := strings.TrimSpace(item.Attributes.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if !slices.Contains(groupNameToIDs[key], id) {
			groupNameToIDs[key] = append(groupNameToIDs[key], id)
		}
	}

	resolved := make([]string, 0, len(inputGroups))
	seen := make(map[string]struct{}, len(inputGroups))
	for _, raw := range inputGroups {
		group := strings.TrimSpace(raw)
		if group == "" {
			continue
		}

		resolvedID := ""
		if _, ok := groupIDs[group]; ok {
			resolvedID = group
		} else {
			matches := groupNameToIDs[strings.ToLower(group)]
			switch len(matches) {
			case 0:
				return nil, fmt.Errorf("beta group %q not found", group)
			case 1:
				resolvedID = matches[0]
			default:
				return nil, fmt.Errorf("multiple beta groups named %q; use group ID", group)
			}
		}

		if _, ok := seen[resolvedID]; ok {
			continue
		}
		seen[resolvedID] = struct{}{}
		resolved = append(resolved, resolvedID)
	}

	if len(resolved) == 0 {
		return nil, fmt.Errorf("at least one beta group is required")
	}

	return resolved, nil
}
