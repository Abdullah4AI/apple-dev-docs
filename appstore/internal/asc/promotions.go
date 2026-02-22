package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AppStoreVersionPromotionAttributes describes promotion metadata.
type AppStoreVersionPromotionAttributes struct {
	CreatedDate string `json:"createdDate,omitempty"`
}

// AppStoreVersionPromotionRelationships describes promotion relationships.
type AppStoreVersionPromotionRelationships struct {
	AppStoreVersion                    *Relationship `json:"appStoreVersion,omitempty"`
	AppStoreVersionExperimentTreatment *Relationship `json:"appStoreVersionExperimentTreatment,omitempty"`
}

// AppStoreVersionPromotionCreateData is the data portion of a promotion create request.
type AppStoreVersionPromotionCreateData struct {
	Type          ResourceType                          `json:"type"`
	Relationships AppStoreVersionPromotionRelationships `json:"relationships"`
}

// AppStoreVersionPromotionCreateRequest is a request to create a version promotion.
type AppStoreVersionPromotionCreateRequest struct {
	Data AppStoreVersionPromotionCreateData `json:"data"`
}

// AppStoreVersionPromotionResponse is the response from promotion create endpoints.
type AppStoreVersionPromotionResponse struct {
	Data  Resource[AppStoreVersionPromotionAttributes] `json:"data"`
	Links Links                                        `json:"links"`
}

// CreateAppStoreVersionPromotion creates a version promotion.
func (c *Client) CreateAppStoreVersionPromotion(ctx context.Context, versionID, treatmentID string) (*AppStoreVersionPromotionResponse, error) {
	versionID = strings.TrimSpace(versionID)
	treatmentID = strings.TrimSpace(treatmentID)
	if versionID == "" {
		return nil, fmt.Errorf("version ID is required")
	}
	if treatmentID == "" {
		return nil, fmt.Errorf("treatment ID is required")
	}

	payload := AppStoreVersionPromotionCreateRequest{
		Data: AppStoreVersionPromotionCreateData{
			Type: ResourceTypeAppStoreVersionPromotions,
			Relationships: AppStoreVersionPromotionRelationships{
				AppStoreVersion: &Relationship{
					Data: ResourceData{
						Type: ResourceTypeAppStoreVersions,
						ID:   versionID,
					},
				},
				AppStoreVersionExperimentTreatment: &Relationship{
					Data: ResourceData{
						Type: ResourceTypeAppStoreVersionExperimentTreatments,
						ID:   treatmentID,
					},
				},
			},
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/appStoreVersionPromotions", body)
	if err != nil {
		return nil, err
	}

	var response AppStoreVersionPromotionResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
