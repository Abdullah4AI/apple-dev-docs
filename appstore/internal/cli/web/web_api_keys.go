package web

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/rootfs"
	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
	webref "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web/reference"
)

const webAPIKeyDownloadAttempts = 3

var (
	newWebAPIKeyClientFn = webcore.NewClient
	createWebAPIKeyFn    = func(ctx context.Context, client *webcore.Client, attrs webcore.APIKeyCreateAttributes) (*webcore.APIKey, error) {
		return client.CreateAPIKey(ctx, attrs)
	}
	downloadWebAPIKeyFn = func(ctx context.Context, client *webcore.Client, keyID string) ([]byte, error) {
		return client.DownloadAPIKey(ctx, keyID)
	}
	getWebAPIKeyFn = func(ctx context.Context, client *webcore.Client, keyID string) (*webcore.APIKey, error) {
		return client.GetAPIKey(ctx, keyID)
	}
	waitWebAPIKeyRetryFn = waitForWebAPIKeyRetry
)

type webAPIKeyCreateResult struct {
	KeyID          string   `json:"keyId"`
	Name           string   `json:"name"`
	IssuerID       string   `json:"issuerId"`
	TeamID         string   `json:"teamId,omitempty"`
	Roles          []string `json:"roles"`
	Active         bool     `json:"active"`
	AllAppsVisible bool     `json:"allAppsVisible"`
	KeyType        string   `json:"keyType"`
	P8Path         string   `json:"p8Path"`
}

// WebAPIKeysCommand returns the web-session API keys command group.
func WebAPIKeysCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web api-keys", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "api-keys",
		ShortUsage: "asc web api-keys <subcommand> [flags]",
		ShortHelp:  "Manage App Store Connect API keys via web sessions.",
		LongHelp: `WEB SESSION WORKFLOWS

Manage App Store Connect API keys using a cached Apple Account web session.

`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			WebAPIKeysCreateCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// WebAPIKeysCreateCommand creates a team API key and saves its one-time P8.
func WebAPIKeysCreateCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web api-keys create", flag.ExitOnError)

	name := fs.String("name", "", "API key name")
	role := fs.String("role", "ADMIN", "API key role")
	outputDir := fs.String("output-dir", ".", "Directory for AuthKey_<KEY_ID>.p8")
	authFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "asc web api-keys create --name NAME [--role ROLE] [--output-dir DIR] [flags]",
		ShortHelp:  "Create a team API key and save its one-time P8.",
		LongHelp: `WEB SESSION WORKFLOWS

Create an all-apps App Store Connect team API key using a cached Apple Account
web session. The one-time P8 is saved as AuthKey_<KEY_ID>.p8 with mode 0600.
The P8 contents are never written to command output.

Account Holder or Admin access is required. The role defaults to ADMIN and must
be one of Apple's selectable team-key roles.

Examples:
  asc web api-keys create --name "Release automation"
  asc web api-keys create --name "CI uploads" --role APP_MANAGER --output-dir ~/.asc/keys --output json

`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageError("web api-keys create does not accept positional arguments")
			}
			if _, err := shared.ValidateOutputFormat(*output.Output, *output.Pretty); err != nil {
				return shared.UsageError(err.Error())
			}

			nameValue := strings.TrimSpace(*name)
			if nameValue == "" {
				return shared.UsageError("--name is required")
			}
			roleValue, err := normalizeWebAPIKeyRole(*role)
			if err != nil {
				return shared.UsageError(err.Error())
			}
			outputDirValue := strings.TrimSpace(*outputDir)
			if outputDirValue == "" {
				return shared.UsageError("--output-dir is required")
			}
			outputRoot, err := rootfs.New(outputDirValue)
			if err != nil {
				return shared.UsageErrorf("invalid --output-dir: %v", err)
			}
			if err := outputRoot.MkdirAll(".", 0o700); err != nil {
				return fmt.Errorf("prepare API key output directory: %w", err)
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()
			session, err := resolveWebSessionForCommand(requestCtx, authFlags)
			if err != nil {
				return err
			}
			client := newWebAPIKeyClientFn(session)

			var created *webcore.APIKey
			err = withWebSpinner("Creating App Store Connect API key", func() error {
				var createErr error
				created, createErr = createWebAPIKeyFn(requestCtx, client, webcore.APIKeyCreateAttributes{
					Nickname: nameValue,
					Role:     roleValue,
				})
				return createErr
			})
			if err != nil {
				return withWebAuthHint(err, "web api-keys create")
			}
			if created == nil || strings.TrimSpace(created.KeyID) == "" {
				return fmt.Errorf("web api-keys create failed: create response did not include a key id")
			}

			fileName := fmt.Sprintf("AuthKey_%s.p8", strings.TrimSpace(created.KeyID))
			p8Path := filepath.Join(outputRoot.Path(), fileName)
			if err := outputRoot.CreateNewFile(fileName, nil, 0o600); err != nil {
				return fmt.Errorf(
					"API key %q (%s) was created, but destination %q could not be reserved before its one-time P8 download; the P8 was not downloaded: %w",
					nameValue,
					created.KeyID,
					p8Path,
					err,
				)
			}

			var p8 []byte
			err = withWebSpinner("Downloading one-time API key P8", func() error {
				var downloadErr error
				p8, downloadErr = downloadWebAPIKeyWithRetry(requestCtx, client, created.KeyID)
				return downloadErr
			})
			if err != nil {
				downloadFailure := fmt.Errorf(
					"API key %q (%s) was created, but its one-time P8 could not be downloaded: %w",
					nameValue,
					created.KeyID,
					withWebAuthHint(err, "web api-keys create"),
				)
				if removeErr := os.Remove(p8Path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
					return errors.Join(
						downloadFailure,
						fmt.Errorf("remove empty reserved P8 file %q: %w", p8Path, removeErr),
					)
				}
				return downloadFailure
			}

			if err := outputRoot.WriteFile(fileName, p8, 0o600); err != nil {
				return fmt.Errorf(
					"API key %q (%s) was created and its one-time P8 was downloaded, but saving %q failed; the P8 cannot be downloaded again: %w",
					nameValue,
					created.KeyID,
					p8Path,
					err,
				)
			}

			var details *webcore.APIKey
			err = withWebSpinner("Loading API key issuer ID", func() error {
				var getErr error
				details, getErr = getWebAPIKeyFn(requestCtx, client, created.KeyID)
				return getErr
			})
			if err != nil {
				return fmt.Errorf(
					"API key %q (%s) was created and its P8 was saved to %q, but the issuer ID could not be loaded: %w",
					nameValue,
					created.KeyID,
					p8Path,
					err,
				)
			}
			if details == nil || strings.TrimSpace(details.IssuerID) == "" {
				return fmt.Errorf(
					"API key %q (%s) was created and its P8 was saved to %q, but the response did not include an issuer ID",
					nameValue,
					created.KeyID,
					p8Path,
				)
			}

			result := &webAPIKeyCreateResult{
				KeyID:          created.KeyID,
				Name:           firstNonEmpty(details.Name, created.Name, nameValue),
				IssuerID:       details.IssuerID,
				TeamID:         strings.TrimSpace(session.TeamID),
				Roles:          firstNonEmptyRoles(details.Roles, created.Roles, []string{roleValue}),
				Active:         details.Active,
				AllAppsVisible: details.AllAppsVisible,
				KeyType:        firstNonEmpty(details.KeyType, created.KeyType, "PUBLIC_API"),
				P8Path:         p8Path,
			}
			return shared.PrintOutputWithRenderers(
				result,
				*output.Output,
				*output.Pretty,
				func() error { return renderWebAPIKeyCreateTable(result) },
				func() error { return renderWebAPIKeyCreateMarkdown(result) },
			)
		},
	}
}

func normalizeWebAPIKeyRole(value string) (string, error) {
	role := strings.ToUpper(strings.TrimSpace(value))
	if role == "" {
		return "", fmt.Errorf("--role is required")
	}
	reference, err := webref.Load()
	if err != nil {
		return "", fmt.Errorf("load API key role reference: %w", err)
	}
	for _, selectable := range reference.APIKeyNotes.Team.SelectableRoles {
		if role == selectable {
			return role, nil
		}
	}
	return "", fmt.Errorf("--role must be one of %s", strings.Join(reference.APIKeyNotes.Team.SelectableRoles, ", "))
}

func downloadWebAPIKeyWithRetry(ctx context.Context, client *webcore.Client, keyID string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < webAPIKeyDownloadAttempts; attempt++ {
		p8, err := downloadWebAPIKeyFn(ctx, client, keyID)
		if err == nil {
			return p8, nil
		}
		lastErr = err
		if attempt == webAPIKeyDownloadAttempts-1 || !webcore.IsAPIKeyDownloadRetryable(err) {
			break
		}
		if err := waitWebAPIKeyRetryFn(ctx, time.Duration(attempt+1)*time.Second); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func waitForWebAPIKeyRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonEmptyRoles(values ...[]string) []string {
	for _, roles := range values {
		if len(roles) > 0 {
			return append([]string(nil), roles...)
		}
	}
	return nil
}

func webAPIKeyCreateRows(result *webAPIKeyCreateResult) ([]string, [][]string) {
	headers := []string{"Key ID", "Name", "Issuer ID", "Team ID", "Roles", "P8 Path"}
	rows := [][]string{{
		result.KeyID,
		result.Name,
		result.IssuerID,
		result.TeamID,
		strings.Join(result.Roles, ", "),
		result.P8Path,
	}}
	return headers, rows
}

func renderWebAPIKeyCreateTable(result *webAPIKeyCreateResult) error {
	headers, rows := webAPIKeyCreateRows(result)
	asc.RenderTable(headers, rows)
	return nil
}

func renderWebAPIKeyCreateMarkdown(result *webAPIKeyCreateResult) error {
	headers, rows := webAPIKeyCreateRows(result)
	asc.RenderMarkdown(headers, rows)
	return nil
}
