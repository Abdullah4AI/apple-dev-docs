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

var getAgreementsStatusFn = func(ctx context.Context, client *webcore.Client) (*asc.WebAgreementsStatusResult, error) {
	return client.GetAgreementsStatus(ctx)
}

var acceptAgreementsFn = func(ctx context.Context, client *webcore.Client, req webcore.AgreementsAcceptRequest) (*asc.WebAgreementsAcceptResult, error) {
	return client.AcceptAgreements(ctx, req)
}

// WebAgreementsCommand returns the Apple Developer Program agreements group.
func WebAgreementsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web agreements", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "agreements",
		ShortUsage: "asc web agreements <subcommand> [flags]",
		ShortHelp:  "[experimental] Check and accept Apple Developer Program agreements.",
		LongHelp: `WEB SESSION WORKFLOWS

This command is experimental.

Check and accept Apple Developer Program agreements, such as the Apple
Developer Program License Agreement, through Apple web-session endpoints.
The public App Store Connect API has no endpoint for these agreements.

Examples:
  asc web agreements status
  asc web agreements accept --agreement-id "AGREEMENT_ID" --confirm

`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			WebAgreementsStatusCommand(),
			WebAgreementsAcceptCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}

// WebAgreementsStatusCommand reports pending and accepted program agreements.
func WebAgreementsStatusCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web agreements status", flag.ExitOnError)

	authFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "asc web agreements status [flags]",
		ShortHelp:  "[experimental] Show Apple Developer Program agreement status.",
		LongHelp: `WEB SESSION WORKFLOWS

This command is experimental.

Show the App Store Connect agreement alert banner and the team's Apple
Developer Program agreement history, including whether an updated agreement
is waiting for the Account Holder.

An agreement is reported as pending when App Store Connect shows an agreement
alert banner or when the agreement's accepted date is older than its
effective date.

Example:
  asc web agreements status

`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageError("web agreements status does not accept positional arguments")
			}

			requestCtx, cancel := shared.ContextWithTimeout(ctx)
			defer cancel()

			session, err := resolveWebSessionForCommand(requestCtx, authFlags)
			if err != nil {
				return err
			}
			client := newWebClientFn(session)

			var result *asc.WebAgreementsStatusResult
			err = withWebSpinner("Fetching Apple Developer Program agreement status", func() error {
				var statusErr error
				result, statusErr = getAgreementsStatusFn(requestCtx, client)
				return statusErr
			})
			if err != nil {
				return withWebAuthHint(err, "web agreements status")
			}
			if result == nil {
				return fmt.Errorf("web agreements status failed: missing status result")
			}
			// Developer Portal bootstrap can add origin-specific cookies to the
			// shared jar. Cache them best-effort after the operation succeeds.
			_ = persistWebSessionFn(session)

			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}

// WebAgreementsAcceptCommand accepts a pending program agreement.
func WebAgreementsAcceptCommand() *ffcli.Command {
	fs := flag.NewFlagSet("web agreements accept", flag.ExitOnError)

	agreementID := fs.String("agreement-id", "", "[experimental] Developer Portal agreement ID to accept (from `asc web agreements status`)")
	confirm := fs.Bool("confirm", false, "[experimental] Confirm accepting the agreement on behalf of the Account Holder")
	authFlags := bindWebSessionFlags(fs)
	output := shared.BindOutputFlags(fs)

	return &ffcli.Command{
		Name:       "accept",
		ShortUsage: "asc web agreements accept --agreement-id AGREEMENT_ID --confirm [flags]",
		ShortHelp:  "[experimental] Accept an Apple Developer Program agreement.",
		LongHelp: `WEB SESSION WORKFLOWS

This command is experimental.

Accept an Apple Developer Program agreement, such as an updated Apple
Developer Program License Agreement, for the web session's team.

Accepting an agreement is a legal action. Apple only allows the team's
Account Holder to accept agreements; sessions for other roles fail with an
Apple error. Find pending agreement IDs with:
  asc web agreements status

Example:
  asc web agreements accept --agreement-id "AGREEMENT_ID" --confirm

`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return shared.UsageError("web agreements accept does not accept positional arguments")
			}

			resolvedAgreementID := strings.TrimSpace(*agreementID)
			switch {
			case resolvedAgreementID == "":
				return shared.UsageError("--agreement-id is required")
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

			var result *asc.WebAgreementsAcceptResult
			err = withWebSpinner("Accepting Apple Developer Program agreement", func() error {
				var acceptErr error
				result, acceptErr = acceptAgreementsFn(requestCtx, client, webcore.AgreementsAcceptRequest{
					AgreementIDs: []string{resolvedAgreementID},
				})
				return acceptErr
			})
			if err != nil {
				return withWebAuthHint(err, "web agreements accept")
			}
			if result == nil {
				return fmt.Errorf("web agreements accept failed: missing accept result")
			}
			// Developer Portal bootstrap can add origin-specific cookies to the
			// shared jar. Cache them best-effort after the operation succeeds.
			_ = persistWebSessionFn(session)

			return shared.PrintOutput(result, *output.Output, *output.Pretty)
		},
	}
}
