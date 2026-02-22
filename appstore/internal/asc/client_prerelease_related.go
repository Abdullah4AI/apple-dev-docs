package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// GetPreReleaseVersionApp retrieves the app for a pre-release version.
func (c *Client) GetPreReleaseVersionApp(ctx context.Context, versionID string) (*AppResponse, error) {
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return nil, fmt.Errorf("versionID is required")
	}

	path := fmt.Sprintf("/v1/preReleaseVersions/%s/app", versionID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AppResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetPreReleaseVersionBuilds retrieves builds for a pre-release version.
func (c *Client) GetPreReleaseVersionBuilds(ctx context.Context, versionID string, opts ...PreReleaseVersionBuildsOption) (*BuildsResponse, error) {
	query := &listQuery{}
	for _, opt := range opts {
		opt(query)
	}

	versionID = strings.TrimSpace(versionID)
	if query.nextURL == "" && versionID == "" {
		return nil, fmt.Errorf("versionID is required")
	}

	path := fmt.Sprintf("/v1/preReleaseVersions/%s/builds", versionID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("preReleaseVersionBuilds: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildListQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response BuildsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
