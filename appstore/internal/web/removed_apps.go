package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	// DefaultRemovedAppsLimit is the default page size for removed-apps listing.
	DefaultRemovedAppsLimit = 48
	// MaxRemovedAppsLimit is the largest accepted page size for removed-apps listing.
	MaxRemovedAppsLimit              = 200
	removedAppsDisplayableVersionMax = 20
)

// RemovedAppsListOptions controls the removed-apps IRIS listing.
type RemovedAppsListOptions struct {
	Limit    int
	Next     string
	Paginate bool
}

// RemovedAppsListResponse is the normalized output for removed apps.
type RemovedAppsListResponse struct {
	Data  []RemovedApp      `json:"data"`
	Links *RemovedAppsLinks `json:"links,omitempty"`
}

// RemovedAppsLinks contains pagination links returned by IRIS.
type RemovedAppsLinks struct {
	Self string `json:"self,omitempty"`
	Next string `json:"next,omitempty"`
}

// RemovedApp summarizes one app from App Store Connect's Removed Apps view.
type RemovedApp struct {
	ID                   string              `json:"id"`
	Type                 string              `json:"type,omitempty"`
	Name                 string              `json:"name,omitempty"`
	BundleID             string              `json:"bundleId,omitempty"`
	SKU                  string              `json:"sku,omitempty"`
	PrimaryLocale        string              `json:"primaryLocale,omitempty"`
	Removed              bool                `json:"removed"`
	Status               string              `json:"status,omitempty"`
	AppStoreLegacyStatus string              `json:"appStoreLegacyStatus,omitempty"`
	Marketplace          string              `json:"marketplace,omitempty"`
	VersionSummary       string              `json:"versionSummary,omitempty"`
	DisplayableVersions  []RemovedAppVersion `json:"displayableVersions,omitempty"`
}

// RemovedAppVersion summarizes one displayable version attached to a removed app.
type RemovedAppVersion struct {
	ID              string `json:"id"`
	Type            string `json:"type,omitempty"`
	Platform        string `json:"platform,omitempty"`
	VersionString   string `json:"versionString,omitempty"`
	AppStoreState   string `json:"appStoreState,omitempty"`
	AppVersionState string `json:"appVersionState,omitempty"`
	CreatedDate     string `json:"createdDate,omitempty"`
	IsWatchOnly     bool   `json:"isWatchOnly,omitempty"`
}

// ListRemovedApps lists apps from App Store Connect's Removed Apps web view.
func (c *Client) ListRemovedApps(ctx context.Context, opts RemovedAppsListOptions) (*RemovedAppsListResponse, error) {
	path, err := c.removedAppsListPath(opts)
	if err != nil {
		return nil, err
	}

	result := &RemovedAppsListResponse{Data: []RemovedApp{}}
	visited := map[string]struct{}{}
	for strings.TrimSpace(path) != "" {
		if _, seen := visited[path]; seen {
			return nil, fmt.Errorf("removed apps pagination loop detected")
		}
		visited[path] = struct{}{}

		page, err := c.fetchRemovedAppsPage(ctx, path)
		if err != nil {
			return nil, err
		}
		result.Data = append(result.Data, page.Data...)
		result.Links = page.Links
		if !opts.Paginate {
			break
		}

		nextPath, err := nextLookupPagePath(page.rawLinks, c.baseURL, "removed apps")
		if err != nil {
			return nil, err
		}
		if err := validateRemovedAppsNextPath(nextPath); err != nil {
			return nil, err
		}
		path = nextPath
	}

	return result, nil
}

func (c *Client) removedAppsListPath(opts RemovedAppsListOptions) (string, error) {
	next := strings.TrimSpace(opts.Next)
	if next != "" {
		return removedAppsNextPath(next, c.baseURL)
	}

	limit := opts.Limit
	if limit == 0 {
		limit = DefaultRemovedAppsLimit
	}
	if limit < 1 || limit > MaxRemovedAppsLimit {
		return "", fmt.Errorf("limit must be between 1 and %d", MaxRemovedAppsLimit)
	}

	values := url.Values{}
	values.Set("include", "appStoreIcon,displayableVersions")
	values.Set("limit", strconv.Itoa(limit))
	values.Set("filter[removed]", "true")
	values.Set("fields[apps]", "name,bundleId,primaryLocale,sku,removed,appStoreLegacyStatus,marketplace,appStoreIcon,displayableVersions")
	values.Set("fields[appStoreVersions]", "platform,versionString,appStoreState,storeIcon,watchStoreIcon,isWatchOnly,createdDate,appVersionState")
	values.Set("limit[displayableVersions]", strconv.Itoa(removedAppsDisplayableVersionMax))
	return "/apps?" + values.Encode(), nil
}

func removedAppsNextPath(next, baseURL string) (string, error) {
	path, err := normalizeNextPath(next, baseURL)
	if err != nil {
		return "", err
	}
	if err := validateRemovedAppsNextPath(path); err != nil {
		return "", err
	}
	return path, nil
}

func validateRemovedAppsNextPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	parsed, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("invalid removed apps next link: %w", err)
	}
	if parsed.EscapedPath() != "/apps" {
		return fmt.Errorf("removed apps next link must point to /apps")
	}
	if parsed.Query().Get("filter[removed]") != "true" {
		return fmt.Errorf("removed apps next link must include filter[removed]=true")
	}
	return nil
}

type removedAppsPage struct {
	Data     []RemovedApp
	Links    *RemovedAppsLinks
	rawLinks map[string]any
}

func (c *Client) fetchRemovedAppsPage(ctx context.Context, path string) (removedAppsPage, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return removedAppsPage{}, err
	}

	var payload jsonAPIListPayload
	if err := json.Unmarshal(respBody, &payload); err != nil {
		return removedAppsPage{}, fmt.Errorf("failed to parse removed apps response: %w", err)
	}

	included := make(map[string]jsonAPIResource, len(payload.Included))
	for _, resource := range payload.Included {
		included[jsonAPIResourceKey(resource.Type, resource.ID)] = resource
	}

	apps := make([]RemovedApp, 0, len(payload.Data))
	for _, resource := range payload.Data {
		apps = append(apps, decodeRemovedAppResource(resource, included))
	}

	return removedAppsPage{
		Data:     apps,
		Links:    decodeRemovedAppsLinks(payload.Links),
		rawLinks: payload.Links,
	}, nil
}

func decodeRemovedAppResource(resource jsonAPIResource, included map[string]jsonAPIResource) RemovedApp {
	app := RemovedApp{
		ID:                   strings.TrimSpace(resource.ID),
		Type:                 strings.TrimSpace(resource.Type),
		Name:                 stringAttr(resource.Attributes, "name"),
		BundleID:             stringAttr(resource.Attributes, "bundleId"),
		SKU:                  stringAttr(resource.Attributes, "sku"),
		PrimaryLocale:        stringAttr(resource.Attributes, "primaryLocale"),
		Removed:              boolAttr(resource.Attributes, "removed"),
		AppStoreLegacyStatus: stringAttr(resource.Attributes, "appStoreLegacyStatus"),
		Marketplace:          stringAttr(resource.Attributes, "marketplace"),
	}

	for _, ref := range relationshipRefs(resource, "displayableVersions") {
		versionResource, ok := included[jsonAPIResourceKey(ref.Type, ref.ID)]
		if !ok {
			continue
		}
		app.DisplayableVersions = append(app.DisplayableVersions, decodeRemovedAppVersionResource(versionResource))
	}
	app.VersionSummary = removedAppVersionSummary(app.DisplayableVersions)
	app.Status = removedAppStatus(app)
	return app
}

func decodeRemovedAppVersionResource(resource jsonAPIResource) RemovedAppVersion {
	return RemovedAppVersion{
		ID:              strings.TrimSpace(resource.ID),
		Type:            strings.TrimSpace(resource.Type),
		Platform:        stringAttr(resource.Attributes, "platform"),
		VersionString:   stringAttr(resource.Attributes, "versionString"),
		AppStoreState:   stringAttr(resource.Attributes, "appStoreState"),
		AppVersionState: stringAttr(resource.Attributes, "appVersionState"),
		CreatedDate:     stringAttr(resource.Attributes, "createdDate"),
		IsWatchOnly:     boolAttr(resource.Attributes, "isWatchOnly"),
	}
}

func decodeRemovedAppsLinks(raw map[string]any) *RemovedAppsLinks {
	self := extractLinkValue(raw, "self")
	next, _ := extractNextLink(raw)
	if self == "" && next == "" {
		return nil
	}
	return &RemovedAppsLinks{Self: self, Next: next}
}

func extractLinkValue(links map[string]any, key string) string {
	if len(links) == 0 {
		return ""
	}
	raw, ok := links[key]
	if !ok || raw == nil {
		return ""
	}
	switch value := raw.(type) {
	case string:
		return strings.TrimSpace(value)
	case map[string]any:
		if href, ok := value["href"].(string); ok {
			return strings.TrimSpace(href)
		}
		if urlValue, ok := value["url"].(string); ok {
			return strings.TrimSpace(urlValue)
		}
	}
	return ""
}

func removedAppVersionSummary(versions []RemovedAppVersion) string {
	if len(versions) == 0 {
		return ""
	}
	first := versions[0]
	parts := []string{}
	if platform := displayPlatform(first.Platform); platform != "" {
		parts = append(parts, platform)
	}
	if version := strings.TrimSpace(first.VersionString); version != "" {
		parts = append(parts, version)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	if state := strings.TrimSpace(first.AppStoreState); state != "" {
		return state
	}
	return strings.TrimSpace(first.AppVersionState)
}

func removedAppStatus(app RemovedApp) string {
	if len(app.DisplayableVersions) > 0 {
		first := app.DisplayableVersions[0]
		if state := strings.TrimSpace(first.AppStoreState); state != "" {
			return state
		}
		if state := strings.TrimSpace(first.AppVersionState); state != "" {
			return state
		}
	}
	return strings.TrimSpace(app.AppStoreLegacyStatus)
}

func displayPlatform(platform string) string {
	switch strings.ToUpper(strings.TrimSpace(platform)) {
	case "IOS":
		return "iOS"
	case "MAC_OS":
		return "macOS"
	case "TV_OS":
		return "tvOS"
	case "VISION_OS":
		return "visionOS"
	default:
		return strings.TrimSpace(platform)
	}
}
