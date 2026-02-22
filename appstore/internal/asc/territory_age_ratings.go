package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// TerritoryAgeRatingAttributes describes territory age rating attributes.
type TerritoryAgeRatingAttributes struct {
	AppStoreAgeRating AppStoreAgeRating `json:"appStoreAgeRating,omitempty"`
}

// TerritoryAgeRatingRelationships describes territory age rating relationships.
type TerritoryAgeRatingRelationships struct {
	Territory Relationship `json:"territory"`
}

// TerritoryAgeRatingsResponse is the response from territory age ratings endpoints.
type TerritoryAgeRatingsResponse = Response[TerritoryAgeRatingAttributes]

// GetAppInfoTerritoryAgeRatings retrieves territory age ratings for an app info.
func (c *Client) GetAppInfoTerritoryAgeRatings(ctx context.Context, appInfoID string, opts ...TerritoryAgeRatingsOption) (*TerritoryAgeRatingsResponse, error) {
	query := &territoryAgeRatingsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	appInfoID = strings.TrimSpace(appInfoID)
	if query.nextURL == "" && appInfoID == "" {
		return nil, fmt.Errorf("appInfoID is required")
	}

	path := fmt.Sprintf("/v1/appInfos/%s/territoryAgeRatings", appInfoID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("territoryAgeRatings: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildTerritoryAgeRatingsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response TerritoryAgeRatingsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
