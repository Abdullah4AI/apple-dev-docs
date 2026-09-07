package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// CIScmProvider is one SCM provider returned by the Xcode Cloud web API.
//
// The API is private and its response can gain fields independently of the
// typed client. Raw therefore keeps the original provider object available to
// JSON callers while the known fields provide stable values for human output.
// Boolean pointers distinguish an omitted value from an explicit false value.
type CIScmProvider struct {
	Raw                 json.RawMessage `json:"-"`
	ID                  string          `json:"id"`
	Provider            string          `json:"provider"`
	ProviderDisplayName string          `json:"provider_display_name"`
	IsRegistered        *bool           `json:"is_registered,omitempty"`
	IsUserConnected     *bool           `json:"is_user_connected,omitempty"`
}

// UnmarshalJSON captures the source object while decoding fields used by the
// table and Markdown renderers.
func (p *CIScmProvider) UnmarshalJSON(data []byte) error {
	type alias CIScmProvider
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	decoded.Raw = append(json.RawMessage(nil), data...)
	*p = CIScmProvider(decoded)
	return nil
}

// MarshalJSON returns Apple's original provider object when it came from the
// API, retaining unknown fields and the source snake_case keys.
func (p CIScmProvider) MarshalJSON() ([]byte, error) {
	if raw := bytes.TrimSpace(p.Raw); len(raw) > 0 {
		if !json.Valid(raw) {
			return nil, fmt.Errorf("invalid raw CI SCM provider response")
		}
		return append([]byte(nil), raw...), nil
	}
	type alias CIScmProvider
	return json.Marshal(alias(p))
}

// CIScmConnectionStatus is the health response for one known SCM provider.
// Error remains opaque because its schema is unverified; Raw preserves the
// complete response, including unknown top-level fields.
type CIScmConnectionStatus struct {
	Raw    json.RawMessage `json:"-"`
	Status string          `json:"status"`
	Error  json.RawMessage `json:"error,omitempty"`
}

// UnmarshalJSON captures the source status object while retaining an opaque
// error value exactly as JSON.
func (s *CIScmConnectionStatus) UnmarshalJSON(data []byte) error {
	type alias CIScmConnectionStatus
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	decoded.Raw = append(json.RawMessage(nil), data...)
	decoded.Error = append(json.RawMessage(nil), decoded.Error...)
	*s = CIScmConnectionStatus(decoded)
	return nil
}

// MarshalJSON returns Apple's original connection-status object when it came
// from the API, retaining unknown fields and the source response shape.
func (s CIScmConnectionStatus) MarshalJSON() ([]byte, error) {
	if raw := bytes.TrimSpace(s.Raw); len(raw) > 0 {
		if !json.Valid(raw) {
			return nil, fmt.Errorf("invalid raw CI SCM connection status response")
		}
		return append([]byte(nil), raw...), nil
	}
	type alias CIScmConnectionStatus
	return json.Marshal(alias(s))
}

func ciScmProvidersPath(teamID string) string {
	return "/teams/" + url.PathEscape(teamID) + "/scm-providers-v2"
}

func ciScmConnectionStatusPath(teamID, scmProviderID string) string {
	return "/teams/" + url.PathEscape(teamID) + "/scm-providers/" + url.PathEscape(scmProviderID) + "/connection-v2"
}

// GetCIScmProviders returns the web Xcode Cloud SCM provider inventory.
// The endpoint returns a plain JSON array and does not expose pagination.
func (c *Client) GetCIScmProviders(ctx context.Context, teamID string) ([]CIScmProvider, error) {
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		return nil, fmt.Errorf("team id is required")
	}
	body, err := c.doRequest(ctx, "GET", ciScmProvidersPath(teamID), nil)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("failed to decode ci scm providers: response must be a JSON array")
	}
	var result []CIScmProvider
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode ci scm providers: %w", err)
	}
	return result, nil
}

// GetCIScmConnectionStatus returns the web connection health for one SCM
// provider selected by its private provider ID.
func (c *Client) GetCIScmConnectionStatus(ctx context.Context, teamID, scmProviderID string) (*CIScmConnectionStatus, error) {
	teamID = strings.TrimSpace(teamID)
	if teamID == "" {
		return nil, fmt.Errorf("team id is required")
	}
	scmProviderID = strings.TrimSpace(scmProviderID)
	if scmProviderID == "" {
		return nil, fmt.Errorf("scm provider id is required")
	}
	body, err := c.doRequest(ctx, "GET", ciScmConnectionStatusPath(teamID, scmProviderID), nil)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("failed to decode ci scm connection status: response must be a JSON object")
	}
	var result CIScmConnectionStatus
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode ci scm connection status: %w", err)
	}
	if strings.TrimSpace(result.Status) == "" {
		return nil, fmt.Errorf("failed to decode ci scm connection status: response missing status")
	}
	return &result, nil
}
