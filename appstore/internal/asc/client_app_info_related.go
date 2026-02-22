package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// GetAppInfoPrimaryCategory retrieves the primary category for an app info.
func (c *Client) GetAppInfoPrimaryCategory(ctx context.Context, appInfoID string) (*AppCategoryResponse, error) {
	appInfoID = strings.TrimSpace(appInfoID)
	if appInfoID == "" {
		return nil, fmt.Errorf("appInfoID is required")
	}

	path := fmt.Sprintf("/v1/appInfos/%s/primaryCategory", appInfoID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AppCategoryResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAppInfoPrimarySubcategoryOne retrieves the primary subcategory one for an app info.
func (c *Client) GetAppInfoPrimarySubcategoryOne(ctx context.Context, appInfoID string) (*AppCategoryResponse, error) {
	appInfoID = strings.TrimSpace(appInfoID)
	if appInfoID == "" {
		return nil, fmt.Errorf("appInfoID is required")
	}

	path := fmt.Sprintf("/v1/appInfos/%s/primarySubcategoryOne", appInfoID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AppCategoryResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAppInfoPrimarySubcategoryTwo retrieves the primary subcategory two for an app info.
func (c *Client) GetAppInfoPrimarySubcategoryTwo(ctx context.Context, appInfoID string) (*AppCategoryResponse, error) {
	appInfoID = strings.TrimSpace(appInfoID)
	if appInfoID == "" {
		return nil, fmt.Errorf("appInfoID is required")
	}

	path := fmt.Sprintf("/v1/appInfos/%s/primarySubcategoryTwo", appInfoID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AppCategoryResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAppInfoSecondaryCategory retrieves the secondary category for an app info.
func (c *Client) GetAppInfoSecondaryCategory(ctx context.Context, appInfoID string) (*AppCategoryResponse, error) {
	appInfoID = strings.TrimSpace(appInfoID)
	if appInfoID == "" {
		return nil, fmt.Errorf("appInfoID is required")
	}

	path := fmt.Sprintf("/v1/appInfos/%s/secondaryCategory", appInfoID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AppCategoryResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAppInfoSecondarySubcategoryOne retrieves the secondary subcategory one for an app info.
func (c *Client) GetAppInfoSecondarySubcategoryOne(ctx context.Context, appInfoID string) (*AppCategoryResponse, error) {
	appInfoID = strings.TrimSpace(appInfoID)
	if appInfoID == "" {
		return nil, fmt.Errorf("appInfoID is required")
	}

	path := fmt.Sprintf("/v1/appInfos/%s/secondarySubcategoryOne", appInfoID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AppCategoryResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetAppInfoSecondarySubcategoryTwo retrieves the secondary subcategory two for an app info.
func (c *Client) GetAppInfoSecondarySubcategoryTwo(ctx context.Context, appInfoID string) (*AppCategoryResponse, error) {
	appInfoID = strings.TrimSpace(appInfoID)
	if appInfoID == "" {
		return nil, fmt.Errorf("appInfoID is required")
	}

	path := fmt.Sprintf("/v1/appInfos/%s/secondarySubcategoryTwo", appInfoID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AppCategoryResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
