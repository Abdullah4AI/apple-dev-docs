package asc

import "fmt"

// WebIAPTaxCategorySetResult is the receipt for a verified In-App Purchase
// tax-category selection made through an authenticated web session.
type WebIAPTaxCategorySetResult struct {
	IAPID        string   `json:"iapId"`
	CategoryID   string   `json:"categoryId"`
	CategoryName string   `json:"categoryName,omitempty"`
	ConditionIDs []string `json:"conditionIds"`
	Changed      bool     `json:"changed"`
	Verified     bool     `json:"verified"`
}

func webIAPTaxCategorySetResultRows(result *WebIAPTaxCategorySetResult) ([]string, [][]string) {
	if result == nil {
		return []string{"Field", "Value"}, nil
	}
	return []string{"Field", "Value"}, [][]string{
		{"IAP ID", result.IAPID},
		{"Category ID", result.CategoryID},
		{"Category Name", result.CategoryName},
		{"Condition IDs", joinOutputStrings(result.ConditionIDs)},
		{"Changed", fmt.Sprintf("%t", result.Changed)},
		{"Verified", fmt.Sprintf("%t", result.Verified)},
	}
}

// WebIAPTaxCategoryResetResult is the receipt for a verified reset to the
// inherited In-App Purchase tax-category state.
type WebIAPTaxCategoryResetResult struct {
	IAPID    string `json:"iapId"`
	Changed  bool   `json:"changed"`
	Verified bool   `json:"verified"`
}

func webIAPTaxCategoryResetResultRows(result *WebIAPTaxCategoryResetResult) ([]string, [][]string) {
	if result == nil {
		return []string{"Field", "Value"}, nil
	}
	return []string{"Field", "Value"}, [][]string{
		{"IAP ID", result.IAPID},
		{"Changed", fmt.Sprintf("%t", result.Changed)},
		{"Verified", fmt.Sprintf("%t", result.Verified)},
	}
}
