package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GameCenterDetailGameCenterGroupLinkageResponse is the response for gameCenterGroup relationships on a detail.
type GameCenterDetailGameCenterGroupLinkageResponse struct {
	Data  ResourceData `json:"data"`
	Links Links        `json:"links"`
}

// GetGameCenterDetailAchievementReleasesRelationships retrieves achievement release linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailAchievementReleasesRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "achievementReleases", opts...)
}

// GetGameCenterDetailActivityReleasesRelationships retrieves activity release linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailActivityReleasesRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "activityReleases", opts...)
}

// GetGameCenterDetailChallengeReleasesRelationships retrieves challenge release linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailChallengeReleasesRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "challengeReleases", opts...)
}

// GetGameCenterDetailGameCenterAchievementsRelationships retrieves achievement linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailGameCenterAchievementsRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "gameCenterAchievements", opts...)
}

// GetGameCenterDetailGameCenterAchievementsV2Relationships retrieves v2 achievement linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailGameCenterAchievementsV2Relationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "gameCenterAchievementsV2", opts...)
}

// GetGameCenterDetailGameCenterActivitiesRelationships retrieves activity linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailGameCenterActivitiesRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "gameCenterActivities", opts...)
}

// GetGameCenterDetailGameCenterAppVersionsRelationships retrieves app version linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailGameCenterAppVersionsRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "gameCenterAppVersions", opts...)
}

// GetGameCenterDetailGameCenterChallengesRelationships retrieves challenge linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailGameCenterChallengesRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "gameCenterChallenges", opts...)
}

// GetGameCenterDetailGameCenterGroupRelationship retrieves the group linkage for a Game Center detail.
func (c *Client) GetGameCenterDetailGameCenterGroupRelationship(ctx context.Context, detailID string) (*GameCenterDetailGameCenterGroupLinkageResponse, error) {
	detailID = strings.TrimSpace(detailID)
	if detailID == "" {
		return nil, fmt.Errorf("detailID is required")
	}

	path := fmt.Sprintf("/v1/gameCenterDetails/%s/relationships/gameCenterGroup", detailID)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterDetailGameCenterGroupLinkageResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse gameCenterGroup relationship response: %w", err)
	}

	return &response, nil
}

// GetGameCenterDetailGameCenterLeaderboardSetsRelationships retrieves leaderboard set linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailGameCenterLeaderboardSetsRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "gameCenterLeaderboardSets", opts...)
}

// GetGameCenterDetailGameCenterLeaderboardSetsV2Relationships retrieves v2 leaderboard set linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailGameCenterLeaderboardSetsV2Relationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "gameCenterLeaderboardSetsV2", opts...)
}

// GetGameCenterDetailGameCenterLeaderboardsRelationships retrieves leaderboard linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailGameCenterLeaderboardsRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "gameCenterLeaderboards", opts...)
}

// GetGameCenterDetailGameCenterLeaderboardsV2Relationships retrieves v2 leaderboard linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailGameCenterLeaderboardsV2Relationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "gameCenterLeaderboardsV2", opts...)
}

// GetGameCenterDetailLeaderboardReleasesRelationships retrieves leaderboard release linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailLeaderboardReleasesRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "leaderboardReleases", opts...)
}

// GetGameCenterDetailLeaderboardSetReleasesRelationships retrieves leaderboard set release linkages for a Game Center detail.
func (c *Client) GetGameCenterDetailLeaderboardSetReleasesRelationships(ctx context.Context, detailID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterDetailLinkages(ctx, detailID, "leaderboardSetReleases", opts...)
}

// UpdateGameCenterDetailChallengesMinimumPlatformVersionsRelationship replaces the challengesMinimumPlatformVersions relationship.
func (c *Client) UpdateGameCenterDetailChallengesMinimumPlatformVersionsRelationship(ctx context.Context, detailID string, versionIDs []string) error {
	return c.updateGameCenterDetailToManyRelationship(ctx, detailID, "challengesMinimumPlatformVersions", ResourceTypeAppStoreVersions, versionIDs)
}

// UpdateGameCenterDetailGameCenterAchievementsRelationship replaces the gameCenterAchievements relationship.
func (c *Client) UpdateGameCenterDetailGameCenterAchievementsRelationship(ctx context.Context, detailID string, achievementIDs []string) error {
	return c.updateGameCenterDetailToManyRelationship(ctx, detailID, "gameCenterAchievements", ResourceTypeGameCenterAchievements, achievementIDs)
}

// UpdateGameCenterDetailGameCenterAchievementsV2Relationship replaces the gameCenterAchievementsV2 relationship.
func (c *Client) UpdateGameCenterDetailGameCenterAchievementsV2Relationship(ctx context.Context, detailID string, achievementIDs []string) error {
	return c.updateGameCenterDetailToManyRelationship(ctx, detailID, "gameCenterAchievementsV2", ResourceTypeGameCenterAchievements, achievementIDs)
}

// UpdateGameCenterDetailGameCenterLeaderboardSetsRelationship replaces the gameCenterLeaderboardSets relationship.
func (c *Client) UpdateGameCenterDetailGameCenterLeaderboardSetsRelationship(ctx context.Context, detailID string, setIDs []string) error {
	return c.updateGameCenterDetailToManyRelationship(ctx, detailID, "gameCenterLeaderboardSets", ResourceTypeGameCenterLeaderboardSets, setIDs)
}

// UpdateGameCenterDetailGameCenterLeaderboardSetsV2Relationship replaces the gameCenterLeaderboardSetsV2 relationship.
func (c *Client) UpdateGameCenterDetailGameCenterLeaderboardSetsV2Relationship(ctx context.Context, detailID string, setIDs []string) error {
	return c.updateGameCenterDetailToManyRelationship(ctx, detailID, "gameCenterLeaderboardSetsV2", ResourceTypeGameCenterLeaderboardSets, setIDs)
}

// UpdateGameCenterDetailGameCenterLeaderboardsRelationship replaces the gameCenterLeaderboards relationship.
func (c *Client) UpdateGameCenterDetailGameCenterLeaderboardsRelationship(ctx context.Context, detailID string, leaderboardIDs []string) error {
	return c.updateGameCenterDetailToManyRelationship(ctx, detailID, "gameCenterLeaderboards", ResourceTypeGameCenterLeaderboards, leaderboardIDs)
}

// UpdateGameCenterDetailGameCenterLeaderboardsV2Relationship replaces the gameCenterLeaderboardsV2 relationship.
func (c *Client) UpdateGameCenterDetailGameCenterLeaderboardsV2Relationship(ctx context.Context, detailID string, leaderboardIDs []string) error {
	return c.updateGameCenterDetailToManyRelationship(ctx, detailID, "gameCenterLeaderboardsV2", ResourceTypeGameCenterLeaderboards, leaderboardIDs)
}

func (c *Client) getGameCenterDetailLinkages(ctx context.Context, detailID, relationship string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getResourceLinkages(
		ctx,
		detailID,
		relationship,
		"detailID",
		"/v1/gameCenterDetails/%s/relationships/%s",
		"gameCenterDetailRelationships",
		opts...,
	)
}

func (c *Client) updateGameCenterDetailToManyRelationship(ctx context.Context, detailID, relationship string, resourceType ResourceType, ids []string) error {
	detailID = strings.TrimSpace(detailID)
	if detailID == "" {
		return fmt.Errorf("detailID is required")
	}

	payload := RelationshipRequest{
		Data: buildRelationshipData(resourceType, ids),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterDetails/%s/relationships/%s", detailID, relationship)
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}
