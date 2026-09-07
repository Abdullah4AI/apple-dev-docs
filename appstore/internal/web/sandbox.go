package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
)

// SandboxAccountCreateAttributes defines inputs for creating a sandbox tester
// via App Store Connect's private web session endpoints.
type SandboxAccountCreateAttributes struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	AccountName     string `json:"acAccountName"`
	AccountPassword string `json:"acAccountPassword"`
	StoreFront      string `json:"storeFront"`
}

const sandboxAccountListLimit = 50

// SandboxAccount is the small account projection needed by the private
// sandbox delete flow. IsInFamily is a pointer so a missing field cannot be
// mistaken for Apple's explicit false value during destructive preflight.
type SandboxAccount struct {
	ID          string `json:"id"`
	IsInFamily  *bool  `json:"isInFamily"`
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	AccountName string `json:"acAccountName,omitempty"`
	StoreFront  string `json:"storeFront,omitempty"`
}

// SandboxAccountListResponse is the response from Apple's private sandbox
// account collection endpoint.
type SandboxAccountListResponse struct {
	TotalAccounts         int              `json:"totalAccounts"`
	TotalInactiveAccounts int              `json:"totalInactiveAccounts"`
	Accounts              []SandboxAccount `json:"accounts"`
}

func normalizeSandboxAccountCreateAttributes(attrs SandboxAccountCreateAttributes) (SandboxAccountCreateAttributes, error) {
	attrs.FirstName = strings.TrimSpace(attrs.FirstName)
	attrs.LastName = strings.TrimSpace(attrs.LastName)
	attrs.AccountName = strings.TrimSpace(attrs.AccountName)
	attrs.AccountPassword = strings.TrimSpace(attrs.AccountPassword)
	attrs.StoreFront = strings.ToUpper(strings.TrimSpace(attrs.StoreFront))

	if attrs.FirstName == "" {
		return attrs, fmt.Errorf("first name is required")
	}
	if attrs.LastName == "" {
		return attrs, fmt.Errorf("last name is required")
	}
	if attrs.AccountName == "" {
		return attrs, fmt.Errorf("account name is required")
	}
	parsedAddress, err := mail.ParseAddress(attrs.AccountName)
	if err != nil || parsedAddress == nil || strings.TrimSpace(parsedAddress.Address) != attrs.AccountName {
		return attrs, fmt.Errorf("account name must be a valid email address")
	}
	if attrs.AccountPassword == "" {
		return attrs, fmt.Errorf("account password is required")
	}
	if attrs.StoreFront == "" {
		return attrs, fmt.Errorf("store front is required")
	}
	return attrs, nil
}

func sandboxOriginBaseURL(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return appStoreBaseURL
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return appStoreBaseURL
	}
	return parsed.Scheme + "://" + parsed.Host
}

func (c *Client) doSandboxRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	origin := sandboxOriginBaseURL(c.baseURL)

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", "*/*")
	headers.Set("Origin", origin)
	headers.Set("Referer", origin+"/access/users/sandbox")

	return c.doRequestBase(ctx, origin, method, path, body, headers)
}

func (c *Client) validateSandboxAccountFields(ctx context.Context, body map[string]string) error {
	if _, err := c.doSandboxRequest(ctx, http.MethodPost, "/sandbox/v2/account/validateFields", body); err != nil {
		return err
	}
	return nil
}

// CreateSandboxAccount creates a sandbox tester by mirroring the current App
// Store Connect web flow: validate fields twice, then submit the create request.
func (c *Client) CreateSandboxAccount(ctx context.Context, attrs SandboxAccountCreateAttributes) error {
	normalized, err := normalizeSandboxAccountCreateAttributes(attrs)
	if err != nil {
		return err
	}

	validateBody := map[string]string{
		"firstName":     normalized.FirstName,
		"lastName":      normalized.LastName,
		"acAccountName": normalized.AccountName,
	}
	if err := c.validateSandboxAccountFields(ctx, validateBody); err != nil {
		return err
	}

	validateWithPasswordBody := map[string]string{
		"firstName":         normalized.FirstName,
		"lastName":          normalized.LastName,
		"acAccountName":     normalized.AccountName,
		"acAccountPassword": normalized.AccountPassword,
	}
	if err := c.validateSandboxAccountFields(ctx, validateWithPasswordBody); err != nil {
		return err
	}

	createBody := map[string]string{
		"firstName":         normalized.FirstName,
		"lastName":          normalized.LastName,
		"acAccountName":     normalized.AccountName,
		"acAccountPassword": normalized.AccountPassword,
		"storeFront":        normalized.StoreFront,
	}
	if _, err := c.doSandboxRequest(ctx, http.MethodPost, "/sandbox/v2/account/create", createBody); err != nil {
		return err
	}
	return nil
}

// ListSandboxAccounts reads the private sandbox account collection used by
// App Store Connect's web tester management screen. The captured web flow
// requests the first 50 accounts and does not expose a verified continuation
// contract, so callers must treat a larger total as an incomplete snapshot.
func (c *Client) ListSandboxAccounts(ctx context.Context) (*SandboxAccountListResponse, error) {
	path := fmt.Sprintf("/sandbox/v2/provider/account/list?limit=%d", sandboxAccountListLimit)
	data, err := c.doSandboxRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var raw struct {
		TotalAccounts         *int             `json:"totalAccounts"`
		TotalInactiveAccounts *int             `json:"totalInactiveAccounts"`
		Accounts              []SandboxAccount `json:"accounts"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse sandbox account list response: %w", err)
	}
	if raw.TotalAccounts == nil || raw.TotalInactiveAccounts == nil || raw.Accounts == nil {
		return nil, fmt.Errorf("failed to parse sandbox account list response: missing totals or accounts")
	}

	return &SandboxAccountListResponse{
		TotalAccounts:         *raw.TotalAccounts,
		TotalInactiveAccounts: *raw.TotalInactiveAccounts,
		Accounts:              raw.Accounts,
	}, nil
}

func normalizeSandboxAccountIDs(ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one sandbox account ID is required")
	}

	seen := make(map[string]struct{}, len(ids))
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, fmt.Errorf("sandbox account ID must not be empty")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized, nil
}

// DeleteSandboxAccounts deletes the selected private sandbox accounts. Apple
// accepts the selected IDs as one JSON object; the response body is ignored to
// mirror the web client, so callers must verify the result with a fresh list.
func (c *Client) DeleteSandboxAccounts(ctx context.Context, ids []string) error {
	normalized, err := normalizeSandboxAccountIDs(ids)
	if err != nil {
		return err
	}

	body := struct {
		IDs []string `json:"ids"`
	}{IDs: normalized}
	_, err = c.doSandboxRequest(ctx, http.MethodPost, "/sandbox/v2/account/delete", body)
	return err
}
