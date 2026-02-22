package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// GetBetaGroupApp retrieves the app for a beta group.
func (c *Client) GetBetaGroupApp(ctx context.Context, groupID string) (*AppResponse, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, fmt.Errorf("groupID is required")
	}

	path := fmt.Sprintf("/v1/betaGroups/%s/app", groupID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AppResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetBetaGroupBetaRecruitmentCriteria retrieves recruitment criteria for a beta group.
func (c *Client) GetBetaGroupBetaRecruitmentCriteria(ctx context.Context, groupID string) (*BetaRecruitmentCriteriaResponse, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, fmt.Errorf("groupID is required")
	}

	path := fmt.Sprintf("/v1/betaGroups/%s/betaRecruitmentCriteria", groupID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response BetaRecruitmentCriteriaResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetBetaGroupBetaRecruitmentCriterionCompatibleBuildCheck retrieves compatible build check for a beta group.
func (c *Client) GetBetaGroupBetaRecruitmentCriterionCompatibleBuildCheck(ctx context.Context, groupID string) (*BetaRecruitmentCriterionCompatibleBuildCheckResponse, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return nil, fmt.Errorf("groupID is required")
	}

	path := fmt.Sprintf("/v1/betaGroups/%s/betaRecruitmentCriterionCompatibleBuildCheck", groupID)
	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response BetaRecruitmentCriterionCompatibleBuildCheckResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetBetaTesterApps retrieves apps for a beta tester.
func (c *Client) GetBetaTesterApps(ctx context.Context, testerID string, opts ...BetaTesterAppsOption) (*AppsResponse, error) {
	query := &listQuery{}
	for _, opt := range opts {
		opt(query)
	}

	testerID = strings.TrimSpace(testerID)
	if query.nextURL == "" && testerID == "" {
		return nil, fmt.Errorf("testerID is required")
	}

	path := fmt.Sprintf("/v1/betaTesters/%s/apps", testerID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("betaTesterApps: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildListQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response AppsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetBetaTesterBetaGroups retrieves beta groups for a beta tester.
func (c *Client) GetBetaTesterBetaGroups(ctx context.Context, testerID string, opts ...BetaTesterBetaGroupsOption) (*BetaGroupsResponse, error) {
	query := &listQuery{}
	for _, opt := range opts {
		opt(query)
	}

	testerID = strings.TrimSpace(testerID)
	if query.nextURL == "" && testerID == "" {
		return nil, fmt.Errorf("testerID is required")
	}

	path := fmt.Sprintf("/v1/betaTesters/%s/betaGroups", testerID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("betaTesterBetaGroups: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildListQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response BetaGroupsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetBetaTesterBuilds retrieves builds for a beta tester.
func (c *Client) GetBetaTesterBuilds(ctx context.Context, testerID string, opts ...BetaTesterBuildsOption) (*BuildsResponse, error) {
	query := &listQuery{}
	for _, opt := range opts {
		opt(query)
	}

	testerID = strings.TrimSpace(testerID)
	if query.nextURL == "" && testerID == "" {
		return nil, fmt.Errorf("testerID is required")
	}

	path := fmt.Sprintf("/v1/betaTesters/%s/builds", testerID)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("betaTesterBuilds: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildListQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response BuildsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
