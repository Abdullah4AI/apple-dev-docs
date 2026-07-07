package web

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

type webAppDeleteResult struct {
	AppID    string `json:"appId"`
	Name     string `json:"name,omitempty"`
	BundleID string `json:"bundleId,omitempty"`
	Removed  bool   `json:"removed"`
}

// WebAppsDeleteCommand removes apps using the internal web API.
func WebAppsDeleteCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web apps delete", flag.ExitOnError)

	app := fs.String("app", "", "App Store Connect app ID or exact bundle ID")
	expectedBundleID := fs.String("expected-bundle-id", "", "Require the app bundle ID to match before deleting")
	expectedName := fs.String("expected-name", "", "Require the app name to match before deleting")
	confirm := fs.Bool("confirm", false, "Confirm deleting this app")
	authFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "delete",
		ShortUsage: "asc web apps delete --app APP_ID_OR_BUNDLE_ID --confirm [flags]",
		ShortHelp:  "Delete an app via Apple web API.",
		LongHelp: `WEB SESSION WORKFLOWS

Delete an app through Apple's web API using a web-session login.

The public App Store Connect API does not expose app deletion. This command uses
the same web-session PATCH observed from App Store Connect: it marks the app
resource as removed. Pass an explicit App Store Connect app ID, or an exact
bundle ID for a web API lookup before deletion. Exact identity guard flags can
be used to stop before mutation if the resolved app is not the one expected.

Examples:
  asc web apps delete --app "1234567890" --confirm
  asc web apps delete --app "com.example.throwaway" --confirm
  asc web apps delete --app "1234567890" --expected-bundle-id "com.example.throwaway" --confirm
  asc web apps delete --app "1234567890" --expected-name "Throwaway" --confirm`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageError("web apps delete does not accept positional arguments")
			}

			selector := strings.TrimSpace(*app)
			wantBundleID := strings.TrimSpace(*expectedBundleID)
			wantName := strings.TrimSpace(*expectedName)
			switch {
			case selector == "":
				return shared.UsageError("--app is required")
			case !*confirm:
				return shared.UsageError("--confirm is required")
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, authFlags)
			if err != nil {
				return err
			}
			client := newWebClientFn(session)

			appID, loaded, err := resolveWebAppDeleteTarget(requestCtx, client, selector)
			if err != nil {
				return err
			}

			if wantBundleID != "" || wantName != "" {
				if loaded == nil || loaded.Data.Attributes == nil {
					loaded, err = getWebAppFn(requestCtx, client, appID)
					if err != nil {
						return withWebAuthHint(err, "web apps delete")
					}
				}
				if err := validateWebAppDeleteGuards(loaded, wantBundleID, wantName); err != nil {
					return err
				}
			}

			deleted, err := deleteWebAppFn(requestCtx, client, appID)
			if err != nil {
				return withWebAuthHint(err, "web apps delete")
			}

			result := webAppDeleteResultFromResponse(appID, deleted, loaded)
			return shared.PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error { return renderWebAppDeleteTable(result) },
				func() error { return renderWebAppDeleteMarkdown(result) },
			)
		},
	}
}

func resolveWebAppDeleteTarget(ctx context.Context, client *webcore.Client, selector string) (string, *webcore.AppResponse, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return "", nil, fmt.Errorf("app selector is required")
	}
	if shared.IsNumericAppID(selector) {
		return selector, nil, nil
	}

	app, err := findWebAppFn(ctx, client, selector)
	if err != nil {
		return "", nil, withWebAuthHint(err, "web apps delete")
	}
	if app == nil || strings.TrimSpace(app.Data.ID) == "" {
		return "", nil, fmt.Errorf("web apps delete failed: app %q not found by exact bundle ID; pass the App Store Connect app ID", selector)
	}
	actualBundleID := webAppAttrString(app, "bundleId")
	if actualBundleID == "" {
		return "", nil, fmt.Errorf("web apps delete failed: Apple did not return a bundle ID for app %q; pass the App Store Connect app ID", selector)
	}
	if actualBundleID != selector {
		return "", nil, fmt.Errorf("web apps delete failed: exact bundle ID lookup for %q returned %q; pass the App Store Connect app ID", selector, actualBundleID)
	}
	return strings.TrimSpace(app.Data.ID), app, nil
}

func validateWebAppDeleteGuards(app *webcore.AppResponse, expectedBundleID, expectedName string) error {
	if app == nil {
		return fmt.Errorf("web apps delete failed: app identity could not be loaded")
	}
	if expectedBundleID != "" {
		actual := webAppAttrString(app, "bundleId")
		if actual == "" {
			return fmt.Errorf("web apps delete failed: expected bundle ID %q, but Apple did not return a bundle ID for app %q", expectedBundleID, strings.TrimSpace(app.Data.ID))
		}
		if actual != expectedBundleID {
			return fmt.Errorf("web apps delete failed: expected bundle ID %q, got %q for app %q", expectedBundleID, actual, strings.TrimSpace(app.Data.ID))
		}
	}
	if expectedName != "" {
		actual := webAppAttrString(app, "name")
		if actual == "" {
			return fmt.Errorf("web apps delete failed: expected name %q, but Apple did not return a name for app %q", expectedName, strings.TrimSpace(app.Data.ID))
		}
		if actual != expectedName {
			return fmt.Errorf("web apps delete failed: expected name %q, got %q for app %q", expectedName, actual, strings.TrimSpace(app.Data.ID))
		}
	}
	return nil
}

func webAppDeleteResultFromResponse(appID string, deleted, fallback *webcore.AppResponse) webAppDeleteResult {
	source := deleted
	if source == nil || source.Data.Attributes == nil {
		source = fallback
	}

	result := webAppDeleteResult{
		AppID:   strings.TrimSpace(appID),
		Removed: true,
	}
	if deleted != nil && strings.TrimSpace(deleted.Data.ID) != "" {
		result.AppID = strings.TrimSpace(deleted.Data.ID)
	}
	if source != nil {
		result.Name = webAppAttrString(source, "name")
		result.BundleID = webAppAttrString(source, "bundleId")
	}
	if removed, ok := webAppAttrBool(deleted, "removed"); ok {
		result.Removed = removed
	}
	return result
}

func webAppAttrString(app *webcore.AppResponse, key string) string {
	if app == nil || app.Data.Attributes == nil {
		return ""
	}
	value, ok := app.Data.Attributes[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func webAppAttrBool(app *webcore.AppResponse, key string) (bool, bool) {
	if app == nil || app.Data.Attributes == nil {
		return false, false
	}
	switch value := app.Data.Attributes[key].(type) {
	case bool:
		return value, true
	case string:
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "true" {
			return true, true
		}
		if normalized == "false" {
			return false, true
		}
	}
	return false, false
}

func webAppDeleteRows(result webAppDeleteResult) [][]string {
	return [][]string{{
		result.AppID,
		result.Name,
		result.BundleID,
		fmt.Sprintf("%t", result.Removed),
	}}
}

func renderWebAppDeleteTable(result webAppDeleteResult) error {
	asc.RenderTable([]string{"App ID", "Name", "Bundle ID", "Removed"}, webAppDeleteRows(result))
	return nil
}

func renderWebAppDeleteMarkdown(result webAppDeleteResult) error {
	asc.RenderMarkdown([]string{"App ID", "Name", "Bundle ID", "Removed"}, webAppDeleteRows(result))
	return nil
}
