package asc

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateBuildBetaNotification(t *testing.T) {
	response := jsonResponse(http.StatusCreated, `{"data":{"type":"buildBetaNotifications","id":"notif-1"}}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/v1/buildBetaNotifications" {
			t.Fatalf("expected path /v1/buildBetaNotifications, got %s", req.URL.Path)
		}
		var payload BuildBetaNotificationCreateRequest
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if payload.Data.Type != ResourceTypeBuildBetaNotifications {
			t.Fatalf("expected type buildBetaNotifications, got %q", payload.Data.Type)
		}
		if payload.Data.Relationships.Build.Data.Type != ResourceTypeBuilds {
			t.Fatalf("expected build relationship type builds, got %q", payload.Data.Relationships.Build.Data.Type)
		}
		if payload.Data.Relationships.Build.Data.ID != "build-1" {
			t.Fatalf("expected build ID build-1, got %q", payload.Data.Relationships.Build.Data.ID)
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.CreateBuildBetaNotification(context.Background(), "build-1"); err != nil {
		t.Fatalf("CreateBuildBetaNotification() error: %v", err)
	}
}

func TestCreateBuildBetaNotification_ValidationErrors(t *testing.T) {
	client := newTestClient(t, nil, nil)
	if _, err := client.CreateBuildBetaNotification(context.Background(), ""); err == nil {
		t.Fatalf("expected error for missing buildID, got nil")
	}
}
