package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

const (
	developerPortalAgreementHistoryPath = "/services-account/QH65B2/account/getAgreementHistory"
	developerPortalAcceptAgreementsPath = "/services-account/QH65B2/account/acceptAgreements"
	olympusContractMessagesPath         = "/contractMessages"
)

// AgreementsAcceptRequest accepts one or more Developer Portal agreements.
type AgreementsAcceptRequest struct {
	AgreementIDs []string
}

// developerAgreementsEnvelope is the QH65B2 account services response shape.
// These endpoints answer HTTP 200 even for failures; resultCode carries the
// outcome (0 success; for example 3050 missing team, 3100 unknown team).
type developerAgreementsEnvelope struct {
	ResultCode   *int                       `json:"resultCode"`
	ResultString string                     `json:"resultString"`
	UserString   string                     `json:"userString"`
	Agreements   []developerAgreementRecord `json:"agreements"`
}

type developerAgreementRecord struct {
	AgreementID          string `json:"agreementId"`
	Title                string `json:"title"`
	Status               string `json:"status"`
	Version              string `json:"version"`
	DateEffective        int64  `json:"dateEffective"`
	DateAccepted         int64  `json:"dateAccepted"`
	DateAgreeBy          int64  `json:"dateAgreeBy"`
	IsAgreementPLA       bool   `json:"isAgreementPLA"`
	AgreementDownloadURL string `json:"agreementDownloadUrl"`
}

// DeveloperPortalAgreementsResultError reports an agreement-services failure
// returned inside an otherwise successful HTTP response.
type DeveloperPortalAgreementsResultError struct {
	ResultCode int
	Message    string
}

func (e *DeveloperPortalAgreementsResultError) Error() string {
	return fmt.Sprintf("developer portal agreements request failed (resultCode %d): %s", e.ResultCode, e.Message)
}

// GetAgreementsStatus reports the App Store Connect agreement banner and the
// team's Developer Portal agreement history in one pending-aware summary.
func (c *Client) GetAgreementsStatus(ctx context.Context) (*asc.WebAgreementsStatusResult, error) {
	messages, err := c.getContractMessages(ctx)
	if err != nil {
		return nil, err
	}
	result := &asc.WebAgreementsStatusResult{
		Pending:          len(messages) > 0,
		ContractMessages: messages,
	}
	if err := c.ensureDeveloperPortalSession(ctx); err != nil {
		return result, err
	}
	teamID := c.developerPortalTeamID()
	if teamID == "" {
		return result, fmt.Errorf("developer portal team is not selected; %s", developerPortalAuthHint)
	}
	result.TeamID = teamID
	envelope, err := c.doDeveloperPortalAgreementsRequest(ctx, developerPortalAgreementHistoryPath, map[string]string{"teamId": teamID})
	if err != nil {
		return result, err
	}

	agreements := make([]asc.WebAgreement, 0, len(envelope.Agreements))
	for _, record := range envelope.Agreements {
		agreement := c.newAgreement(record)
		if agreement.Pending {
			result.Pending = true
		}
		agreements = append(agreements, agreement)
	}
	result.Agreements = agreements
	return result, nil
}

// AcceptAgreements accepts the given Developer Portal agreements for the web
// session's team. Apple only allows the Account Holder to accept agreements.
func (c *Client) AcceptAgreements(ctx context.Context, req AgreementsAcceptRequest) (*asc.WebAgreementsAcceptResult, error) {
	agreementIDs := make([]string, 0, len(req.AgreementIDs))
	for _, id := range req.AgreementIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			agreementIDs = append(agreementIDs, id)
		}
	}
	if len(agreementIDs) == 0 {
		return nil, fmt.Errorf("at least one agreement id is required")
	}

	if err := c.ensureDeveloperPortalSession(ctx); err != nil {
		return nil, err
	}
	teamID := c.developerPortalTeamID()
	if teamID == "" {
		return nil, fmt.Errorf("developer portal team is not selected; %s", developerPortalAuthHint)
	}
	payload := struct {
		TeamID       string   `json:"teamId"`
		AgreementIDs []string `json:"agreementIds"`
	}{TeamID: teamID, AgreementIDs: agreementIDs}
	envelope, err := c.doDeveloperPortalAgreementsRequest(ctx, developerPortalAcceptAgreementsPath, payload)
	if err != nil {
		return nil, err
	}

	agreements := make([]asc.WebAgreement, 0, len(envelope.Agreements))
	for _, record := range envelope.Agreements {
		agreements = append(agreements, c.newAgreement(record))
	}
	return &asc.WebAgreementsAcceptResult{
		TeamID:       teamID,
		AgreementIDs: agreementIDs,
		Status:       "accepted",
		Agreements:   agreements,
	}, nil
}

func (c *Client) getContractMessages(ctx context.Context) ([]asc.WebAgreementContractMessage, error) {
	body, err := c.doOlympusGet(ctx, olympusContractMessagesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch App Store Connect contract messages: %w", err)
	}
	var messages []asc.WebAgreementContractMessage
	if err := json.Unmarshal(body, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse App Store Connect contract messages: %w", err)
	}
	return messages, nil
}

func (c *Client) doDeveloperPortalAgreementsRequest(ctx context.Context, path string, payload any) (*developerAgreementsEnvelope, error) {
	headers := developerPortalAgreementsHeaders()
	csrf, csrfTS := c.developerCSRFTokens()
	if csrf != "" {
		headers.Set("csrf", csrf)
	}
	if csrfTS != "" {
		headers.Set("csrf_ts", csrfTS)
	}
	body, response, err := c.doDeveloperPortalHTTP(ctx, http.MethodPost, c.developerPortalOrigin()+path, payload, headers)
	if err != nil {
		return nil, err
	}
	c.captureDeveloperCSRFTokens(response.Header)
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, developerPortalSessionError(response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, &APIError{
			Status:         response.StatusCode,
			AppleRequestID: extractAppleRequestID(response.Header),
			CorrelationKey: strings.TrimSpace(response.Header.Get("X-Apple-Jingle-Correlation-Key")),
			rawBody:        body,
		}
	}
	var envelope developerAgreementsEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse Developer Portal agreements response: %w", err)
	}
	if envelope.ResultCode == nil {
		return nil, fmt.Errorf("developer portal agreements response is missing resultCode")
	}
	if *envelope.ResultCode != 0 {
		message := strings.TrimSpace(envelope.UserString)
		if message == "" {
			message = strings.TrimSpace(envelope.ResultString)
		}
		if message == "" {
			message = "unknown Developer Portal error"
		}
		return nil, &DeveloperPortalAgreementsResultError{ResultCode: *envelope.ResultCode, Message: message}
	}
	return &envelope, nil
}

func (c *Client) newAgreement(record developerAgreementRecord) asc.WebAgreement {
	downloadURL := strings.TrimSpace(record.AgreementDownloadURL)
	if downloadURL != "" && strings.HasPrefix(downloadURL, "/") {
		downloadURL = c.developerPortalOrigin() + downloadURL
	}
	return asc.WebAgreement{
		AgreementID:               strings.TrimSpace(record.AgreementID),
		Title:                     strings.TrimSpace(record.Title),
		Status:                    strings.TrimSpace(record.Status),
		Version:                   strings.TrimSpace(record.Version),
		IsProgramLicenseAgreement: record.IsAgreementPLA,
		Pending:                   record.DateAccepted < record.DateEffective,
		DateEffective:             formatAgreementDate(record.DateEffective),
		DateAccepted:              formatAgreementDate(record.DateAccepted),
		DateAgreeBy:               formatAgreementDate(record.DateAgreeBy),
		DownloadURL:               downloadURL,
	}
}

func formatAgreementDate(milliseconds int64) string {
	if milliseconds <= 0 {
		return ""
	}
	return time.UnixMilli(milliseconds).UTC().Format(time.RFC3339)
}

func developerPortalAgreementsHeaders() http.Header {
	headers := make(http.Header)
	headers.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	headers.Set("Content-Type", "application/json")
	headers.Set("Referer", developerPortalBaseURL+"/account")
	headers.Set("User-Agent", "App-Store-Connect-CLI")
	headers.Set("X-Requested-With", "XMLHttpRequest")
	return headers
}
