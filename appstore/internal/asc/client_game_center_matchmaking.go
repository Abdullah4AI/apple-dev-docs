package asc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetGameCenterMatchmakingQueues retrieves the list of matchmaking queues.
func (c *Client) GetGameCenterMatchmakingQueues(ctx context.Context, opts ...GCMatchmakingQueuesOption) (*GameCenterMatchmakingQueuesResponse, error) {
	query := &gcMatchmakingQueuesQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := "/v1/gameCenterMatchmakingQueues"
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-queues: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingQueuesQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingQueuesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterMatchmakingQueue retrieves a matchmaking queue by ID.
func (c *Client) GetGameCenterMatchmakingQueue(ctx context.Context, queueID string) (*GameCenterMatchmakingQueueResponse, error) {
	path := fmt.Sprintf("/v1/gameCenterMatchmakingQueues/%s", strings.TrimSpace(queueID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingQueueResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateGameCenterMatchmakingQueue creates a new matchmaking queue.
func (c *Client) CreateGameCenterMatchmakingQueue(ctx context.Context, attrs GameCenterMatchmakingQueueCreateAttributes, ruleSetID string, experimentRuleSetID string) (*GameCenterMatchmakingQueueResponse, error) {
	relationships := &GameCenterMatchmakingQueueRelationships{
		RuleSet: &Relationship{
			Data: ResourceData{
				Type: ResourceTypeGameCenterMatchmakingRuleSets,
				ID:   strings.TrimSpace(ruleSetID),
			},
		},
	}
	if strings.TrimSpace(experimentRuleSetID) != "" {
		relationships.ExperimentRuleSet = &Relationship{
			Data: ResourceData{
				Type: ResourceTypeGameCenterMatchmakingRuleSets,
				ID:   strings.TrimSpace(experimentRuleSetID),
			},
		}
	}

	payload := GameCenterMatchmakingQueueCreateRequest{
		Data: GameCenterMatchmakingQueueCreateData{
			Type:          ResourceTypeGameCenterMatchmakingQueues,
			Attributes:    attrs,
			Relationships: relationships,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/gameCenterMatchmakingQueues", body)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingQueueResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateGameCenterMatchmakingQueue updates a matchmaking queue.
func (c *Client) UpdateGameCenterMatchmakingQueue(ctx context.Context, queueID string, attrs GameCenterMatchmakingQueueUpdateAttributes, ruleSetID string, experimentRuleSetID string) (*GameCenterMatchmakingQueueResponse, error) {
	var relationships *GameCenterMatchmakingQueueRelationships
	if strings.TrimSpace(ruleSetID) != "" || strings.TrimSpace(experimentRuleSetID) != "" {
		relationships = &GameCenterMatchmakingQueueRelationships{}
		if strings.TrimSpace(ruleSetID) != "" {
			relationships.RuleSet = &Relationship{
				Data: ResourceData{
					Type: ResourceTypeGameCenterMatchmakingRuleSets,
					ID:   strings.TrimSpace(ruleSetID),
				},
			}
		}
		if strings.TrimSpace(experimentRuleSetID) != "" {
			relationships.ExperimentRuleSet = &Relationship{
				Data: ResourceData{
					Type: ResourceTypeGameCenterMatchmakingRuleSets,
					ID:   strings.TrimSpace(experimentRuleSetID),
				},
			}
		}
	}

	payload := GameCenterMatchmakingQueueUpdateRequest{
		Data: GameCenterMatchmakingQueueUpdateData{
			Type:          ResourceTypeGameCenterMatchmakingQueues,
			ID:            strings.TrimSpace(queueID),
			Attributes:    &attrs,
			Relationships: relationships,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingQueues/%s", strings.TrimSpace(queueID))
	data, err := c.do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingQueueResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// DeleteGameCenterMatchmakingQueue deletes a matchmaking queue.
func (c *Client) DeleteGameCenterMatchmakingQueue(ctx context.Context, queueID string) error {
	path := fmt.Sprintf("/v1/gameCenterMatchmakingQueues/%s", strings.TrimSpace(queueID))
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	return err
}

// GetGameCenterMatchmakingRuleSets retrieves the list of matchmaking rule sets.
func (c *Client) GetGameCenterMatchmakingRuleSets(ctx context.Context, opts ...GCMatchmakingRuleSetsOption) (*GameCenterMatchmakingRuleSetsResponse, error) {
	query := &gcMatchmakingRuleSetsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := "/v1/gameCenterMatchmakingRuleSets"
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-rule-sets: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingRuleSetsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingRuleSetsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterMatchmakingRuleSet retrieves a matchmaking rule set by ID.
func (c *Client) GetGameCenterMatchmakingRuleSet(ctx context.Context, ruleSetID string) (*GameCenterMatchmakingRuleSetResponse, error) {
	path := fmt.Sprintf("/v1/gameCenterMatchmakingRuleSets/%s", strings.TrimSpace(ruleSetID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingRuleSetResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterMatchmakingRuleSetQueues retrieves the queues for a rule set.
func (c *Client) GetGameCenterMatchmakingRuleSetQueues(ctx context.Context, ruleSetID string, opts ...GCMatchmakingQueuesOption) (*GameCenterMatchmakingQueuesResponse, error) {
	query := &gcMatchmakingQueuesQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingRuleSets/%s/matchmakingQueues", strings.TrimSpace(ruleSetID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-rule-set-queues: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingQueuesQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingQueuesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateGameCenterMatchmakingRuleSet creates a new matchmaking rule set.
func (c *Client) CreateGameCenterMatchmakingRuleSet(ctx context.Context, attrs GameCenterMatchmakingRuleSetCreateAttributes) (*GameCenterMatchmakingRuleSetResponse, error) {
	payload := GameCenterMatchmakingRuleSetCreateRequest{
		Data: GameCenterMatchmakingRuleSetCreateData{
			Type:       ResourceTypeGameCenterMatchmakingRuleSets,
			Attributes: attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/gameCenterMatchmakingRuleSets", body)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingRuleSetResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateGameCenterMatchmakingRuleSet updates a matchmaking rule set.
func (c *Client) UpdateGameCenterMatchmakingRuleSet(ctx context.Context, ruleSetID string, attrs GameCenterMatchmakingRuleSetUpdateAttributes) (*GameCenterMatchmakingRuleSetResponse, error) {
	payload := GameCenterMatchmakingRuleSetUpdateRequest{
		Data: GameCenterMatchmakingRuleSetUpdateData{
			Type:       ResourceTypeGameCenterMatchmakingRuleSets,
			ID:         strings.TrimSpace(ruleSetID),
			Attributes: &attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingRuleSets/%s", strings.TrimSpace(ruleSetID))
	data, err := c.do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingRuleSetResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// DeleteGameCenterMatchmakingRuleSet deletes a matchmaking rule set.
func (c *Client) DeleteGameCenterMatchmakingRuleSet(ctx context.Context, ruleSetID string) error {
	path := fmt.Sprintf("/v1/gameCenterMatchmakingRuleSets/%s", strings.TrimSpace(ruleSetID))
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	return err
}

// GetGameCenterMatchmakingRules retrieves the list of rules for a rule set.
func (c *Client) GetGameCenterMatchmakingRules(ctx context.Context, ruleSetID string, opts ...GCMatchmakingRulesOption) (*GameCenterMatchmakingRulesResponse, error) {
	query := &gcMatchmakingRulesQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingRuleSets/%s/rules", strings.TrimSpace(ruleSetID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-rules: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingRulesQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingRulesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateGameCenterMatchmakingRule creates a new rule for a rule set.
func (c *Client) CreateGameCenterMatchmakingRule(ctx context.Context, ruleSetID string, attrs GameCenterMatchmakingRuleCreateAttributes) (*GameCenterMatchmakingRuleResponse, error) {
	payload := GameCenterMatchmakingRuleCreateRequest{
		Data: GameCenterMatchmakingRuleCreateData{
			Type:       ResourceTypeGameCenterMatchmakingRules,
			Attributes: attrs,
			Relationships: &GameCenterMatchmakingRuleRelationships{
				RuleSet: &Relationship{
					Data: ResourceData{
						Type: ResourceTypeGameCenterMatchmakingRuleSets,
						ID:   strings.TrimSpace(ruleSetID),
					},
				},
			},
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/gameCenterMatchmakingRules", body)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingRuleResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateGameCenterMatchmakingRule updates a matchmaking rule.
func (c *Client) UpdateGameCenterMatchmakingRule(ctx context.Context, ruleID string, attrs GameCenterMatchmakingRuleUpdateAttributes) (*GameCenterMatchmakingRuleResponse, error) {
	payload := GameCenterMatchmakingRuleUpdateRequest{
		Data: GameCenterMatchmakingRuleUpdateData{
			Type:       ResourceTypeGameCenterMatchmakingRules,
			ID:         strings.TrimSpace(ruleID),
			Attributes: &attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingRules/%s", strings.TrimSpace(ruleID))
	data, err := c.do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingRuleResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// DeleteGameCenterMatchmakingRule deletes a matchmaking rule.
func (c *Client) DeleteGameCenterMatchmakingRule(ctx context.Context, ruleID string) error {
	path := fmt.Sprintf("/v1/gameCenterMatchmakingRules/%s", strings.TrimSpace(ruleID))
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	return err
}

// GetGameCenterMatchmakingTeams retrieves the list of teams for a rule set.
func (c *Client) GetGameCenterMatchmakingTeams(ctx context.Context, ruleSetID string, opts ...GCMatchmakingTeamsOption) (*GameCenterMatchmakingTeamsResponse, error) {
	query := &gcMatchmakingTeamsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingRuleSets/%s/teams", strings.TrimSpace(ruleSetID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-teams: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingTeamsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingTeamsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateGameCenterMatchmakingTeam creates a new team for a rule set.
func (c *Client) CreateGameCenterMatchmakingTeam(ctx context.Context, ruleSetID string, attrs GameCenterMatchmakingTeamCreateAttributes) (*GameCenterMatchmakingTeamResponse, error) {
	payload := GameCenterMatchmakingTeamCreateRequest{
		Data: GameCenterMatchmakingTeamCreateData{
			Type:       ResourceTypeGameCenterMatchmakingTeams,
			Attributes: attrs,
			Relationships: &GameCenterMatchmakingTeamRelationships{
				RuleSet: &Relationship{
					Data: ResourceData{
						Type: ResourceTypeGameCenterMatchmakingRuleSets,
						ID:   strings.TrimSpace(ruleSetID),
					},
				},
			},
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/gameCenterMatchmakingTeams", body)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingTeamResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateGameCenterMatchmakingTeam updates a matchmaking team.
func (c *Client) UpdateGameCenterMatchmakingTeam(ctx context.Context, teamID string, attrs GameCenterMatchmakingTeamUpdateAttributes) (*GameCenterMatchmakingTeamResponse, error) {
	payload := GameCenterMatchmakingTeamUpdateRequest{
		Data: GameCenterMatchmakingTeamUpdateData{
			Type:       ResourceTypeGameCenterMatchmakingTeams,
			ID:         strings.TrimSpace(teamID),
			Attributes: &attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingTeams/%s", strings.TrimSpace(teamID))
	data, err := c.do(ctx, http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingTeamResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// DeleteGameCenterMatchmakingTeam deletes a matchmaking team.
func (c *Client) DeleteGameCenterMatchmakingTeam(ctx context.Context, teamID string) error {
	path := fmt.Sprintf("/v1/gameCenterMatchmakingTeams/%s", strings.TrimSpace(teamID))
	_, err := c.do(ctx, http.MethodDelete, path, nil)
	return err
}

// GetGameCenterMatchmakingQueueSizes retrieves queue sizes metrics.
func (c *Client) GetGameCenterMatchmakingQueueSizes(ctx context.Context, queueID string, opts ...GCMatchmakingMetricsOption) (*GameCenterMatchmakingQueueSizesResponse, error) {
	query := &gcMatchmakingMetricsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingQueues/%s/metrics/matchmakingQueueSizes", strings.TrimSpace(queueID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-queue-sizes: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingQueueSizesQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingQueueSizesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterMatchmakingQueueRequests retrieves queue requests metrics.
func (c *Client) GetGameCenterMatchmakingQueueRequests(ctx context.Context, queueID string, opts ...GCMatchmakingMetricsOption) (*GameCenterMatchmakingQueueRequestsResponse, error) {
	query := &gcMatchmakingMetricsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingQueues/%s/metrics/matchmakingRequests", strings.TrimSpace(queueID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-queue-requests: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingQueueRequestsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingQueueRequestsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterMatchmakingQueueSessions retrieves matchmaking sessions metrics.
func (c *Client) GetGameCenterMatchmakingQueueSessions(ctx context.Context, queueID string, opts ...GCMatchmakingMetricsOption) (*GameCenterMatchmakingQueueSessionsResponse, error) {
	query := &gcMatchmakingMetricsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingQueues/%s/metrics/matchmakingSessions", strings.TrimSpace(queueID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-queue-sessions: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingQueueSessionsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingQueueSessionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterMatchmakingQueueExperimentSizes retrieves experiment queue sizes metrics.
func (c *Client) GetGameCenterMatchmakingQueueExperimentSizes(ctx context.Context, queueID string, opts ...GCMatchmakingMetricsOption) (*GameCenterMatchmakingQueueExperimentSizesResponse, error) {
	query := &gcMatchmakingMetricsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingQueues/%s/metrics/experimentMatchmakingQueueSizes", strings.TrimSpace(queueID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-experiment-queue-sizes: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingQueueSizesQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingQueueExperimentSizesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterMatchmakingQueueExperimentRequests retrieves experiment queue requests metrics.
func (c *Client) GetGameCenterMatchmakingQueueExperimentRequests(ctx context.Context, queueID string, opts ...GCMatchmakingMetricsOption) (*GameCenterMatchmakingQueueExperimentRequestsResponse, error) {
	query := &gcMatchmakingMetricsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingQueues/%s/metrics/experimentMatchmakingRequests", strings.TrimSpace(queueID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-experiment-requests: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingQueueRequestsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingQueueExperimentRequestsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterMatchmakingBooleanRuleResults retrieves boolean rule results metrics.
func (c *Client) GetGameCenterMatchmakingBooleanRuleResults(ctx context.Context, ruleID string, opts ...GCMatchmakingMetricsOption) (*GameCenterMatchmakingBooleanRuleResultsResponse, error) {
	query := &gcMatchmakingMetricsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingRules/%s/metrics/matchmakingBooleanRuleResults", strings.TrimSpace(ruleID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-boolean-rule-results: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingRuleMetricsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingBooleanRuleResultsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterMatchmakingNumberRuleResults retrieves number rule results metrics.
func (c *Client) GetGameCenterMatchmakingNumberRuleResults(ctx context.Context, ruleID string, opts ...GCMatchmakingMetricsOption) (*GameCenterMatchmakingNumberRuleResultsResponse, error) {
	query := &gcMatchmakingMetricsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingRules/%s/metrics/matchmakingNumberRuleResults", strings.TrimSpace(ruleID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-number-rule-results: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingRuleMetricsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingNumberRuleResultsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetGameCenterMatchmakingRuleErrors retrieves rule errors metrics.
func (c *Client) GetGameCenterMatchmakingRuleErrors(ctx context.Context, ruleID string, opts ...GCMatchmakingMetricsOption) (*GameCenterMatchmakingRuleErrorsResponse, error) {
	query := &gcMatchmakingMetricsQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := fmt.Sprintf("/v1/gameCenterMatchmakingRules/%s/metrics/matchmakingRuleErrors", strings.TrimSpace(ruleID))
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("game-center-matchmaking-rule-errors: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildGCMatchmakingRuleMetricsQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingRuleErrorsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateGameCenterMatchmakingRuleSetTest creates a rule set test from a JSON payload.
func (c *Client) CreateGameCenterMatchmakingRuleSetTest(ctx context.Context, payload json.RawMessage) (*GameCenterMatchmakingRuleSetTestResponse, error) {
	if len(bytes.TrimSpace(payload)) == 0 {
		return nil, fmt.Errorf("empty rule set test payload")
	}

	data, err := c.do(ctx, http.MethodPost, "/v1/gameCenterMatchmakingRuleSetTests", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	var response GameCenterMatchmakingRuleSetTestResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
