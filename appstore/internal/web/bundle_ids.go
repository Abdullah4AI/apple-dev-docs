package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
type AppClipBundleIDCapabilitySyncResult struct {
	BundleID       string `json:"bundleId"`
	ParentBundleID string `json:"parentBundleId"`
	Capability     string `json:"capability"`
	Enabled        bool   `json:"enabled"`
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

type webBundleIDCapabilityRelationship struct {
	ID            string                                 `json:"id,omitempty"`
	Type          string                                 `json:"type"`
	Attributes    any                                    `json:"attributes,omitempty"`
	Relationships map[string]webBundleIDRelationshipData `json:"relationships,omitempty"`
}

type webBundleIDRelationshipData struct {
	Data relationshipData `json:"data"`
}

func (r webBundleIDCapabilityRelationship) capabilityID() string {
	if r.Relationships == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(r.Relationships["capability"].Data.ID))
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
// the parentBundleId relationship required by App Clip targets.
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

	payload := buildAppClipBundleIDCapabilityPatchRequest(current, req)
	if _, err := c.doIrisV1Request(ctx, http.MethodPatch, fmt.Sprintf("/bundleIds/%s", req.BundleID), payload); err != nil {
		return nil, err
	}

	return &AppClipBundleIDCapabilitySyncResult{
		BundleID:       req.BundleID,
		ParentBundleID: req.ParentBundleID,
		Capability:     req.Capability,
		Enabled:        req.Enabled,
	}, nil
}

func buildAppClipBundleIDCapabilityPatchRequest(current webBundleIDResponse, req AppClipBundleIDCapabilitySyncRequest) webBundleIDPatchRequest {
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
		Relationships: map[string]webBundleIDRelationshipData{
			"capability": {
				Data: relationshipData{Type: "capabilities", ID: req.Capability},
			},
			"parentBundleId": {
				Data: relationshipData{Type: "bundleIds", ID: req.ParentBundleID},
			},
		},
	}
	payload.Data.Relationships.BundleIDCapabilities.Data = appendPreservedBundleIDCapabilities(currentBundleIDCapabilities(current), capability, req.SettingsProvided)
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

func appendPreservedBundleIDCapabilities(existing []webBundleIDCapabilityRelationship, synced webBundleIDCapabilityRelationship, settingsProvided bool) []webBundleIDCapabilityRelationship {
	capabilityID := synced.capabilityID()
	capabilities := make([]webBundleIDCapabilityRelationship, 0, len(existing)+1)
	for _, capability := range existing {
		if capability.capabilityID() == capabilityID {
			if synced.ID == "" {
				synced.ID = capability.ID
			}
			if !settingsProvided {
				preserveBundleIDCapabilitySettings(capability, &synced)
			}
			continue
		}
		capabilities = append(capabilities, capability)
	}
	return append(capabilities, synced)
}

func preserveBundleIDCapabilitySettings(existing webBundleIDCapabilityRelationship, synced *webBundleIDCapabilityRelationship) {
	existingAttributes, ok := existing.Attributes.(map[string]any)
	if !ok {
		return
	}
	existingSettings, ok := existingAttributes["settings"]
	if !ok {
		return
	}
	syncedAttributes, ok := synced.Attributes.(map[string]any)
	if !ok {
		return
	}
	syncedAttributes["settings"] = existingSettings
}
