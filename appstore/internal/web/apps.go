package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	defaultPrimaryLocale = "en-US"
	defaultPlatform      = "IOS"
	defaultVersion       = "1.0"
)

// AppCreateAttributes defines app creation inputs for the internal web API.
type AppCreateAttributes struct {
	Name          string `json:"-"`
	SKU           string `json:"sku"`
	PrimaryLocale string `json:"primaryLocale"`
	BundleID      string `json:"bundleId"`
	CompanyName   string `json:"companyName,omitempty"`
	Platform      string `json:"-"`
	VersionString string `json:"-"`
}

// AppResponse is the app response payload from internal create/find calls.
type AppResponse struct {
	Data struct {
		ID         string         `json:"id"`
		Type       string         `json:"type"`
		Attributes map[string]any `json:"attributes"`
	} `json:"data"`
}

type relationshipData struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type appCreateRequest struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			SKU           string `json:"sku"`
			PrimaryLocale string `json:"primaryLocale"`
			BundleID      string `json:"bundleId"`
			CompanyName   string `json:"companyName,omitempty"`
		} `json:"attributes"`
		Relationships struct {
			AppStoreVersions struct {
				Data []relationshipData `json:"data"`
			} `json:"appStoreVersions"`
			AppInfos struct {
				Data []relationshipData `json:"data"`
			} `json:"appInfos"`
		} `json:"relationships"`
	} `json:"data"`
	Included []any `json:"included"`
}

type appDeleteRequest struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Removed bool `json:"removed"`
		} `json:"attributes"`
	} `json:"data"`
}

// RestoreApp marks an app as available again.
func (c *Client) RestoreApp(ctx context.Context, appID string) (*AppResponse, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("app id is required")
	}
	req := map[string]any{"data": map[string]any{"type": "apps", "id": appID, "attributes": map[string]any{"removed": false}}}
	body, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/apps/%s", url.PathEscape(appID)), req)
	if err != nil {
		return nil, err
	}
	var result AppResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse app response: %w", err)
	}
	return &result, nil
}

// SetUserAppPermission grants or revokes access for all siloable users.
func (c *Client) SetUserAppPermission(ctx context.Context, appID, access string) error {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return fmt.Errorf("app id is required")
	}
	operation := "REVOKE"
	mode := strings.ToLower(strings.TrimSpace(access))
	if mode != "limited" && mode != "full" {
		return fmt.Errorf("access must be limited or full")
	}
	if mode == "full" {
		operation = "GRANT"
	}
	req := map[string]any{"data": map[string]any{"type": "userAppPermissions", "attributes": map[string]any{"appAdamId": appID, "operationType": operation, "userOperationType": "ALL_SILOABLE_USERS"}}}
	_, err := c.doRequest(ctx, "POST", "/userAppPermissions", req)
	return err
}

func normalizeCreateAttrs(attrs AppCreateAttributes) (AppCreateAttributes, error) {
	attrs.Name = strings.TrimSpace(attrs.Name)
	attrs.SKU = strings.TrimSpace(attrs.SKU)
	attrs.PrimaryLocale = strings.TrimSpace(attrs.PrimaryLocale)
	attrs.BundleID = strings.TrimSpace(attrs.BundleID)
	attrs.CompanyName = strings.TrimSpace(attrs.CompanyName)
	attrs.Platform = strings.ToUpper(strings.TrimSpace(attrs.Platform))
	attrs.VersionString = strings.TrimSpace(attrs.VersionString)

	if attrs.Name == "" {
		return attrs, fmt.Errorf("name is required")
	}
	if attrs.BundleID == "" {
		return attrs, fmt.Errorf("bundle id is required")
	}
	if attrs.SKU == "" {
		return attrs, fmt.Errorf("sku is required")
	}
	if attrs.PrimaryLocale == "" {
		attrs.PrimaryLocale = defaultPrimaryLocale
	}
	if attrs.Platform == "" {
		attrs.Platform = defaultPlatform
	}
	if attrs.VersionString == "" {
		attrs.VersionString = defaultVersion
	}

	switch attrs.Platform {
	case "IOS", "MAC_OS", "UNIVERSAL", "TV_OS":
	default:
		return attrs, fmt.Errorf("platform must be one of IOS, MAC_OS, TV_OS, UNIVERSAL")
	}
	return attrs, nil
}

func buildAppCreateRequest(attrs AppCreateAttributes) *appCreateRequest {
	req := &appCreateRequest{}
	req.Data.Type = "apps"
	req.Data.Attributes.SKU = attrs.SKU
	req.Data.Attributes.PrimaryLocale = attrs.PrimaryLocale
	req.Data.Attributes.BundleID = attrs.BundleID
	req.Data.Attributes.CompanyName = attrs.CompanyName

	storeVersionID := "${new-appStoreVersion}"
	storeVersionLocalizationID := "${new-appStoreVersionLocalization}"
	appInfoID := "${new-appInfo}"
	appInfoLocalizationID := "${new-appInfoLocalization}"

	req.Data.Relationships.AppStoreVersions.Data = []relationshipData{
		{Type: "appStoreVersions", ID: storeVersionID},
	}
	req.Data.Relationships.AppInfos.Data = []relationshipData{
		{Type: "appInfos", ID: appInfoID},
	}

	type appStoreVersionData struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			VersionString string `json:"versionString"`
			Platform      string `json:"platform"`
		} `json:"attributes"`
		Relationships struct {
			AppStoreVersionLocalizations struct {
				Data []relationshipData `json:"data"`
			} `json:"appStoreVersionLocalizations"`
		} `json:"relationships"`
	}
	version := appStoreVersionData{Type: "appStoreVersions", ID: storeVersionID}
	version.Attributes.VersionString = attrs.VersionString
	version.Attributes.Platform = attrs.Platform
	version.Relationships.AppStoreVersionLocalizations.Data = []relationshipData{
		{Type: "appStoreVersionLocalizations", ID: storeVersionLocalizationID},
	}

	type appStoreVersionLocalizationData struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Locale string `json:"locale"`
		} `json:"attributes"`
	}
	versionLoc := appStoreVersionLocalizationData{
		Type: "appStoreVersionLocalizations",
		ID:   storeVersionLocalizationID,
	}
	versionLoc.Attributes.Locale = attrs.PrimaryLocale

	type appInfoData struct {
		Type          string `json:"type"`
		ID            string `json:"id"`
		Relationships struct {
			AppInfoLocalizations struct {
				Data []relationshipData `json:"data"`
			} `json:"appInfoLocalizations"`
		} `json:"relationships"`
	}
	info := appInfoData{Type: "appInfos", ID: appInfoID}
	info.Relationships.AppInfoLocalizations.Data = []relationshipData{
		{Type: "appInfoLocalizations", ID: appInfoLocalizationID},
	}

	type appInfoLocalizationData struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Locale string `json:"locale"`
			Name   string `json:"name"`
		} `json:"attributes"`
	}
	infoLoc := appInfoLocalizationData{
		Type: "appInfoLocalizations",
		ID:   appInfoLocalizationID,
	}
	infoLoc.Attributes.Locale = attrs.PrimaryLocale
	infoLoc.Attributes.Name = attrs.Name

	req.Included = []any{
		version,
		versionLoc,
		info,
		infoLoc,
	}
	return req
}

// CreateApp creates an app with the internal web API.
func (c *Client) CreateApp(ctx context.Context, attrs AppCreateAttributes) (*AppResponse, error) {
	normalized, err := normalizeCreateAttrs(attrs)
	if err != nil {
		return nil, err
	}
	req := buildAppCreateRequest(normalized)

	respBody, err := c.doRequest(ctx, "POST", "/apps", req)
	if err != nil {
		return nil, err
	}

	var result AppResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse app response: %w", err)
	}
	return &result, nil
}

// GetApp retrieves an app by ID using the internal web API.
func (c *Client) GetApp(ctx context.Context, appID string) (*AppResponse, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("app id is required")
	}

	respBody, err := c.doRequest(ctx, "GET", fmt.Sprintf("/apps/%s", url.PathEscape(appID)), nil)
	if err != nil {
		return nil, err
	}

	var result AppResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse app response: %w", err)
	}
	return &result, nil
}

// DeleteApp marks an app as removed with the internal App Store Connect web API.
func (c *Client) DeleteApp(ctx context.Context, appID string) (*AppResponse, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("app id is required")
	}

	req := &appDeleteRequest{}
	req.Data.Type = "apps"
	req.Data.ID = appID
	req.Data.Attributes.Removed = true

	respBody, err := c.doRequest(ctx, "PATCH", fmt.Sprintf("/apps/%s", url.PathEscape(appID)), req)
	if err != nil {
		return nil, err
	}

	var result AppResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse app response: %w", err)
	}
	return &result, nil
}

// AppRemovalState is the read model used to preflight and verify web app removal.
// Field names match the captured removed-apps listing on GET /apps.
type AppRemovalState struct {
	ID                        string
	Name                      string
	BundleID                  string
	Removed                   bool
	RemovedKnown              bool
	AppStoreLegacyStatus      string
	Marketplace               string
	VersionStates             []string
	DisplayableVersionsLoaded bool
}

// GetAppRemovalState reads the app attributes needed to check removal
// eligibility and to verify the post-PATCH removed state.
func (c *Client) GetAppRemovalState(ctx context.Context, appID string) (*AppRemovalState, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("app id is required")
	}

	values := url.Values{}
	values.Set("include", "displayableVersions")
	values.Set("fields[apps]", "name,bundleId,removed,appStoreLegacyStatus,marketplace,displayableVersions")
	values.Set("fields[appStoreVersions]", "platform,versionString,appStoreState,appVersionState")
	values.Set("limit[displayableVersions]", fmt.Sprintf("%d", removedAppsDisplayableVersionMax))
	path := queryPath("/apps/"+url.PathEscape(appID), values)

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Data     jsonAPIResource   `json:"data"`
		Included []jsonAPIResource `json:"included"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse app response: %w", err)
	}
	if strings.TrimSpace(payload.Data.ID) == "" {
		return nil, fmt.Errorf("app %q was not found", appID)
	}
	if strings.TrimSpace(payload.Data.ID) != appID {
		return nil, fmt.Errorf("app %q response returned id %q", appID, strings.TrimSpace(payload.Data.ID))
	}

	included := make(map[string]jsonAPIResource, len(payload.Included))
	for _, resource := range payload.Included {
		included[jsonAPIResourceKey(resource.Type, resource.ID)] = resource
	}
	decoded := decodeRemovedAppResource(payload.Data, included)
	removed, removedKnown := boolAttrKnown(payload.Data.Attributes, "removed")

	versionStates := make([]string, 0, len(decoded.DisplayableVersions)*2)
	versionsHaveState := true
	for _, version := range decoded.DisplayableVersions {
		storeState := strings.TrimSpace(version.AppStoreState)
		versionState := strings.TrimSpace(version.AppVersionState)
		if storeState == "" && versionState == "" {
			versionsHaveState = false
		}
		if storeState != "" {
			versionStates = append(versionStates, storeState)
		}
		if versionState != "" {
			versionStates = append(versionStates, versionState)
		}
	}

	return &AppRemovalState{
		ID:                        decoded.ID,
		Name:                      decoded.Name,
		BundleID:                  decoded.BundleID,
		Removed:                   removed,
		RemovedKnown:              removedKnown,
		AppStoreLegacyStatus:      decoded.AppStoreLegacyStatus,
		Marketplace:               decoded.Marketplace,
		VersionStates:             versionStates,
		DisplayableVersionsLoaded: displayableVersionsIncluded(payload.Data, included) && versionsHaveState,
	}, nil
}

func displayableVersionsIncluded(resource jsonAPIResource, included map[string]jsonAPIResource) bool {
	relationship, ok := resource.Relationships["displayableVersions"]
	if !ok {
		return false
	}
	trimmed := strings.TrimSpace(string(relationship.Data))
	if trimmed == "" || trimmed == "null" {
		return false
	}
	for _, ref := range parseRelationshipRefs(relationship.Data) {
		if _, found := included[jsonAPIResourceKey(ref.Type, ref.ID)]; !found {
			return false
		}
	}
	return true
}

// FindApp finds an existing app by bundle ID.
func (c *Client) FindApp(ctx context.Context, bundleID string) (*AppResponse, error) {
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return nil, fmt.Errorf("bundle id is required")
	}
	path := fmt.Sprintf("/apps?filter[bundleId]=%s", url.QueryEscape(bundleID))

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Data []struct {
			ID         string         `json:"id"`
			Type       string         `json:"type"`
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse app response: %w", err)
	}
	if len(payload.Data) == 0 {
		return nil, nil
	}

	result := &AppResponse{}
	result.Data.ID = payload.Data[0].ID
	result.Data.Type = payload.Data[0].Type
	result.Data.Attributes = payload.Data[0].Attributes
	return result, nil
}
