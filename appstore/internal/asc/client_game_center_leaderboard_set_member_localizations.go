package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetGameCenterLeaderboardSetMemberLocalizations retrieves leaderboard set member localizations.
func (c *Client) GetGameCenterLeaderboardSetMemberLocalizations(ctx context.Context, opts ...GCLeaderboardSetMemberLocalizationsOption) (*GameCenterLeaderboardSetMemberLocalizationsResponse, error) {
	query := &gcLeaderboardSetMemberLocalizationsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := "/v1/gameCenterLeaderboardSetMemberLocalizations"
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-leaderboard-set-member-localizations: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCLeaderboardSetMemberLocalizationsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardSetMemberLocalizationsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterLeaderboardSetMemberLocalization retrieves a leaderboard set member localization by ID.
func (c *Client) GetGameCenterLeaderboardSetMemberLocalization(ctx context.Context, localizationID string) (*GameCenterLeaderboardSetMemberLocalizationResponse, error) {
	path := fmt.Sprintf("/v1/gameCenterLeaderboardSetMemberLocalizations/%s", strings.TrimSpace(localizationID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardSetMemberLocalizationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateGameCenterLeaderboardSetMemberLocalization creates a new leaderboard set member localization.
func (c *Client) CreateGameCenterLeaderboardSetMemberLocalization(ctx context.Context, leaderboardSetID, leaderboardID string, attrs GameCenterLeaderboardSetMemberLocalizationCreateAttributes) (*GameCenterLeaderboardSetMemberLocalizationResponse, error) {
	leaderboardSetID = strings.TrimSpace(leaderboardSetID)
	if leaderboardSetID == "" {
		return nil, fmt.Errorf("leaderboardSetID is required")
	}
	leaderboardID = strings.TrimSpace(leaderboardID)
	if leaderboardID == "" {
		return nil, fmt.Errorf("leaderboardID is required")
	}

	attrs.Name = strings.TrimSpace(attrs.Name)
	if attrs.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	attrs.Locale = strings.TrimSpace(attrs.Locale)
	if attrs.Locale == "" {
		return nil, fmt.Errorf("locale is required")
	}

	payload := GameCenterLeaderboardSetMemberLocalizationCreateRequest{
		Data: GameCenterLeaderboardSetMemberLocalizationCreateData{
			Type:       ResourceTypeGameCenterLeaderboardSetMemberLocalizations,
			Attributes: attrs,
			Relationships: &GameCenterLeaderboardSetMemberLocalizationRelationships{
				GameCenterLeaderboardSet: &Relationship{
					Data: ResourceData{
						Type: ResourceTypeGameCenterLeaderboardSets,
						ID:   leaderboardSetID,
					},
				},
				GameCenterLeaderboard: &Relationship{
					Data: ResourceData{
						Type: ResourceTypeGameCenterLeaderboards,
						ID:   leaderboardID,
					},
				},
			},
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/gameCenterLeaderboardSetMemberLocalizations", body)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardSetMemberLocalizationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateGameCenterLeaderboardSetMemberLocalization updates an existing leaderboard set member localization.
func (c *Client) UpdateGameCenterLeaderboardSetMemberLocalization(ctx context.Context, localizationID string, attrs GameCenterLeaderboardSetMemberLocalizationUpdateAttributes) (*GameCenterLeaderboardSetMemberLocalizationResponse, error) {
	localizationID = strings.TrimSpace(localizationID)
	if localizationID == "" {
		return nil, fmt.Errorf("localizationID is required")
	}
	if attrs.Name == nil {
		return nil, fmt.Errorf("at least one attribute is required")
	}
	trimmedName := strings.TrimSpace(*attrs.Name)
	if trimmedName == "" {
		return nil, fmt.Errorf("name must not be empty")
	}
	attrs.Name = &trimmedName

	payload := GameCenterLeaderboardSetMemberLocalizationUpdateRequest{
		Data: GameCenterLeaderboardSetMemberLocalizationUpdateData{
			Type:       ResourceTypeGameCenterLeaderboardSetMemberLocalizations,
			ID:         localizationID,
			Attributes: &attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboardSetMemberLocalizations/%s", localizationID)
	data, err := c.do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardSetMemberLocalizationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// DeleteGameCenterLeaderboardSetMemberLocalization deletes a leaderboard set member localization.
func (c *Client) DeleteGameCenterLeaderboardSetMemberLocalization(ctx context.Context, localizationID string) error {
	localizationID = strings.TrimSpace(localizationID)
	if localizationID == "" {
		return fmt.Errorf("localizationID is required")
	}
	path := fmt.Sprintf("/v1/gameCenterLeaderboardSetMemberLocalizations/%s", localizationID)
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	return err
}

// GetGameCenterLeaderboardSetMemberLocalizationLeaderboard retrieves the leaderboard for a member localization.
func (c *Client) GetGameCenterLeaderboardSetMemberLocalizationLeaderboard(ctx context.Context, localizationID string) (*GameCenterLeaderboardResponse, error) {
	path := fmt.Sprintf("/v1/gameCenterLeaderboardSetMemberLocalizations/%s/gameCenterLeaderboard", strings.TrimSpace(localizationID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterLeaderboardSetMemberLocalizationLeaderboardSet retrieves the leaderboard set for a member localization.
func (c *Client) GetGameCenterLeaderboardSetMemberLocalizationLeaderboardSet(ctx context.Context, localizationID string) (*GameCenterLeaderboardSetResponse, error) {
	path := fmt.Sprintf("/v1/gameCenterLeaderboardSetMemberLocalizations/%s/gameCenterLeaderboardSet", strings.TrimSpace(localizationID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardSetResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
