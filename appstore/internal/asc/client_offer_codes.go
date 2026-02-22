package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetSubscriptionOfferCodeCustomCode retrieves a custom code by ID.
func (c *Client) GetSubscriptionOfferCodeCustomCode(ctx context.Context, customCodeID string) (*SubscriptionOfferCodeCustomCodeResponse, error) {
	path := fmt.Sprintf("/v1/subscriptionOfferCodeCustomCodes/%s", strings.TrimSpace(customCodeID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response SubscriptionOfferCodeCustomCodeResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateSubscriptionOfferCodeCustomCode creates a custom code.
func (c *Client) CreateSubscriptionOfferCodeCustomCode(ctx context.Context, req SubscriptionOfferCodeCustomCodeCreateRequest) (*SubscriptionOfferCodeCustomCodeResponse, error) {
	body, err := BuildRequestBody(req)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/subscriptionOfferCodeCustomCodes", body)
	if err != nil {
		return nil, err
	}

	var response SubscriptionOfferCodeCustomCodeResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateSubscriptionOfferCodeCustomCode updates a custom code.
func (c *Client) UpdateSubscriptionOfferCodeCustomCode(ctx context.Context, customCodeID string, attrs SubscriptionOfferCodeCustomCodeUpdateAttributes) (*SubscriptionOfferCodeCustomCodeResponse, error) {
	payload := SubscriptionOfferCodeCustomCodeUpdateRequest{
		Data: SubscriptionOfferCodeCustomCodeUpdateData{
			Type:       ResourceTypeSubscriptionOfferCodeCustomCodes,
			ID:         strings.TrimSpace(customCodeID),
			Attributes: attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/subscriptionOfferCodeCustomCodes/%s", strings.TrimSpace(customCodeID))
	data, err := c.do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var response SubscriptionOfferCodeCustomCodeResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateSubscriptionOfferCodeOneTimeUseCode updates a one-time use code batch.
func (c *Client) UpdateSubscriptionOfferCodeOneTimeUseCode(ctx context.Context, oneTimeUseCodeID string, attrs SubscriptionOfferCodeOneTimeUseCodeUpdateAttributes) (*SubscriptionOfferCodeOneTimeUseCodeResponse, error) {
	payload := SubscriptionOfferCodeOneTimeUseCodeUpdateRequest{
		Data: SubscriptionOfferCodeOneTimeUseCodeUpdateData{
			Type:       ResourceTypeSubscriptionOfferCodeOneTimeUseCodes,
			ID:         strings.TrimSpace(oneTimeUseCodeID),
			Attributes: attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/subscriptionOfferCodeOneTimeUseCodes/%s", strings.TrimSpace(oneTimeUseCodeID))
	data, err := c.do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var response SubscriptionOfferCodeOneTimeUseCodeResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
