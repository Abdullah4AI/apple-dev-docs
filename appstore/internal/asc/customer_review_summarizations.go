package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// CustomerReviewSummarizationAttributes describes a customer review summarization.
type CustomerReviewSummarizationAttributes struct {
	CreatedDate string   `json:"createdDate"`
	Locale      string   `json:"locale"`
	Platform    Platform `json:"platform"`
	Text        string   `json:"text"`
}

// CustomerReviewSummarizationRelationships describes customer review summarization relationships.
type CustomerReviewSummarizationRelationships struct {
	Territory *Relationship `json:"territory,omitempty"`
}

// CustomerReviewSummarizationResource represents a customer review summarization resource.
type CustomerReviewSummarizationResource struct {
	Type          ResourceType                              `json:"type"`
	ID            string                                    `json:"id"`
	Attributes    CustomerReviewSummarizationAttributes     `json:"attributes"`
	Relationships *CustomerReviewSummarizationRelationships `json:"relationships,omitempty"`
}

// CustomerReviewSummarizationsResponse is the response from customer review summarizations endpoints.
type CustomerReviewSummarizationsResponse struct {
	Data     []CustomerReviewSummarizationResource `json:"data"`
	Links    Links                                 `json:"links"`
	Included json.RawMessage                       `json:"included,omitempty"`
	Meta     json.RawMessage                       `json:"meta,omitempty"`
}

// GetCustomerReviewSummarizations retrieves review summarizations for an app.
func (c *Client) GetCustomerReviewSummarizations(ctx context.Context, appID string, opts ...CustomerReviewSummarizationsOption) (*CustomerReviewSummarizationsResponse, error) {
	query := &customerReviewSummarizationsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	appID = strings.TrimSpace(appID)

	path := fmt.Sprintf("/v1/apps/%s/customerReviewSummarizations", appID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("customerReviewSummarizations: %w", err)
		}
		path = query.nextURL
	} else {
		if appID == "" {
			return nil, fmt.Errorf("appID is required")
		}
	}
	if query.nextURL == "" {
		if queryString := buildCustomerReviewSummarizationsQuery(query); queryString != "" {
			path += "?" + queryString
		}
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response CustomerReviewSummarizationsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetLinks returns the pagination links for review summarizations.
func (r *CustomerReviewSummarizationsResponse) GetLinks() *Links {
	return &r.Links
}

// GetData returns the data field for pagination aggregation.
func (r *CustomerReviewSummarizationsResponse) GetData() any {
	return r.Data
}
