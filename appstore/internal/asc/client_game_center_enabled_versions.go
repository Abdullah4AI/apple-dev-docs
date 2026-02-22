package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetAppGameCenterEnabledVersions retrieves enabled Game Center versions for an app.
func (c *Client) GetAppGameCenterEnabledVersions(ctx context.Context, appID string, opts ...GCEnabledVersionsOption) (*GameCenterEnabledVersionsResponse, error) {
	query := &gcEnabledVersionsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	appID = strings.TrimSpace(appID)
	if query.nextURL == "" && appID == "" {
		return nil, fmt.Errorf("appID is required")
	}

	path := fmt.Sprintf("/v1/apps/%s/gameCenterEnabledVersions", appID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-enabled-versions: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCEnabledVersionsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterEnabledVersionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterEnabledVersionCompatibleVersions retrieves compatible enabled versions.
func (c *Client) GetGameCenterEnabledVersionCompatibleVersions(ctx context.Context, enabledVersionID string, opts ...GCEnabledVersionsOption) (*GameCenterEnabledVersionsResponse, error) {
	query := &gcEnabledVersionsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	enabledVersionID = strings.TrimSpace(enabledVersionID)
	if query.nextURL == "" && enabledVersionID == "" {
		return nil, fmt.Errorf("enabledVersionID is required")
	}

	path := fmt.Sprintf("/v1/gameCenterEnabledVersions/%s/compatibleVersions", enabledVersionID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-enabled-versions-compatibility: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCEnabledVersionsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterEnabledVersionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
