package asc

import (
	"context"
	"net/http"
	"testing"
)

func TestGetInAppPurchasesV2_WithStateTypeAndSortFilters(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[{"type":"inAppPurchases","id":"iap-1","attributes":{"name":"Pro","productId":"com.example.pro","inAppPurchaseType":"CONSUMABLE","state":"READY_TO_SUBMIT"}}]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/apps/123/inAppPurchasesV2" {
			t.Fatalf("expected path /v1/apps/123/inAppPurchasesV2, got %s", req.URL.Path)
		}
		values := req.URL.Query()
		if values.Get("filter[state]") != "READY_TO_SUBMIT,APPROVED" {
			t.Fatalf("expected filter[state]=READY_TO_SUBMIT,APPROVED, got %q", values.Get("filter[state]"))
		}
		if values.Get("filter[inAppPurchaseType]") != "CONSUMABLE,NON_CONSUMABLE" {
			t.Fatalf("expected filter[inAppPurchaseType]=CONSUMABLE,NON_CONSUMABLE, got %q", values.Get("filter[inAppPurchaseType]"))
		}
		if values.Get("sort") != "name,-inAppPurchaseType" {
			t.Fatalf("expected sort=name,-inAppPurchaseType, got %q", values.Get("sort"))
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetInAppPurchasesV2(
		context.Background(),
		"123",
		WithIAPStates([]string{"READY_TO_SUBMIT", "APPROVED"}),
		WithIAPTypes([]string{"CONSUMABLE", "NON_CONSUMABLE"}),
		WithIAPSort([]string{"name", "-inAppPurchaseType"}),
	); err != nil {
		t.Fatalf("GetInAppPurchasesV2() error: %v", err)
	}
}
