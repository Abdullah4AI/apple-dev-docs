package web

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

var syncAppClipBundleIDCapabilityFn = func(ctx context.Context, client *webcore.Client, req webcore.AppClipBundleIDCapabilitySyncRequest) (*webcore.AppClipBundleIDCapabilitySyncResult, error) {
	return client.SyncAppClipBundleIDCapability(ctx, req)
}

// WebBundleIDsCommand returns the private Bundle ID command group.
func WebBundleIDsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web bundle-ids", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "bundle-ids",
		ShortUsage: "asc web bundle-ids <subcommand> [flags]",
		ShortHelp:  "[experimental] Manage Bundle IDs via private web-session endpoints.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Manage Bundle ID operations that are only available through Apple's private
web-session endpoints.

` + webWarningText,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			WebBundleIDCapabilitiesCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// WebBundleIDCapabilitiesCommand returns the private Bundle ID capabilities group.
func WebBundleIDCapabilitiesCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web bundle-ids capabilities", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "capabilities",
		ShortUsage: "asc web bundle-ids capabilities <subcommand> [flags]",
		ShortHelp:  "[experimental] Sync Bundle ID capabilities via web sessions.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Sync Bundle ID capabilities through Apple's private Bundle ID patch endpoint.

` + webWarningText,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			WebBundleIDCapabilitiesSyncAppClipCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// WebBundleIDCapabilitiesSyncAppClipCommand syncs one App Clip capability relationship.
func WebBundleIDCapabilitiesSyncAppClipCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web bundle-ids capabilities sync-app-clip", flag.ExitOnError)

	bundleID := fs.String("bundle-id", "", "Opaque App Clip Bundle ID resource ID")
	parentBundleID := fs.String("parent-bundle-id", "", "Opaque parent app Bundle ID resource ID")
	capability := fs.String("capability", "", "Capability ID (for example: PUSH_NOTIFICATIONS)")
	settingsJSON := fs.String("settings-json", "", "Optional JSON array of capability settings")
	authFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "sync-app-clip",
		ShortUsage: "asc web bundle-ids capabilities sync-app-clip --bundle-id BUNDLE_ID --parent-bundle-id PARENT_BUNDLE_ID --capability CAPABILITY [flags]",
		ShortHelp:  "[experimental] Sync an App Clip capability with parentBundleId.",
		LongHelp: `EXPERIMENTAL / UNOFFICIAL / DISCOURAGED

Patch an App Clip Bundle ID capability through Apple's private Bundle ID update
payload and include the parentBundleId relationship required for App Clip
targets. This mirrors the App Store Connect web-session shape used for App Clip
Bundle IDs, not the public API-key capability endpoint.

Examples:
  asc web bundle-ids capabilities sync-app-clip --bundle-id "CLIP_BUNDLE_ID" --parent-bundle-id "PARENT_BUNDLE_ID" --capability "PUSH_NOTIFICATIONS"
  asc web bundle-ids capabilities sync-app-clip --bundle-id "CLIP_BUNDLE_ID" --parent-bundle-id "PARENT_BUNDLE_ID" --capability "PUSH_NOTIFICATIONS" --settings-json '[{"key":"PUSH_NOTIFICATION_FEATURES","options":[{"key":"PUSH_NOTIFICATION_FEATURE_BROADCAST","enabled":true}]}]'

` + webWarningText,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageError("web bundle-ids capabilities sync-app-clip does not accept positional arguments")
			}

			resolvedBundleID := strings.TrimSpace(*bundleID)
			resolvedParentBundleID := strings.TrimSpace(*parentBundleID)
			resolvedCapability := strings.ToUpper(strings.TrimSpace(*capability))
			if resolvedBundleID == "" {
				return shared.UsageError("--bundle-id is required")
			}
			if resolvedParentBundleID == "" {
				return shared.UsageError("--parent-bundle-id is required")
			}
			if resolvedCapability == "" {
				return shared.UsageError("--capability is required")
			}

			settings, err := parseWebBundleIDCapabilitySettings(*settingsJSON)
			if err != nil {
				return shared.UsageErrorf("--settings-json must be a JSON array of capability settings: %v", err)
			}

			session, err := resolveWebSessionForCommand(ctx, authFlags)
			if err != nil {
				return err
			}
			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			client := newWebClientFn(session)
			var result *webcore.AppClipBundleIDCapabilitySyncResult
			err = withWebSpinner("Syncing App Clip Bundle ID capability", func() error {
				var err error
				result, err = syncAppClipBundleIDCapabilityFn(requestCtx, client, webcore.AppClipBundleIDCapabilitySyncRequest{
					BundleID:         resolvedBundleID,
					ParentBundleID:   resolvedParentBundleID,
					Capability:       resolvedCapability,
					Enabled:          true,
					Settings:         settings,
					SettingsProvided: strings.TrimSpace(*settingsJSON) != "",
				})
				return err
			})
			if err != nil {
				return withWebAuthHint(err, "web bundle-ids capabilities sync-app-clip")
			}
			if result == nil {
				return fmt.Errorf("web bundle-ids capabilities sync-app-clip failed: missing sync result")
			}

			return shared.PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error { return renderWebBundleIDCapabilitySyncTable(result) },
				func() error { return renderWebBundleIDCapabilitySyncMarkdown(result) },
			)
		},
	}
}

func parseWebBundleIDCapabilitySettings(value string) ([]webcore.BundleIDCapabilitySetting, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return []webcore.BundleIDCapabilitySetting{}, nil
	}
	var settings []webcore.BundleIDCapabilitySetting
	decoder := json.NewDecoder(strings.NewReader(value))
	if err := decoder.Decode(&settings); err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, fmt.Errorf("expected JSON array, got null")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("multiple JSON values are not supported")
	}
	return settings, nil
}

func renderWebBundleIDCapabilitySyncTable(result *webcore.AppClipBundleIDCapabilitySyncResult) error {
	asc.RenderTable(
		[]string{"Bundle ID", "Parent Bundle ID", "Capability", "Enabled"},
		[][]string{{
			result.BundleID,
			result.ParentBundleID,
			result.Capability,
			fmt.Sprintf("%t", result.Enabled),
		}},
	)
	return nil
}

func renderWebBundleIDCapabilitySyncMarkdown(result *webcore.AppClipBundleIDCapabilitySyncResult) error {
	asc.RenderMarkdown(
		[]string{"Bundle ID", "Parent Bundle ID", "Capability", "Enabled"},
		[][]string{{
			result.BundleID,
			result.ParentBundleID,
			result.Capability,
			fmt.Sprintf("%t", result.Enabled),
		}},
	)
	return nil
}
