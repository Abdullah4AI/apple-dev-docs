package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GameCenterLeaderboardSetGroupLeaderboardSetLinkageResponse is the response for groupLeaderboardSet relationships.
type GameCenterLeaderboardSetGroupLeaderboardSetLinkageResponse struct {
	Data  ResourceData `json:"data"`
	Links Links        `json:"links"`
}

// GameCenterLeaderboardGroupLeaderboardLinkageResponse is the response for groupLeaderboard relationships.
type GameCenterLeaderboardGroupLeaderboardLinkageResponse struct {
	Data  ResourceData `json:"data"`
	Links Links        `json:"links"`
}

// GetGameCenterLeaderboardSetMembersRelationships retrieves leaderboard linkages for a leaderboard set.
func (c *Client) GetGameCenterLeaderboardSetMembersRelationships(ctx context.Context, setID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterLeaderboardSetLinkages(ctx, setID, "gameCenterLeaderboards", opts...)
}

// GetGameCenterLeaderboardSetGroupLeaderboardSetRelationship retrieves the group leaderboard set linkage for a leaderboard set.
func (c *Client) GetGameCenterLeaderboardSetGroupLeaderboardSetRelationship(ctx context.Context, setID string) (*GameCenterLeaderboardSetGroupLeaderboardSetLinkageResponse, error) {
	setID = strings.TrimSpace(setID)
	if setID == "" {
		return nil, fmt.Errorf("setID is required")
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboardSets/%s/relationships/groupLeaderboardSet", setID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardSetGroupLeaderboardSetLinkageResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse groupLeaderboardSet relationship response: %w", err)
	}
	return &response, nil
}

// GetGameCenterLeaderboardSetLocalizationsRelationships retrieves localization linkages for a leaderboard set.
func (c *Client) GetGameCenterLeaderboardSetLocalizationsRelationships(ctx context.Context, setID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterLeaderboardSetLinkages(ctx, setID, "localizations", opts...)
}

// GetGameCenterLeaderboardSetReleasesRelationships retrieves release linkages for a leaderboard set.
func (c *Client) GetGameCenterLeaderboardSetReleasesRelationships(ctx context.Context, setID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterLeaderboardSetLinkages(ctx, setID, "releases", opts...)
}

// AddGameCenterLeaderboardSetMembers adds leaderboards to a leaderboard set.
func (c *Client) AddGameCenterLeaderboardSetMembers(ctx context.Context, setID string, leaderboardIDs []string) error {
	setID = strings.TrimSpace(setID)
	leaderboardIDs = normalizeList(leaderboardIDs)
	if setID == "" {
		return fmt.Errorf("setID is required")
	}
	if len(leaderboardIDs) == 0 {
		return fmt.Errorf("leaderboardIDs are required")
	}

	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterLeaderboards, leaderboardIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboardSets/%s/relationships/gameCenterLeaderboards", setID)
	_, err = c.do(ctx, http.MethodPost, path, body)
	return err
}

// RemoveGameCenterLeaderboardSetMembers removes leaderboards from a leaderboard set.
func (c *Client) RemoveGameCenterLeaderboardSetMembers(ctx context.Context, setID string, leaderboardIDs []string) error {
	setID = strings.TrimSpace(setID)
	leaderboardIDs = normalizeList(leaderboardIDs)
	if setID == "" {
		return fmt.Errorf("setID is required")
	}
	if len(leaderboardIDs) == 0 {
		return fmt.Errorf("leaderboardIDs are required")
	}

	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterLeaderboards, leaderboardIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboardSets/%s/relationships/gameCenterLeaderboards", setID)
	_, err = c.do(ctx, http.MethodDelete, path, body)
	return err
}

// UpdateGameCenterLeaderboardSetGroupLeaderboardSetRelationship updates the groupLeaderboardSet relationship on a leaderboard set.
func (c *Client) UpdateGameCenterLeaderboardSetGroupLeaderboardSetRelationship(ctx context.Context, setID, groupSetID string) error {
	setID = strings.TrimSpace(setID)
	groupSetID = strings.TrimSpace(groupSetID)
	if setID == "" {
		return fmt.Errorf("setID is required")
	}
	if groupSetID == "" {
		return fmt.Errorf("groupSetID is required")
	}

	payload := struct {
		Data ResourceData `json:"data"`
	}{
		Data: ResourceData{
			Type: ResourceTypeGameCenterLeaderboardSets,
			ID:   groupSetID,
		},
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboardSets/%s/relationships/groupLeaderboardSet", setID)
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// GetGameCenterLeaderboardGroupLeaderboardRelationship retrieves the group leaderboard linkage for a leaderboard.
func (c *Client) GetGameCenterLeaderboardGroupLeaderboardRelationship(ctx context.Context, leaderboardID string) (*GameCenterLeaderboardGroupLeaderboardLinkageResponse, error) {
	leaderboardID = strings.TrimSpace(leaderboardID)
	if leaderboardID == "" {
		return nil, fmt.Errorf("leaderboardID is required")
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboards/%s/relationships/groupLeaderboard", leaderboardID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardGroupLeaderboardLinkageResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse groupLeaderboard relationship response: %w", err)
	}
	return &response, nil
}

// GetGameCenterLeaderboardLocalizationsRelationships retrieves localization linkages for a leaderboard.
func (c *Client) GetGameCenterLeaderboardLocalizationsRelationships(ctx context.Context, leaderboardID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterLeaderboardLinkages(ctx, leaderboardID, "localizations", opts...)
}

// GetGameCenterLeaderboardReleasesRelationships retrieves release linkages for a leaderboard.
func (c *Client) GetGameCenterLeaderboardReleasesRelationships(ctx context.Context, leaderboardID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterLeaderboardLinkages(ctx, leaderboardID, "releases", opts...)
}

// UpdateGameCenterLeaderboardActivityRelationship updates the activity relationship on a leaderboard.
func (c *Client) UpdateGameCenterLeaderboardActivityRelationship(ctx context.Context, leaderboardID, activityID string) error {
	leaderboardID = strings.TrimSpace(leaderboardID)
	activityID = strings.TrimSpace(activityID)
	if leaderboardID == "" {
		return fmt.Errorf("leaderboardID is required")
	}
	if activityID == "" {
		return fmt.Errorf("activityID is required")
	}

	payload := struct {
		Data ResourceData `json:"data"`
	}{
		Data: ResourceData{
			Type: ResourceTypeGameCenterActivities,
			ID:   activityID,
		},
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboards/%s/relationships/activity", leaderboardID)
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// UpdateGameCenterLeaderboardChallengeRelationship updates the challenge relationship on a leaderboard.
func (c *Client) UpdateGameCenterLeaderboardChallengeRelationship(ctx context.Context, leaderboardID, challengeID string) error {
	leaderboardID = strings.TrimSpace(leaderboardID)
	challengeID = strings.TrimSpace(challengeID)
	if leaderboardID == "" {
		return fmt.Errorf("leaderboardID is required")
	}
	if challengeID == "" {
		return fmt.Errorf("challengeID is required")
	}

	payload := struct {
		Data ResourceData `json:"data"`
	}{
		Data: ResourceData{
			Type: ResourceTypeGameCenterChallenges,
			ID:   challengeID,
		},
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboards/%s/relationships/challenge", leaderboardID)
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// UpdateGameCenterLeaderboardGroupLeaderboardRelationship updates the groupLeaderboard relationship on a leaderboard.
func (c *Client) UpdateGameCenterLeaderboardGroupLeaderboardRelationship(ctx context.Context, leaderboardID, groupLeaderboardID string) error {
	leaderboardID = strings.TrimSpace(leaderboardID)
	groupLeaderboardID = strings.TrimSpace(groupLeaderboardID)
	if leaderboardID == "" {
		return fmt.Errorf("leaderboardID is required")
	}
	if groupLeaderboardID == "" {
		return fmt.Errorf("groupLeaderboardID is required")
	}

	payload := struct {
		Data ResourceData `json:"data"`
	}{
		Data: ResourceData{
			Type: ResourceTypeGameCenterLeaderboards,
			ID:   groupLeaderboardID,
		},
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterLeaderboards/%s/relationships/groupLeaderboard", leaderboardID)
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

func (c *Client) getGameCenterLeaderboardSetLinkages(ctx context.Context, setID, relationship string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getResourceLinkages(
		ctx,
		setID,
		relationship,
		"setID",
		"/v1/gameCenterLeaderboardSets/%s/relationships/%s",
		"gameCenterLeaderboardSetRelationships",
		opts...,
	)
}

func (c *Client) getGameCenterLeaderboardLinkages(ctx context.Context, leaderboardID, relationship string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getResourceLinkages(
		ctx,
		leaderboardID,
		relationship,
		"leaderboardID",
		"/v1/gameCenterLeaderboards/%s/relationships/%s",
		"gameCenterLeaderboardRelationships",
		opts...,
	)
}
