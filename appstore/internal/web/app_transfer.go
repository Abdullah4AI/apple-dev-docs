package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AppTransferStatus preserves the app response, with a summary for human output.
// Presence is unknown when Apple omits linkage, none for explicit null, and
// present when Apple returns a transfer reference. It does not imply eligibility.
type AppTransferStatus struct {
	Raw       json.RawMessage
	AppID     string
	Presence  string
	RequestID string
	State     string
}

// MarshalJSON returns Apple's envelope without flattening or dropping fields.
func (s AppTransferStatus) MarshalJSON() ([]byte, error) { return s.Raw, nil }

// GetAppTransferStatus reads the app-attached transfer request through the
// private web API. It never follows the legacy transfer page or action links.
func (c *Client) GetAppTransferStatus(ctx context.Context, appID string) (*AppTransferStatus, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("app id is required")
	}
	body, err := c.doRequest(ctx, http.MethodGet, "/apps/"+url.PathEscape(appID)+"?include=appTransferRequest", nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Data     jsonAPIResource   `json:"data"`
		Included []jsonAPIResource `json:"included"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse app transfer status: %w", err)
	}
	if payload.Data.Type != "apps" || payload.Data.ID != appID {
		return nil, fmt.Errorf("app transfer status response does not identify requested app %q", appID)
	}
	result := &AppTransferStatus{Raw: body, AppID: appID, Presence: "unknown"}
	relation, ok := payload.Data.Relationships["appTransferRequest"]
	if !ok || len(relation.Data) == 0 {
		return result, nil
	}
	if bytes.Equal(bytes.TrimSpace(relation.Data), []byte("null")) {
		result.Presence = "none"
		return result, nil
	}
	var ref resourceRef
	if err := json.Unmarshal(relation.Data, &ref); err != nil {
		return nil, fmt.Errorf("parse app transfer relationship: %w", err)
	}
	if strings.TrimSpace(ref.ID) == "" || strings.TrimSpace(ref.Type) == "" {
		return nil, fmt.Errorf("app transfer relationship is missing resource identity")
	}
	result.Presence = "present"
	result.RequestID = ref.ID
	found := false
	for _, resource := range payload.Included {
		if resource.ID != ref.ID || resource.Type != ref.Type {
			continue
		}
		if found {
			return nil, fmt.Errorf("app transfer response contains duplicate referenced resources")
		}
		found = true
		if state, ok := resource.Attributes["state"].(string); ok {
			result.State = state
		}
	}
	return result, nil
}
