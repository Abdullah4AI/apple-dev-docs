package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GameCenterLeaderboardLocalizationImageLinkageResponse is the response for leaderboard localization image relationships.
type GameCenterLeaderboardLocalizationImageLinkageResponse struct {
	Data  ResourceData `json:"data"`
	Links Links        `json:"links"`
}

// GameCenterLeaderboardSetMemberLocalizationLeaderboardLinkageResponse is the response for member localization leaderboard relationships.
type GameCenterLeaderboardSetMemberLocalizationLeaderboardLinkageResponse struct {
	Data  ResourceData `json:"data"`
	Links Links        `json:"links"`
}

// GameCenterLeaderboardSetMemberLocalizationLeaderboardSetLinkageResponse is the response for member localization leaderboard set relationships.
type GameCenterLeaderboardSetMemberLocalizationLeaderboardSetLinkageResponse struct {
	Data  ResourceData `json:"data"`
	Links Links        `json:"links"`
}

// GetGameCenterLeaderboardLocalizationImageRelationship retrieves the image linkage for a leaderboard localization.
func (c *Client) GetGameCenterLeaderboardLocalizationImageRelationship(ctx context.Context, localizationID string) (*GameCenterLeaderboardLocalizationImageLinkageResponse, error) {
	localizationID = strings.TrimSpace(localizationID)
	if localizationID == "" {
		return nil, fmt.Errorf("localizationID is required")
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboardLocalizations/%s/relationships/gameCenterLeaderboardImage", localizationID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardLocalizationImageLinkageResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse gameCenterLeaderboardImage relationship response: %w", err)
	}

	return &response, nil
}

// GetGameCenterLeaderboardSetMemberLocalizationLeaderboardRelationship retrieves the leaderboard linkage for a member localization.
func (c *Client) GetGameCenterLeaderboardSetMemberLocalizationLeaderboardRelationship(ctx context.Context, localizationID string) (*GameCenterLeaderboardSetMemberLocalizationLeaderboardLinkageResponse, error) {
	localizationID = strings.TrimSpace(localizationID)
	if localizationID == "" {
		return nil, fmt.Errorf("localizationID is required")
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboardSetMemberLocalizations/%s/relationships/gameCenterLeaderboard", localizationID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardSetMemberLocalizationLeaderboardLinkageResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse gameCenterLeaderboard relationship response: %w", err)
	}

	return &response, nil
}

// GetGameCenterLeaderboardSetMemberLocalizationLeaderboardSetRelationship retrieves the leaderboard set linkage for a member localization.
func (c *Client) GetGameCenterLeaderboardSetMemberLocalizationLeaderboardSetRelationship(ctx context.Context, localizationID string) (*GameCenterLeaderboardSetMemberLocalizationLeaderboardSetLinkageResponse, error) {
	localizationID = strings.TrimSpace(localizationID)
	if localizationID == "" {
		return nil, fmt.Errorf("localizationID is required")
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboardSetMemberLocalizations/%s/relationships/gameCenterLeaderboardSet", localizationID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardSetMemberLocalizationLeaderboardSetLinkageResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse gameCenterLeaderboardSet relationship response: %w", err)
	}

	return &response, nil
}
