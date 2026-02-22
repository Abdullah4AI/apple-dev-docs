package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetGameCenterAppVersions retrieves the list of Game Center app versions.
func (c *Client) GetGameCenterAppVersions(ctx context.Context, opts ...GCAppVersionsOption) (*GameCenterAppVersionsResponse, error) {
	query := &gcAppVersionsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := "/v1/gameCenterAppVersions"
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-app-versions: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCAppVersionsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterAppVersionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterAppVersion retrieves a Game Center app version by ID.
func (c *Client) GetGameCenterAppVersion(ctx context.Context, appVersionID string) (*GameCenterAppVersionResponse, error) {
	path := fmt.Sprintf("/v1/gameCenterAppVersions/%s", strings.TrimSpace(appVersionID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterAppVersionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterDetailGameCenterAppVersions retrieves Game Center app versions for a detail.
func (c *Client) GetGameCenterDetailGameCenterAppVersions(ctx context.Context, detailID string, opts ...GCAppVersionsOption) (*GameCenterAppVersionsResponse, error) {
	query := &gcAppVersionsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	detailID = strings.TrimSpace(detailID)
	if query.nextURL == "" && detailID == "" {
		return nil, fmt.Errorf("detailID is required")
	}

	path := fmt.Sprintf("/v1/gameCenterDetails/%s/gameCenterAppVersions", detailID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-detail-app-versions: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCAppVersionsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterAppVersionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterAppVersionAppStoreVersion retrieves the related App Store version.
func (c *Client) GetGameCenterAppVersionAppStoreVersion(ctx context.Context, appVersionID string) (*AppStoreVersionResponse, error) {
	path := fmt.Sprintf("/v1/gameCenterAppVersions/%s/appStoreVersion", strings.TrimSpace(appVersionID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response AppStoreVersionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterAppVersionCompatibilityVersions retrieves compatible Game Center app versions.
func (c *Client) GetGameCenterAppVersionCompatibilityVersions(ctx context.Context, appVersionID string, opts ...GCAppVersionsOption) (*GameCenterAppVersionsResponse, error) {
	query := &gcAppVersionsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	appVersionID = strings.TrimSpace(appVersionID)
	if query.nextURL == "" && appVersionID == "" {
		return nil, fmt.Errorf("appVersionID is required")
	}

	path := fmt.Sprintf("/v1/gameCenterAppVersions/%s/compatibilityVersions", appVersionID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-app-version-compatibility: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCAppVersionsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterAppVersionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateGameCenterAppVersion creates a new Game Center app version.
func (c *Client) CreateGameCenterAppVersion(ctx context.Context, appStoreVersionID string) (*GameCenterAppVersionResponse, error) {
	appStoreVersionID = strings.TrimSpace(appStoreVersionID)
	if appStoreVersionID == "" {
		return nil, fmt.Errorf("appStoreVersionID is required")
	}

	payload := GameCenterAppVersionCreateRequest{
		Data: GameCenterAppVersionCreateData{
			Type: ResourceTypeGameCenterAppVersions,
			Relationships: &GameCenterAppVersionRelationships{
				AppStoreVersion: &Relationship{
					Data: ResourceData{
						Type: ResourceTypeAppStoreVersions,
						ID:   appStoreVersionID,
					},
				},
			},
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/gameCenterAppVersions", body)
	if err != nil {
		return nil, err
	}

	var response GameCenterAppVersionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateGameCenterAppVersion updates an existing Game Center app version.
func (c *Client) UpdateGameCenterAppVersion(ctx context.Context, appVersionID string, attrs GameCenterAppVersionUpdateAttributes) (*GameCenterAppVersionResponse, error) {
	appVersionID = strings.TrimSpace(appVersionID)
	if appVersionID == "" {
		return nil, fmt.Errorf("appVersionID is required")
	}
	if attrs.Enabled == nil {
		return nil, fmt.Errorf("at least one attribute is required")
	}

	payload := GameCenterAppVersionUpdateRequest{
		Data: GameCenterAppVersionUpdateData{
			Type:       ResourceTypeGameCenterAppVersions,
			ID:         appVersionID,
			Attributes: &attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/gameCenterAppVersions/%s", appVersionID)
	data, err := c.do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var response GameCenterAppVersionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAppStoreVersionGameCenterAppVersion retrieves the related Game Center app version.
func (c *Client) GetAppStoreVersionGameCenterAppVersion(ctx context.Context, versionID string) (*GameCenterAppVersionResponse, error) {
	path := fmt.Sprintf("/v1/appStoreVersions/%s/gameCenterAppVersion", strings.TrimSpace(versionID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterAppVersionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
