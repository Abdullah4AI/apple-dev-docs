package asc

import "fmt"

// WebAppTaxCategoryViewResult is the computed read result for an app's
// explicit App Information tax category and its effective UI category.
type WebAppTaxCategoryViewResult struct {
	AppID                 string   `json:"appId"`
	Configured            bool     `json:"configured"`
	CategoryID            string   `json:"categoryId,omitempty"`
	CategoryName          string   `json:"categoryName,omitempty"`
	EffectiveCategoryID   string   `json:"effectiveCategoryId,omitempty"`
	EffectiveCategoryName string   `json:"effectiveCategoryName"`
	EnabledConditionIDs   []string `json:"enabledConditionIds"`
}

func webAppTaxCategoryViewResultRows(result *WebAppTaxCategoryViewResult) ([]string, [][]string) {
	return []string{"Field", "Value"}, [][]string{
		{"App ID", result.AppID},
		{"Configured", fmt.Sprintf("%t", result.Configured)},
		{"Category ID", result.CategoryID},
		{"Category Name", result.CategoryName},
		{"Effective Category ID", result.EffectiveCategoryID},
		{"Effective Category Name", result.EffectiveCategoryName},
		{"Enabled Condition IDs", joinOutputStrings(result.EnabledConditionIDs)},
	}
}

// WebAppTaxCategorySetResult is the receipt for a verified App Information
// tax-category selection made through an authenticated web session.
type WebAppTaxCategorySetResult struct {
	AppID        string   `json:"appId"`
	CategoryID   string   `json:"categoryId"`
	CategoryName string   `json:"categoryName,omitempty"`
	ConditionIDs []string `json:"conditionIds"`
	Changed      bool     `json:"changed"`
	Verified     bool     `json:"verified"`
}

func webAppTaxCategorySetResultRows(result *WebAppTaxCategorySetResult) ([]string, [][]string) {
	return []string{"Field", "Value"}, [][]string{
		{"App ID", result.AppID},
		{"Category ID", result.CategoryID},
		{"Category Name", result.CategoryName},
		{"Condition IDs", joinOutputStrings(result.ConditionIDs)},
		{"Changed", fmt.Sprintf("%t", result.Changed)},
		{"Verified", fmt.Sprintf("%t", result.Verified)},
	}
}

func joinOutputStrings(values []string) string {
	if len(values) == 0 {
		return ""
	}
	result := values[0]
	for _, value := range values[1:] {
		result += "," + value
	}
	return result
}
