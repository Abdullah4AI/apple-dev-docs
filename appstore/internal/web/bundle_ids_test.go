package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestSyncAppClipBundleIDCapabilityAddsParentBundleRelationship(t *testing.T) {
	var patchBody []byte
	client := &Client{
		httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/iris/v1/bundleIds/clip-bundle":
				if r.URL.Query().Get("include") != "bundleIdCapabilities" {
					t.Fatalf("expected include=bundleIdCapabilities, got %q", r.URL.RawQuery)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(bytes.NewBufferString(`{
						"data":{
							"id":"clip-bundle",
							"type":"bundleIds",
							"attributes":{
								"name":"Example Clip",
								"identifier":"com.example.app.Clip",
								"seedId":"TEAMID"
							}
						}
					}`)),
				}, nil
			case r.Method == http.MethodPatch && r.URL.Path == "/iris/v1/bundleIds/clip-bundle":
				var err error
				patchBody, err = io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("ReadAll patch body error: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(bytes.NewBufferString(`{"data":{"id":"clip-bundle","type":"bundleIds"}}`)),
				}, nil
			default:
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
				return nil, nil
			}
		})},
	}

	enabled := true
	result, err := client.SyncAppClipBundleIDCapability(context.Background(), AppClipBundleIDCapabilitySyncRequest{
		BundleID:       "clip-bundle",
		ParentBundleID: "parent-bundle",
		Capability:     "PUSH_NOTIFICATIONS",
		Enabled:        true,
		Settings: []BundleIDCapabilitySetting{{
			Key: "PUSH_NOTIFICATION_FEATURES",
			Options: []BundleIDCapabilityOption{{
				Key:     "PUSH_NOTIFICATION_FEATURE_BROADCAST",
				Enabled: &enabled,
			}},
		}},
	})
	if err != nil {
		t.Fatalf("SyncAppClipBundleIDCapability error: %v", err)
	}
	if result.ParentBundleID != "parent-bundle" || result.Capability != "PUSH_NOTIFICATIONS" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(patchBody) == 0 {
		t.Fatal("expected patch body")
	}

	var payload struct {
		Data struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				Name       string `json:"name"`
				Identifier string `json:"identifier"`
				SeedID     string `json:"seedId"`
			} `json:"attributes"`
			Relationships struct {
				BundleIDCapabilities struct {
					Data []struct {
						Type       string `json:"type"`
						Attributes struct {
							Enabled  bool `json:"enabled"`
							Settings []struct {
								Key string `json:"key"`
							} `json:"settings"`
						} `json:"attributes"`
						Relationships struct {
							Capability     webBundleIDRelationshipData `json:"capability"`
							ParentBundleID webBundleIDRelationshipData `json:"parentBundleId"`
						} `json:"relationships"`
					} `json:"data"`
				} `json:"bundleIdCapabilities"`
			} `json:"relationships"`
		} `json:"data"`
	}
	if err := json.Unmarshal(patchBody, &payload); err != nil {
		t.Fatalf("json.Unmarshal patch body error: %v; body=%s", err, patchBody)
	}
	if payload.Data.Type != "bundleIds" || payload.Data.ID != "clip-bundle" {
		t.Fatalf("unexpected bundle data: %+v", payload.Data)
	}
	if payload.Data.Attributes.Identifier != "com.example.app.Clip" || payload.Data.Attributes.SeedID != "TEAMID" {
		t.Fatalf("expected current bundle attributes in patch, got %+v", payload.Data.Attributes)
	}
	caps := payload.Data.Relationships.BundleIDCapabilities.Data
	if len(caps) != 1 {
		t.Fatalf("expected one capability relationship, got %d", len(caps))
	}
	if caps[0].Relationships.Capability.Data != (relationshipData{Type: "capabilities", ID: "PUSH_NOTIFICATIONS"}) {
		t.Fatalf("unexpected capability relationship: %+v", caps[0].Relationships.Capability.Data)
	}
	if caps[0].Relationships.ParentBundleID.Data != (relationshipData{Type: "bundleIds", ID: "parent-bundle"}) {
		t.Fatalf("unexpected parentBundleId relationship: %+v", caps[0].Relationships.ParentBundleID.Data)
	}
	if len(caps[0].Attributes.Settings) != 1 || caps[0].Attributes.Settings[0].Key != "PUSH_NOTIFICATION_FEATURES" {
		t.Fatalf("expected settings to be preserved, got %+v", caps[0].Attributes.Settings)
	}
}

func TestSyncAppClipBundleIDCapabilityPreservesExistingCapabilities(t *testing.T) {
	var patchBody []byte
	client := &Client{
		httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/iris/v1/bundleIds/clip-bundle":
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(bytes.NewBufferString(`{
						"data":{
							"id":"clip-bundle",
							"type":"bundleIds",
							"attributes":{
								"name":"Example Clip",
								"identifier":"com.example.app.Clip"
							},
							"relationships":{
								"bundleIdCapabilities":{
									"data":[
										{"id":"existing-icloud","type":"bundleIdCapabilities"},
										{"id":"existing-push","type":"bundleIdCapabilities"}
									]
								}
							}
						},
						"included":[
							{
								"id":"existing-icloud",
								"type":"bundleIdCapabilities",
								"attributes":{"enabled":true,"settings":[]},
								"relationships":{
									"capability":{"data":{"type":"capabilities","id":"ICLOUD"}}
								}
							},
							{
								"id":"existing-push",
								"type":"bundleIdCapabilities",
								"attributes":{"enabled":false,"settings":[{"key":"PUSH_NOTIFICATION_FEATURES"}]},
								"relationships":{
									"capability":{"data":{"type":"capabilities","id":"PUSH_NOTIFICATIONS"}}
								}
							}
						]
					}`)),
				}, nil
			case r.Method == http.MethodPatch && r.URL.Path == "/iris/v1/bundleIds/clip-bundle":
				var err error
				patchBody, err = io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("ReadAll patch body error: %v", err)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(bytes.NewBufferString(`{"data":{"id":"clip-bundle","type":"bundleIds"}}`)),
				}, nil
			default:
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
				return nil, nil
			}
		})},
	}

	if _, err := client.SyncAppClipBundleIDCapability(context.Background(), AppClipBundleIDCapabilitySyncRequest{
		BundleID:       "clip-bundle",
		ParentBundleID: "parent-bundle",
		Capability:     "PUSH_NOTIFICATIONS",
		Enabled:        true,
	}); err != nil {
		t.Fatalf("SyncAppClipBundleIDCapability error: %v", err)
	}

	var payload struct {
		Data struct {
			Relationships struct {
				BundleIDCapabilities struct {
					Data []webBundleIDCapabilityRelationship `json:"data"`
				} `json:"bundleIdCapabilities"`
			} `json:"relationships"`
		} `json:"data"`
	}
	if err := json.Unmarshal(patchBody, &payload); err != nil {
		t.Fatalf("json.Unmarshal patch body error: %v; body=%s", err, patchBody)
	}
	caps := payload.Data.Relationships.BundleIDCapabilities.Data
	if len(caps) != 2 {
		t.Fatalf("expected existing ICLOUD plus synced PUSH_NOTIFICATIONS, got %d: %+v", len(caps), caps)
	}
	if caps[0].ID != "existing-icloud" || caps[0].capabilityID() != "ICLOUD" {
		t.Fatalf("expected existing ICLOUD capability to be preserved first, got %+v", caps[0])
	}
	if caps[1].ID != "existing-push" || caps[1].capabilityID() != "PUSH_NOTIFICATIONS" {
		t.Fatalf("expected synced PUSH_NOTIFICATIONS capability to replace existing entry, got %+v", caps[1])
	}
	if caps[1].Relationships["parentBundleId"].Data != (relationshipData{Type: "bundleIds", ID: "parent-bundle"}) {
		t.Fatalf("unexpected synced parentBundleId relationship: %+v", caps[1].Relationships["parentBundleId"].Data)
	}
	attributes, ok := caps[1].Attributes.(map[string]any)
	if !ok {
		t.Fatalf("expected synced attributes map, got %T", caps[1].Attributes)
	}
	settings, ok := attributes["settings"].([]any)
	if !ok || len(settings) != 1 {
		t.Fatalf("expected existing settings to be preserved, got %+v", attributes["settings"])
	}
	setting, ok := settings[0].(map[string]any)
	if !ok || setting["key"] != "PUSH_NOTIFICATION_FEATURES" {
		t.Fatalf("expected existing PUSH_NOTIFICATION_FEATURES setting, got %+v", settings[0])
	}
}
