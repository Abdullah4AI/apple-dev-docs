package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ActorAttributes describes an actor resource.
type ActorAttributes struct {
	ActorType     string `json:"actorType,omitempty"`
	UserFirstName string `json:"userFirstName,omitempty"`
	UserLastName  string `json:"userLastName,omitempty"`
	UserEmail     string `json:"userEmail,omitempty"`
	APIKeyID      string `json:"apiKeyId,omitempty"`
}

// ActorsResponse is the response from actors endpoint.
type ActorsResponse = Response[ActorAttributes]

// ActorResponse is the response from actor detail endpoint.
type ActorResponse = SingleResponse[ActorAttributes]

// GetActors retrieves actors filtered by IDs.
func (c *Client) GetActors(ctx context.Context, opts ...ActorsOption) (*ActorsResponse, error) {
	query := &actorsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := "/v1/actors"
	if query.nextURL != "" {
		// Validate nextURL to prevent credential exfiltration
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("actors: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildActorsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response ActorsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetActor retrieves a single actor by ID.
func (c *Client) GetActor(ctx context.Context, actorID string, fields []string) (*ActorResponse, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, fmt.Errorf("actorID is required")
	}

	var lastNotFound error
	for _, candidate := range actorIDCandidates(actorID) {
		response, err := c.getActorByID(ctx, candidate, fields)
		if err == nil {
			return response, nil
		}
		if IsNotFound(err) {
			lastNotFound = err
			continue
		}
		return nil, err
	}
	if lastNotFound != nil {
		return nil, lastNotFound
	}
	return nil, fmt.Errorf("actor not found")
}

func (c *Client) getActorByID(ctx context.Context, actorID string, fields []string) (*ActorResponse, error) {
	path := fmt.Sprintf("/v1/actors/%s", actorID)
	if queryString := buildActorsFieldsQuery(fields); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response ActorResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

func actorIDCandidates(actorID string) []string {
	if strings.Contains(actorID, ":") {
		return []string{actorID}
	}
	normalized := strings.TrimSpace(actorID)
	if normalized == "" {
		return nil
	}
	if strings.EqualFold(normalized, "APPLE") {
		return []string{normalized}
	}
	return []string{
		normalized,
		"USER:" + normalized,
		"API_KEY:" + normalized,
		"XCODE_CLOUD:" + normalized,
	}
}
