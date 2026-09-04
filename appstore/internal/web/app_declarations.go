package web

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// AppDeclaration reports one App Store Regulations & Permits requirement that
// App Store Connect tracks for an app.
type AppDeclaration struct {
	AppID           string `json:"appId"`
	RequirementID   string `json:"requirementId"`
	RequirementName string `json:"requirementName"`
	Ref             string `json:"ref,omitempty"`
	Status          string `json:"status,omitempty"`
	FormID          string `json:"formId,omitempty"`
	DateSigned      string `json:"dateSigned,omitempty"`
	Required        bool   `json:"required"`
}

// MedicalDeviceDeclarationState reports the stored regulated medical device
// declaration for an app.
type MedicalDeviceDeclarationState struct {
	AppID              string   `json:"appId"`
	RequirementID      string   `json:"requirementId"`
	RequirementName    string   `json:"requirementName"`
	Status             string   `json:"status,omitempty"`
	FormID             string   `json:"formId,omitempty"`
	Required           bool     `json:"required"`
	Declaration        string   `json:"declaration,omitempty"`
	CountriesOrRegions []string `json:"countriesOrRegions,omitempty"`
}

// ListAppDeclarations lists the compliance requirements App Store Connect
// tracks for an app under App Information -> App Store Regulations & Permits.
func (c *Client) ListAppDeclarations(ctx context.Context, accountID, appID string) ([]AppDeclaration, error) {
	requirements, err := c.listComplianceRequirements(ctx, accountID, appID)
	if err != nil {
		return nil, err
	}

	trimmedAppID := strings.TrimSpace(appID)
	declarations := make([]AppDeclaration, 0, len(requirements))
	for _, requirement := range requirements {
		declarations = append(declarations, AppDeclaration{
			AppID:           trimmedAppID,
			RequirementID:   requirement.ID,
			RequirementName: requirement.Name,
			Ref:             requirement.Ref,
			Status:          requirement.Status,
			FormID:          requirement.FormID,
			DateSigned:      requirement.DateSigned,
			Required:        requirement.IsRequired,
		})
	}
	return declarations, nil
}

// GetMedicalDeviceDeclaration reads the stored regulated medical device
// declaration for an app.
func (c *Client) GetMedicalDeviceDeclaration(ctx context.Context, accountID, appID string) (*MedicalDeviceDeclarationState, error) {
	requirement, form, err := c.medicalDeviceRequirementAndForm(ctx, accountID, appID)
	if err != nil {
		return nil, err
	}

	return &MedicalDeviceDeclarationState{
		AppID:              strings.TrimSpace(appID),
		RequirementID:      requirement.ID,
		RequirementName:    requirement.Name,
		Status:             requirement.Status,
		FormID:             requirement.FormID,
		Required:           requirement.IsRequired,
		Declaration:        form.declaration(),
		CountriesOrRegions: form.countriesOrRegions(),
	}, nil
}

func (c *Client) medicalDeviceRequirementAndForm(ctx context.Context, accountID, appID string) (*complianceRequirement, *medicalDeviceFormResponse, error) {
	requirements, err := c.listComplianceRequirements(ctx, accountID, appID)
	if err != nil {
		return nil, nil, err
	}
	requirement := findComplianceRequirement(requirements, medicalDeviceRequirementName)
	if requirement == nil {
		return nil, nil, fmt.Errorf("regulated medical device requirement was not found for app %q", strings.TrimSpace(appID))
	}

	form, err := c.getMedicalDeviceForm(ctx, accountID, appID, requirement.ID)
	if err != nil {
		return nil, nil, err
	}
	return requirement, form, nil
}

// medicalDeviceFormData is the stored answer App Store Connect returns for the
// medical device form. Apple returns it either at the top level, under a `data`
// object, or under a single-element `data` array, so all three are accepted.
type medicalDeviceFormData struct {
	CountriesOrRegions []string `json:"countriesOrRegions"`
	MedicalDeviceData  struct {
		Declaration string `json:"declaration"`
	} `json:"medicalDeviceData"`
}

func (r *medicalDeviceFormResponse) storedData() *medicalDeviceFormData {
	if r == nil {
		return nil
	}
	if declaration := strings.TrimSpace(r.MedicalDeviceData.Declaration); declaration != "" {
		return &medicalDeviceFormData{
			CountriesOrRegions: r.CountriesOrRegions,
			MedicalDeviceData:  r.MedicalDeviceData,
		}
	}
	if len(r.Data) == 0 {
		return nil
	}

	var single medicalDeviceFormData
	if err := json.Unmarshal(r.Data, &single); err == nil {
		return &single
	}
	var list []medicalDeviceFormData
	if err := json.Unmarshal(r.Data, &list); err != nil || len(list) == 0 {
		return nil
	}
	return &list[0]
}

func (r *medicalDeviceFormResponse) declaration() string {
	data := r.storedData()
	if data == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(data.MedicalDeviceData.Declaration))
}

func (r *medicalDeviceFormResponse) countriesOrRegions() []string {
	data := r.storedData()
	if data == nil {
		return nil
	}
	regions := make([]string, 0, len(data.CountriesOrRegions))
	for _, region := range data.CountriesOrRegions {
		if normalized := normalizeMedicalDeviceRegion(region); normalized != "" {
			regions = append(regions, normalized)
		}
	}
	if len(regions) == 0 {
		return nil
	}
	slices.Sort(regions)
	return slices.Compact(regions)
}
