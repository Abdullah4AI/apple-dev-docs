package web

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

func stubWebAgreementsSession(t *testing.T) {
	t.Helper()
	origResolveSession := resolveSessionFn
	origNewWebClient := newWebClientFn
	origPersist := persistWebSessionFn
	t.Cleanup(func() {
		resolveSessionFn = origResolveSession
		newWebClientFn = origNewWebClient
		persistWebSessionFn = origPersist
	})
	resolveSessionFn = func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{}, "cache", nil
	}
	newWebClientFn = func(session *webcore.AuthSession) *webcore.Client {
		return &webcore.Client{}
	}
	persistWebSessionFn = func(session *webcore.AuthSession) error {
		return nil
	}
}

func TestWebAgreementsAcceptValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "missing agreement id", args: []string{"--confirm"}, wantErr: "--agreement-id is required"},
		{name: "missing confirm", args: []string{"--agreement-id", "XG8DNV4HYY"}, wantErr: "--confirm is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := WebAgreementsAcceptCommand()
			if err := cmd.FlagSet.Parse(tc.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			stdout, stderr := captureWebCommandOutput(t, func() {
				err := cmd.Exec(context.Background(), nil)
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected flag.ErrHelp, got %v", err)
				}
			})
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, tc.wantErr) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.wantErr, stderr)
			}
		})
	}
}

func TestWebAgreementsRejectPositionalArgs(t *testing.T) {
	tests := []struct {
		name    string
		cmd     func(t *testing.T) (exec func(context.Context, []string) error)
		wantErr string
	}{
		{
			name: "status",
			cmd: func(t *testing.T) func(context.Context, []string) error {
				t.Helper()
				return WebAgreementsStatusCommand().Exec
			},
			wantErr: "web agreements status does not accept positional arguments",
		},
		{
			name: "accept",
			cmd: func(t *testing.T) func(context.Context, []string) error {
				t.Helper()
				cmd := WebAgreementsAcceptCommand()
				if err := cmd.FlagSet.Parse([]string{"--agreement-id", "XG8DNV4HYY", "--confirm"}); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				return cmd.Exec
			},
			wantErr: "web agreements accept does not accept positional arguments",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exec := tc.cmd(t)
			stdout, stderr := captureWebCommandOutput(t, func() {
				err := exec(context.Background(), []string{"unexpected"})
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected flag.ErrHelp, got %v", err)
				}
			})
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, tc.wantErr) {
				t.Fatalf("expected stderr to contain %q, got %q", tc.wantErr, stderr)
			}
		})
	}
}

func TestWebAgreementsCommandsAndFlagsAreExperimental(t *testing.T) {
	group := WebAgreementsCommand()
	if !strings.HasPrefix(group.ShortHelp, "[experimental] ") {
		t.Fatalf("group ShortHelp = %q, want experimental prefix", group.ShortHelp)
	}

	status := WebAgreementsStatusCommand()
	if !strings.HasPrefix(status.ShortHelp, "[experimental] ") {
		t.Fatalf("status ShortHelp = %q, want experimental prefix", status.ShortHelp)
	}

	accept := WebAgreementsAcceptCommand()
	if !strings.HasPrefix(accept.ShortHelp, "[experimental] ") {
		t.Fatalf("accept ShortHelp = %q, want experimental prefix", accept.ShortHelp)
	}
	for _, name := range []string{"agreement-id", "confirm"} {
		flag := accept.FlagSet.Lookup(name)
		if flag == nil {
			t.Fatalf("expected --%s flag", name)
		}
		if !strings.HasPrefix(flag.Usage, "[experimental] ") {
			t.Fatalf("--%s usage = %q, want experimental prefix", name, flag.Usage)
		}
	}
}

func TestWebAgreementsStatusNonJSONOutputIncludesBannerOnlyPendingState(t *testing.T) {
	stubWebAgreementsSession(t)

	origStatus := getAgreementsStatusFn
	t.Cleanup(func() { getAgreementsStatusFn = origStatus })
	getAgreementsStatusFn = func(ctx context.Context, client *webcore.Client) (*asc.WebAgreementsStatusResult, error) {
		return &asc.WebAgreementsStatusResult{
			TeamID:  "TEAM123456",
			Pending: true,
			ContractMessages: []asc.WebAgreementContractMessage{{
				ID:      "contract_message",
				Group:   "Alert",
				Subject: "Apple Developer Program License Agreement Updated",
			}},
		}, nil
	}

	for _, format := range []string{"table", "markdown"} {
		t.Run(format, func(t *testing.T) {
			cmd := WebAgreementsStatusCommand()
			if err := cmd.FlagSet.Parse([]string{"--output", format}); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			stdout, _ := captureWebCommandOutput(t, func() {
				if err := cmd.Exec(context.Background(), nil); err != nil {
					t.Fatalf("Exec() error: %v", err)
				}
			})
			for _, want := range []string{"TEAM123456", "true", "Apple Developer Program License Agreement Updated"} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("%s output missing %q: %q", format, want, stdout)
				}
			}
		})
	}
}

func TestWebAgreementsStatusPrintsJSON(t *testing.T) {
	stubWebAgreementsSession(t)

	origStatus := getAgreementsStatusFn
	t.Cleanup(func() { getAgreementsStatusFn = origStatus })
	getAgreementsStatusFn = func(ctx context.Context, client *webcore.Client) (*asc.WebAgreementsStatusResult, error) {
		return &asc.WebAgreementsStatusResult{
			TeamID:  "TEAM123456",
			Pending: true,
			ContractMessages: []asc.WebAgreementContractMessage{{
				ID:      "contract_message",
				Group:   "Alert",
				Subject: "Apple Developer Program License Agreement Updated",
			}},
			Agreements: []asc.WebAgreement{{
				AgreementID:               "XG8DNV4HYY",
				Title:                     "Apple Developer Program License Agreement",
				Status:                    "active",
				Version:                   "5031",
				IsProgramLicenseAgreement: true,
				Pending:                   true,
				DateEffective:             "2026-08-14T09:38:53Z",
				DateAgreeBy:               "2026-10-01T23:59:59Z",
			}},
		}, nil
	}

	cmd := WebAgreementsStatusCommand()
	if err := cmd.FlagSet.Parse([]string{"--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, _ := captureWebCommandOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("Exec() error: %v", err)
		}
	})

	var payload struct {
		TeamID           string `json:"teamId"`
		Pending          bool   `json:"pending"`
		ContractMessages []struct {
			Subject string `json:"subject"`
		} `json:"contractMessages"`
		Agreements []struct {
			AgreementID string `json:"agreementId"`
			Pending     bool   `json:"pending"`
		} `json:"agreements"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("unmarshal stdout %q: %v", stdout, err)
	}
	if payload.TeamID != "TEAM123456" || !payload.Pending {
		t.Fatalf("payload = %+v, want pending TEAM123456", payload)
	}
	if len(payload.ContractMessages) != 1 || !strings.Contains(payload.ContractMessages[0].Subject, "License Agreement Updated") {
		t.Fatalf("contractMessages = %+v, want banner entry", payload.ContractMessages)
	}
	if len(payload.Agreements) != 1 || payload.Agreements[0].AgreementID != "XG8DNV4HYY" || !payload.Agreements[0].Pending {
		t.Fatalf("agreements = %+v, want pending XG8DNV4HYY", payload.Agreements)
	}
}

func TestWebAgreementsAcceptCallsClient(t *testing.T) {
	stubWebAgreementsSession(t)

	var gotReq webcore.AgreementsAcceptRequest
	origAccept := acceptAgreementsFn
	t.Cleanup(func() { acceptAgreementsFn = origAccept })
	acceptAgreementsFn = func(ctx context.Context, client *webcore.Client, req webcore.AgreementsAcceptRequest) (*asc.WebAgreementsAcceptResult, error) {
		gotReq = req
		return &asc.WebAgreementsAcceptResult{
			TeamID:       "TEAM123456",
			AgreementIDs: []string{"XG8DNV4HYY"},
			Status:       "accepted",
			Agreements: []asc.WebAgreement{{
				AgreementID:  "XG8DNV4HYY",
				Version:      "5031",
				Status:       "active",
				DateAccepted: "2026-08-19T16:56:47Z",
			}},
		}, nil
	}

	cmd := WebAgreementsAcceptCommand()
	if err := cmd.FlagSet.Parse([]string{"--agreement-id", "XG8DNV4HYY", "--confirm", "--output", "json"}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	stdout, _ := captureWebCommandOutput(t, func() {
		if err := cmd.Exec(context.Background(), nil); err != nil {
			t.Fatalf("Exec() error: %v", err)
		}
	})

	if len(gotReq.AgreementIDs) != 1 || gotReq.AgreementIDs[0] != "XG8DNV4HYY" {
		t.Fatalf("accept request = %+v, want agreement XG8DNV4HYY", gotReq)
	}

	var payload struct {
		TeamID       string   `json:"teamId"`
		AgreementIDs []string `json:"agreementIds"`
		Status       string   `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("unmarshal stdout %q: %v", stdout, err)
	}
	if payload.Status != "accepted" || payload.TeamID != "TEAM123456" || len(payload.AgreementIDs) != 1 {
		t.Fatalf("payload = %+v, want accepted receipt", payload)
	}
}
