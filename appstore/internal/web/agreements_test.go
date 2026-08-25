package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func agreementsTestClient(t *testing.T, fn roundTripFunc) *Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New() error: %v", err)
	}
	return &Client{httpClient: &http.Client{Jar: jar, Transport: fn}}
}

func contractMessagesFixture() string {
	return `[{"id":"contract_message","group":"Alert","subject":"Apple Developer Program License Agreement Updated","message":"The Apple Developer Program License Agreement has been updated and needs to be reviewed.","priority":null}]`
}

func agreementHistoryFixture(accepted bool) string {
	dateAccepted := "0"
	if accepted {
		dateAccepted = "1787158607000"
	}
	return `{"resultCode":0,"agreements":[{"agreementDownloadUrl":"/services-account/agreement/XG8DNV4HYY/content/pdf","dateEffective":1787060333000,"dateAccepted":` + dateAccepted + `,"dateAgreeBy":1790899199000,"status":"active","version":"5031","isAgreementPLA":true,"agreementId":"XG8DNV4HYY","title":"Apple Developer Program License Agreement"}]}`
}

func TestGetAgreementsStatusReportsPendingAgreement(t *testing.T) {
	requestCount := 0
	client := agreementsTestClient(t, func(r *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if r.Method != http.MethodGet || r.URL.String() != "https://appstoreconnect.apple.com/olympus/v1/contractMessages" {
				t.Fatalf("unexpected contract messages request %s %s", r.Method, r.URL.String())
			}
			return developerPortalTestResponse(http.StatusOK, contractMessagesFixture(), nil), nil
		case 2:
			if r.Method != http.MethodPost || r.URL.Path != "/services-account/QH65B2/account/listTeams.action" {
				t.Fatalf("unexpected bootstrap request %s %s", r.Method, r.URL.String())
			}
			return developerPortalTestResponse(http.StatusOK, developerPortalTeamsFixture(), http.Header{"csrf": []string{"bootstrap-csrf"}, "csrf_ts": []string{"bootstrap-ts"}}), nil
		case 3:
			if r.Method != http.MethodPost || r.URL.String() != "https://developer.apple.com/services-account/QH65B2/account/getAgreementHistory" {
				t.Fatalf("unexpected agreement history request %s %s", r.Method, r.URL.String())
			}
			if got := r.Header.Get("csrf"); got != "bootstrap-csrf" {
				t.Fatalf("agreement history request csrf header = %q, want bootstrap-csrf", got)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode agreement history request: %v", err)
			}
			if payload["teamId"] != "TEAM123456" {
				t.Fatalf("agreement history teamId = %v, want TEAM123456", payload["teamId"])
			}
			return developerPortalTestResponse(http.StatusOK, agreementHistoryFixture(false), nil), nil
		default:
			t.Fatalf("unexpected extra request %s %s", r.Method, r.URL.String())
			return nil, nil
		}
	})

	result, err := client.GetAgreementsStatus(context.Background())
	if err != nil {
		t.Fatalf("GetAgreementsStatus() error: %v", err)
	}
	if requestCount != 3 {
		t.Fatalf("request count = %d, want 3", requestCount)
	}
	if result.TeamID != "TEAM123456" {
		t.Fatalf("TeamID = %q, want TEAM123456", result.TeamID)
	}
	if !result.Pending {
		t.Fatal("Pending = false, want true")
	}
	if len(result.ContractMessages) != 1 || result.ContractMessages[0].Subject != "Apple Developer Program License Agreement Updated" {
		t.Fatalf("ContractMessages = %+v, want the license agreement banner", result.ContractMessages)
	}
	if len(result.Agreements) != 1 {
		t.Fatalf("Agreements length = %d, want 1", len(result.Agreements))
	}
	agreement := result.Agreements[0]
	if agreement.AgreementID != "XG8DNV4HYY" {
		t.Fatalf("AgreementID = %q, want XG8DNV4HYY", agreement.AgreementID)
	}
	if !agreement.IsProgramLicenseAgreement {
		t.Fatal("IsProgramLicenseAgreement = false, want true")
	}
	if !agreement.Pending {
		t.Fatal("agreement Pending = false, want true")
	}
	if agreement.Version != "5031" || agreement.Status != "active" {
		t.Fatalf("agreement version/status = %q/%q, want 5031/active", agreement.Version, agreement.Status)
	}
	wantEffective := time.UnixMilli(1787060333000).UTC().Format(time.RFC3339)
	if agreement.DateEffective != wantEffective {
		t.Fatalf("DateEffective = %q, want %q", agreement.DateEffective, wantEffective)
	}
	if agreement.DateAccepted != "" {
		t.Fatalf("DateAccepted = %q, want empty for a pending agreement", agreement.DateAccepted)
	}
	wantAgreeBy := time.UnixMilli(1790899199000).UTC().Format(time.RFC3339)
	if agreement.DateAgreeBy != wantAgreeBy {
		t.Fatalf("DateAgreeBy = %q, want %q", agreement.DateAgreeBy, wantAgreeBy)
	}
	if agreement.DownloadURL != "https://developer.apple.com/services-account/agreement/XG8DNV4HYY/content/pdf" {
		t.Fatalf("DownloadURL = %q, want absolute developer portal URL", agreement.DownloadURL)
	}
}

func TestGetAgreementsStatusNotPendingWhenAccepted(t *testing.T) {
	requestCount := 0
	client := agreementsTestClient(t, func(r *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			return developerPortalTestResponse(http.StatusOK, `[ ]`, nil), nil
		case 2:
			return developerPortalTestResponse(http.StatusOK, developerPortalTeamsFixture(), nil), nil
		case 3:
			return developerPortalTestResponse(http.StatusOK, agreementHistoryFixture(true), nil), nil
		default:
			t.Fatalf("unexpected extra request %s %s", r.Method, r.URL.String())
			return nil, nil
		}
	})

	result, err := client.GetAgreementsStatus(context.Background())
	if err != nil {
		t.Fatalf("GetAgreementsStatus() error: %v", err)
	}
	if result.Pending {
		t.Fatal("Pending = true, want false")
	}
	if len(result.ContractMessages) != 0 {
		t.Fatalf("ContractMessages = %+v, want empty", result.ContractMessages)
	}
	if len(result.Agreements) != 1 || result.Agreements[0].Pending {
		t.Fatalf("Agreements = %+v, want one accepted agreement", result.Agreements)
	}
	if result.Agreements[0].DateAccepted == "" {
		t.Fatal("DateAccepted is empty, want RFC3339 timestamp")
	}
}

func TestGetAgreementsStatusSurfacesResultCodeError(t *testing.T) {
	requestCount := 0
	client := agreementsTestClient(t, func(r *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			return developerPortalTestResponse(http.StatusOK, `[ ]`, nil), nil
		case 2:
			return developerPortalTestResponse(http.StatusOK, developerPortalTeamsFixture(), nil), nil
		default:
			return developerPortalTestResponse(http.StatusOK, `{"resultCode":3050,"resultString":"Please select a team.","userString":"Please select a team."}`, nil), nil
		}
	})

	_, err := client.GetAgreementsStatus(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Please select a team.") {
		t.Fatalf("GetAgreementsStatus() error = %v, want Apple resultCode message", err)
	}
	if err != nil && !strings.Contains(err.Error(), "3050") {
		t.Fatalf("GetAgreementsStatus() error = %v, want resultCode in message", err)
	}
	var resultErr *DeveloperPortalAgreementsResultError
	if !errors.As(err, &resultErr) {
		t.Fatalf("GetAgreementsStatus() error = %T, want *DeveloperPortalAgreementsResultError", err)
	}
	if resultErr.ResultCode != 3050 || resultErr.Message != "Please select a team." {
		t.Fatalf("result error = %+v, want code 3050 and Apple message", resultErr)
	}
}

func TestGetAgreementsStatusSurfacesHTTPStatusErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		wantText   string
		wantAPIErr bool
	}{
		{name: "expired session", status: http.StatusUnauthorized, wantText: "web session is unauthorized or expired for Developer Portal"},
		{name: "server error", status: http.StatusInternalServerError, wantText: "web api error", wantAPIErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requestCount := 0
			client := agreementsTestClient(t, func(r *http.Request) (*http.Response, error) {
				requestCount++
				switch requestCount {
				case 1:
					return developerPortalTestResponse(http.StatusOK, `[]`, nil), nil
				case 2:
					return developerPortalTestResponse(http.StatusOK, developerPortalTeamsFixture(), nil), nil
				default:
					return developerPortalTestResponse(tc.status, `{}`, nil), nil
				}
			})

			_, err := client.GetAgreementsStatus(context.Background())
			if err == nil || !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("GetAgreementsStatus() error = %v, want text %q", err, tc.wantText)
			}
			var apiErr *APIError
			if errors.As(err, &apiErr) != tc.wantAPIErr {
				t.Fatalf("GetAgreementsStatus() error = %T, APIError match = %t, want %t", err, errors.As(err, &apiErr), tc.wantAPIErr)
			}
		})
	}
}

func TestAcceptAgreementsSendsTeamAndAgreementIDs(t *testing.T) {
	requestCount := 0
	client := agreementsTestClient(t, func(r *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/services-account/QH65B2/account/listTeams.action" {
				t.Fatalf("unexpected bootstrap request %s %s", r.Method, r.URL.String())
			}
			return developerPortalTestResponse(http.StatusOK, developerPortalTeamsFixture(), nil), nil
		case 2:
			if r.Method != http.MethodPost || r.URL.String() != "https://developer.apple.com/services-account/QH65B2/account/acceptAgreements" {
				t.Fatalf("unexpected accept request %s %s", r.Method, r.URL.String())
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read accept request body: %v", err)
			}
			var payload struct {
				TeamID       string   `json:"teamId"`
				AgreementIDs []string `json:"agreementIds"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode accept request body %q: %v", string(body), err)
			}
			if payload.TeamID != "TEAM123456" {
				t.Fatalf("accept teamId = %q, want TEAM123456", payload.TeamID)
			}
			if len(payload.AgreementIDs) != 1 || payload.AgreementIDs[0] != "XG8DNV4HYY" {
				t.Fatalf("accept agreementIds = %v, want [XG8DNV4HYY]", payload.AgreementIDs)
			}
			return developerPortalTestResponse(http.StatusOK, agreementHistoryFixture(true), nil), nil
		default:
			t.Fatalf("unexpected extra request %s %s", r.Method, r.URL.String())
			return nil, nil
		}
	})

	result, err := client.AcceptAgreements(context.Background(), AgreementsAcceptRequest{AgreementIDs: []string{" XG8DNV4HYY "}})
	if err != nil {
		t.Fatalf("AcceptAgreements() error: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want 2", requestCount)
	}
	if result.TeamID != "TEAM123456" {
		t.Fatalf("TeamID = %q, want TEAM123456", result.TeamID)
	}
	if result.Status != "accepted" {
		t.Fatalf("Status = %q, want accepted", result.Status)
	}
	if len(result.AgreementIDs) != 1 || result.AgreementIDs[0] != "XG8DNV4HYY" {
		t.Fatalf("AgreementIDs = %v, want [XG8DNV4HYY]", result.AgreementIDs)
	}
	if len(result.Agreements) != 1 || result.Agreements[0].Pending || result.Agreements[0].DateAccepted == "" {
		t.Fatalf("Agreements = %+v, want one accepted agreement", result.Agreements)
	}
}

func TestAcceptAgreementsRequiresAgreementID(t *testing.T) {
	client := agreementsTestClient(t, func(r *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		return nil, nil
	})

	_, err := client.AcceptAgreements(context.Background(), AgreementsAcceptRequest{AgreementIDs: []string{"  "}})
	if err == nil || !strings.Contains(err.Error(), "agreement id is required") {
		t.Fatalf("AcceptAgreements() error = %v, want missing agreement id error", err)
	}
}

func TestAcceptAgreementsSurfacesResultCodeError(t *testing.T) {
	requestCount := 0
	client := agreementsTestClient(t, func(r *http.Request) (*http.Response, error) {
		requestCount++
		if requestCount == 1 {
			return developerPortalTestResponse(http.StatusOK, developerPortalTeamsFixture(), nil), nil
		}
		return developerPortalTestResponse(http.StatusOK, `{"resultCode":2200,"resultString":"Not allowed.","userString":"Only the Account Holder can accept this agreement."}`, nil), nil
	})

	_, err := client.AcceptAgreements(context.Background(), AgreementsAcceptRequest{AgreementIDs: []string{"XG8DNV4HYY"}})
	if err == nil || !strings.Contains(err.Error(), "Only the Account Holder can accept this agreement.") {
		t.Fatalf("AcceptAgreements() error = %v, want Apple user message", err)
	}
}

func TestAcceptAgreementsRejectsMissingResultCode(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch requestCount {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != developerPortalTeamsPath {
				t.Errorf("bootstrap request = %s %s, want POST %s", r.Method, r.URL.Path, developerPortalTeamsPath)
			}
			if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
				t.Errorf("bootstrap Content-Type = %q, want application/x-www-form-urlencoded", got)
			}
			_, _ = io.WriteString(w, developerPortalTeamsFixture())
		case 2:
			if r.Method != http.MethodPost || r.URL.Path != developerPortalAcceptAgreementsPath {
				t.Errorf("accept request = %s %s, want POST %s", r.Method, r.URL.Path, developerPortalAcceptAgreementsPath)
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("accept Content-Type = %q, want application/json", got)
			}
			if got := r.Header.Get("X-Requested-With"); got != "XMLHttpRequest" {
				t.Errorf("accept X-Requested-With = %q, want XMLHttpRequest", got)
			}
			var payload struct {
				TeamID       string   `json:"teamId"`
				AgreementIDs []string `json:"agreementIds"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode accept payload: %v", err)
			}
			if payload.TeamID != "TEAM123456" || len(payload.AgreementIDs) != 1 || payload.AgreementIDs[0] != "XG8DNV4HYY" {
				t.Errorf("accept payload = %+v, want selected team and agreement", payload)
			}
			_, _ = io.WriteString(w, `{"agreements":[]}`)
		default:
			t.Errorf("unexpected request %d: %s %s", requestCount, r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	client := &Client{httpClient: server.Client(), developerPortalURL: server.URL}

	result, err := client.AcceptAgreements(context.Background(), AgreementsAcceptRequest{AgreementIDs: []string{"XG8DNV4HYY"}})
	if err == nil || !strings.Contains(err.Error(), "missing resultCode") {
		t.Fatalf("AcceptAgreements() result/error = %+v/%v, want missing resultCode error", result, err)
	}
}
