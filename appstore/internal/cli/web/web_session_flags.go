package web

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

type webSessionFlags struct {
	appleID              *string
	twoFactorCode        *string
	twoFactorCodeCommand *string
	providerID           *int64
	publicProviderID     *string
}

const deprecatedTwoFactorCodeFlagName = "two-factor-code"

func bindDeprecatedTwoFactorCodeFlag(fs *flag.FlagSet) *string {
	return fs.String(deprecatedTwoFactorCodeFlagName, "", "Deprecated: direct 2FA code if verification is required; prefer --two-factor-code-command")
}

func bindWebSessionFlags(fs *flag.FlagSet) webSessionFlags {
	return webSessionFlags{
		appleID:              fs.String("apple-id", "", "Apple Account email used to scope a user-owned session cache (optional when a cached session exists)"),
		twoFactorCode:        bindDeprecatedTwoFactorCodeFlag(fs),
		twoFactorCodeCommand: fs.String("two-factor-code-command", "", "Shell command that prints the 2FA code to stdout if verification is required"),
		providerID:           fs.Int64("provider-id", 0, "Numeric App Store Connect provider ID to select for this web session"),
		publicProviderID:     fs.String("public-provider-id", "", "Public App Store Connect provider/team ID to select for this web session"),
	}
}

func warnDeprecatedTwoFactorCodeFlag(twoFactorCode string) {
	if strings.TrimSpace(twoFactorCode) == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "Warning: `--%s` is deprecated. Use `--two-factor-code-command` or `%s` for automation.\n", deprecatedTwoFactorCodeFlagName, webTwoFactorCodeCommandEnv)
}

func resolveWebSessionForCommand(ctx context.Context, flags webSessionFlags) (*webcore.AuthSession, error) {
	warnDeprecatedTwoFactorCodeFlag(*flags.twoFactorCode)
	selection := providerSelectionFromFlags(flags)
	session, _, err := callResolveSessionForProviderSelection(
		ctx,
		*flags.appleID,
		"",
		*flags.twoFactorCode,
		*flags.twoFactorCodeCommand,
		selection,
	)
	if err != nil {
		return nil, err
	}
	if err := selectResolvedWebSessionProvider(ctx, session, selection); err != nil {
		return nil, err
	}
	return session, nil
}

func providerSelectionFromFlags(flags webSessionFlags) webcore.ProviderSelection {
	var providerID int64
	if flags.providerID != nil {
		providerID = *flags.providerID
	}
	var publicProviderID string
	if flags.publicProviderID != nil {
		publicProviderID = *flags.publicProviderID
	}
	return webcore.ProviderSelection{
		ProviderID:       providerID,
		PublicProviderID: publicProviderID,
	}
}

func withWebAuthHint(err error, operation string) error {
	if err == nil {
		return nil
	}
	if strings.HasPrefix(err.Error(), operation+" failed:") {
		return err
	}
	var apiErr *webcore.APIError
	if errors.As(err, &apiErr) && (apiErr.Status == 401 || apiErr.Status == 403) {
		return fmt.Errorf("%s failed: web session is unauthorized or expired (run 'asc web auth login'): %w", operation, err)
	}
	return fmt.Errorf("%s failed: %w", operation, err)
}
