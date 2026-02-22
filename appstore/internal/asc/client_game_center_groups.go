package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetGameCenterGroups retrieves the list of Game Center groups.
func (c *Client) GetGameCenterGroups(ctx context.Context, opts ...GCGroupsOption) (*GameCenterGroupsResponse, error) {
	query := &gcGroupsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := "/v1/gameCenterGroups"
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-groups: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCGroupsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterGroupsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterGroup retrieves a Game Center group by ID.
func (c *Client) GetGameCenterGroup(ctx context.Context, groupID string) (*GameCenterGroupResponse, error) {
	path := fmt.Sprintf("/v1/gameCenterGroups/%s", strings.TrimSpace(groupID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterGroupResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateGameCenterGroup creates a new Game Center group.
func (c *Client) CreateGameCenterGroup(ctx context.Context, referenceName *string) (*GameCenterGroupResponse, error) {
	var attrs *GameCenterGroupCreateAttributes
	if referenceName != nil {
		value := strings.TrimSpace(*referenceName)
		attrs = &GameCenterGroupCreateAttributes{
			ReferenceName: &value,
		}
	}

	payload := GameCenterGroupCreateRequest{
		Data: GameCenterGroupCreateData{
			Type:       ResourceTypeGameCenterGroups,
			Attributes: attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/gameCenterGroups", body)
	if err != nil {
		return nil, err
	}

	var response GameCenterGroupResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateGameCenterGroup updates an existing Game Center group.
func (c *Client) UpdateGameCenterGroup(ctx context.Context, groupID string, referenceName *string) (*GameCenterGroupResponse, error) {
	var attrs *GameCenterGroupUpdateAttributes
	if referenceName != nil {
		value := strings.TrimSpace(*referenceName)
		attrs = &GameCenterGroupUpdateAttributes{
			ReferenceName: &value,
		}
	}

	payload := GameCenterGroupUpdateRequest{
		Data: GameCenterGroupUpdateData{
			Type:       ResourceTypeGameCenterGroups,
			ID:         strings.TrimSpace(groupID),
			Attributes: attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s", strings.TrimSpace(groupID))
	data, err := c.do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var response GameCenterGroupResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// DeleteGameCenterGroup deletes a Game Center group.
func (c *Client) DeleteGameCenterGroup(ctx context.Context, groupID string) error {
	path := fmt.Sprintf("/v1/gameCenterGroups/%s", strings.TrimSpace(groupID))
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	return err
}

// GetGameCenterGroupAchievements retrieves the group's achievements.
func (c *Client) GetGameCenterGroupAchievements(ctx context.Context, groupID string, opts ...GCAchievementsOption) (*GameCenterAchievementsResponse, error) {
	query := &gcAchievementsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/gameCenterAchievements", strings.TrimSpace(groupID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-group-achievements: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCAchievementsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterAchievementsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterGroupAchievementsV2 retrieves the group's v2 achievements.
func (c *Client) GetGameCenterGroupAchievementsV2(ctx context.Context, groupID string, opts ...GCAchievementsOption) (*GameCenterAchievementsResponse, error) {
	query := &gcAchievementsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/gameCenterAchievementsV2", strings.TrimSpace(groupID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-group-achievements-v2: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCAchievementsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterAchievementsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterGroupActivities retrieves the group's activities.
func (c *Client) GetGameCenterGroupActivities(ctx context.Context, groupID string, opts ...GCActivitiesOption) (*GameCenterActivitiesResponse, error) {
	query := &gcActivitiesQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/gameCenterActivities", strings.TrimSpace(groupID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-group-activities: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCActivitiesQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterActivitiesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterGroupChallenges retrieves the group's challenges.
func (c *Client) GetGameCenterGroupChallenges(ctx context.Context, groupID string, opts ...GCChallengesOption) (*GameCenterChallengesResponse, error) {
	query := &gcChallengesQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/gameCenterChallenges", strings.TrimSpace(groupID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-group-challenges: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCChallengesQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterChallengesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterGroupLeaderboards retrieves the group's leaderboards.
func (c *Client) GetGameCenterGroupLeaderboards(ctx context.Context, groupID string, opts ...GCLeaderboardsOption) (*GameCenterLeaderboardsResponse, error) {
	query := &gcLeaderboardsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/gameCenterLeaderboards", strings.TrimSpace(groupID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-group-leaderboards: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCLeaderboardsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterGroupLeaderboardsV2 retrieves the group's v2 leaderboards.
func (c *Client) GetGameCenterGroupLeaderboardsV2(ctx context.Context, groupID string, opts ...GCLeaderboardsOption) (*GameCenterLeaderboardsResponse, error) {
	query := &gcLeaderboardsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/gameCenterLeaderboardsV2", strings.TrimSpace(groupID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-group-leaderboards-v2: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCLeaderboardsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterGroupLeaderboardSets retrieves the group's leaderboard sets.
func (c *Client) GetGameCenterGroupLeaderboardSets(ctx context.Context, groupID string, opts ...GCLeaderboardSetsOption) (*GameCenterLeaderboardSetsResponse, error) {
	query := &gcLeaderboardSetsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/gameCenterLeaderboardSets", strings.TrimSpace(groupID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-group-leaderboard-sets: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCLeaderboardSetsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardSetsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterGroupLeaderboardSetsV2 retrieves the group's v2 leaderboard sets.
func (c *Client) GetGameCenterGroupLeaderboardSetsV2(ctx context.Context, groupID string, opts ...GCLeaderboardSetsOption) (*GameCenterLeaderboardSetsResponse, error) {
	query := &gcLeaderboardSetsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/gameCenterLeaderboardSetsV2", strings.TrimSpace(groupID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-group-leaderboard-sets-v2: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCLeaderboardSetsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterLeaderboardSetsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateGameCenterGroupAchievements replaces the group's achievements.
func (c *Client) UpdateGameCenterGroupAchievements(ctx context.Context, groupID string, achievementIDs []string) error {
	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterAchievements, achievementIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/gameCenterAchievements", strings.TrimSpace(groupID))
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// UpdateGameCenterGroupAchievementsV2 replaces the group's v2 achievements.
func (c *Client) UpdateGameCenterGroupAchievementsV2(ctx context.Context, groupID string, achievementIDs []string) error {
	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterAchievements, achievementIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/gameCenterAchievementsV2", strings.TrimSpace(groupID))
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// UpdateGameCenterGroupLeaderboards replaces the group's leaderboards.
func (c *Client) UpdateGameCenterGroupLeaderboards(ctx context.Context, groupID string, leaderboardIDs []string) error {
	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterLeaderboards, leaderboardIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/gameCenterLeaderboards", strings.TrimSpace(groupID))
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// UpdateGameCenterGroupLeaderboardsV2 replaces the group's v2 leaderboards.
func (c *Client) UpdateGameCenterGroupLeaderboardsV2(ctx context.Context, groupID string, leaderboardIDs []string) error {
	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterLeaderboards, leaderboardIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/gameCenterLeaderboardsV2", strings.TrimSpace(groupID))
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}

// UpdateGameCenterGroupChallenges replaces the group's challenges.
func (c *Client) UpdateGameCenterGroupChallenges(ctx context.Context, groupID string, challengeIDs []string) error {
	payload := RelationshipRequest{
		Data: buildRelationshipData(ResourceTypeGameCenterChallenges, challengeIDs),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/gameCenterChallenges", strings.TrimSpace(groupID))
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}
