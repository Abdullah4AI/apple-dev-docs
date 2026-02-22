package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetAccessibilityDeclarations retrieves accessibility declarations for an app.
func (c *Client) GetAccessibilityDeclarations(ctx context.Context, appID string, opts ...AccessibilityDeclarationsOption) (*AccessibilityDeclarationsResponse, error) {
	query := &accessibilityDeclarationsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/apps/%s/accessibilityDeclarations", strings.TrimSpace(appID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("accessibility-declarations: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildAccessibilityDeclarationsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response AccessibilityDeclarationsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAccessibilityDeclaration retrieves a single accessibility declaration by ID.
func (c *Client) GetAccessibilityDeclaration(ctx context.Context, declarationID string, fields []string) (*AccessibilityDeclarationResponse, error) {
	declarationID = strings.TrimSpace(declarationID)
	path := fmt.Sprintf("/v1/accessibilityDeclarations/%s", declarationID)
	if queryString := buildAccessibilityDeclarationsFieldsQuery(fields); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response AccessibilityDeclarationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateAccessibilityDeclaration creates a new accessibility declaration.
func (c *Client) CreateAccessibilityDeclaration(ctx context.Context, appID string, attrs AccessibilityDeclarationCreateAttributes) (*AccessibilityDeclarationResponse, error) {
	payload := AccessibilityDeclarationCreateRequest{
		Data: AccessibilityDeclarationCreateData{
			Type:       ResourceTypeAccessibilityDeclarations,
			Attributes: attrs,
			Relationships: &AccessibilityDeclarationRelationships{
				App: &Relationship{
					Data: ResourceData{
						Type: ResourceTypeApps,
						ID:   strings.TrimSpace(appID),
					},
				},
			},
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/accessibilityDeclarations", body)
	if err != nil {
		return nil, err
	}

	var response AccessibilityDeclarationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateAccessibilityDeclaration updates an accessibility declaration by ID.
func (c *Client) UpdateAccessibilityDeclaration(ctx context.Context, declarationID string, attrs AccessibilityDeclarationUpdateAttributes) (*AccessibilityDeclarationResponse, error) {
	declarationID = strings.TrimSpace(declarationID)
	payload := AccessibilityDeclarationUpdateRequest{
		Data: AccessibilityDeclarationUpdateData{
			Type: ResourceTypeAccessibilityDeclarations,
			ID:   declarationID,
		},
	}
	if HasAccessibilityDeclarationUpdates(attrs) {
		payload.Data.Attributes = &attrs
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/v1/accessibilityDeclarations/%s", declarationID), body)
	if err != nil {
		return nil, err
	}

	var response AccessibilityDeclarationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// DeleteAccessibilityDeclaration deletes an accessibility declaration by ID.
func (c *Client) DeleteAccessibilityDeclaration(ctx context.Context, declarationID string) error {
	path := fmt.Sprintf("/v1/accessibilityDeclarations/%s", strings.TrimSpace(declarationID))
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	return err
}

// HasAccessibilityDeclarationUpdates reports whether any update attributes are set.
func HasAccessibilityDeclarationUpdates(attrs AccessibilityDeclarationUpdateAttributes) bool {
	return attrs.Publish != nil ||
		attrs.SupportsAudioDescriptions != nil ||
		attrs.SupportsCaptions != nil ||
		attrs.SupportsDarkInterface != nil ||
		attrs.SupportsDifferentiateWithoutColorAlone != nil ||
		attrs.SupportsLargerText != nil ||
		attrs.SupportsReducedMotion != nil ||
		attrs.SupportsSufficientContrast != nil ||
		attrs.SupportsVoiceControl != nil ||
		attrs.SupportsVoiceover != nil
}
