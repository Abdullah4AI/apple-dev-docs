package asc

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// GetGameCenterGroupGameCenterAchievementsRelationships retrieves achievement linkages for a Game Center group.
func (c *Client) GetGameCenterGroupGameCenterAchievementsRelationships(ctx context.Context, groupID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterGroupLinkages(ctx, groupID, "gameCenterAchievements", opts...)
}

// GetGameCenterGroupGameCenterAchievementsV2Relationships retrieves v2 achievement linkages for a Game Center group.
func (c *Client) GetGameCenterGroupGameCenterAchievementsV2Relationships(ctx context.Context, groupID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterGroupLinkages(ctx, groupID, "gameCenterAchievementsV2", opts...)
}

// GetGameCenterGroupGameCenterActivitiesRelationships retrieves activity linkages for a Game Center group.
func (c *Client) GetGameCenterGroupGameCenterActivitiesRelationships(ctx context.Context, groupID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterGroupLinkages(ctx, groupID, "gameCenterActivities", opts...)
}

// GetGameCenterGroupGameCenterChallengesRelationships retrieves challenge linkages for a Game Center group.
func (c *Client) GetGameCenterGroupGameCenterChallengesRelationships(ctx context.Context, groupID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterGroupLinkages(ctx, groupID, "gameCenterChallenges", opts...)
}

// GetGameCenterGroupGameCenterDetailsRelationships retrieves detail linkages for a Game Center group.
func (c *Client) GetGameCenterGroupGameCenterDetailsRelationships(ctx context.Context, groupID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterGroupLinkages(ctx, groupID, "gameCenterDetails", opts...)
}

// GetGameCenterGroupGameCenterLeaderboardSetsRelationships retrieves leaderboard set linkages for a Game Center group.
func (c *Client) GetGameCenterGroupGameCenterLeaderboardSetsRelationships(ctx context.Context, groupID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterGroupLinkages(ctx, groupID, "gameCenterLeaderboardSets", opts...)
}

// GetGameCenterGroupGameCenterLeaderboardSetsV2Relationships retrieves v2 leaderboard set linkages for a Game Center group.
func (c *Client) GetGameCenterGroupGameCenterLeaderboardSetsV2Relationships(ctx context.Context, groupID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterGroupLinkages(ctx, groupID, "gameCenterLeaderboardSetsV2", opts...)
}

// GetGameCenterGroupGameCenterLeaderboardsRelationships retrieves leaderboard linkages for a Game Center group.
func (c *Client) GetGameCenterGroupGameCenterLeaderboardsRelationships(ctx context.Context, groupID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterGroupLinkages(ctx, groupID, "gameCenterLeaderboards", opts...)
}

// GetGameCenterGroupGameCenterLeaderboardsV2Relationships retrieves v2 leaderboard linkages for a Game Center group.
func (c *Client) GetGameCenterGroupGameCenterLeaderboardsV2Relationships(ctx context.Context, groupID string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getGameCenterGroupLinkages(ctx, groupID, "gameCenterLeaderboardsV2", opts...)
}

// UpdateGameCenterGroupGameCenterLeaderboardSetsRelationship replaces the gameCenterLeaderboardSets relationship.
func (c *Client) UpdateGameCenterGroupGameCenterLeaderboardSetsRelationship(ctx context.Context, groupID string, setIDs []string) error {
	return c.updateGameCenterGroupToManyRelationship(ctx, groupID, "gameCenterLeaderboardSets", ResourceTypeGameCenterLeaderboardSets, setIDs)
}

// UpdateGameCenterGroupGameCenterLeaderboardSetsV2Relationship replaces the gameCenterLeaderboardSetsV2 relationship.
func (c *Client) UpdateGameCenterGroupGameCenterLeaderboardSetsV2Relationship(ctx context.Context, groupID string, setIDs []string) error {
	return c.updateGameCenterGroupToManyRelationship(ctx, groupID, "gameCenterLeaderboardSetsV2", ResourceTypeGameCenterLeaderboardSets, setIDs)
}

func (c *Client) getGameCenterGroupLinkages(ctx context.Context, groupID, relationship string, opts ...LinkagesOption) (*LinkagesResponse, error) {
	return c.getResourceLinkages(
		ctx,
		groupID,
		relationship,
		"groupID",
		"/v1/gameCenterGroups/%s/relationships/%s",
		"gameCenterGroupRelationships",
		opts...,
	)
}

func (c *Client) updateGameCenterGroupToManyRelationship(ctx context.Context, groupID, relationship string, resourceType ResourceType, ids []string) error {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return fmt.Errorf("groupID is required")
	}

	payload := RelationshipRequest{
		Data: buildRelationshipData(resourceType, ids),
	}
	body, err := BuildRequestBody(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/v1/gameCenterGroups/%s/relationships/%s", groupID, relationship)
	_, err = c.do(ctx, http.MethodPatch, path, body)
	return err
}
