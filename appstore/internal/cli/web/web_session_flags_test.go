package web

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"strings"
	"testing"

	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

func TestBindWebSessionFlagsIncludesDeprecatedTwoFactorAlias(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flags := bindWebSessionFlags(fs)

	if flags.twoFactorCode == nil {
		t.Fatal("expected deprecated two-factor-code pointer to be populated")
	}

	twoFactorCodeFlag := fs.Lookup(deprecatedTwoFactorCodeFlagName)
	if twoFactorCodeFlag == nil {
		t.Fatalf("expected --%s to be registered", deprecatedTwoFactorCodeFlagName)
		return
	}
	if !strings.Contains(twoFactorCodeFlag.Usage, "Deprecated:") {
		t.Fatalf("expected deprecated help text, got %q", twoFactorCodeFlag.Usage)
	}

	if fs.Lookup("two-factor-code-command") == nil {
		t.Fatal("expected --two-factor-code-command to remain registered")
	}
	if fs.Lookup("provider-id") == nil {
		t.Fatal("expected --provider-id to be registered")
	}
	if fs.Lookup("public-provider-id") == nil {
		t.Fatal("expected --public-provider-id to be registered")
	}
}

func TestResolveWebSessionForCommandPassesTwoFactorCodeCommand(t *testing.T) {
	restoreResolve := SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode, twoFactorCodeCommand string) (*webcore.AuthSession, string, error) {
		if appleID != "user@example.com" {
			t.Fatalf("appleID = %q, want %q", appleID, "user@example.com")
		}
		if twoFactorCode != "" {
			t.Fatalf("twoFactorCode = %q, want empty", twoFactorCode)
		}
		if twoFactorCodeCommand != "osascript /tmp/get-apple-2fa-code.scpt" {
			t.Fatalf("twoFactorCodeCommand = %q, want osascript helper", twoFactorCodeCommand)
		}
		return &webcore.AuthSession{}, "test", nil
	})
	t.Cleanup(restoreResolve)

	flags := webSessionFlags{
		appleID:              ptrTo("user@example.com"),
		twoFactorCode:        ptrTo(""),
		twoFactorCodeCommand: ptrTo("osascript /tmp/get-apple-2fa-code.scpt"),
	}

	session, err := resolveWebSessionForCommand(context.Background(), flags)
	if err != nil {
		t.Fatalf("resolveWebSessionForCommand() error = %v", err)
	}
	if session == nil {
		t.Fatal("expected session")
	}
}

func TestResolveWebSessionForCommandSelectsProvider(t *testing.T) {
	expected := &webcore.AuthSession{UserEmail: "user@example.com"}
	restoreResolve := SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode, twoFactorCodeCommand string) (*webcore.AuthSession, string, error) {
		return expected, "cache", nil
	})
	t.Cleanup(restoreResolve)

	origSelectProvider := selectWebProviderFn
	origPersist := persistWebSessionFn
	t.Cleanup(func() {
		selectWebProviderFn = origSelectProvider
		persistWebSessionFn = origPersist
	})

	selected := false
	selectWebProviderFn = func(ctx context.Context, session *webcore.AuthSession, selection webcore.ProviderSelection) error {
		selected = true
		if session != expected {
			t.Fatal("expected resolved session to be selected")
		}
		if selection.ProviderID != 123456 {
			t.Fatalf("ProviderID = %d, want 123456", selection.ProviderID)
		}
		if selection.PublicProviderID != "TEAM123" {
			t.Fatalf("PublicProviderID = %q, want TEAM123", selection.PublicProviderID)
		}
		session.ProviderID = selection.ProviderID
		session.PublicProviderID = selection.PublicProviderID
		return nil
	}
	persisted := false
	persistWebSessionFn = func(session *webcore.AuthSession) error {
		persisted = true
		if session != expected {
			t.Fatal("expected selected session to be persisted")
		}
		return nil
	}

	providerID := int64(123456)
	flags := webSessionFlags{
		appleID:              ptrTo("user@example.com"),
		twoFactorCode:        ptrTo(""),
		twoFactorCodeCommand: ptrTo(""),
		providerID:           &providerID,
		publicProviderID:     ptrTo("TEAM123"),
	}

	session, err := resolveWebSessionForCommand(context.Background(), flags)
	if err != nil {
		t.Fatalf("resolveWebSessionForCommand() error = %v", err)
	}
	if session != expected {
		t.Fatal("expected selected session")
	}
	if !selected {
		t.Fatal("expected provider selection")
	}
	if !persisted {
		t.Fatal("expected selected provider session to be persisted")
	}
}

func TestResolveWebSessionForCommandDoesNotPersistBeforeProviderSelection(t *testing.T) {
	origTryResume := tryResumeSessionFn
	origTryResumeLast := tryResumeLastFn
	origLoadCachedSession := loadCachedSessionFn
	origLoadLastCachedSession := loadLastCachedSessionFn
	origWebLogin := webLoginFn
	origWebLoginWithClient := webLoginWithClientFn
	origSelectProvider := selectWebProviderFn
	origPersist := persistWebSessionFn
	t.Cleanup(func() {
		tryResumeSessionFn = origTryResume
		tryResumeLastFn = origTryResumeLast
		loadCachedSessionFn = origLoadCachedSession
		loadLastCachedSessionFn = origLoadLastCachedSession
		webLoginFn = origWebLogin
		webLoginWithClientFn = origWebLoginWithClient
		selectWebProviderFn = origSelectProvider
		persistWebSessionFn = origPersist
	})

	t.Setenv(webPasswordEnv, "secret")

	providerID := int64(123456)
	flags := webSessionFlags{
		appleID:              ptrTo("user@example.com"),
		twoFactorCode:        ptrTo(""),
		twoFactorCodeCommand: ptrTo(""),
		providerID:           &providerID,
		publicProviderID:     ptrTo("TEAM123"),
	}

	selectErr := errors.New("provider unavailable")
	selectWebProviderFn = func(ctx context.Context, session *webcore.AuthSession, selection webcore.ProviderSelection) error {
		return selectErr
	}
	persistWebSessionFn = func(session *webcore.AuthSession) error {
		t.Fatal("did not expect session cache persistence before provider selection succeeds")
		return nil
	}

	t.Run("fresh login", func(t *testing.T) {
		tryResumeSessionFn = func(ctx context.Context, username string) (*webcore.AuthSession, bool, error) {
			return nil, false, nil
		}
		tryResumeLastFn = func(ctx context.Context) (*webcore.AuthSession, bool, error) {
			t.Fatal("did not expect last-session cache lookup when apple-id is provided")
			return nil, false, nil
		}
		webLoginFn = func(ctx context.Context, creds webcore.LoginCredentials) (*webcore.AuthSession, error) {
			if creds.Username != "user@example.com" {
				t.Fatalf("Username = %q, want user@example.com", creds.Username)
			}
			return &webcore.AuthSession{UserEmail: creds.Username}, nil
		}

		_, err := resolveWebSessionForCommand(context.Background(), flags)
		if !errors.Is(err, selectErr) {
			t.Fatalf("expected provider selection error, got %v", err)
		}
	})

	t.Run("auto reauth", func(t *testing.T) {
		cachedClient := &http.Client{}
		tryResumeSessionFn = func(ctx context.Context, username string) (*webcore.AuthSession, bool, error) {
			return nil, false, webcore.ErrCachedSessionExpired
		}
		tryResumeLastFn = func(ctx context.Context) (*webcore.AuthSession, bool, error) {
			t.Fatal("did not expect last-session cache lookup when apple-id is provided")
			return nil, false, nil
		}
		loadCachedSessionFn = func(username string) (*webcore.AuthSession, bool, error) {
			return &webcore.AuthSession{Client: cachedClient, UserEmail: username}, true, nil
		}
		loadLastCachedSessionFn = func() (*webcore.AuthSession, bool, error) {
			t.Fatal("did not expect last cached-session load when apple-id is provided")
			return nil, false, nil
		}
		webLoginWithClientFn = func(ctx context.Context, client *http.Client, creds webcore.LoginCredentials) (*webcore.AuthSession, error) {
			if client != cachedClient {
				t.Fatal("expected cached client to be reused")
			}
			return &webcore.AuthSession{Client: client, UserEmail: creds.Username}, nil
		}

		_, err := resolveWebSessionForCommand(context.Background(), flags)
		if !errors.Is(err, selectErr) {
			t.Fatalf("expected provider selection error, got %v", err)
		}
	})
}

func ptrTo(value string) *string {
	return &value
}
