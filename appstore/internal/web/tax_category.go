package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	taxCategoryCatalogInclude = "subcategories,conditions"
	appTaxCategoryInclude     = "category,enabledConditions"
	taxCategoryRelationship   = "category"
	taxConditionRelationship  = "conditions"
)

// TaxCategoryReference identifies a tax category or condition related to a
// tax category. Names are populated from JSON:API included resources when
// Apple returns them.
type TaxCategoryReference struct {
	ID   string `json:"id"`
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

// TaxCondition describes an application tax condition available in the
// captured tax-category catalog.
type TaxCondition struct {
	ID   string `json:"id"`
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

// TaxCategory describes an application tax category and its related choices.
// ContentProviders is retained as returned by Apple's web API because its
// shape is not stable in the captured responses.
type TaxCategory struct {
	ID                  string                 `json:"id"`
	Type                string                 `json:"type,omitempty"`
	Name                string                 `json:"name,omitempty"`
	ProductType         string                 `json:"productType,omitempty"`
	SubcategoryRequired bool                   `json:"subcategoryRequired"`
	ContentProviders    any                    `json:"contentProviders,omitempty"`
	Subcategories       []TaxCategoryReference `json:"subcategories,omitempty"`
	Conditions          []TaxCategoryReference `json:"conditions,omitempty"`
}

// TaxCategoryCatalog is the application tax category and condition catalog.
type TaxCategoryCatalog struct {
	Categories []TaxCategory  `json:"categories"`
	Conditions []TaxCondition `json:"conditions"`
	// Raw preserves the complete JSON:API catalog envelope returned by Apple.
	// It is omitted from the typed shape and emitted verbatim for JSON output,
	// so included resources, links, metadata, and unknown top-level members are
	// not lost while parsing table rows.
	Raw json.RawMessage `json:"-"`
}

// MarshalJSON preserves Apple's raw catalog envelope for JSON output. The
// typed fields remain available to callers that need validation or tables.
func (c TaxCategoryCatalog) MarshalJSON() ([]byte, error) {
	if len(c.Raw) > 0 {
		return append([]byte(nil), c.Raw...), nil
	}
	type catalogWithoutRaw TaxCategoryCatalog
	return json.Marshal(catalogWithoutRaw(c))
}

// AppTaxCategory describes the explicit tax category assigned to an app.
// Configured is false when Apple's appTaxCategories relationship is absent
// (Apple reports that state as a 404 for apps using the UI default).
type AppTaxCategory struct {
	ID                  string   `json:"id,omitempty"`
	AppID               string   `json:"appId"`
	CategoryID          string   `json:"categoryId,omitempty"`
	CategoryName        string   `json:"categoryName,omitempty"`
	EnabledConditionIDs []string `json:"enabledConditionIds,omitempty"`
	Configured          bool     `json:"configured"`
}

func taxCategoryReferenceFromRef(ref resourceRef, included map[string]jsonAPIResource) TaxCategoryReference {
	reference := TaxCategoryReference{
		ID:   strings.TrimSpace(ref.ID),
		Type: strings.TrimSpace(ref.Type),
	}
	if includedResource, ok := included[jsonAPIResourceKey(reference.Type, reference.ID)]; ok {
		reference.Name = stringAttr(includedResource.Attributes, "name")
	}
	return reference
}

func decodeTaxCategoryResource(resource jsonAPIResource, included map[string]jsonAPIResource) TaxCategory {
	category := TaxCategory{
		ID:                  strings.TrimSpace(resource.ID),
		Type:                strings.TrimSpace(resource.Type),
		Name:                stringAttr(resource.Attributes, "name"),
		ProductType:         stringAttr(resource.Attributes, "productType"),
		SubcategoryRequired: boolAttr(resource.Attributes, "subcategoryRequired"),
		ContentProviders:    resource.Attributes["contentProviders"],
	}
	for _, ref := range relationshipRefs(resource, "subcategories") {
		if strings.TrimSpace(ref.ID) == "" {
			continue
		}
		category.Subcategories = append(category.Subcategories, taxCategoryReferenceFromRef(ref, included))
	}
	for _, ref := range relationshipRefs(resource, taxConditionRelationship) {
		if strings.TrimSpace(ref.ID) == "" {
			continue
		}
		category.Conditions = append(category.Conditions, taxCategoryReferenceFromRef(ref, included))
	}
	return category
}

func decodeTaxConditionResource(resource jsonAPIResource) TaxCondition {
	name := stringAttr(resource.Attributes, "description")
	if name == "" {
		name = stringAttr(resource.Attributes, "name")
	}
	return TaxCondition{
		ID:   strings.TrimSpace(resource.ID),
		Type: strings.TrimSpace(resource.Type),
		Name: name,
	}
}

func appendUniqueTaxResource(resources []jsonAPIResource, seen map[string]struct{}, resource jsonAPIResource) []jsonAPIResource {
	key := jsonAPIResourceKey(resource.Type, resource.ID)
	if strings.TrimSpace(resource.Type) == "" || strings.TrimSpace(resource.ID) == "" {
		return resources
	}
	if _, exists := seen[key]; exists {
		return resources
	}
	seen[key] = struct{}{}
	return append(resources, resource)
}

func decodeTaxCategoryCatalog(payload jsonAPIListPayload) TaxCategoryCatalog {
	included := buildIncludedMap(payload.Included)
	categoryResources := make([]jsonAPIResource, 0, len(payload.Data)+len(payload.Included))
	conditionResources := make([]jsonAPIResource, 0, len(payload.Included))
	categorySeen := make(map[string]struct{})
	conditionSeen := make(map[string]struct{})
	for _, resource := range payload.Data {
		if resource.Type == "taxCategories" {
			categoryResources = appendUniqueTaxResource(categoryResources, categorySeen, resource)
		}
		if resource.Type == "taxConditions" {
			conditionResources = appendUniqueTaxResource(conditionResources, conditionSeen, resource)
		}
	}
	for _, resource := range payload.Included {
		switch resource.Type {
		case "taxCategories":
			categoryResources = appendUniqueTaxResource(categoryResources, categorySeen, resource)
		case "taxConditions":
			conditionResources = appendUniqueTaxResource(conditionResources, conditionSeen, resource)
		}
	}

	catalog := TaxCategoryCatalog{
		Categories: make([]TaxCategory, 0, len(categoryResources)),
		Conditions: make([]TaxCondition, 0, len(conditionResources)),
	}
	for _, resource := range categoryResources {
		catalog.Categories = append(catalog.Categories, decodeTaxCategoryResource(resource, included))
	}
	for _, resource := range conditionResources {
		catalog.Conditions = append(catalog.Conditions, decodeTaxConditionResource(resource))
	}
	return catalog
}

// ListTaxCategories reads the application tax category catalog used by the
// App Information tax UI. This is an internal web-session endpoint; it is not
// part of Apple's public App Store Connect API.
func (c *Client) ListTaxCategories(ctx context.Context) (TaxCategoryCatalog, error) {
	return c.listTaxCategoriesForProduct(ctx, "APPLICATION")
}

func (c *Client) listTaxCategoriesForProduct(ctx context.Context, productType string) (TaxCategoryCatalog, error) {
	query := url.Values{}
	query.Set("filter[productType]", productType)
	query.Set("include", taxCategoryCatalogInclude)
	query.Set("limit[subcategories]", "100")
	query.Set("limit[conditions]", "100")
	path := queryPath("/taxCategories", query)
	responseBody, err := c.doJSONAPIRequest(ctx, c.baseURL, path)
	if err != nil {
		return TaxCategoryCatalog{}, err
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return TaxCategoryCatalog{}, fmt.Errorf("failed to parse tax categories response: %w", err)
	}
	trimmedData := bytes.TrimSpace(envelope.Data)
	if len(trimmedData) == 0 || bytes.Equal(trimmedData, []byte("null")) {
		return TaxCategoryCatalog{}, fmt.Errorf("tax categories response missing non-null data")
	}
	var payload jsonAPIListPayload
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return TaxCategoryCatalog{}, fmt.Errorf("failed to parse tax categories response: %w", err)
	}
	catalog := decodeTaxCategoryCatalog(payload)
	catalog.Raw = append(json.RawMessage(nil), responseBody...)
	return catalog, nil
}

func decodeAppTaxCategoryResource(resource jsonAPIResource, included []jsonAPIResource, appID string) *AppTaxCategory {
	includedMap := buildIncludedMap(included)
	result := &AppTaxCategory{
		ID:                  strings.TrimSpace(resource.ID),
		AppID:               strings.TrimSpace(appID),
		EnabledConditionIDs: make([]string, 0),
		Configured:          true,
	}
	if result.AppID == "" {
		result.AppID = strings.TrimSpace(resource.ID)
	}
	if ref := firstRelationshipRef(resource, taxCategoryRelationship); ref != nil {
		result.CategoryID = strings.TrimSpace(ref.ID)
		if related, ok := includedMap[jsonAPIResourceKey(ref.Type, ref.ID)]; ok {
			result.CategoryName = stringAttr(related.Attributes, "name")
		}
	}
	for _, ref := range relationshipRefs(resource, "enabledConditions") {
		if strings.TrimSpace(ref.ID) != "" {
			result.EnabledConditionIDs = append(result.EnabledConditionIDs, strings.TrimSpace(ref.ID))
		}
	}
	return result
}

func (c *Client) verifiedUnconfiguredAppTaxCategory(ctx context.Context, appID string) (*AppTaxCategory, error) {
	verifiedApp, verifyErr := c.GetApp(ctx, appID)
	if verifyErr != nil {
		return nil, fmt.Errorf("failed to verify app %q after missing tax category: %w", appID, verifyErr)
	}
	if verifiedApp == nil || strings.TrimSpace(verifiedApp.Data.ID) != appID || strings.TrimSpace(verifiedApp.Data.Type) != "apps" {
		var verifiedID, verifiedType string
		if verifiedApp != nil {
			verifiedID = strings.TrimSpace(verifiedApp.Data.ID)
			verifiedType = strings.TrimSpace(verifiedApp.Data.Type)
		}
		return nil, fmt.Errorf("failed to verify app %q after missing tax category: response identified app %q of type %q", appID, verifiedID, verifiedType)
	}
	return &AppTaxCategory{
		AppID:               appID,
		EnabledConditionIDs: []string{},
		Configured:          false,
	}, nil
}

// GetAppTaxCategory reads an app's explicit tax category and enabled
// conditions. A tax-category 404 is treated as Apple's App Store Software UI
// default only after a successful app resource read verifies the app exists.
func (c *Client) GetAppTaxCategory(ctx context.Context, appID string) (*AppTaxCategory, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("app id is required")
	}
	query := url.Values{}
	query.Set("include", appTaxCategoryInclude)
	query.Set("limit[enabledConditions]", "100")
	path := queryPath("/appTaxCategories/"+url.PathEscape(appID), query)
	responseBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		if IsNotFound(err) {
			return c.verifiedUnconfiguredAppTaxCategory(ctx, appID)
		}
		return nil, err
	}
	var payload jsonAPISingleResourcePayload
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse app tax category response: %w", err)
	}
	trimmedData := strings.TrimSpace(string(payload.Data))
	if trimmedData == "" || trimmedData == "null" {
		return c.verifiedUnconfiguredAppTaxCategory(ctx, appID)
	}
	var resource jsonAPIResource
	if err := json.Unmarshal(payload.Data, &resource); err != nil {
		return nil, fmt.Errorf("failed to parse app tax category resource: %w", err)
	}
	if strings.TrimSpace(resource.ID) == "" {
		return nil, fmt.Errorf("app tax category response did not include a resource id")
	}
	if strings.TrimSpace(resource.ID) != appID || strings.TrimSpace(resource.Type) != "appTaxCategories" {
		return nil, fmt.Errorf("app tax category response identified resource %q of type %q, want app tax category for %q", resource.ID, resource.Type, appID)
	}
	return decodeAppTaxCategoryResource(resource, payload.Included, appID), nil
}

func normalizeTaxConditionIDs(conditionIDs []string) ([]string, error) {
	normalized := make([]string, 0, len(conditionIDs))
	seen := make(map[string]struct{}, len(conditionIDs))
	for _, conditionID := range conditionIDs {
		conditionID = strings.TrimSpace(conditionID)
		if conditionID == "" {
			return nil, fmt.Errorf("condition id cannot be empty")
		}
		if _, exists := seen[conditionID]; exists {
			continue
		}
		seen[conditionID] = struct{}{}
		normalized = append(normalized, conditionID)
	}
	return normalized, nil
}

// SaveAppTaxCategory writes an app's complete desired category and condition
// set. The explicit empty enabledConditions relationship is intentional: an
// omitted condition selection means clear the current conditions rather than
// preserve stale values when the category changes.
func (c *Client) SaveAppTaxCategory(ctx context.Context, appID, categoryID string, conditionIDs []string, configured bool) error {
	appID = strings.TrimSpace(appID)
	categoryID = strings.TrimSpace(categoryID)
	if appID == "" {
		return fmt.Errorf("app id is required")
	}
	if categoryID == "" {
		return fmt.Errorf("category id is required")
	}
	normalizedConditions, err := normalizeTaxConditionIDs(conditionIDs)
	if err != nil {
		return err
	}
	conditionData := make([]map[string]string, 0, len(normalizedConditions))
	for _, conditionID := range normalizedConditions {
		conditionData = append(conditionData, map[string]string{
			"type": "taxConditions",
			"id":   conditionID,
		})
	}
	relationships := map[string]any{
		taxCategoryRelationship: map[string]any{
			"data": map[string]string{
				"type": "taxCategories",
				"id":   categoryID,
			},
		},
		"enabledConditions": map[string]any{
			"data": conditionData,
		},
	}
	method := http.MethodPost
	path := "/appTaxCategories"
	data := map[string]any{
		"type":          "appTaxCategories",
		"relationships": relationships,
	}
	if configured {
		method = http.MethodPatch
		path += "/" + url.PathEscape(appID)
		data["id"] = appID
	} else {
		relationships["app"] = map[string]any{
			"data": map[string]string{
				"type": "apps",
				"id":   appID,
			},
		}
	}
	_, err = c.doRequest(ctx, method, path, map[string]any{"data": data})
	return err
}
