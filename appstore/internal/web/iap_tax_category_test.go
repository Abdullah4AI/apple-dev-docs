package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIAPTaxCategoryRead(t *testing.T) {
	for _, tc := range []struct {
		name, linkage, record string
		configured, wantErr   bool
	}{
		{name: "inherited", linkage: `null`},
		{name: "configured", linkage: `{"type":"inAppPurchaseTaxCategoryInfos","id":"tax-1"}`, record: `{"data":{"type":"inAppPurchaseTaxCategoryInfos","id":"tax-1","relationships":{"inAppPurchaseV2":{"data":{"type":"inAppPurchases","id":"iap-1"}},"category":{"data":{"type":"taxCategories","id":"C009"}},"enabledConditions":{"data":[{"type":"taxConditions","id":"C009-Q01"}]}}},"meta":{"future":true}}`, configured: true},
		{name: "missing linkage", linkage: ``, wantErr: true},
		{name: "wrong reference type", linkage: `{"type":"apps","id":"tax-1"}`, wantErr: true},
		{name: "wrong IAP", linkage: `{"type":"inAppPurchaseTaxCategoryInfos","id":"tax-1"}`, record: `{"data":{"type":"inAppPurchaseTaxCategoryInfos","id":"tax-1","relationships":{"inAppPurchaseV2":{"data":{"type":"inAppPurchases","id":"another"}}}}}`, wantErr: true},
		{name: "unknown conditions", linkage: `{"type":"inAppPurchaseTaxCategoryInfos","id":"tax-1"}`, record: `{"data":{"type":"inAppPurchaseTaxCategoryInfos","id":"tax-1","relationships":{"inAppPurchaseV2":{"data":{"type":"inAppPurchases","id":"iap-1"}},"category":{"data":{"type":"taxCategories","id":"C009"}},"enabledConditions":{"links":{}}}}}`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			discovery := `{"data":{"type":"inAppPurchases","id":"iap-1","relationships":{"inAppPurchaseTaxCategoryInfo":{}}},"meta":{"future":true}}`
			if tc.linkage != "" {
				discovery = fmt.Sprintf(`{"data":{"type":"inAppPurchases","id":"iap-1","relationships":{"inAppPurchaseTaxCategoryInfo":{"data":%s}}},"meta":{"future":true}}`, tc.linkage)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("unexpected method %s", r.Method)
				}
				switch r.URL.Path {
				case "/iris/v2/inAppPurchases/iap-1":
					if r.URL.Query().Get("include") != "inAppPurchaseTaxCategoryInfo" {
						t.Error(r.URL)
					}
					_, _ = io.WriteString(w, discovery)
				case "/inAppPurchaseTaxCategoryInfos/tax-1":
					if r.URL.Query().Get("include") != "category,enabledConditions,inAppPurchaseV2" || r.URL.Query().Get("limit[enabledConditions]") != "100" {
						t.Error(r.URL)
					}
					_, _ = io.WriteString(w, tc.record)
				default:
					t.Error(r.URL)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			result, err := testWebClient(server).GetIAPTaxCategory(context.Background(), "iap-1")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Configured != tc.configured || result.IAPID != "iap-1" {
				t.Fatalf("result %+v", result)
			}
			raw, err := json.Marshal(result)
			if err != nil || !strings.Contains(string(raw), `"future":true`) {
				t.Fatalf("lost envelope: %s %v", raw, err)
			}
			if tc.configured && (result.ID != "tax-1" || result.CategoryID != "C009" || len(result.EnabledConditionIDs) != 1) {
				t.Fatalf("result %+v", result)
			}
		})
	}
}

func TestIAPTaxCategoryWritesUseCapturedContract(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body struct {
			Data struct {
				ID, Type      string
				Relationships map[string]json.RawMessage
			}
		}
		if r.Method == "DELETE" {
			if r.URL.Path != "/inAppPurchaseTaxCategoryInfos/tax-1" {
				t.Error(r.URL)
			}
			w.WriteHeader(204)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Data.Type != "inAppPurchaseTaxCategoryInfos" {
			t.Errorf("type %s", body.Data.Type)
		}
		if string(body.Data.Relationships["enabledConditions"]) != `{"data":[]}` {
			t.Errorf("conditions %s", body.Data.Relationships["enabledConditions"])
		}
		switch r.Method {
		case "POST":
			if body.Data.ID != "" || !strings.Contains(string(body.Data.Relationships["inAppPurchaseV2"]), `"iap-1"`) {
				t.Errorf("body %+v", body)
			}
		case "PATCH":
			if body.Data.ID != "tax-1" || len(body.Data.Relationships["inAppPurchaseV2"]) != 0 {
				t.Errorf("body %+v", body)
			}
		default:
			t.Error(r.Method)
		}
		_, _ = io.WriteString(w, `{"data":{"type":"inAppPurchaseTaxCategoryInfos","id":"tax-1"}}`)
	}))
	defer server.Close()
	client := testWebClient(server)
	ctx := context.Background()
	inherited := &IAPTaxCategory{IAPID: "iap-1"}
	if err := client.SaveIAPTaxCategory(ctx, "iap-1", "C003", nil, inherited); err != nil {
		t.Fatal(err)
	}
	configured := &IAPTaxCategory{IAPID: "iap-1", ID: "tax-1", Configured: true}
	if err := client.SaveIAPTaxCategory(ctx, "iap-1", "C003", nil, configured); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteIAPTaxCategory(ctx, "iap-1", configured); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteIAPTaxCategory(ctx, "other", configured); err == nil {
		t.Fatal("wrong IAP accepted")
	}
	if calls != 3 {
		t.Fatalf("calls %d", calls)
	}
}

func TestListIAPTaxCategoriesUsesAddonCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/taxCategories" || r.URL.Query().Get("filter[productType]") != "ADDON" {
			t.Error(r.URL)
		}
		_, _ = io.WriteString(w, `{"data":[{"type":"taxCategories","id":"C003","attributes":{"productType":"ADDON"},"relationships":{"conditions":{"data":[{"type":"taxConditions","id":"condition-1"}]}}}],"included":[{"type":"taxConditions","id":"condition-1","attributes":{"description":"A condition from Apple"}}],"meta":{"future":true}}`)
	}))
	defer server.Close()
	result, err := testWebClient(server).ListIAPTaxCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conditions) != 1 || result.Conditions[0].Name != "A condition from Apple" {
		t.Fatalf("condition descriptions missing: %+v", result.Conditions)
	}
	if len(result.Categories) != 1 || result.Categories[0].ProductType != "ADDON" {
		t.Fatalf("result %+v", result)
	}
}

func TestIAPTaxCategoryWriteFailureIsNotRetried(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"errors":[{"code":"UNAVAILABLE"}]}`)
	}))
	defer server.Close()
	err := testWebClient(server).SaveIAPTaxCategory(context.Background(), "iap-1", "C003", nil, &IAPTaxCategory{IAPID: "iap-1"})
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestIAPTaxCategoryRejectsIncompleteConditions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/inAppPurchases/") {
			_, _ = io.WriteString(w, `{"data":{"type":"inAppPurchases","id":"iap-1","relationships":{"inAppPurchaseTaxCategoryInfo":{"data":{"type":"inAppPurchaseTaxCategoryInfos","id":"tax-1"}}}}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"type":"inAppPurchaseTaxCategoryInfos","id":"tax-1","relationships":{"inAppPurchaseV2":{"data":{"type":"inAppPurchases","id":"iap-1"}},"category":{"data":{"type":"taxCategories","id":"C009"}},"enabledConditions":{"data":[],"meta":{"paging":{"total":1}}}}}}`)
	}))
	defer server.Close()
	_, err := testWebClient(server).GetIAPTaxCategory(context.Background(), "iap-1")
	if err == nil || !strings.Contains(err.Error(), "incomplete condition") {
		t.Fatalf("err=%v", err)
	}
}
