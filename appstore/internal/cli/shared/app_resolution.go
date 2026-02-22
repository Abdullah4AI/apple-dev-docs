package shared

import (
	"context"
	"fmt"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

// ResolveAppStoreVersionID finds a version ID by version string and platform.
func ResolveAppStoreVersionID(ctx context.Context, client *asc.Client, appID, version, platform string) (string, error) {
	opts := []asc.AppStoreVersionsOption{
		asc.WithAppStoreVersionsVersionStrings([]string{version}),
		asc.WithAppStoreVersionsPlatforms([]string{platform}),
		asc.WithAppStoreVersionsLimit(10),
	}
	resp, err := client.GetAppStoreVersions(ctx, appID, opts...)
	if err != nil {
		return "", err
	}
	if resp == nil || len(resp.Data) == 0 {
		return "", fmt.Errorf("app store version not found for version %q and platform %q", version, platform)
	}
	if len(resp.Data) > 1 {
		return "", fmt.Errorf("multiple app store versions found for version %q and platform %q (use --version-id)", version, platform)
	}
	return resp.Data[0].ID, nil
}

// ResolveAppInfoID resolves the app info ID, optionally using a provided override.
func ResolveAppInfoID(ctx context.Context, client *asc.Client, appID, appInfoID string) (string, error) {
	if strings.TrimSpace(appInfoID) != "" {
		return strings.TrimSpace(appInfoID), nil
	}

	resp, err := client.GetAppInfos(ctx, appID)
	if err != nil {
		return "", err
	}
	if len(resp.Data) == 0 {
		return "", fmt.Errorf("no app info found for app %q", appID)
	}
	if len(resp.Data) > 1 {
		return "", fmt.Errorf("multiple app infos found for app %q; use --app-info", appID)
	}
	return resp.Data[0].ID, nil
}
