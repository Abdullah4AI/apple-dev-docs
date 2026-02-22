package asc

import (
	"context"
	"net/http"
	"testing"
)

func TestGetBackgroundAssetVersionAppStoreRelease_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":{"type":"backgroundAssetVersionAppStoreReleases","id":"rel-1","attributes":{"state":"READY_FOR_REVIEW"}}}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/backgroundAssetVersionAppStoreReleases/rel-1" {
			t.Fatalf("expected path /v1/backgroundAssetVersionAppStoreReleases/rel-1, got %s", req.URL.Path)
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetBackgroundAssetVersionAppStoreRelease(context.Background(), "rel-1"); err != nil {
		t.Fatalf("GetBackgroundAssetVersionAppStoreRelease() error: %v", err)
	}
}

func TestGetBackgroundAssetVersionExternalBetaRelease_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":{"type":"backgroundAssetVersionExternalBetaReleases","id":"rel-2","attributes":{"state":"READY_FOR_TESTING"}}}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/backgroundAssetVersionExternalBetaReleases/rel-2" {
			t.Fatalf("expected path /v1/backgroundAssetVersionExternalBetaReleases/rel-2, got %s", req.URL.Path)
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetBackgroundAssetVersionExternalBetaRelease(context.Background(), "rel-2"); err != nil {
		t.Fatalf("GetBackgroundAssetVersionExternalBetaRelease() error: %v", err)
	}
}

func TestGetBackgroundAssetVersionInternalBetaRelease_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":{"type":"backgroundAssetVersionInternalBetaReleases","id":"rel-3","attributes":{"state":"READY_FOR_TESTING"}}}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/backgroundAssetVersionInternalBetaReleases/rel-3" {
			t.Fatalf("expected path /v1/backgroundAssetVersionInternalBetaReleases/rel-3, got %s", req.URL.Path)
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetBackgroundAssetVersionInternalBetaRelease(context.Background(), "rel-3"); err != nil {
		t.Fatalf("GetBackgroundAssetVersionInternalBetaRelease() error: %v", err)
	}
}
