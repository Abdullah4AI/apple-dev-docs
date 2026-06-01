package appleads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// PageDetail is the Apple Ads offset pagination envelope.
type PageDetail struct {
	ItemsPerPage int `json:"itemsPerPage"`
	StartIndex   int `json:"startIndex"`
	TotalResults int `json:"totalResults"`
}

type paginatedEnvelope struct {
	Data       []json.RawMessage `json:"data"`
	Pagination PageDetail        `json:"pagination"`
}

// PaginateAll fetches all pages for an offset-paginated endpoint.
func (c *Client) PaginateAll(ctx context.Context, spec EndpointSpec, pathParams map[string]string, query url.Values, startOffset, pageSize int, body json.RawMessage) (RawResponse, error) {
	maxLimit := MaxPageLimit(spec)
	if pageSize <= 0 {
		pageSize = maxLimit
	}
	if pageSize > maxLimit {
		pageSize = maxLimit
	}

	offset := startOffset
	if offset < 0 {
		offset = 0
	}
	var aggregated []json.RawMessage
	total := -1
	for {
		pageQuery := cloneValues(query)
		pageQuery.Set("limit", strconv.Itoa(pageSize))
		pageQuery.Set("offset", strconv.Itoa(offset))
		raw, err := c.Do(ctx, spec, pathParams, pageQuery, body)
		if err != nil {
			return nil, err
		}
		var page paginatedEnvelope
		if err := json.Unmarshal(raw, &page); err != nil {
			return nil, fmt.Errorf("parse paginated response: %w", err)
		}
		aggregated = append(aggregated, page.Data...)
		itemsPerPage := page.Pagination.ItemsPerPage
		if itemsPerPage <= 0 {
			itemsPerPage = len(page.Data)
		}
		total = page.Pagination.TotalResults
		if len(page.Data) == 0 {
			break
		}
		nextOffset := page.Pagination.StartIndex + itemsPerPage
		if total >= 0 && nextOffset >= total {
			break
		}
		if nextOffset <= offset {
			nextOffset = offset + len(page.Data)
		}
		offset = nextOffset
	}

	out := paginatedEnvelope{
		Data: aggregated,
		Pagination: PageDetail{
			ItemsPerPage: pageSize,
			StartIndex:   max(0, startOffset),
			TotalResults: total,
		},
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return RawResponse(data), nil
}

// MaxPageLimit returns the endpoint-specific maximum page size.
func MaxPageLimit(spec EndpointSpec) int {
	maxLimit := maxAppleAdsPageLimit
	for _, param := range spec.QueryParams {
		if param.Name == "limit" && param.Max > 0 {
			maxLimit = param.Max
			break
		}
	}
	return maxLimit
}

func cloneValues(values url.Values) url.Values {
	cloned := url.Values{}
	for key, items := range values {
		for _, item := range items {
			cloned.Add(key, item)
		}
	}
	return cloned
}
