package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AppDistribution captures the app-level distribution method attributes that
// App Store Connect exposes only through the internal web API.
type AppDistribution struct {
	AppID                 string `json:"appId"`
	Name                  string `json:"name,omitempty"`
	BundleID              string `json:"bundleId,omitempty"`
	DistributionType      string `json:"distributionType,omitempty"`
	EducationDiscountType string `json:"educationDiscountType,omitempty"`
}

// GetAppDistribution retrieves the app-level distribution method settings.
//
// The public App Store Connect API does not expose distributionType, so this
// reads the internal apps resource, which returns the attribute verbatim.
func (c *Client) GetAppDistribution(ctx context.Context, appID string) (*AppDistribution, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("app id is required")
	}

	responseBody, err := c.doRequest(ctx, http.MethodGet, "/apps/"+url.PathEscape(appID), nil)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Data jsonAPIResource `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse app distribution response: %w", err)
	}

	result := &AppDistribution{
		AppID:                 strings.TrimSpace(payload.Data.ID),
		Name:                  stringAttr(payload.Data.Attributes, "name"),
		BundleID:              stringAttr(payload.Data.Attributes, "bundleId"),
		DistributionType:      stringAttr(payload.Data.Attributes, "distributionType"),
		EducationDiscountType: stringAttr(payload.Data.Attributes, "educationDiscountType"),
	}
	if result.AppID == "" {
		result.AppID = appID
	}
	return result, nil
}
