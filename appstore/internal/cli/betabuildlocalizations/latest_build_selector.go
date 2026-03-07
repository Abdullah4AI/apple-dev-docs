package betabuildlocalizations

import (
	"context"
	"fmt"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func resolveLatestBuildIDForBetaBuildLocalizations(
	ctx context.Context,
	client *asc.Client,
	appInput string,
	stateValues []string,
) (string, error) {
	resolvedAppID := shared.ResolveAppID(appInput)
	if resolvedAppID == "" {
		return "", shared.UsageError("--app is required with --latest")
	}

	resolvedAppID, err := shared.ResolveAppIDWithLookup(ctx, client, resolvedAppID)
	if err != nil {
		return "", err
	}

	opts := []asc.BuildsOption{
		asc.WithBuildsSort("-uploadedDate"),
		asc.WithBuildsLimit(1),
	}
	if len(stateValues) > 0 {
		opts = append(opts, asc.WithBuildsProcessingStates(stateValues))
	}

	builds, err := client.GetBuilds(ctx, resolvedAppID, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest build: %w", err)
	}
	if len(builds.Data) == 0 {
		if len(stateValues) > 0 {
			return "", fmt.Errorf("no builds found for app %s matching state filter", resolvedAppID)
		}
		return "", fmt.Errorf("no builds found for app %s", resolvedAppID)
	}

	buildID := strings.TrimSpace(builds.Data[0].ID)
	if buildID == "" {
		return "", fmt.Errorf("latest build is missing an ID for app %s", resolvedAppID)
	}

	return buildID, nil
}

func normalizeLatestBuildProcessingStateFilter(raw string) ([]string, error) {
	return shared.NormalizeBuildProcessingStateFilter(raw, shared.BuildProcessingStateFilterOptions{
		FlagName:          "--state",
		AllowedValuesHelp: "PROCESSING, FAILED, INVALID, VALID, COMPLETE, or all",
		Aliases: map[string]string{
			"COMPLETE": asc.BuildProcessingStateValid,
		},
	})
}
