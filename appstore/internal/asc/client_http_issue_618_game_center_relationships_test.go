package asc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestIssue618_GameCenterRelationshipEndpoints_GET(t *testing.T) {
	ctx := context.Background()

	const (
		linkagesOK = `{"data":[{"type":"apps","id":"1"}],"links":{}}`
		toOneOK    = `{"data":{"type":"apps","id":"1"},"links":{}}`
	)

	tests := []struct {
		name     string
		wantPath string
		body     string
		call     func(*Client) error
	}{
		{
			name:     "GetGameCenterAchievementLocalizationAchievementRelationship",
			wantPath: "/v1/gameCenterAchievementLocalizations/loc-1/relationships/gameCenterAchievement",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterAchievementLocalizationAchievementRelationship(ctx, "loc-1")
				return err
			},
		},
		{
			name:     "GetGameCenterAchievementLocalizationImageRelationship",
			wantPath: "/v1/gameCenterAchievementLocalizations/loc-1/relationships/gameCenterAchievementImage",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterAchievementLocalizationImageRelationship(ctx, "loc-1")
				return err
			},
		},
		{
			name:     "GetGameCenterAchievementGroupAchievementRelationship",
			wantPath: "/v1/gameCenterAchievements/ach-1/relationships/groupAchievement",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterAchievementGroupAchievementRelationship(ctx, "ach-1")
				return err
			},
		},
		{
			name:     "GetGameCenterAchievementLocalizationsRelationships",
			wantPath: "/v1/gameCenterAchievements/ach-1/relationships/localizations",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterAchievementLocalizationsRelationships(ctx, "ach-1")
				return err
			},
		},
		{
			name:     "GetGameCenterAchievementReleasesRelationships",
			wantPath: "/v1/gameCenterAchievements/ach-1/relationships/releases",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterAchievementReleasesRelationships(ctx, "ach-1")
				return err
			},
		},
		{
			name:     "GetGameCenterActivityVersionsRelationships",
			wantPath: "/v1/gameCenterActivities/act-1/relationships/versions",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterActivityVersionsRelationships(ctx, "act-1")
				return err
			},
		},
		{
			name:     "GetGameCenterActivityLocalizationImageRelationship",
			wantPath: "/v1/gameCenterActivityLocalizations/loc-1/relationships/image",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterActivityLocalizationImageRelationship(ctx, "loc-1")
				return err
			},
		},
		{
			name:     "GetGameCenterActivityVersionDefaultImageRelationship",
			wantPath: "/v1/gameCenterActivityVersions/ver-1/relationships/defaultImage",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterActivityVersionDefaultImageRelationship(ctx, "ver-1")
				return err
			},
		},
		{
			name:     "GetGameCenterActivityVersionLocalizationsRelationships",
			wantPath: "/v1/gameCenterActivityVersions/ver-1/relationships/localizations",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterActivityVersionLocalizationsRelationships(ctx, "ver-1")
				return err
			},
		},
		{
			name:     "GetGameCenterAppVersionAppStoreVersionRelationship",
			wantPath: "/v1/gameCenterAppVersions/appver-1/relationships/appStoreVersion",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterAppVersionAppStoreVersionRelationship(ctx, "appver-1")
				return err
			},
		},
		{
			name:     "GetGameCenterAppVersionCompatibilityVersionsRelationships",
			wantPath: "/v1/gameCenterAppVersions/appver-1/relationships/compatibilityVersions",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterAppVersionCompatibilityVersionsRelationships(ctx, "appver-1")
				return err
			},
		},
		{
			name:     "GetGameCenterChallengeLocalizationImageRelationship",
			wantPath: "/v1/gameCenterChallengeLocalizations/loc-1/relationships/image",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterChallengeLocalizationImageRelationship(ctx, "loc-1")
				return err
			},
		},
		{
			name:     "GetGameCenterChallengeVersionDefaultImageRelationship",
			wantPath: "/v1/gameCenterChallengeVersions/ver-1/relationships/defaultImage",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterChallengeVersionDefaultImageRelationship(ctx, "ver-1")
				return err
			},
		},
		{
			name:     "GetGameCenterChallengeVersionLocalizationsRelationships",
			wantPath: "/v1/gameCenterChallengeVersions/ver-1/relationships/localizations",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterChallengeVersionLocalizationsRelationships(ctx, "ver-1")
				return err
			},
		},
		{
			name:     "GetGameCenterChallengeVersionsRelationships",
			wantPath: "/v1/gameCenterChallenges/chal-1/relationships/versions",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterChallengeVersionsRelationships(ctx, "chal-1")
				return err
			},
		},
		{
			name:     "GetGameCenterEnabledVersionCompatibleVersionsRelationships",
			wantPath: "/v1/gameCenterEnabledVersions/env-1/relationships/compatibleVersions",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterEnabledVersionCompatibleVersionsRelationships(ctx, "env-1")
				return err
			},
		},
		{
			name:     "GetGameCenterLeaderboardLocalizationImageRelationship",
			wantPath: "/v1/gameCenterLeaderboardLocalizations/loc-1/relationships/gameCenterLeaderboardImage",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterLeaderboardLocalizationImageRelationship(ctx, "loc-1")
				return err
			},
		},
		{
			name:     "GetGameCenterLeaderboardSetMemberLocalizationLeaderboardRelationship",
			wantPath: "/v1/gameCenterLeaderboardSetMemberLocalizations/loc-1/relationships/gameCenterLeaderboard",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterLeaderboardSetMemberLocalizationLeaderboardRelationship(ctx, "loc-1")
				return err
			},
		},
		{
			name:     "GetGameCenterLeaderboardSetMemberLocalizationLeaderboardSetRelationship",
			wantPath: "/v1/gameCenterLeaderboardSetMemberLocalizations/loc-1/relationships/gameCenterLeaderboardSet",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterLeaderboardSetMemberLocalizationLeaderboardSetRelationship(ctx, "loc-1")
				return err
			},
		},
		{
			name:     "GetGameCenterMatchmakingRuleSetQueuesRelationships",
			wantPath: "/v1/gameCenterMatchmakingRuleSets/rs-1/relationships/matchmakingQueues",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterMatchmakingRuleSetQueuesRelationships(ctx, "rs-1")
				return err
			},
		},
		{
			name:     "GetGameCenterMatchmakingRuleSetRulesRelationships",
			wantPath: "/v1/gameCenterMatchmakingRuleSets/rs-1/relationships/rules",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterMatchmakingRuleSetRulesRelationships(ctx, "rs-1")
				return err
			},
		},
		{
			name:     "GetGameCenterMatchmakingRuleSetTeamsRelationships",
			wantPath: "/v1/gameCenterMatchmakingRuleSets/rs-1/relationships/teams",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterMatchmakingRuleSetTeamsRelationships(ctx, "rs-1")
				return err
			},
		},
		{
			name:     "GetGameCenterAchievementVersionsRelationshipsV2",
			wantPath: "/v2/gameCenterAchievements/ach-1/relationships/versions",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetGameCenterAchievementVersionsRelationshipsV2(ctx, "ach-1")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(req *http.Request) {
				if req.Method != http.MethodGet {
					t.Fatalf("expected GET, got %s", req.Method)
				}
				if req.URL.Path != tt.wantPath {
					t.Fatalf("expected path %s, got %s", tt.wantPath, req.URL.Path)
				}
				assertAuthorized(t, req)
			}, jsonResponse(http.StatusOK, tt.body))

			if err := tt.call(client); err != nil {
				t.Fatalf("request error: %v", err)
			}
		})
	}
}

func TestIssue618_GameCenterRelationshipEndpoints_Mutations(t *testing.T) {
	ctx := context.Background()

	type toOneRequest struct {
		Data ResourceData `json:"data"`
	}

	tests := []struct {
		name       string
		wantMethod string
		wantPath   string
		wantBodyFn func(*testing.T, []byte)
		call       func(*Client) error
	}{
		{
			name:       "AddGameCenterActivityAchievementsV2",
			wantMethod: http.MethodPost,
			wantPath:   "/v1/gameCenterActivities/act-1/relationships/achievementsV2",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 1 || got.Data[0].Type != ResourceTypeGameCenterAchievements || got.Data[0].ID != "ach-1" {
					t.Fatalf("unexpected body: %#v", got)
				}
			},
			call: func(client *Client) error {
				return client.AddGameCenterActivityAchievementsV2(ctx, "act-1", []string{"ach-1"})
			},
		},
		{
			name:       "RemoveGameCenterActivityAchievementsV2",
			wantMethod: http.MethodDelete,
			wantPath:   "/v1/gameCenterActivities/act-1/relationships/achievementsV2",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 1 || got.Data[0].Type != ResourceTypeGameCenterAchievements || got.Data[0].ID != "ach-1" {
					t.Fatalf("unexpected body: %#v", got)
				}
			},
			call: func(client *Client) error {
				return client.RemoveGameCenterActivityAchievementsV2(ctx, "act-1", []string{"ach-1"})
			},
		},
		{
			name:       "AddGameCenterActivityLeaderboardsV2",
			wantMethod: http.MethodPost,
			wantPath:   "/v1/gameCenterActivities/act-1/relationships/leaderboardsV2",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 1 || got.Data[0].Type != ResourceTypeGameCenterLeaderboards || got.Data[0].ID != "lb-1" {
					t.Fatalf("unexpected body: %#v", got)
				}
			},
			call: func(client *Client) error {
				return client.AddGameCenterActivityLeaderboardsV2(ctx, "act-1", []string{"lb-1"})
			},
		},
		{
			name:       "RemoveGameCenterActivityLeaderboardsV2",
			wantMethod: http.MethodDelete,
			wantPath:   "/v1/gameCenterActivities/act-1/relationships/leaderboardsV2",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 1 || got.Data[0].Type != ResourceTypeGameCenterLeaderboards || got.Data[0].ID != "lb-1" {
					t.Fatalf("unexpected body: %#v", got)
				}
			},
			call: func(client *Client) error {
				return client.RemoveGameCenterActivityLeaderboardsV2(ctx, "act-1", []string{"lb-1"})
			},
		},
		{
			name:       "AddGameCenterAppVersionCompatibilityVersions",
			wantMethod: http.MethodPost,
			wantPath:   "/v1/gameCenterAppVersions/appver-1/relationships/compatibilityVersions",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 1 || got.Data[0].Type != ResourceTypeGameCenterAppVersions || got.Data[0].ID != "appver-2" {
					t.Fatalf("unexpected body: %#v", got)
				}
			},
			call: func(client *Client) error {
				return client.AddGameCenterAppVersionCompatibilityVersions(ctx, "appver-1", []string{"appver-2"})
			},
		},
		{
			name:       "RemoveGameCenterAppVersionCompatibilityVersions",
			wantMethod: http.MethodDelete,
			wantPath:   "/v1/gameCenterAppVersions/appver-1/relationships/compatibilityVersions",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 1 || got.Data[0].Type != ResourceTypeGameCenterAppVersions || got.Data[0].ID != "appver-2" {
					t.Fatalf("unexpected body: %#v", got)
				}
			},
			call: func(client *Client) error {
				return client.RemoveGameCenterAppVersionCompatibilityVersions(ctx, "appver-1", []string{"appver-2"})
			},
		},
		{
			name:       "AddGameCenterEnabledVersionCompatibleVersions",
			wantMethod: http.MethodPost,
			wantPath:   "/v1/gameCenterEnabledVersions/env-1/relationships/compatibleVersions",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 1 || got.Data[0].Type != ResourceTypeGameCenterEnabledVersions || got.Data[0].ID != "env-2" {
					t.Fatalf("unexpected body: %#v", got)
				}
			},
			call: func(client *Client) error {
				return client.AddGameCenterEnabledVersionCompatibleVersions(ctx, "env-1", []string{"env-2"})
			},
		},
		{
			name:       "RemoveGameCenterEnabledVersionCompatibleVersions",
			wantMethod: http.MethodDelete,
			wantPath:   "/v1/gameCenterEnabledVersions/env-1/relationships/compatibleVersions",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 1 || got.Data[0].Type != ResourceTypeGameCenterEnabledVersions || got.Data[0].ID != "env-2" {
					t.Fatalf("unexpected body: %#v", got)
				}
			},
			call: func(client *Client) error {
				return client.RemoveGameCenterEnabledVersionCompatibleVersions(ctx, "env-1", []string{"env-2"})
			},
		},
		{
			name:       "UpdateGameCenterEnabledVersionCompatibleVersionsRelationship",
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/gameCenterEnabledVersions/env-1/relationships/compatibleVersions",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 2 {
					t.Fatalf("expected 2 items, got %d", len(got.Data))
				}
				if got.Data[0].Type != ResourceTypeGameCenterEnabledVersions || got.Data[0].ID != "env-2" {
					t.Fatalf("unexpected first item: %#v", got.Data[0])
				}
				if got.Data[1].Type != ResourceTypeGameCenterEnabledVersions || got.Data[1].ID != "env-3" {
					t.Fatalf("unexpected second item: %#v", got.Data[1])
				}
			},
			call: func(client *Client) error {
				return client.UpdateGameCenterEnabledVersionCompatibleVersionsRelationship(ctx, "env-1", []string{"env-2", "env-3"})
			},
		},
		{
			name:       "UpdateGameCenterAchievementActivityRelationship",
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/gameCenterAchievements/ach-1/relationships/activity",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got toOneRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if got.Data.Type != ResourceTypeGameCenterActivities || got.Data.ID != "act-1" {
					t.Fatalf("unexpected data: %#v", got.Data)
				}
			},
			call: func(client *Client) error {
				return client.UpdateGameCenterAchievementActivityRelationship(ctx, "ach-1", "act-1")
			},
		},
		{
			name:       "UpdateGameCenterAchievementGroupAchievementRelationship",
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/gameCenterAchievements/ach-1/relationships/groupAchievement",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got toOneRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if got.Data.Type != ResourceTypeGameCenterAchievements || got.Data.ID != "ach-2" {
					t.Fatalf("unexpected data: %#v", got.Data)
				}
			},
			call: func(client *Client) error {
				return client.UpdateGameCenterAchievementGroupAchievementRelationship(ctx, "ach-1", "ach-2")
			},
		},
		{
			name:       "UpdateGameCenterChallengeLeaderboardRelationship",
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/gameCenterChallenges/chal-1/relationships/leaderboard",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got toOneRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if got.Data.Type != ResourceTypeGameCenterLeaderboards || got.Data.ID != "lb-1" {
					t.Fatalf("unexpected data: %#v", got.Data)
				}
			},
			call: func(client *Client) error {
				return client.UpdateGameCenterChallengeLeaderboardRelationship(ctx, "chal-1", "lb-1")
			},
		},
		{
			name:       "UpdateGameCenterChallengeLeaderboardV2Relationship",
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/gameCenterChallenges/chal-1/relationships/leaderboardV2",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got toOneRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if got.Data.Type != ResourceTypeGameCenterLeaderboards || got.Data.ID != "lb-2" {
					t.Fatalf("unexpected data: %#v", got.Data)
				}
			},
			call: func(client *Client) error {
				return client.UpdateGameCenterChallengeLeaderboardV2Relationship(ctx, "chal-1", "lb-2")
			},
		},
		{
			name:       "UpdateGameCenterAchievementActivityRelationshipV2",
			wantMethod: http.MethodPatch,
			wantPath:   "/v2/gameCenterAchievements/ach-1/relationships/activity",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got toOneRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if got.Data.Type != ResourceTypeGameCenterActivities || got.Data.ID != "act-1" {
					t.Fatalf("unexpected data: %#v", got.Data)
				}
			},
			call: func(client *Client) error {
				return client.UpdateGameCenterAchievementActivityRelationshipV2(ctx, "ach-1", "act-1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(req *http.Request) {
				if req.Method != tt.wantMethod {
					t.Fatalf("expected %s, got %s", tt.wantMethod, req.Method)
				}
				if req.URL.Path != tt.wantPath {
					t.Fatalf("expected path %s, got %s", tt.wantPath, req.URL.Path)
				}
				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				tt.wantBodyFn(t, body)
				assertAuthorized(t, req)
			}, jsonResponse(http.StatusNoContent, ""))

			if err := tt.call(client); err != nil {
				t.Fatalf("request error: %v", err)
			}
		})
	}
}
