package asc

import (
	"fmt"
	"strings"
)

// WebAppDeclaration is one App Store Regulations & Permits requirement from
// `asc web apps declarations list`. JSON remains a bare array of these objects.
type WebAppDeclaration struct {
	AppID           string `json:"appId"`
	RequirementID   string `json:"requirementId"`
	RequirementName string `json:"requirementName"`
	Ref             string `json:"ref,omitempty"`
	Status          string `json:"status,omitempty"`
	FormID          string `json:"formId,omitempty"`
	DateSigned      string `json:"dateSigned,omitempty"`
	Required        bool   `json:"required"`
}

// WebAppDeclarationList is the computed listing for `asc web apps declarations list`.
type WebAppDeclarationList []WebAppDeclaration

// WebMedicalDeviceDeclarationState is the computed read from
// `asc web apps medical-device view`.
type WebMedicalDeviceDeclarationState struct {
	AppID              string   `json:"appId"`
	RequirementID      string   `json:"requirementId"`
	RequirementName    string   `json:"requirementName"`
	Status             string   `json:"status,omitempty"`
	FormID             string   `json:"formId,omitempty"`
	Required           bool     `json:"required"`
	Declaration        string   `json:"declaration,omitempty"`
	CountriesOrRegions []string `json:"countriesOrRegions,omitempty"`
}

// WebMedicalDeviceDeclarationResult is the mutation receipt from
// `asc web apps medical-device set`.
type WebMedicalDeviceDeclarationResult struct {
	AppID              string   `json:"appId"`
	RequirementID      string   `json:"requirementId"`
	RequirementName    string   `json:"requirementName"`
	Status             string   `json:"status,omitempty"`
	FormID             string   `json:"formId,omitempty"`
	Declared           bool     `json:"declared"`
	Changed            bool     `json:"changed"`
	CountriesOrRegions []string `json:"countriesOrRegions,omitempty"`
}

// WebMedicalDeviceRegionResult is the mutation receipt from
// `asc web apps medical-device region set`.
type WebMedicalDeviceRegionResult struct {
	AppID           string `json:"appId"`
	RequirementID   string `json:"requirementId"`
	RequirementName string `json:"requirementName"`
	Status          string `json:"status,omitempty"`
	FormID          string `json:"formId,omitempty"`
	Region          string `json:"region"`
	Declared        bool   `json:"declared"`
	Changed         bool   `json:"changed"`
}

func webAppDeclarationListRows(result *WebAppDeclarationList) ([]string, [][]string) {
	headers := []string{"Requirement", "Status", "Required", "Requirement ID", "Form ID"}
	if result == nil {
		return headers, nil
	}
	rows := make([][]string, 0, len(*result))
	for _, declaration := range *result {
		rows = append(rows, []string{
			declaration.RequirementName,
			webDeclarationText(declaration.Status),
			fmt.Sprintf("%t", declaration.Required),
			webDeclarationText(declaration.RequirementID),
			webDeclarationText(declaration.FormID),
		})
	}
	return headers, rows
}

func webMedicalDeviceDeclarationStateRows(result *WebMedicalDeviceDeclarationState) ([]string, [][]string) {
	headers := []string{"App ID", "Requirement", "Declaration", "Status", "Required", "Countries/Regions"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.AppID,
		result.RequirementName,
		webDeclarationText(result.Declaration),
		webDeclarationText(result.Status),
		fmt.Sprintf("%t", result.Required),
		webDeclarationText(strings.Join(result.CountriesOrRegions, ",")),
	}}
}

func webMedicalDeviceDeclarationResultRows(result *WebMedicalDeviceDeclarationResult) ([]string, [][]string) {
	headers := []string{"App ID", "Requirement", "Declared", "Changed", "Status", "Countries/Regions"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.AppID,
		result.RequirementName,
		fmt.Sprintf("%t", result.Declared),
		fmt.Sprintf("%t", result.Changed),
		webDeclarationText(result.Status),
		webDeclarationText(strings.Join(result.CountriesOrRegions, ",")),
	}}
}

func webMedicalDeviceRegionResultRows(result *WebMedicalDeviceRegionResult) ([]string, [][]string) {
	headers := []string{"App ID", "Requirement", "Region", "Declared", "Changed", "Status"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.AppID,
		result.RequirementName,
		result.Region,
		fmt.Sprintf("%t", result.Declared),
		fmt.Sprintf("%t", result.Changed),
		webDeclarationText(result.Status),
	}}
}

func webDeclarationText(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "n/a"
	}
	return trimmed
}
