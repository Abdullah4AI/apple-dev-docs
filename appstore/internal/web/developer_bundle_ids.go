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
	privateCloudCompute = "PRIVATE_CLOUD_COMPUTE"
)

var supportedDeveloperBundleIDCapabilities = map[string]struct{}{
	privateCloudCompute: {},
}

var developerBundleIDIncludes = []string{
	"bundleIdCapabilities",
	"bundleIdCapabilities.capability",
	"bundleIdCapabilities.associatedBundleIds",
	"bundleIdCapabilities.appGroups",
	"bundleIdCapabilities.merchantIds",
	"bundleIdCapabilities.cloudContainers",
	"bundleIdCapabilities.certificates",
	"bundleIdCapabilities.appConsentBundleId",
	"bundleIdCapabilities.macBundleId",
	"bundleIdCapabilities.relatedAppConsentBundleIds",
	"bundleIdCapabilities.parentBundleId",
	"bundleIdCapabilities.mediaSharingProtocolIds",
}

// DeveloperBundleIDCapabilityEnableRequest enables one supported Developer
// Portal-only capability on an existing Bundle ID resource.
type DeveloperBundleIDCapabilityEnableRequest struct {
	BundleID   string
	Capability string
}

// DeveloperBundleIDCapabilityEnableResult summarizes a Developer Portal
// capability enable operation. Changed is false when the capability was already
// enabled and no PATCH was sent.
type DeveloperBundleIDCapabilityEnableResult struct {
	BundleID   string `json:"bundleId"`
	Capability string `json:"capability"`
	Enabled    bool   `json:"enabled"`
	Changed    bool   `json:"changed"`
	Status     string `json:"status"`
}

type developerCapabilityMetadataResponse struct {
	Data []struct {
		ID         string                                `json:"id"`
		Type       string                                `json:"type"`
		Attributes developerCapabilityMetadataAttributes `json:"attributes"`
	} `json:"data"`
}

type developerCapabilityMetadataAttributes struct {
	Name         string `json:"name"`
	Entitlement  string `json:"entitlement"`
	IsPublic     bool   `json:"isPublic"`
	Editable     bool   `json:"editable"`
	CanRequest   bool   `json:"canRequestFromPortal"`
	EnabledByDef bool   `json:"enabledByDefault"`
}

type developerResource struct {
	ID            string                     `json:"id,omitempty"`
	Type          string                     `json:"type"`
	Attributes    json.RawMessage            `json:"attributes,omitempty"`
	Relationships map[string]json.RawMessage `json:"relationships,omitempty"`
}

type developerResourceRelationship struct {
	Data []developerResource `json:"data"`
}

type developerBundleIDResponse struct {
	Data struct {
		ID            string                     `json:"id"`
		Type          string                     `json:"type"`
		Attributes    json.RawMessage            `json:"attributes"`
		Relationships map[string]json.RawMessage `json:"relationships"`
	} `json:"data"`
	Included []developerResource `json:"included"`
}

type developerBundleIDPatchRequest struct {
	Data struct {
		ID            string                     `json:"id"`
		Type          string                     `json:"type"`
		Attributes    json.RawMessage            `json:"attributes"`
		Relationships map[string]json.RawMessage `json:"relationships"`
	} `json:"data"`
}

func normalizeDeveloperBundleIDCapabilityEnableRequest(req DeveloperBundleIDCapabilityEnableRequest) (DeveloperBundleIDCapabilityEnableRequest, error) {
	req.BundleID = strings.TrimSpace(req.BundleID)
	req.Capability = strings.ToUpper(strings.TrimSpace(req.Capability))
	if req.BundleID == "" {
		return req, fmt.Errorf("bundle id is required")
	}
	if req.Capability == "" {
		return req, fmt.Errorf("capability is required")
	}
	if _, ok := supportedDeveloperBundleIDCapabilities[req.Capability]; !ok {
		return req, fmt.Errorf("unsupported Developer Portal capability %q (supported: %s)", req.Capability, privateCloudCompute)
	}
	return req, nil
}

// EnableDeveloperBundleIDCapability enables a supported Developer Portal-only
// Bundle ID capability while preserving Apple's complete current capability
// relationship payload.
func (c *Client) EnableDeveloperBundleIDCapability(ctx context.Context, req DeveloperBundleIDCapabilityEnableRequest) (*DeveloperBundleIDCapabilityEnableResult, error) {
	req, err := normalizeDeveloperBundleIDCapabilityEnableRequest(req)
	if err != nil {
		return nil, err
	}
	if err := c.ensureDeveloperPortalSession(ctx); err != nil {
		return nil, err
	}

	metadata, err := c.loadDeveloperCapabilityMetadata(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	capabilityMetadata, ok := findDeveloperCapability(metadata, req.Capability)
	if !ok {
		return nil, fmt.Errorf("capability %q is not available in Developer Portal for this account", req.Capability)
	}
	if !capabilityMetadata.Editable {
		return nil, fmt.Errorf("capability %q is not editable in Developer Portal for this account", req.Capability)
	}

	current, err := c.loadDeveloperBundleID(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	payload, alreadyEnabled, err := buildDeveloperBundleIDCapabilityPatchRequest(current, req)
	if err != nil {
		return nil, err
	}
	if alreadyEnabled {
		return &DeveloperBundleIDCapabilityEnableResult{
			BundleID:   req.BundleID,
			Capability: req.Capability,
			Enabled:    true,
			Changed:    false,
			Status:     "already-enabled",
		}, nil
	}
	teamID := c.developerPortalTeamID()
	if teamID == "" {
		return nil, fmt.Errorf("developer portal team is not selected; %s", developerPortalAuthHint)
	}
	payload, err = addDeveloperPortalTeamID(payload, teamID)
	if err != nil {
		return nil, err
	}

	csrf, csrfTS := c.developerCSRFTokens()
	if csrf == "" || csrfTS == "" {
		return nil, fmt.Errorf("missing Developer Portal CSRF headers; %s", developerPortalAuthHint)
	}
	path := "/bundleIds/" + url.PathEscape(req.BundleID)
	if _, err := c.doDeveloperPortalRequest(ctx, http.MethodPatch, path, payload, developerPortalHeaders(req.BundleID), true); err != nil {
		return nil, err
	}

	return &DeveloperBundleIDCapabilityEnableResult{
		BundleID:   req.BundleID,
		Capability: req.Capability,
		Enabled:    true,
		Changed:    true,
		Status:     "enabled",
	}, nil
}

func (c *Client) loadDeveloperCapabilityMetadata(ctx context.Context, bundleID string) (developerCapabilityMetadataResponse, error) {
	query := make(url.Values)
	query.Set("filter[capabilityType]", "capability,service")
	query.Set("filter[includeRequestable]", "true")
	body, err := c.doDeveloperPortalProxyRead(ctx, "/capabilities", query, developerPortalHeaders(bundleID))
	if err != nil {
		return developerCapabilityMetadataResponse{}, err
	}
	var response developerCapabilityMetadataResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("failed to parse Developer Portal capabilities response: %w", err)
	}
	return response, nil
}

func findDeveloperCapability(response developerCapabilityMetadataResponse, capabilityID string) (developerCapabilityMetadataAttributes, bool) {
	for _, capability := range response.Data {
		if capability.Type == "capabilities" && strings.EqualFold(strings.TrimSpace(capability.ID), capabilityID) {
			return capability.Attributes, true
		}
	}
	return developerCapabilityMetadataAttributes{}, false
}

func (c *Client) loadDeveloperBundleID(ctx context.Context, bundleID string) (developerBundleIDResponse, error) {
	query := make(url.Values)
	query.Set("fields[bundleIds]", "name,identifier,platform,seedId,wildcard,~permissions.delete,~permissions.edit")
	query.Set("include", strings.Join(developerBundleIDIncludes, ","))
	path := "/bundleIds/" + url.PathEscape(bundleID)
	body, err := c.doDeveloperPortalProxyRead(ctx, path, query, developerPortalHeaders(bundleID))
	if err != nil {
		return developerBundleIDResponse{}, err
	}
	var response developerBundleIDResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return response, fmt.Errorf("failed to parse Developer Portal Bundle ID response: %w", err)
	}
	if strings.TrimSpace(response.Data.ID) == "" || response.Data.Type != "bundleIds" || len(response.Data.Attributes) == 0 {
		return response, fmt.Errorf("incomplete Bundle ID resource returned by Developer Portal")
	}
	return response, nil
}

func buildDeveloperBundleIDCapabilityPatchRequest(current developerBundleIDResponse, req DeveloperBundleIDCapabilityEnableRequest) (developerBundleIDPatchRequest, bool, error) {
	capabilities, err := developerBundleIDCapabilities(current)
	if err != nil {
		return developerBundleIDPatchRequest{}, false, err
	}

	capabilityIDs := make([]string, len(capabilities))
	targetIndex := -1
	targetEnabled := false
	for index, capability := range capabilities {
		capabilityID, err := developerBundleIDCapabilityID(capability)
		if err != nil {
			return developerBundleIDPatchRequest{}, false, err
		}
		capabilityIDs[index] = capabilityID
		if capabilityID != req.Capability {
			continue
		}
		enabled, err := developerBundleIDCapabilityEnabled(capability)
		if err != nil {
			return developerBundleIDPatchRequest{}, false, err
		}
		if targetIndex == -1 || (enabled && !targetEnabled) {
			targetIndex = index
			targetEnabled = enabled
		}
	}
	if targetEnabled {
		return developerBundleIDPatchRequest{}, true, nil
	}

	updated := make([]developerResource, 0, len(capabilities)+1)
	for index, capability := range capabilities {
		if capabilityIDs[index] != req.Capability {
			updated = append(updated, capability)
			continue
		}
		if index != targetIndex {
			continue
		}
		var err error
		capability.Attributes, err = setDeveloperCapabilityEnabled(capability.Attributes)
		if err != nil {
			return developerBundleIDPatchRequest{}, false, err
		}
		updated = append(updated, capability)
	}
	if targetIndex == -1 {
		updated = append(updated, newDeveloperBundleIDCapability(req.Capability))
	}

	relationshipBody, err := marshalDeveloperBundleIDCapabilitiesForPatch(updated)
	if err != nil {
		return developerBundleIDPatchRequest{}, false, err
	}
	relationships := cloneRawMessageMap(current.Data.Relationships)
	if relationships == nil {
		relationships = make(map[string]json.RawMessage)
	}
	relationships["bundleIdCapabilities"] = relationshipBody

	var payload developerBundleIDPatchRequest
	payload.Data.ID = current.Data.ID
	payload.Data.Type = current.Data.Type
	payload.Data.Attributes = append(json.RawMessage(nil), current.Data.Attributes...)
	payload.Data.Relationships = relationships
	return payload, false, nil
}

func addDeveloperPortalTeamID(payload developerBundleIDPatchRequest, teamID string) (developerBundleIDPatchRequest, error) {
	var attributes map[string]json.RawMessage
	if len(payload.Data.Attributes) > 0 {
		if err := json.Unmarshal(payload.Data.Attributes, &attributes); err != nil {
			return payload, fmt.Errorf("failed to parse Bundle ID attributes for Developer Portal team: %w", err)
		}
	}
	if attributes == nil {
		attributes = make(map[string]json.RawMessage)
	}
	for key := range attributes {
		if key == "permissions" || strings.HasPrefix(key, "~permissions.") {
			delete(attributes, key)
		}
	}
	encodedTeamID, err := json.Marshal(strings.TrimSpace(teamID))
	if err != nil {
		return payload, fmt.Errorf("failed to encode Developer Portal team: %w", err)
	}
	attributes["teamId"] = encodedTeamID
	encodedAttributes, err := json.Marshal(attributes)
	if err != nil {
		return payload, fmt.Errorf("failed to encode Bundle ID attributes for Developer Portal team: %w", err)
	}
	payload.Data.Attributes = encodedAttributes
	return payload, nil
}

func marshalDeveloperBundleIDCapabilitiesForPatch(capabilities []developerResource) (json.RawMessage, error) {
	for index := range capabilities {
		if len(capabilities[index].Attributes) == 0 {
			continue
		}
		var attributes map[string]json.RawMessage
		if err := json.Unmarshal(capabilities[index].Attributes, &attributes); err != nil {
			return nil, fmt.Errorf("failed to parse Bundle ID capability %q attributes for patch: %w", capabilities[index].ID, err)
		}
		writable := make(map[string]json.RawMessage, 2)
		for _, key := range []string{"enabled", "settings"} {
			if value, ok := attributes[key]; ok {
				writable[key] = append(json.RawMessage(nil), value...)
			}
		}
		encoded, err := json.Marshal(writable)
		if err != nil {
			return nil, fmt.Errorf("failed to encode Bundle ID capability %q attributes for patch: %w", capabilities[index].ID, err)
		}
		capabilities[index].Attributes = encoded
	}

	encoded, err := json.Marshal(developerResourceRelationship{Data: capabilities})
	if err != nil {
		return nil, fmt.Errorf("failed to build Bundle ID capability relationships: %w", err)
	}
	return encoded, nil
}

func developerBundleIDCapabilities(current developerBundleIDResponse) ([]developerResource, error) {
	var relationship developerResourceRelationship
	rawRelationship, ok := current.Data.Relationships["bundleIdCapabilities"]
	if ok {
		if err := json.Unmarshal(rawRelationship, &relationship); err != nil {
			return nil, fmt.Errorf("failed to parse current Bundle ID capability relationships: %w", err)
		}
	}

	includedByID := make(map[string]developerResource)
	includedOrder := make([]string, 0)
	for _, resource := range current.Included {
		if resource.Type != "bundleIdCapabilities" || strings.TrimSpace(resource.ID) == "" {
			continue
		}
		if _, exists := includedByID[resource.ID]; !exists {
			includedOrder = append(includedOrder, resource.ID)
		}
		includedByID[resource.ID] = resource
	}

	capabilities := make([]developerResource, 0, len(relationship.Data))
	seen := make(map[string]struct{})
	for _, resource := range relationship.Data {
		if resource.Type != "bundleIdCapabilities" {
			continue
		}
		if resource.ID != "" {
			if _, duplicate := seen[resource.ID]; duplicate {
				continue
			}
			seen[resource.ID] = struct{}{}
			if included, ok := includedByID[resource.ID]; ok {
				resource = included
			}
		}
		if _, err := developerBundleIDCapabilityID(resource); err != nil {
			return nil, fmt.Errorf("cannot safely preserve Bundle ID capability %q: %w", resource.ID, err)
		}
		capabilities = append(capabilities, resource)
	}
	for _, id := range includedOrder {
		if _, ok := seen[id]; ok {
			continue
		}
		resource := includedByID[id]
		if _, err := developerBundleIDCapabilityID(resource); err != nil {
			return nil, fmt.Errorf("cannot safely preserve Bundle ID capability %q: %w", resource.ID, err)
		}
		seen[id] = struct{}{}
		capabilities = append(capabilities, resource)
	}
	return capabilities, nil
}

func developerBundleIDCapabilityID(resource developerResource) (string, error) {
	raw, ok := resource.Relationships["capability"]
	if !ok {
		return "", fmt.Errorf("missing capability relationship")
	}
	var relationship struct {
		Data relationshipData `json:"data"`
	}
	if err := json.Unmarshal(raw, &relationship); err != nil {
		return "", fmt.Errorf("invalid capability relationship: %w", err)
	}
	id := strings.ToUpper(strings.TrimSpace(relationship.Data.ID))
	if relationship.Data.Type != "capabilities" || id == "" {
		return "", fmt.Errorf("invalid capability relationship data")
	}
	return id, nil
}

func developerBundleIDCapabilityEnabled(resource developerResource) (bool, error) {
	if len(resource.Attributes) == 0 {
		return false, fmt.Errorf("bundle ID capability %q is missing attributes", resource.ID)
	}
	var attributes map[string]json.RawMessage
	if err := json.Unmarshal(resource.Attributes, &attributes); err != nil {
		return false, fmt.Errorf("failed to parse Bundle ID capability %q attributes: %w", resource.ID, err)
	}
	var enabled bool
	raw, ok := attributes["enabled"]
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal(raw, &enabled); err != nil {
		return false, fmt.Errorf("failed to parse Bundle ID capability %q enabled state: %w", resource.ID, err)
	}
	return enabled, nil
}

func setDeveloperCapabilityEnabled(raw json.RawMessage) (json.RawMessage, error) {
	return setDeveloperCapabilityEnabledValue(raw, true)
}

func setDeveloperCapabilityEnabledValue(raw json.RawMessage, enabled bool) (json.RawMessage, error) {
	var attributes map[string]json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &attributes); err != nil {
			return nil, fmt.Errorf("failed to parse existing capability attributes: %w", err)
		}
	}
	if attributes == nil {
		attributes = make(map[string]json.RawMessage)
	}
	attributes["enabled"] = json.RawMessage(strconv.FormatBool(enabled))
	if _, ok := attributes["settings"]; !ok {
		attributes["settings"] = json.RawMessage("[]")
	}
	updated, err := json.Marshal(attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to encode capability attributes: %w", err)
	}
	return updated, nil
}

func newDeveloperBundleIDCapability(capability string) developerResource {
	capabilityRelationship, _ := json.Marshal(struct {
		Data relationshipData `json:"data"`
	}{Data: relationshipData{Type: "capabilities", ID: capability}})
	return developerResource{
		Type:       "bundleIdCapabilities",
		Attributes: json.RawMessage(`{"enabled":true,"settings":[]}`),
		Relationships: map[string]json.RawMessage{
			"capability": capabilityRelationship,
		},
	}
}

func cloneRawMessageMap(source map[string]json.RawMessage) map[string]json.RawMessage {
	if source == nil {
		return nil
	}
	cloned := make(map[string]json.RawMessage, len(source))
	for key, value := range source {
		cloned[key] = append(json.RawMessage(nil), value...)
	}
	return cloned
}
