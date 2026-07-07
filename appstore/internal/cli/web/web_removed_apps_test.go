package web

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"strings"
	"testing"

	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

func TestWebRemovedAppsCommandHierarchy(t *testing.T) {
	cmd := WebRemovedAppsCommand()
	if cmd.Name != "removed-apps" {
		t.Fatalf("expected command name %q, got %q", "removed-apps", cmd.Name)
	}
	if cmd.UsageFunc == nil {
		t.Fatal("expected command usage func")
	}
	if len(cmd.Subcommands) != 1 || cmd.Subcommands[0].Name != "list" {
		t.Fatalf("expected list subcommand, got %+v", cmd.Subcommands)
	}
}

func TestWebRemovedAppsListOutputsJSON(t *testing.T) {
	restoreSession := SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode, twoFactorCodeCommand string) (*webcore.AuthSession, string, error) {
		if appleID != "user@example.com" {
			t.Fatalf("expected apple id %q, got %q", "user@example.com", appleID)
		}
		return &webcore.AuthSession{}, "cache", nil
	})
	origNewWebClient := newWebClientFn
	origListRemoved := listRemovedWebAppsFn
	t.Cleanup(func() {
		restoreSession()
		newWebClientFn = origNewWebClient
		listRemovedWebAppsFn = origListRemoved
	})

	newWebClientFn = func(session *webcore.AuthSession) *webcore.Client {
		return &webcore.Client{}
	}
	listRemovedWebAppsFn = func(ctx context.Context, client *webcore.Client, opts webcore.RemovedAppsListOptions) (*webcore.RemovedAppsListResponse, error) {
		if opts.Limit != 10 {
			t.Fatalf("expected limit 10, got %d", opts.Limit)
		}
		return &webcore.RemovedAppsListResponse{
			Data: []webcore.RemovedApp{{
				ID:                   "1234567890",
				Name:                 "Throwaway",
				BundleID:             "com.example.throwaway",
				SKU:                  "THROWAWAY",
				PrimaryLocale:        "en-US",
				Removed:              true,
				AppStoreLegacyStatus: "PREPARE_FOR_SUBMISSION",
				DisplayableVersions: []webcore.RemovedAppVersion{{
					ID:            "version-1",
					Platform:      "IOS",
					VersionString: "1.0",
				}},
			}},
		}, nil
	}

	cmd := WebRemovedAppsListCommand()
	if err := cmd.FlagSet.Parse([]string{"--apple-id", "user@example.com", "--limit", "10", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, stderr := captureWebCommandOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var payload webcore.RemovedAppsListResponse
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("expected JSON output, got error %v and output %q", err, stdout)
	}
	if len(payload.Data) != 1 || payload.Data[0].Name != "Throwaway" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestWebRemovedAppsListTableIncludesCoreColumns(t *testing.T) {
	restoreSession := SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode, twoFactorCodeCommand string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{}, "cache", nil
	})
	origNewWebClient := newWebClientFn
	origListRemoved := listRemovedWebAppsFn
	t.Cleanup(func() {
		restoreSession()
		newWebClientFn = origNewWebClient
		listRemovedWebAppsFn = origListRemoved
	})

	newWebClientFn = func(session *webcore.AuthSession) *webcore.Client {
		return &webcore.Client{}
	}
	listRemovedWebAppsFn = func(ctx context.Context, client *webcore.Client, opts webcore.RemovedAppsListOptions) (*webcore.RemovedAppsListResponse, error) {
		return &webcore.RemovedAppsListResponse{
			Data: []webcore.RemovedApp{{
				ID:             "1234567890",
				Name:           "Throwaway",
				BundleID:       "com.example.throwaway",
				VersionSummary: "iOS 1.0",
				Status:         "PREPARE_FOR_SUBMISSION",
				Removed:        true,
			}},
		}, nil
	}

	cmd := WebRemovedAppsListCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "table"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, _ := captureWebCommandOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})
	for _, want := range []string{"ID", "Name", "Bundle ID", "Version", "Status", "Throwaway", "com.example.throwaway", "iOS 1.0", "PREPARE_FOR_SUBMISSION"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected table output to contain %q, got %q", want, stdout)
		}
	}
}

func TestWebRemovedAppsListValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "positional", args: []string{"extra"}, wantErr: "web removed-apps list does not accept positional arguments"},
		{name: "limit low", args: []string{"--limit", "0"}, wantErr: "--limit must be between 1 and 200"},
		{name: "next with paginate", args: []string{"--next", "https://appstoreconnect.apple.com/iris/v1/apps?page=2", "--paginate"}, wantErr: "--next cannot be combined with --paginate"},
		{name: "bad next host", args: []string{"--next", "https://example.com/iris/v1/apps?page=2"}, wantErr: "--next must be an App Store Connect web URL"},
		{name: "protocol relative next host", args: []string{"--next", "//example.com/iris/v1/apps?filter[removed]=true"}, wantErr: "--next must be an App Store Connect web URL"},
		{name: "next missing removed filter", args: []string{"--next", "https://appstoreconnect.apple.com/iris/v1/apps?limit=48"}, wantErr: "--next must include filter[removed]=true"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := WebRemovedAppsListCommand()
			if err := cmd.FlagSet.Parse(tc.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			_, stderr := captureWebCommandOutput(t, func() {
				err := cmd.Exec(context.Background(), cmd.FlagSet.Args())
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected flag.ErrHelp, got %v", err)
				}
			})
			if !strings.Contains(stderr, tc.wantErr) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.wantErr, stderr)
			}
		})
	}
}

func TestValidateRemovedAppsNextURLAcceptsSafeRelativeLinks(t *testing.T) {
	for _, next := range []string{
		"/iris/v1/apps?filter[removed]=true&cursor=abc",
		"/apps?filter[removed]=true&cursor=abc",
		"https://appstoreconnect.apple.com/iris/v1/apps?filter[removed]=true&cursor=abc",
	} {
		t.Run(next, func(t *testing.T) {
			if err := validateRemovedAppsNextURL(next); err != nil {
				t.Fatalf("expected %q to validate, got %v", next, err)
			}
		})
	}
}
