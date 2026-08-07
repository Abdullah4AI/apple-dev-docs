package web

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const apiKeyTypePublic = "PUBLIC_API"

// ErrAPIKeyResponseInvalid reports a malformed or incomplete one-time P8 response.
var ErrAPIKeyResponseInvalid = errors.New("invalid api key download response")

// APIKeyCreateAttributes describes a team API key created through a web session.
type APIKeyCreateAttributes struct {
	Nickname string
	Role     string
}

// APIKey contains non-secret metadata for an App Store Connect API key.
type APIKey struct {
	KeyID          string   `json:"keyId"`
	Name           string   `json:"name,omitempty"`
	IssuerID       string   `json:"issuerId,omitempty"`
	Roles          []string `json:"roles,omitempty"`
	Active         bool     `json:"active"`
	AllAppsVisible bool     `json:"allAppsVisible"`
	CanDownload    bool     `json:"canDownload"`
	KeyType        string   `json:"keyType,omitempty"`
	LastUsed       string   `json:"lastUsed,omitempty"`
	RevokingDate   string   `json:"revokingDate,omitempty"`
}

type apiKeyResource struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes struct {
		Nickname       string   `json:"nickname"`
		Roles          []string `json:"roles"`
		AllAppsVisible bool     `json:"allAppsVisible"`
		CanDownload    bool     `json:"canDownload"`
		IsActive       bool     `json:"isActive"`
		KeyType        string   `json:"keyType"`
		LastUsed       string   `json:"lastUsed"`
		RevokingDate   string   `json:"revokingDate"`
		PrivateKey     string   `json:"privateKey"`
		Provider       *struct {
			ID string `json:"id"`
		} `json:"provider"`
	} `json:"attributes"`
	Relationships struct {
		Provider struct {
			Data *struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"provider"`
	} `json:"relationships"`
}

type apiKeyPayload struct {
	Data     apiKeyResource   `json:"data"`
	Included []apiKeyResource `json:"included"`
}

// CreateAPIKey creates an all-apps team API key using the selected web session.
func (c *Client) CreateAPIKey(ctx context.Context, attrs APIKeyCreateAttributes) (*APIKey, error) {
	nickname := strings.TrimSpace(attrs.Nickname)
	if nickname == "" {
		return nil, fmt.Errorf("api key nickname is required")
	}
	role := strings.ToUpper(strings.TrimSpace(attrs.Role))
	if role == "" {
		return nil, fmt.Errorf("api key role is required")
	}

	request := map[string]any{
		"data": map[string]any{
			"type": "apiKeys",
			"attributes": map[string]any{
				"nickname":       nickname,
				"roles":          []string{role},
				"allAppsVisible": true,
				"keyType":        apiKeyTypePublic,
			},
		},
	}
	body, err := c.doIrisV1Request(ctx, http.MethodPost, "/apiKeys", request)
	if err != nil {
		return nil, err
	}
	return parseAPIKeyResponse(body, "create api key")
}

// GetAPIKey returns API key metadata, including its issuer/provider ID.
func (c *Client) GetAPIKey(ctx context.Context, keyID string) (*APIKey, error) {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return nil, fmt.Errorf("api key id is required")
	}
	path := "/apiKeys/" + url.PathEscape(keyID) + "?include=provider"
	body, err := c.doIrisV1Request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return parseAPIKeyResponse(body, "get api key")
}

// DownloadAPIKey downloads and decodes the one-time P8 for an API key.
func (c *Client) DownloadAPIKey(ctx context.Context, keyID string) ([]byte, error) {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return nil, fmt.Errorf("api key id is required")
	}
	query := url.Values{}
	query.Set("fields[apiKeys]", "privateKey")
	path := "/apiKeys/" + url.PathEscape(keyID) + "?" + query.Encode()
	body, err := c.doIrisV1Request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var payload apiKeyPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("%w: failed to parse JSON: %w", ErrAPIKeyResponseInvalid, err)
	}
	encoded := strings.TrimSpace(payload.Data.Attributes.PrivateKey)
	if encoded == "" {
		return nil, fmt.Errorf("%w: response did not include a P8", ErrAPIKeyResponseInvalid)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(encoded)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode P8: %w", ErrAPIKeyResponseInvalid, err)
	}
	if !bytes.Contains(decoded, []byte("-----BEGIN PRIVATE KEY-----")) || !bytes.Contains(decoded, []byte("-----END PRIVATE KEY-----")) {
		return nil, fmt.Errorf("%w: response contained an invalid P8", ErrAPIKeyResponseInvalid)
	}
	return decoded, nil
}

// IsAPIKeyDownloadRetryable reports whether a newly created key download may
// succeed after Apple's API-key resource finishes propagating.
func IsAPIKeyDownloadRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrAPIKeyResponseInvalid) {
		return false
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return true
	}
	switch apiErr.Status {
	case http.StatusNotFound, http.StatusConflict, http.StatusTooManyRequests:
		return true
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusBadRequest:
		return false
	default:
		return apiErr.Status >= http.StatusInternalServerError
	}
}

func parseAPIKeyResponse(body []byte, operation string) (*APIKey, error) {
	var payload apiKeyPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse %s response: %w", operation, err)
	}
	if strings.TrimSpace(payload.Data.ID) == "" {
		return nil, fmt.Errorf("%s response did not include a key id", operation)
	}

	issuerID := ""
	if payload.Data.Attributes.Provider != nil {
		issuerID = strings.TrimSpace(payload.Data.Attributes.Provider.ID)
	}
	if issuerID == "" && payload.Data.Relationships.Provider.Data != nil {
		issuerID = strings.TrimSpace(payload.Data.Relationships.Provider.Data.ID)
	}

	return &APIKey{
		KeyID:          strings.TrimSpace(payload.Data.ID),
		Name:           strings.TrimSpace(payload.Data.Attributes.Nickname),
		IssuerID:       issuerID,
		Roles:          append([]string(nil), payload.Data.Attributes.Roles...),
		Active:         payload.Data.Attributes.IsActive,
		AllAppsVisible: payload.Data.Attributes.AllAppsVisible,
		CanDownload:    payload.Data.Attributes.CanDownload,
		KeyType:        strings.TrimSpace(payload.Data.Attributes.KeyType),
		LastUsed:       strings.TrimSpace(payload.Data.Attributes.LastUsed),
		RevokingDate:   strings.TrimSpace(payload.Data.Attributes.RevokingDate),
	}, nil
}
