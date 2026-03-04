package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func getTypedResourceLinkages[T any](
	c *Client,
	ctx context.Context,
	resourceID string,
	relationship string,
	resourceIDName string,
	pathFmt string,
	nextURLErrorContext string,
	opts ...LinkagesOption,
) (*T, error) {
	query := &linkagesQuery{}
	for _, opt := range opts {
		opt(query)
	}

	resourceID = strings.TrimSpace(resourceID)
	if query.nextURL == "" && resourceID == "" {
		return nil, fmt.Errorf("%s is required", resourceIDName)
	}

	path := fmt.Sprintf(pathFmt, resourceID, relationship)
	if query.nextURL != "" {
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("%s: %w", nextURLErrorContext, err)
		}
		path = query.nextURL
	} else if queryString := buildLinkagesQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response T
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse %s relationship response: %w", relationship, err)
	}

	return &response, nil
}

// getResourceLinkages is a small internal helper for relationship linkages endpoints.
// Many ASC resources share identical pagination and nextURL validation behavior.
func (c *Client) getResourceLinkages(
	ctx context.Context,
	resourceID string,
	relationship string,
	resourceIDName string,
	pathFmt string,
	nextURLErrorContext string,
	opts ...LinkagesOption,
) (*LinkagesResponse, error) {
	return getTypedResourceLinkages[LinkagesResponse](
		c,
		ctx,
		resourceID,
		relationship,
		resourceIDName,
		pathFmt,
		nextURLErrorContext,
		opts...,
	)
}
