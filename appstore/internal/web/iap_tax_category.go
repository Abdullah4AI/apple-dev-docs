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

// IAPTaxCategory preserves Apple's read envelope and summarizes its tax override.
// Configured is false only after explicit null linkage on the selected IAP.
type IAPTaxCategory struct {
	Raw                                 json.RawMessage
	IAPID, ID, CategoryID, CategoryName string
	Configured                          bool
	EnabledConditionIDs                 []string
}

func (s IAPTaxCategory) MarshalJSON() ([]byte, error) { return s.Raw, nil }

// ListIAPTaxCategories reads the ADDON catalog used by Apple's IAP tax picker.
func (c *Client) ListIAPTaxCategories(ctx context.Context) (TaxCategoryCatalog, error) {
	return c.listTaxCategoriesForProduct(ctx, "ADDON")
}

func iapTaxReference(data json.RawMessage, expectedType string) (resourceRef, error) {
	var ref resourceRef
	if err := json.Unmarshal(data, &ref); err != nil {
		return ref, fmt.Errorf("invalid %s relationship: %w", expectedType, err)
	}
	if ref.Type != expectedType || strings.TrimSpace(ref.ID) == "" {
		return ref, fmt.Errorf("invalid %s relationship identity", expectedType)
	}
	return ref, nil
}

// GetIAPTaxCategory discovers the tax record instead of assuming its ID matches
// the IAP. An absent relationship or a failed read never means inheritance.
func (c *Client) GetIAPTaxCategory(ctx context.Context, iapID string) (*IAPTaxCategory, error) {
	iapID = strings.TrimSpace(iapID)
	if iapID == "" {
		return nil, fmt.Errorf("iap id is required")
	}
	body, err := c.doJSONAPIRequest(ctx, c.webIrisV2BaseURL(), "/inAppPurchases/"+url.PathEscape(iapID)+"?include=inAppPurchaseTaxCategoryInfo")
	if err != nil {
		return nil, err
	}
	var payload struct {
		Data     jsonAPIResource   `json:"data"`
		Included []jsonAPIResource `json:"included"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse IAP tax discovery: %w", err)
	}
	if payload.Data.Type != "inAppPurchases" || payload.Data.ID != iapID {
		return nil, fmt.Errorf("IAP tax discovery does not identify requested IAP %q", iapID)
	}
	relation, ok := payload.Data.Relationships["inAppPurchaseTaxCategoryInfo"]
	if !ok || len(relation.Data) == 0 {
		return nil, fmt.Errorf("IAP tax discovery omitted tax-category linkage")
	}
	result := &IAPTaxCategory{Raw: body, IAPID: iapID, EnabledConditionIDs: []string{}}
	if bytes.Equal(bytes.TrimSpace(relation.Data), []byte("null")) {
		return result, nil
	}
	ref, err := iapTaxReference(relation.Data, "inAppPurchaseTaxCategoryInfos")
	if err != nil {
		return nil, err
	}
	query := url.Values{"include": {"category,enabledConditions,inAppPurchaseV2"}, "limit[enabledConditions]": {"100"}}
	body, err = c.doJSONAPIRequest(ctx, c.baseURL, queryPath("/inAppPurchaseTaxCategoryInfos/"+url.PathEscape(ref.ID), query))
	if err != nil {
		return nil, err
	}
	// Use a fresh envelope; unmarshalling into the discovery would retain omitted fields.
	var taxPayload struct {
		Data     jsonAPIResource   `json:"data"`
		Included []jsonAPIResource `json:"included"`
	}
	if err := json.Unmarshal(body, &taxPayload); err != nil {
		return nil, fmt.Errorf("parse IAP tax category: %w", err)
	}
	resource := taxPayload.Data
	if resource.Type != ref.Type || resource.ID != ref.ID {
		return nil, fmt.Errorf("IAP tax response does not identify requested tax record %q", ref.ID)
	}
	owner, err := iapTaxReference(resource.Relationships["inAppPurchaseV2"].Data, "inAppPurchases")
	if err != nil {
		return nil, err
	}
	if owner.ID != iapID {
		return nil, fmt.Errorf("IAP tax record belongs to a different IAP")
	}
	category, err := iapTaxReference(resource.Relationships["category"].Data, "taxCategories")
	if err != nil {
		return nil, err
	}
	conditions := resource.Relationships["enabledConditions"].Data
	if !bytes.HasPrefix(bytes.TrimSpace(conditions), []byte("[")) {
		return nil, fmt.Errorf("IAP tax response omitted explicit condition linkage")
	}
	var refs []resourceRef
	if err := json.Unmarshal(conditions, &refs); err != nil {
		return nil, fmt.Errorf("parse IAP tax conditions: %w", err)
	}
	for _, condition := range refs {
		if condition.Type != "taxConditions" || strings.TrimSpace(condition.ID) == "" {
			return nil, fmt.Errorf("invalid IAP tax condition identity")
		}
		result.EnabledConditionIDs = append(result.EnabledConditionIDs, condition.ID)
	}
	// A partial relationship cannot verify a complete condition replacement.
	var completeness struct {
		Data struct {
			Relationships struct {
				Conditions struct {
					Meta struct {
						Paging struct {
							Total *int `json:"total"`
						} `json:"paging"`
					} `json:"meta"`
					Links struct {
						Next json.RawMessage `json:"next"`
					} `json:"links"`
				} `json:"enabledConditions"`
			} `json:"relationships"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &completeness); err != nil {
		return nil, err
	}
	conditionMeta := completeness.Data.Relationships.Conditions
	if total := conditionMeta.Meta.Paging.Total; total != nil && *total != len(refs) {
		return nil, fmt.Errorf("IAP tax response contains an incomplete condition relationship")
	}
	if next := bytes.TrimSpace(conditionMeta.Links.Next); len(next) > 0 && !bytes.Equal(next, []byte("null")) && !bytes.Equal(next, []byte(`""`)) {
		return nil, fmt.Errorf("IAP tax response contains additional condition pages")
	}
	result.Raw = body
	result.ID = ref.ID
	result.Configured = true
	result.CategoryID = category.ID
	for _, included := range taxPayload.Included {
		if included.Type == category.Type && included.ID == category.ID {
			result.CategoryName = stringAttr(included.Attributes, "name")
			break
		}
	}
	return result, nil
}

func validateIAPTaxCurrent(iapID string, current *IAPTaxCategory) error {
	if iapID == "" {
		return fmt.Errorf("iap id is required")
	}
	if current == nil || current.IAPID != iapID {
		return fmt.Errorf("verified current tax state for selected IAP is required")
	}
	if current.Configured && strings.TrimSpace(current.ID) == "" {
		return fmt.Errorf("configured IAP tax state is missing its record id")
	}
	return nil
}

// SaveIAPTaxCategory sends one create or update with the complete condition set.
// Callers must verify the selected IAP again after this request succeeds.
func (c *Client) SaveIAPTaxCategory(ctx context.Context, iapID, categoryID string, conditions []string, current *IAPTaxCategory) error {
	iapID = strings.TrimSpace(iapID)
	categoryID = strings.TrimSpace(categoryID)
	if err := validateIAPTaxCurrent(iapID, current); err != nil {
		return err
	}
	if categoryID == "" {
		return fmt.Errorf("category id is required")
	}
	normalized, err := normalizeTaxConditionIDs(conditions)
	if err != nil {
		return err
	}
	refs := make([]resourceRef, 0, len(normalized))
	for _, id := range normalized {
		refs = append(refs, resourceRef{Type: "taxConditions", ID: id})
	}
	relationships := map[string]any{"category": map[string]any{"data": resourceRef{Type: "taxCategories", ID: categoryID}}, "enabledConditions": map[string]any{"data": refs}}
	data := map[string]any{"type": "inAppPurchaseTaxCategoryInfos", "relationships": relationships}
	method, path := http.MethodPost, "/inAppPurchaseTaxCategoryInfos"
	if current.Configured {
		method = http.MethodPatch
		path += "/" + url.PathEscape(current.ID)
		data["id"] = current.ID
	} else {
		relationships["inAppPurchaseV2"] = map[string]any{"data": resourceRef{Type: "inAppPurchases", ID: iapID}}
	}
	body, err := c.doRequest(ctx, method, path, map[string]any{"data": data})
	if err != nil {
		return err
	}
	var response struct {
		Data resourceRef `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("parse IAP tax write response: %w", err)
	}
	if response.Data.Type != "inAppPurchaseTaxCategoryInfos" || strings.TrimSpace(response.Data.ID) == "" || (current.Configured && response.Data.ID != current.ID) {
		return fmt.Errorf("IAP tax write response has unexpected resource identity")
	}
	return nil
}

// DeleteIAPTaxCategory removes only the discovered override. The IAP is retained.
func (c *Client) DeleteIAPTaxCategory(ctx context.Context, iapID string, current *IAPTaxCategory) error {
	iapID = strings.TrimSpace(iapID)
	if err := validateIAPTaxCurrent(iapID, current); err != nil {
		return err
	}
	if !current.Configured {
		return nil
	}
	_, err := c.doRequest(ctx, http.MethodDelete, "/inAppPurchaseTaxCategoryInfos/"+url.PathEscape(current.ID), nil)
	return err
}
