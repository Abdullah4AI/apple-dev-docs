package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// BundleIDCapabilityOption describes one option inside a capability setting.
type BundleIDCapabilityOption struct {
	Key              string `json:"key"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	Enabled          *bool  `json:"enabled,omitempty"`
	EnabledByDefault *bool  `json:"enabledByDefault,omitempty"`
	SupportsWildcard *bool  `json:"supportsWildcard,omitempty"`
}

// BundleIDCapabilitySetting describes an App Store Connect bundle ID capability setting.
type BundleIDCapabilitySetting struct {
	Key              string                     `json:"key"`
	Name             string                     `json:"name,omitempty"`
	Description      string                     `json:"description,omitempty"`
	EnabledByDefault *bool                      `json:"enabledByDefault,omitempty"`
	Visible          *bool                      `json:"visible,omitempty"`
	AllowedInstances string                     `json:"allowedInstances,omitempty"`
	MinInstances     *int                       `json:"minInstances,omitempty"`
	Options          []BundleIDCapabilityOption `json:"options,omitempty"`
}

// AppClipBundleIDCapabilitySyncRequest updates an App Clip Bundle ID capability set
// through Apple's web-session bundleIds patch payload.
type AppClipBundleIDCapabilitySyncRequest struct {
	BundleID         string
	ParentBundleID   string
	Capability       string
	Enabled          bool
	Settings         []BundleIDCapabilitySetting
	SettingsProvided bool
}

// AppClipBundleIDCapabilitySyncResult summarizes the private capability sync.
// Changed is false when the requested parent relationship, enabled state, and
// any explicitly provided settings were already in place and no PATCH was sent.
type AppClipBundleIDCapabilitySyncResult struct {
	BundleID       string `json:"bundleId"`
	ParentBundleID string `json:"parentBundleId"`
	Capability     string `json:"capability"`
	Enabled        bool   `json:"enabled"`
	Changed        bool   `json:"changed"`
	Status         string `json:"status"`
}

type webBundleIDResponse struct {
	Data struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			Name       string `json:"name"`
			Identifier string `json:"identifier"`
			SeedID     string `json:"seedId,omitempty"`
		} `json:"attributes"`
		Relationships struct {
			BundleIDCapabilities struct {
				Data []webBundleIDCapabilityRelationship `json:"data"`
			} `json:"bundleIdCapabilities"`
		} `json:"relationships"`
	} `json:"data"`
	Included []webBundleIDCapabilityRelationship `json:"included,omitempty"`
}

type webBundleIDPatchRequest struct {
	Data struct {
		ID            string `json:"id"`
		Type          string `json:"type"`
		Attributes    any    `json:"attributes"`
		Relationships struct {
			BundleIDCapabilities struct {
				Data []webBundleIDCapabilityRelationship `json:"data"`
			} `json:"bundleIdCapabilities"`
		} `json:"relationships"`
	} `json:"data"`
}

// webBundleIDCapabilityRelationship is one bundleIdCapabilities resource as
// Apple returns it. Relationships stay raw so to-one links such as
// parentBundleId and to-many links such as appGroups or cloudContainers are
// preserved byte-for-byte when the graph is written back.
type webBundleIDCapabilityRelationship struct {
	ID            string                     `json:"id,omitempty"`
	Type          string                     `json:"type"`
	Attributes    map[string]any             `json:"attributes,omitempty"`
	Relationships map[string]json.RawMessage `json:"relationships,omitempty"`
}

type webBundleIDRelationshipData struct {
	Data relationshipData `json:"data"`
}

// writableBundleIDCapabilityAttributes lists the capability attributes Apple's
// web-session PATCH accepts. Everything else in a read response is metadata.
var writableBundleIDCapabilityAttributes = []string{"enabled", "settings"}

func (r webBundleIDCapabilityRelationship) toOneRelationshipID(name string) string {
	raw, ok := r.Relationships[name]
	if !ok || len(raw) == 0 {
		return ""
	}
	var relationship webBundleIDRelationshipData
	if err := json.Unmarshal(raw, &relationship); err != nil {
		return ""
	}
	return strings.TrimSpace(relationship.Data.ID)
}

func (r webBundleIDCapabilityRelationship) capabilityID() string {
	return strings.ToUpper(r.toOneRelationshipID("capability"))
}

func (r webBundleIDCapabilityRelationship) parentBundleID() string {
	return r.toOneRelationshipID("parentBundleId")
}

func (r webBundleIDCapabilityRelationship) enabled() bool {
	enabled, ok := r.Attributes["enabled"].(bool)
	return ok && enabled
}

func (r webBundleIDCapabilityRelationship) settings() (any, bool) {
	settings, ok := r.Attributes["settings"]
	return settings, ok
}

func encodeToOneRelationship(resourceType, id string) json.RawMessage {
	encoded, _ := json.Marshal(webBundleIDRelationshipData{Data: relationshipData{Type: resourceType, ID: id}})
	return encoded
}

func normalizeAppClipBundleIDCapabilitySyncRequest(req AppClipBundleIDCapabilitySyncRequest) (AppClipBundleIDCapabilitySyncRequest, error) {
	req.BundleID = strings.TrimSpace(req.BundleID)
	req.ParentBundleID = strings.TrimSpace(req.ParentBundleID)
	req.Capability = strings.ToUpper(strings.TrimSpace(req.Capability))
	if req.BundleID == "" {
		return req, fmt.Errorf("bundle id is required")
	}
	if req.ParentBundleID == "" {
		return req, fmt.Errorf("parent bundle id is required")
	}
	if req.Capability == "" {
		return req, fmt.Errorf("capability is required")
	}
	return req, nil
}

// SyncAppClipBundleIDCapability patches a bundle ID capability relationship with
// the parentBundleId relationship required by App Clip targets. It reads the
// current capability graph first and skips the PATCH when the requested state
// is already in place, because every Bundle ID write invalidates the
// provisioning profiles that contain it.
func (c *Client) SyncAppClipBundleIDCapability(ctx context.Context, req AppClipBundleIDCapabilitySyncRequest) (*AppClipBundleIDCapabilitySyncResult, error) {
	req, err := normalizeAppClipBundleIDCapabilitySyncRequest(req)
	if err != nil {
		return nil, err
	}

	body, err := c.doIrisV1Request(ctx, http.MethodGet, fmt.Sprintf("/bundleIds/%s?include=bundleIdCapabilities", req.BundleID), nil)
	if err != nil {
		return nil, err
	}
	var current webBundleIDResponse
	if err := json.Unmarshal(body, &current); err != nil {
		return nil, fmt.Errorf("failed to parse bundle id response: %w", err)
	}

	result := &AppClipBundleIDCapabilitySyncResult{
		BundleID:       req.BundleID,
		ParentBundleID: req.ParentBundleID,
		Capability:     req.Capability,
		Enabled:        req.Enabled,
	}
	existing := currentBundleIDCapabilities(current)
	if appClipBundleIDCapabilityAlreadySynced(existing, req) {
		result.Changed = false
		result.Status = "already-synced"
		return result, nil
	}

	payload := buildAppClipBundleIDCapabilityPatchRequest(current, existing, req)
	if _, err := c.doIrisV1Request(ctx, http.MethodPatch, fmt.Sprintf("/bundleIds/%s", req.BundleID), payload); err != nil {
		return nil, err
	}
	result.Changed = true
	result.Status = "synced"
	return result, nil
}

// appClipBundleIDCapabilityAlreadySynced reports whether the current graph
// already carries the requested capability with the requested enabled state and
// parentBundleId. When the caller supplied settings explicitly they must match
// too; otherwise the existing settings are preserved and cannot differ. Apple
// can return duplicate records for one capability, so every record is checked
// before concluding that a write is needed.
func appClipBundleIDCapabilityAlreadySynced(existing []webBundleIDCapabilityRelationship, req AppClipBundleIDCapabilitySyncRequest) bool {
	for _, capability := range existing {
		if capability.capabilityID() == req.Capability && appClipBundleIDCapabilityMatches(capability, req) {
			return true
		}
	}
	return false
}

func appClipBundleIDCapabilityMatches(capability webBundleIDCapabilityRelationship, req AppClipBundleIDCapabilitySyncRequest) bool {
	if capability.enabled() != req.Enabled || capability.parentBundleID() != req.ParentBundleID {
		return false
	}
	if !req.SettingsProvided {
		return true
	}
	currentSettings, ok := capability.settings()
	return ok && requestedSettingsApplied(req.Settings, currentSettings)
}

// requestedSettingsApplied reports whether every caller-controlled settings
// field is already present with the same value in Apple's current settings.
// Apple enriches settings with read-only metadata such as name, description,
// visible, and enabledByDefault, so the current value is projected onto the
// keys the caller actually sent instead of being compared as a whole.
func requestedSettingsApplied(requested, current any) bool {
	requestedValue, ok := normalizeJSONValue(requested)
	if !ok {
		return false
	}
	currentValue, ok := normalizeJSONValue(current)
	if !ok {
		return false
	}
	return jsonSubset(requestedValue, currentValue)
}

func normalizeJSONValue(value any) (any, bool) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, false
	}
	return decoded, true
}

// jsonSubset reports whether requested is structurally contained in current.
// Objects may carry extra keys on the current side. Keyed arrays (every
// element is an object with a string "key") are matched entry by entry on that
// key: requested keys must be unique, each requested entry must be contained in
// the current entry with the same key, and Apple may list extra entries such as
// the options the caller did not name. Unkeyed arrays must match in length and
// element order.
func jsonSubset(requested, current any) bool {
	switch requestedValue := requested.(type) {
	case map[string]any:
		currentObject, ok := current.(map[string]any)
		if !ok {
			return false
		}
		for key, value := range requestedValue {
			currentField, ok := currentObject[key]
			if !ok || !jsonSubset(value, currentField) {
				return false
			}
		}
		return true
	case []any:
		currentArray, ok := current.([]any)
		if !ok {
			return false
		}
		if len(requestedValue) == 0 {
			return len(currentArray) == 0
		}
		if jsonArrayIsKeyed(requestedValue) {
			requestedByKey, ok := jsonObjectsByKey(requestedValue)
			if !ok {
				return false
			}
			currentByKey, ok := jsonObjectsByKey(currentArray)
			if !ok {
				return false
			}
			for key, object := range requestedByKey {
				match, ok := currentByKey[key]
				if !ok || !jsonSubset(object, match) {
					return false
				}
			}
			return true
		}
		if len(currentArray) != len(requestedValue) {
			return false
		}
		for index, element := range requestedValue {
			if !jsonSubset(element, currentArray[index]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(requested, current)
	}
}

func jsonArrayIsKeyed(values []any) bool {
	for _, value := range values {
		object, ok := value.(map[string]any)
		if !ok {
			return false
		}
		if _, ok := object["key"].(string); !ok {
			return false
		}
	}
	return true
}

// jsonObjectsByKey indexes keyed objects by their "key" field. It reports false
// when an element is not a keyed object or when a key repeats, so duplicate
// keys on either side are never reported as already applied.
func jsonObjectsByKey(values []any) (map[string]map[string]any, bool) {
	if !jsonArrayIsKeyed(values) {
		return nil, false
	}
	byKey := make(map[string]map[string]any, len(values))
	for _, value := range values {
		object := value.(map[string]any)
		key := object["key"].(string)
		if _, duplicate := byKey[key]; duplicate {
			return nil, false
		}
		byKey[key] = object
	}
	return byKey, true
}

func buildAppClipBundleIDCapabilityPatchRequest(current webBundleIDResponse, existing []webBundleIDCapabilityRelationship, req AppClipBundleIDCapabilitySyncRequest) webBundleIDPatchRequest {
	var payload webBundleIDPatchRequest
	payload.Data.ID = req.BundleID
	payload.Data.Type = "bundleIds"
	payload.Data.Attributes = struct {
		Name       string `json:"name"`
		Identifier string `json:"identifier"`
		SeedID     string `json:"seedId,omitempty"`
	}{
		Name:       current.Data.Attributes.Name,
		Identifier: current.Data.Attributes.Identifier,
		SeedID:     current.Data.Attributes.SeedID,
	}

	capability := webBundleIDCapabilityRelationship{
		Type: "bundleIdCapabilities",
		Attributes: map[string]any{
			"enabled":  req.Enabled,
			"settings": req.Settings,
		},
		Relationships: map[string]json.RawMessage{
			"capability":     encodeToOneRelationship("capabilities", req.Capability),
			"parentBundleId": encodeToOneRelationship("bundleIds", req.ParentBundleID),
		},
	}
	payload.Data.Relationships.BundleIDCapabilities.Data = appendPreservedBundleIDCapabilities(existing, capability, req.SettingsProvided)
	return payload
}

func currentBundleIDCapabilities(current webBundleIDResponse) []webBundleIDCapabilityRelationship {
	capabilities := make([]webBundleIDCapabilityRelationship, 0, len(current.Included)+len(current.Data.Relationships.BundleIDCapabilities.Data))
	seen := make(map[string]struct{})
	for _, capability := range current.Included {
		if capability.Type != "bundleIdCapabilities" {
			continue
		}
		capabilities = append(capabilities, capability)
		if capability.ID != "" {
			seen[capability.ID] = struct{}{}
		}
	}
	for _, capability := range current.Data.Relationships.BundleIDCapabilities.Data {
		if capability.Type != "bundleIdCapabilities" {
			continue
		}
		if capability.ID != "" {
			if _, ok := seen[capability.ID]; ok {
				continue
			}
			seen[capability.ID] = struct{}{}
		}
		capabilities = append(capabilities, capability)
	}
	return capabilities
}

// appendPreservedBundleIDCapabilities rebuilds the capability list for the
// PATCH: unrelated capabilities keep their resource IDs, relationships, and
// writable attributes; the synced capability replaces any existing entry for
// the same capability while inheriting its ID, other relationships, and (unless
// settings were provided explicitly) its current settings.
func appendPreservedBundleIDCapabilities(existing []webBundleIDCapabilityRelationship, synced webBundleIDCapabilityRelationship, settingsProvided bool) []webBundleIDCapabilityRelationship {
	capabilityID := synced.capabilityID()
	capabilities := make([]webBundleIDCapabilityRelationship, 0, len(existing)+1)
	for _, capability := range existing {
		if capability.capabilityID() == capabilityID {
			if synced.ID == "" {
				synced.ID = capability.ID
			}
			if !settingsProvided {
				if existingSettings, ok := capability.settings(); ok {
					synced.Attributes["settings"] = existingSettings
				}
			}
			for name, raw := range capability.Relationships {
				if _, overridden := synced.Relationships[name]; !overridden {
					synced.Relationships[name] = raw
				}
			}
			continue
		}
		capabilities = append(capabilities, withWritableBundleIDCapabilityAttributes(capability))
	}
	return append(capabilities, synced)
}

func withWritableBundleIDCapabilityAttributes(capability webBundleIDCapabilityRelationship) webBundleIDCapabilityRelationship {
	if capability.Attributes == nil {
		return capability
	}
	writable := make(map[string]any, len(writableBundleIDCapabilityAttributes))
	for _, key := range writableBundleIDCapabilityAttributes {
		if value, ok := capability.Attributes[key]; ok {
			writable[key] = value
		}
	}
	capability.Attributes = writable
	return capability
}
