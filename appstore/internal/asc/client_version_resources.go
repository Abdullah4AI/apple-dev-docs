package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetAppStoreVersionAppClipDefaultExperience retrieves the default App Clip experience for a version.
func (c *Client) GetAppStoreVersionAppClipDefaultExperience(ctx context.Context, versionID string) (*AppClipDefaultExperienceResponse, error) {
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return nil, fmt.Errorf("versionID is required")
	}

	path := fmt.Sprintf("/v1/appStoreVersions/%s/appClipDefaultExperience", versionID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response AppClipDefaultExperienceResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAppStoreVersionExperimentsV2ForVersion retrieves v2 experiments for an app store version.
func (c *Client) GetAppStoreVersionExperimentsV2ForVersion(ctx context.Context, versionID string, opts ...AppStoreVersionExperimentsV2Option) (*AppStoreVersionExperimentsV2Response, error) {
	query := &appStoreVersionExperimentsV2Query{}
	for _, opt := range opts {
		opt(query)
	}

	versionID = strings.TrimSpace(versionID)
	if query.nextURL == "" && versionID == "" {
		return nil, fmt.Errorf("versionID is required")
	}

	path := fmt.Sprintf("/v1/appStoreVersions/%s/appStoreVersionExperimentsV2", versionID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("appStoreVersionExperimentsV2: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildAppStoreVersionExperimentsV2Query(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response AppStoreVersionExperimentsV2Response
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAppStoreVersionCustomerReviews retrieves customer reviews for an app store version.
func (c *Client) GetAppStoreVersionCustomerReviews(ctx context.Context, versionID string, opts ...ReviewOption) (*ReviewsResponse, error) {
	query := &reviewQuery{}
	for _, opt := range opts {
		opt(query)
	}

	versionID = strings.TrimSpace(versionID)
	if query.nextURL == "" && versionID == "" {
		return nil, fmt.Errorf("versionID is required")
	}

	path := fmt.Sprintf("/v1/appStoreVersions/%s/customerReviews", versionID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("customerReviews: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildReviewQuery(opts); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response ReviewsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
