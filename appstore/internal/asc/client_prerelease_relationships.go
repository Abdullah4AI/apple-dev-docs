package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// PreReleaseVersionAppLinkageResponse is the response for pre-release version app relationships.
type PreReleaseVersionAppLinkageResponse struct {
	Data  ResourceData `json:"data"`
	Links Links        `json:"links"`
}

// GetPreReleaseVersionAppRelationship retrieves the app linkage for a pre-release version.
func (c *Client) GetPreReleaseVersionAppRelationship(ctx context.Context, versionID string) (*PreReleaseVersionAppLinkageResponse, error) {
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return nil, fmt.Errorf("versionID is required")
	}

	path := fmt.Sprintf("/v1/preReleaseVersions/%s/relationships/app", versionID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response PreReleaseVersionAppLinkageResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse app relationship response: %w", err)
	}

	return &response, nil
}

// GetPreReleaseVersionBuildsRelationships retrieves build linkages for a pre-release version.
func (c *Client) GetPreReleaseVersionBuildsRelationships(ctx context.Context, versionID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getResourceLinkages(
		ctx,
		versionID,
		"builds",
		"versionID",
		"/v1/preReleaseVersions/%s/relationships/%s",
		"preReleaseVersionBuilds",
		opts...,
	)
}
