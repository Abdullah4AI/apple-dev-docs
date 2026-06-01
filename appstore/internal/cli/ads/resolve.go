package ads

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/appleads"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/config"
)

type commonFlags struct {
	AdsProfile *string
	Org        *string
}

func resolveClient(ctx context.Context, flags commonFlags, requiresOrg bool) (*appleads.Client, error) {
	credentials, err := resolveCredentials(flags)
	if err != nil {
		return nil, err
	}
	if requiresOrg {
		orgID, err := resolveOrgID(flags, credentials)
		if err != nil {
			return nil, err
		}
		if orgID == "" {
			return nil, shared.UsageError("--org is required (or set ASC_ADS_ORG_ID or an Ads profile org_id)")
		}
		credentials.OrgID = orgID
	}
	_ = ctx
	return appleads.NewClient(credentials)
}

func resolveCredentials(flags commonFlags) (appleads.Credentials, error) {
	credentials, _, err := resolveCredentialsWithSource(flags)
	return credentials, err
}

func resolveCredentialsWithSource(flags commonFlags) (appleads.Credentials, string, error) {
	profile := strings.TrimSpace(value(flags.AdsProfile))
	profileSource := "--ads-profile"
	if profile == "" {
		profile = strings.TrimSpace(os.Getenv("ASC_ADS_PROFILE"))
		profileSource = "ASC_ADS_PROFILE"
	}
	accessToken := strings.TrimSpace(os.Getenv("ASC_ADS_ACCESS_TOKEN"))
	strict := parseBoolEnv("ASC_ADS_STRICT_AUTH")
	if profile != "" {
		if strict && accessToken != "" {
			return appleads.Credentials{}, "", fmt.Errorf("mixed Apple Ads authentication sources detected: profile and ASC_ADS_ACCESS_TOKEN")
		}
		if strict {
			if _, ok, err := envCredentials(); err != nil {
				return appleads.Credentials{}, "", err
			} else if ok {
				return appleads.Credentials{}, "", fmt.Errorf("mixed Apple Ads authentication sources detected: profile and ASC_ADS_* key credentials")
			}
		}
		credentials, _, err := appleads.GetCredentialsWithSource(profile)
		if err != nil {
			return appleads.Credentials{}, "", err
		}
		return credentials, profileSource, nil
	}
	if accessToken != "" {
		if strict {
			if _, ok, err := envCredentials(); err != nil {
				return appleads.Credentials{}, "", err
			} else if ok {
				return appleads.Credentials{}, "", fmt.Errorf("mixed Apple Ads authentication sources detected: ASC_ADS_ACCESS_TOKEN and ASC_ADS_* key credentials")
			}
		}
		return appleads.Credentials{AccessToken: accessToken}, "ASC_ADS_ACCESS_TOKEN", nil
	}

	env, ok, err := envCredentials()
	if err != nil {
		return appleads.Credentials{}, "", err
	}
	if ok {
		return env, "ASC_ADS_* key credentials", nil
	}

	credentials, _, err := appleads.GetCredentialsWithSource("")
	if err != nil {
		return appleads.Credentials{}, "", err
	}
	return credentials, "default Ads profile", nil
}

func envCredentials() (appleads.Credentials, bool, error) {
	credentials := appleads.Credentials{
		ClientID:       strings.TrimSpace(os.Getenv("ASC_ADS_CLIENT_ID")),
		TeamID:         strings.TrimSpace(os.Getenv("ASC_ADS_TEAM_ID")),
		KeyID:          strings.TrimSpace(os.Getenv("ASC_ADS_KEY_ID")),
		PrivateKeyPath: strings.TrimSpace(os.Getenv("ASC_ADS_PRIVATE_KEY_PATH")),
		PrivateKeyPEM:  strings.TrimSpace(os.Getenv("ASC_ADS_PRIVATE_KEY")),
		OrgID:          strings.TrimSpace(os.Getenv("ASC_ADS_ORG_ID")),
	}
	privateKeyB64 := strings.TrimSpace(os.Getenv("ASC_ADS_PRIVATE_KEY_B64"))
	if privateKeyB64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(privateKeyB64)
		if err != nil {
			return appleads.Credentials{}, false, fmt.Errorf("ASC_ADS_PRIVATE_KEY_B64 is not valid base64: %w", err)
		}
		if credentials.PrivateKeyPEM == "" {
			credentials.PrivateKeyPEM = string(decoded)
		}
	}
	complete := credentials.ClientID != "" &&
		credentials.TeamID != "" &&
		credentials.KeyID != "" &&
		(credentials.PrivateKeyPath != "" || credentials.PrivateKeyPEM != "")
	keyEnvSet := credentials.ClientID != "" ||
		credentials.TeamID != "" ||
		credentials.KeyID != "" ||
		credentials.PrivateKeyPath != "" ||
		credentials.PrivateKeyPEM != ""
	if !complete && keyEnvSet {
		return appleads.Credentials{}, false, fmt.Errorf("incomplete Apple Ads environment credentials: set ASC_ADS_CLIENT_ID, ASC_ADS_TEAM_ID, ASC_ADS_KEY_ID, and one of ASC_ADS_PRIVATE_KEY_PATH, ASC_ADS_PRIVATE_KEY, or ASC_ADS_PRIVATE_KEY_B64")
	}
	return credentials, complete, nil
}

func resolveOrgID(flags commonFlags, credentials appleads.Credentials) (string, error) {
	orgID, _, err := resolveOrgIDWithSource(flags, credentials)
	return orgID, err
}

func resolveOrgIDWithSource(flags commonFlags, credentials appleads.Credentials) (string, string, error) {
	if orgID := value(flags.Org); orgID != "" {
		return orgID, "--org", nil
	}
	if orgID := strings.TrimSpace(os.Getenv("ASC_ADS_ORG_ID")); orgID != "" {
		return orgID, "ASC_ADS_ORG_ID", nil
	}
	if orgID := strings.TrimSpace(credentials.OrgID); orgID != "" {
		if strings.TrimSpace(credentials.Profile) != "" {
			return orgID, "Ads profile org_id", nil
		}
		return orgID, "credential org_id", nil
	}
	cfg, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			return "", "", nil
		}
		return "", "", err
	}
	orgID := strings.TrimSpace(cfg.Ads.OrgID)
	if orgID == "" {
		return "", "", nil
	}
	return orgID, "ads.org_id", nil
}

func requestContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return shared.ContextWithTimeout(ctx)
}

func parseBoolEnv(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func value(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return strings.TrimSpace(*ptr)
}
