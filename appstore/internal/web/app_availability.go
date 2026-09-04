package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

// AppAvailability models the internal web API app availability resource.
type AppAvailability struct {
	ID                             string   `json:"id"`
	Type                           string   `json:"type,omitempty"`
	AvailableInNewTerritories      bool     `json:"availableInNewTerritories"`
	AvailableTerritories           []string `json:"availableTerritories,omitempty"`
	AvailableTerritoriesLoaded     bool     `json:"-"`
	AvailableInNewTerritoriesKnown bool     `json:"-"`
}

type appAvailabilityRelatedReadError struct {
	err error
}

func (e *appAvailabilityRelatedReadError) Error() string {
	if e == nil || e.err == nil {
		return "could not read territoryAvailabilities"
	}
	return e.err.Error()
}

func (e *appAvailabilityRelatedReadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// AppAvailabilityCreateAttributes defines inputs for creating initial app availability.
type AppAvailabilityCreateAttributes struct {
	AppID                     string   `json:"-"`
	AvailableInNewTerritories bool     `json:"-"`
	AvailableTerritories      []string `json:"-"`
}

// IsNotFound reports whether the internal web API returned a not-found response.
func IsNotFound(err error) bool {
	var relatedErr *appAvailabilityRelatedReadError
	if errors.As(err, &relatedErr) {
		return false
	}
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

func normalizeAppAvailabilityCreateAttributes(attrs AppAvailabilityCreateAttributes) (AppAvailabilityCreateAttributes, error) {
	attrs.AppID = strings.TrimSpace(attrs.AppID)
	if attrs.AppID == "" {
		return attrs, fmt.Errorf("app id is required")
	}

	normalizedTerritories := make([]string, 0, len(attrs.AvailableTerritories))
	seen := make(map[string]struct{}, len(attrs.AvailableTerritories))
	for _, territoryID := range attrs.AvailableTerritories {
		normalized := strings.ToUpper(strings.TrimSpace(territoryID))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		normalizedTerritories = append(normalizedTerritories, normalized)
	}
	if len(normalizedTerritories) == 0 {
		return attrs, fmt.Errorf("at least one available territory is required")
	}
	slices.Sort(normalizedTerritories)
	attrs.AvailableTerritories = normalizedTerritories
	return attrs, nil
}

func decodeAppAvailabilityResource(resource jsonAPIResource) AppAvailability {
	inNew, inNewKnown := boolAttrKnown(resource.Attributes, "availableInNewTerritories")
	availability := AppAvailability{
		ID:                             strings.TrimSpace(resource.ID),
		Type:                           strings.TrimSpace(resource.Type),
		AvailableInNewTerritories:      inNew,
		AvailableInNewTerritoriesKnown: inNewKnown,
	}

	relationship, ok := resource.Relationships["availableTerritories"]
	if ok {
		trimmedData := strings.TrimSpace(string(relationship.Data))
		availability.AvailableTerritoriesLoaded = trimmedData != "" && trimmedData != "null"
	}

	refs := parseRelationshipRefs(relationship.Data)
	if len(refs) == 0 {
		return availability
	}

	territories := make([]string, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		territoryID := strings.ToUpper(strings.TrimSpace(ref.ID))
		if territoryID == "" {
			continue
		}
		if _, ok := seen[territoryID]; ok {
			continue
		}
		seen[territoryID] = struct{}{}
		territories = append(territories, territoryID)
	}
	slices.Sort(territories)
	availability.AvailableTerritories = territories
	return availability
}

// GetAppAvailability retrieves the internal web app availability resource for an app.
func (c *Client) GetAppAvailability(ctx context.Context, appID string) (*AppAvailability, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("app id is required")
	}

	path := "/apps/" + url.PathEscape(appID) + "/appAvailabilityV2"
	responseBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Data jsonAPIResource `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse app availability response: %w", err)
	}

	availability := decodeAppAvailabilityResource(payload.Data)
	if availability.AvailableTerritoriesLoaded {
		return &availability, nil
	}
	if strings.TrimSpace(availability.ID) == "" {
		return nil, fmt.Errorf("app availability id missing from response")
	}

	territories, err := c.listAppTerritoryAvailabilities(ctx, availability.ID)
	if err != nil {
		return nil, &appAvailabilityRelatedReadError{
			err: fmt.Errorf("could not read territoryAvailabilities for app availability %q: %w", availability.ID, err),
		}
	}
	availability.AvailableTerritories = territories
	availability.AvailableTerritoriesLoaded = true
	return &availability, nil
}

func (c *Client) webIrisV2BaseURL() string {
	base := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	switch {
	case strings.HasSuffix(base, "/iris/v1"):
		return strings.TrimSuffix(base, "/iris/v1") + "/iris/v2"
	case strings.HasSuffix(base, "/iris/v2"):
		return base
	case base == "":
		return irisV2BaseURL
	default:
		return base + "/iris/v2"
	}
}

func (c *Client) listAppTerritoryAvailabilities(ctx context.Context, availabilityID string) ([]string, error) {
	availabilityID = strings.TrimSpace(availabilityID)
	if availabilityID == "" {
		return nil, fmt.Errorf("app availability id is required")
	}

	query := url.Values{}
	query.Set("include", "territory")
	query.Set("limit", "200")
	path := queryPath("/appAvailabilities/"+url.PathEscape(availabilityID)+"/territoryAvailabilities", query)

	payload, err := c.fetchJSONAPIPagesFromWithRequiredLinks(ctx, c.webIrisV2BaseURL(), path, "territory availabilities")
	if err != nil {
		return nil, err
	}

	territories := make([]string, 0)
	seen := make(map[string]struct{})
	for _, resource := range payload.Data {
		available, known := boolAttrKnown(resource.Attributes, "available")
		if !known {
			id := strings.TrimSpace(resource.ID)
			if id == "" {
				id = "unknown"
			}
			return nil, fmt.Errorf("territory availability %q omitted or mistyped the available attribute", id)
		}
		if !available {
			continue
		}
		territoryID := territoryIDFromAvailabilityResource(resource)
		if territoryID == "" {
			return nil, fmt.Errorf("territory availability %q is available but omitted the territory linkage", strings.TrimSpace(resource.ID))
		}
		if _, ok := seen[territoryID]; ok {
			continue
		}
		seen[territoryID] = struct{}{}
		territories = append(territories, territoryID)
	}
	slices.Sort(territories)
	return territories, nil
}

func territoryIDFromAvailabilityResource(resource jsonAPIResource) string {
	relationship, ok := resource.Relationships["territory"]
	if !ok {
		return ""
	}
	refs := parseRelationshipRefs(relationship.Data)
	if len(refs) == 0 {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(refs[0].ID))
}

// CreateAppAvailability creates the initial app availability via the internal web API.
func (c *Client) CreateAppAvailability(ctx context.Context, attrs AppAvailabilityCreateAttributes) (*AppAvailability, error) {
	normalized, err := normalizeAppAvailabilityCreateAttributes(attrs)
	if err != nil {
		return nil, err
	}

	territories := make([]map[string]string, 0, len(normalized.AvailableTerritories))
	for _, territoryID := range normalized.AvailableTerritories {
		territories = append(territories, map[string]string{
			"type": "territories",
			"id":   territoryID,
		})
	}

	requestBody := map[string]any{
		"data": map[string]any{
			"type": "appAvailabilities",
			"attributes": map[string]bool{
				"availableInNewTerritories": normalized.AvailableInNewTerritories,
			},
			"relationships": map[string]any{
				"app": map[string]any{
					"data": map[string]string{
						"type": "apps",
						"id":   normalized.AppID,
					},
				},
				"availableTerritories": map[string]any{
					"data": territories,
				},
			},
		},
	}

	responseBody, err := c.doRequest(ctx, http.MethodPost, "/appAvailabilities", requestBody)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Data jsonAPIResource `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse app availability create response: %w", err)
	}

	availability := decodeAppAvailabilityResource(payload.Data)
	return &availability, nil
}
