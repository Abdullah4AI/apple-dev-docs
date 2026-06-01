package cmdtest

import (
	"encoding/json"
	"errors"
	"flag"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdsAuthEvalWorkflow(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	keyPath := filepath.Join(t.TempDir(), "apple-ads-private-key.pem")
	writeECDSAPEM(t, keyPath)

	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_ADS_BYPASS_KEYCHAIN", "1")

	stdout, stderr, err := runAdsEvalCommand(
		t,
		"ads", "auth", "login",
		"--bypass-keychain",
		"--name", "Marketing",
		"--client-id", "SEARCHADS.CLIENT",
		"--team-id", "SEARCHADS.TEAM",
		"--key-id", "KEY_ID",
		"--private-key", keyPath,
		"--org", "987654",
	)
	if err != nil {
		t.Fatalf("login error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("login stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Successfully registered Apple Ads API key 'Marketing'") {
		t.Fatalf("login stdout = %q", stdout)
	}

	stdout, stderr, err = runAdsEvalCommand(t, "ads", "auth", "status", "--output", "json")
	if err != nil {
		t.Fatalf("status error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("status stderr = %q, want empty", stderr)
	}
	var status struct {
		Storage     string `json:"storage"`
		Credentials []struct {
			Name     string `json:"name"`
			ClientID string `json:"client_id"`
			OrgID    string `json:"org_id"`
			Default  bool   `json:"default"`
			Source   string `json:"source"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("status stdout is not JSON: %v\n%s", err, stdout)
	}
	if status.Storage != "Config File" || len(status.Credentials) != 1 {
		t.Fatalf("status = %+v, want one config credential", status)
	}
	got := status.Credentials[0]
	if got.Name != "Marketing" || got.ClientID != "SEARCHADS.CLIENT" || got.OrgID != "987654" || !got.Default || got.Source != "config" {
		t.Fatalf("credential = %+v, want stored Marketing profile", got)
	}

	stdout, stderr, err = runAdsEvalCommand(t, "ads", "auth", "switch", "--name", "Marketing")
	if err != nil {
		t.Fatalf("switch error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" || !strings.Contains(stdout, "Default Apple Ads profile set to 'Marketing'") {
		t.Fatalf("switch stdout = %q stderr = %q", stdout, stderr)
	}

	stdout, stderr, err = runAdsEvalCommand(t, "ads", "auth", "logout", "--name", "Marketing")
	if err != nil {
		t.Fatalf("logout error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" || !strings.Contains(stdout, "Successfully removed Apple Ads credential 'Marketing'") {
		t.Fatalf("logout stdout = %q stderr = %q", stdout, stderr)
	}
}

func TestAdsAuthStatusShowsActiveEnvironmentContext(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")
	t.Setenv("ASC_ADS_ORG_ID", "987654")

	stdout, stderr, err := runAdsEvalCommand(t, "ads", "auth", "status", "--output", "json")
	if err != nil {
		t.Fatalf("status error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("status stderr = %q, want empty", stderr)
	}
	if strings.Contains(stdout, `"ACCESS"`) {
		t.Fatalf("status leaked access token: %s", stdout)
	}
	var status struct {
		Active struct {
			Source      string `json:"source"`
			OrgID       string `json:"org_id"`
			OrgIDSource string `json:"org_id_source"`
		} `json:"active"`
		Credentials []struct {
			Name string `json:"name"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("status stdout is not JSON: %v\n%s", err, stdout)
	}
	if status.Active.Source != "ASC_ADS_ACCESS_TOKEN" || status.Active.OrgID != "987654" || status.Active.OrgIDSource != "ASC_ADS_ORG_ID" {
		t.Fatalf("active = %+v, want env access token and org source", status.Active)
	}
	if len(status.Credentials) != 0 {
		t.Fatalf("credentials = %+v, want no stored credentials", status.Credentials)
	}

	stdout, stderr, err = runAdsEvalCommand(t, "ads", "auth", "status")
	if err != nil {
		t.Fatalf("table status error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("table status stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Active auth: ASC_ADS_ACCESS_TOKEN",
		"Org ID: 987654 (ASC_ADS_ORG_ID)",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("table status = %q, missing %q", stdout, want)
		}
	}
}

func TestAdsAuthStatusSurfacesMissingNamedProfile(t *testing.T) {
	configPath := writeAdsEvalPayload(t, "config.json", `{"ads":{"keys":[]}}`)
	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_ADS_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_ADS_PROFILE", "Missing")

	stdout, stderr, err := runAdsEvalCommand(t, "ads", "auth", "status", "--output", "json")
	if err != nil {
		t.Fatalf("status error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("status stderr = %q, want empty", stderr)
	}
	var status struct {
		Active struct {
			Error string `json:"error"`
		} `json:"active"`
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("status stdout is not JSON: %v\n%s", err, stdout)
	}
	if !strings.Contains(status.Active.Error, `credentials not found for profile "Missing"`) {
		t.Fatalf("active.error = %q, want missing named profile", status.Active.Error)
	}

	stdout, stderr, err = runAdsEvalCommand(t, "ads", "auth", "status")
	if err != nil {
		t.Fatalf("table status error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("table status stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, `Active auth: unavailable (credentials not found for profile "Missing")`) {
		t.Fatalf("table status = %q, want missing named profile surfaced", stdout)
	}
}

func TestAdsAuthStatusOmitsBlankConfigOrgSource(t *testing.T) {
	configPath := writeAdsEvalPayload(t, "config.json", `{"ads":{"org_id":"   "}}`)
	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_ADS_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")

	stdout, stderr, err := runAdsEvalCommand(t, "ads", "auth", "status", "--output", "json")
	if err != nil {
		t.Fatalf("status error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("status stderr = %q, want empty", stderr)
	}
	var status struct {
		Active struct {
			Source      string `json:"source"`
			OrgID       string `json:"org_id"`
			OrgIDSource string `json:"org_id_source"`
		} `json:"active"`
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("status stdout is not JSON: %v\n%s", err, stdout)
	}
	if status.Active.Source != "ASC_ADS_ACCESS_TOKEN" {
		t.Fatalf("active.source = %q, want ASC_ADS_ACCESS_TOKEN", status.Active.Source)
	}
	if status.Active.OrgID != "" || status.Active.OrgIDSource != "" {
		t.Fatalf("active org = %+v, want no org ID or source", status.Active)
	}
}

func TestAdsAuthStatusKeepsAuthSourceWhenOptionalOrgConfigIsInvalid(t *testing.T) {
	configPath := writeAdsEvalPayload(t, "config.json", `{"ads":`)
	t.Setenv("ASC_CONFIG_PATH", configPath)
	t.Setenv("ASC_ADS_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")

	stdout, stderr, err := runAdsEvalCommand(t, "ads", "auth", "status", "--output", "json")
	if err != nil {
		t.Fatalf("status error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("status stderr = %q, want empty", stderr)
	}
	var status struct {
		Active struct {
			Source string `json:"source"`
			Error  string `json:"error"`
		} `json:"active"`
		CredentialsError string `json:"credentials_error"`
	}
	if err := json.Unmarshal([]byte(stdout), &status); err != nil {
		t.Fatalf("status stdout is not JSON: %v\n%s", err, stdout)
	}
	if status.Active.Source != "ASC_ADS_ACCESS_TOKEN" {
		t.Fatalf("active.source = %q, want ASC_ADS_ACCESS_TOKEN", status.Active.Source)
	}
	if status.Active.Error == "" || !strings.Contains(status.Active.Error, "failed to parse config") {
		t.Fatalf("active.error = %q, want config parse error", status.Active.Error)
	}
	if status.CredentialsError == "" || !strings.Contains(status.CredentialsError, "failed to parse config") {
		t.Fatalf("credentials_error = %q, want config parse error", status.CredentialsError)
	}

	stdout, stderr, err = runAdsEvalCommand(t, "ads", "auth", "status")
	if err != nil {
		t.Fatalf("table status error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("table status stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Active auth: ASC_ADS_ACCESS_TOKEN",
		"Org ID: unavailable (",
		"Stored credentials: unavailable (",
		"failed to parse config",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("table status = %q, missing %q", stdout, want)
		}
	}

	stdout, stderr, err = runAdsEvalCommand(t, "ads", "auth", "status", "--validate", "--output", "json")
	if _, ok := errors.AsType[ReportedError](err); !ok {
		t.Fatalf("validate status error = %T %v, want ReportedError", err, err)
	}
	if !strings.Contains(err.Error(), "validation skipped because credentials could not be listed") {
		t.Fatalf("validate status error = %v, want validation skipped error", err)
	}
	if stderr != "" {
		t.Fatalf("validate status stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, `"source":"ASC_ADS_ACCESS_TOKEN"`) || !strings.Contains(stdout, `"credentials_error"`) {
		t.Fatalf("validate status stdout = %q, want active source and credentials_error", stdout)
	}
}

func TestAdsAuthEvalValidatesUsageErrors(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("ASC_ADS_BYPASS_KEYCHAIN", "1")

	_, stderr, err := runAdsEvalCommand(t, "ads", "auth", "login")
	if !errors.Is(err, flag.ErrHelp) || !strings.Contains(stderr, "--name is required") {
		t.Fatalf("login error = %v stderr = %q, want missing name usage error", err, stderr)
	}

	_, stderr, err = runAdsEvalCommand(t, "ads", "auth", "login", "--local", "--name", "Marketing")
	if !errors.Is(err, flag.ErrHelp) || !strings.Contains(stderr, "--client-id is required") {
		t.Fatalf("login partial error = %v stderr = %q, want missing client ID usage error", err, stderr)
	}

	_, stderr, err = runAdsEvalCommand(t, "ads", "auth", "status", "--output", "markdown")
	if !errors.Is(err, flag.ErrHelp) || !strings.Contains(stderr, "unsupported format: markdown") {
		t.Fatalf("status error = %v stderr = %q, want invalid output usage error", err, stderr)
	}

	_, stderr, err = runAdsEvalCommand(t, "ads", "auth", "status", "--output", "json", "unexpected")
	if !errors.Is(err, flag.ErrHelp) || !strings.Contains(stderr, "unexpected argument(s): unexpected") {
		t.Fatalf("status args error = %v stderr = %q, want unexpected argument usage error", err, stderr)
	}

	_, stderr, err = runAdsEvalCommand(t, "ads", "auth", "logout", "--all", "--name", "Marketing")
	if !errors.Is(err, flag.ErrHelp) || !strings.Contains(stderr, "--all and --name are mutually exclusive") {
		t.Fatalf("logout error = %v stderr = %q, want mutually exclusive usage error", err, stderr)
	}

	_, stderr, err = runAdsEvalCommand(t, "ads", "auth", "logout", "--all")
	if !errors.Is(err, flag.ErrHelp) || !strings.Contains(stderr, "--all requires --confirm") {
		t.Fatalf("logout error = %v stderr = %q, want confirm usage error", err, stderr)
	}

	_, stderr, err = runAdsEvalCommand(t, "ads", "auth", "logout")
	if !errors.Is(err, flag.ErrHelp) || !strings.Contains(stderr, "provide --name or --all") {
		t.Fatalf("logout error = %v stderr = %q, want explicit target usage error", err, stderr)
	}
}

func TestAdsAuthTokenEvalRequiresConfirm(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("ASC_ADS_ACCESS_TOKEN", "ACCESS")

	_, stderr, err := runAdsEvalCommand(t, "ads", "auth", "token", "--output", "json")
	if !errors.Is(err, flag.ErrHelp) || !strings.Contains(stderr, "--confirm is required") {
		t.Fatalf("token error = %v stderr = %q, want confirm usage error", err, stderr)
	}

	stdout, stderr, err := runAdsEvalCommand(t, "ads", "auth", "token", "--confirm", "--output", "json")
	if err != nil {
		t.Fatalf("token error: %v\nstderr: %s", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("token stderr = %q, want empty", stderr)
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("token stdout is not JSON: %v\n%s", err, stdout)
	}
	if parsed.AccessToken != "ACCESS" {
		t.Fatalf("access_token = %q, want ACCESS", parsed.AccessToken)
	}
}
