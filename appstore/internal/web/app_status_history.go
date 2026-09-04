package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// appStatusHistoryVersionFields limits the app version list to the identity
// fields the status history view needs.
const appStatusHistoryVersionFields = "versionString,platform,appStoreState,appVersionState,createdDate"

// appStatusHistoryVersionSummaryFields keeps the identity attributes and the
// app relationship so --version-id can verify ownership. Apple's sparse
// fieldset omits relationships that are not listed.
const appStatusHistoryVersionSummaryFields = appStatusHistoryVersionFields + ",app"

// AppStatusHistory groups App Store version status changes for one app.
type AppStatusHistory struct {
	AppID    string                    `json:"appId"`
	Versions []AppStatusHistoryVersion `json:"versions"`
}

// AppStatusHistoryVersion holds the status changes recorded for one app version.
type AppStatusHistoryVersion struct {
	VersionID     string            `json:"versionId"`
	VersionString string            `json:"versionString,omitempty"`
	Platform      string            `json:"platform,omitempty"`
	CreatedDate   string            `json:"createdDate,omitempty"`
	Changes       []AppStatusChange `json:"changes"`
}

// AppStatusChange is one recorded App Store version status transition.
type AppStatusChange struct {
	ID              string `json:"id"`
	AppStoreState   string `json:"appStoreState,omitempty"`
	AppVersionState string `json:"appVersionState,omitempty"`
	Date            string `json:"date,omitempty"`
	Initiator       string `json:"initiator,omitempty"`
}

// GetAppStatusHistory reads App Store version status changes for an app.
//
// App Store Connect records status changes per app store version, and exposes
// no app-level history resource, so this lists the app's versions and then
// reads each version's state changes. A non-empty versionID skips the version
// list, verifies that version belongs to appID, and reads that single version.
func (c *Client) GetAppStatusHistory(ctx context.Context, appID, versionID string) (*AppStatusHistory, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("app id is required")
	}
	versionID = strings.TrimSpace(versionID)

	result := &AppStatusHistory{AppID: appID, Versions: make([]AppStatusHistoryVersion, 0)}

	versions, err := c.appStatusHistoryVersions(ctx, appID, versionID)
	if err != nil {
		return nil, err
	}

	for _, version := range versions {
		changes, err := c.appStoreVersionStateChanges(ctx, version.VersionID)
		if err != nil {
			return nil, err
		}
		version.Changes = changes
		result.Versions = append(result.Versions, version)
	}

	return result, nil
}

func (c *Client) appStatusHistoryVersions(ctx context.Context, appID, versionID string) ([]AppStatusHistoryVersion, error) {
	if versionID != "" {
		version, err := c.appStoreVersionSummary(ctx, appID, versionID)
		if err != nil {
			return nil, err
		}
		return []AppStatusHistoryVersion{version}, nil
	}

	query := url.Values{}
	query.Set("fields[appStoreVersions]", appStatusHistoryVersionFields)
	path := "/apps/" + url.PathEscape(appID) + "/appStoreVersions?" + query.Encode()

	payload, err := c.fetchJSONAPIPages(ctx, path, "app status history versions")
	if err != nil {
		return nil, err
	}

	versions := make([]AppStatusHistoryVersion, 0, len(payload.Data))
	for _, resource := range payload.Data {
		versions = append(versions, appStatusHistoryVersionFromResource(resource))
	}
	return versions, nil
}

func (c *Client) appStoreVersionSummary(ctx context.Context, appID, versionID string) (AppStatusHistoryVersion, error) {
	query := url.Values{}
	query.Set("fields[appStoreVersions]", appStatusHistoryVersionSummaryFields)
	query.Set("include", "app")
	path := "/appStoreVersions/" + url.PathEscape(versionID) + "?" + query.Encode()

	responseBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return AppStatusHistoryVersion{}, err
	}

	var payload struct {
		Data jsonAPIResource `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return AppStatusHistoryVersion{}, fmt.Errorf("failed to parse app store version response: %w", err)
	}

	if err := verifyAppStoreVersionAppID(payload.Data, appID, versionID); err != nil {
		return AppStatusHistoryVersion{}, err
	}

	version := appStatusHistoryVersionFromResource(payload.Data)
	if version.VersionID == "" {
		version.VersionID = versionID
	}
	return version, nil
}

func verifyAppStoreVersionAppID(resource jsonAPIResource, appID, versionID string) error {
	ref := firstRelationshipRef(resource, "app")
	if ref == nil || strings.TrimSpace(ref.ID) == "" {
		return fmt.Errorf("app relationship missing for app store version %q", versionID)
	}
	relatedAppID := strings.TrimSpace(ref.ID)
	if !strings.EqualFold(relatedAppID, strings.TrimSpace(appID)) {
		return fmt.Errorf("version %q belongs to app %q, not %q", versionID, relatedAppID, appID)
	}
	return nil
}

func appStatusHistoryVersionFromResource(resource jsonAPIResource) AppStatusHistoryVersion {
	return AppStatusHistoryVersion{
		VersionID:     strings.TrimSpace(resource.ID),
		VersionString: stringAttr(resource.Attributes, "versionString"),
		Platform:      stringAttr(resource.Attributes, "platform"),
		CreatedDate:   stringAttr(resource.Attributes, "createdDate"),
		Changes:       make([]AppStatusChange, 0),
	}
}

func (c *Client) appStoreVersionStateChanges(ctx context.Context, versionID string) ([]AppStatusChange, error) {
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return nil, fmt.Errorf("app store version id is required")
	}

	path := "/appStoreVersions/" + url.PathEscape(versionID) + "/appStoreVersionStateChanges"
	payload, err := c.fetchJSONAPIPages(ctx, path, "app store version state changes")
	if err != nil {
		return nil, err
	}

	changes := make([]AppStatusChange, 0, len(payload.Data))
	for _, resource := range payload.Data {
		changes = append(changes, AppStatusChange{
			ID:              strings.TrimSpace(resource.ID),
			AppStoreState:   stringAttr(resource.Attributes, "appStoreState"),
			AppVersionState: stringAttr(resource.Attributes, "appVersionState"),
			Date:            stringAttr(resource.Attributes, "date"),
			Initiator:       stringAttr(resource.Attributes, "initiator"),
		})
	}
	return changes, nil
}
