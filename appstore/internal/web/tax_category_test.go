package web

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListTaxCategoriesUsesCapturedApplicationCatalogRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/taxCategories" {
			t.Fatalf("unexpected tax catalog request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("filter[productType]"); got != "APPLICATION" {
			t.Fatalf("product type filter = %q", got)
		}
		if got := r.URL.Query().Get("include"); got != "subcategories,conditions" {
			t.Fatalf("include = %q", got)
		}
		if got := r.URL.Query().Get("limit[subcategories]"); got != "100" {
			t.Fatalf("subcategory limit = %q", got)
		}
		if got := r.URL.Query().Get("limit[conditions]"); got != "100" {
			t.Fatalf("condition limit = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"data":[{
				"id":"cat-app","type":"taxCategories",
				"attributes":{"name":"App Store Software","productType":"APPLICATION","subcategoryRequired":true,"contentProviders":["provider-1"]},
				"relationships":{"subcategories":{"data":[{"type":"taxCategories","id":"sub-games"}]},"conditions":{"data":[{"type":"taxConditions","id":"cond-1"}]}}
			}],
			"included":[
				{"id":"sub-games","type":"taxCategories","attributes":{"name":"Games","productType":"APPLICATION"}},
				{"id":"cond-1","type":"taxConditions","attributes":{"name":"Digital goods"}}
			],"links":{},"meta":{},"unknownTopLevel":{"preserve":"this"}
		}`)
	}))
	defer server.Close()

	catalog, err := testWebClient(server).ListTaxCategories(context.Background())
	if err != nil {
		t.Fatalf("ListTaxCategories() error = %v", err)
	}
	if len(catalog.Categories) != 2 || catalog.Categories[0].ID != "cat-app" || catalog.Categories[1].ID != "sub-games" {
		t.Fatalf("unexpected categories: %#v", catalog.Categories)
	}
	if !catalog.Categories[0].SubcategoryRequired || catalog.Categories[0].Name != "App Store Software" {
		t.Fatalf("unexpected root category: %#v", catalog.Categories[0])
	}
	if len(catalog.Categories[0].Subcategories) != 1 || catalog.Categories[0].Subcategories[0].Name != "Games" {
		t.Fatalf("unexpected subcategory refs: %#v", catalog.Categories[0].Subcategories)
	}
	if len(catalog.Conditions) != 1 || catalog.Conditions[0].ID != "cond-1" || catalog.Conditions[0].Name != "Digital goods" {
		t.Fatalf("unexpected conditions: %#v", catalog.Conditions)
	}
	if !bytes.Contains(catalog.Raw, []byte(`"unknownTopLevel":{"preserve":"this"}`)) {
		t.Fatalf("raw catalog envelope lost unknown top-level member: %s", catalog.Raw)
	}
}

func TestGetAppTaxCategoryParsesConfiguredCategoryAndConditions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/appTaxCategories/app-1" {
			t.Fatalf("unexpected app tax request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("include"); got != "category,enabledConditions" {
			t.Fatalf("include = %q", got)
		}
		if got := r.URL.Query().Get("limit[enabledConditions]"); got != "100" {
			t.Fatalf("condition limit = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"data":{"id":"app-1","type":"appTaxCategories","relationships":{
				"category":{"data":{"type":"taxCategories","id":"cat-games"}},
				"enabledConditions":{"data":[{"type":"taxConditions","id":"cond-1"},{"type":"taxConditions","id":"cond-2"}]}
			}},"included":[{"id":"cat-games","type":"taxCategories","attributes":{"name":"Games"}}]
		}`)
	}))
	defer server.Close()

	current, err := testWebClient(server).GetAppTaxCategory(context.Background(), "app-1")
	if err != nil {
		t.Fatalf("GetAppTaxCategory() error = %v", err)
	}
	if !current.Configured || current.AppID != "app-1" || current.CategoryID != "cat-games" {
		t.Fatalf("unexpected current category: %#v", current)
	}
	if len(current.EnabledConditionIDs) != 2 || current.EnabledConditionIDs[1] != "cond-2" {
		t.Fatalf("unexpected enabled conditions: %#v", current.EnabledConditionIDs)
	}
}

func TestGetAppTaxCategoryTreatsNotFoundAsUnconfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/appTaxCategories/app-1":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"errors":[{"code":"RESOURCE_NOT_FOUND"}]}`)
		case "/apps/app-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"data":{"id":"app-1","type":"apps","attributes":{"name":"Example"}}}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	current, err := testWebClient(server).GetAppTaxCategory(context.Background(), "app-1")
	if err != nil {
		t.Fatalf("GetAppTaxCategory() error = %v", err)
	}
	if current == nil || current.Configured || current.AppID != "app-1" {
		t.Fatalf("unexpected unconfigured result: %#v", current)
	}
}

func TestGetAppTaxCategoryRequiresAppVerificationAfterNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/appTaxCategories/app-1", "/apps/app-1":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"errors":[{"code":"RESOURCE_NOT_FOUND"}]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	current, err := testWebClient(server).GetAppTaxCategory(context.Background(), "app-1")
	if err == nil || !strings.Contains(err.Error(), "verify app") {
		t.Fatalf("expected app verification error, got current=%#v err=%v", current, err)
	}
	if current != nil {
		t.Fatalf("expected no unconfigured result when app verification fails: %#v", current)
	}
}

func TestGetAppTaxCategoryRequiresAppVerificationAfterNullData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/appTaxCategories/app-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"data":null}`)
		case "/apps/app-1":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"errors":[{"code":"RESOURCE_NOT_FOUND"}]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	current, err := testWebClient(server).GetAppTaxCategory(context.Background(), "app-1")
	if err == nil || !strings.Contains(err.Error(), "verify app") {
		t.Fatalf("expected app verification error, got current=%#v err=%v", current, err)
	}
	if current != nil {
		t.Fatalf("expected no unconfigured result when app verification fails: %#v", current)
	}
}

func TestSaveAppTaxCategoryUsesCapturedPostBodyForUnconfiguredApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/appTaxCategories" {
			t.Fatalf("unexpected app tax create request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		data := taxCategoryWriteMap(t, body, "data")
		if data["type"] != "appTaxCategories" {
			t.Fatalf("type = %#v", data["type"])
		}
		if _, ok := data["id"]; ok {
			t.Fatalf("POST must omit id: %#v", data)
		}
		relationships := taxCategoryWriteMap(t, data, "relationships")
		app := taxCategoryWriteMap(t, relationships, "app")
		appData := taxCategoryWriteMap(t, app, "data")
		if appData["type"] != "apps" || appData["id"] != "app-1" {
			t.Fatalf("app relationship = %#v", appData)
		}
		category := taxCategoryWriteMap(t, relationships, "category")
		categoryData := taxCategoryWriteMap(t, category, "data")
		if categoryData["type"] != "taxCategories" || categoryData["id"] != "cat-games" {
			t.Fatalf("category relationship = %#v", categoryData)
		}
		conditions := taxCategoryWriteMap(t, relationships, "enabledConditions")
		conditionData, ok := conditions["data"].([]any)
		if !ok || len(conditionData) != 0 {
			t.Fatalf("enabled conditions = %#v, want an explicit empty array", conditions)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	if err := testWebClient(server).SaveAppTaxCategory(context.Background(), "app-1", "cat-games", nil, false); err != nil {
		t.Fatalf("SaveAppTaxCategory() error = %v", err)
	}
}

func TestSaveAppTaxCategoryUsesCapturedPatchBodyForConfiguredApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/appTaxCategories/app-1" {
			t.Fatalf("unexpected app tax update request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		data := taxCategoryWriteMap(t, body, "data")
		if data["type"] != "appTaxCategories" || data["id"] != "app-1" {
			t.Fatalf("patch data = %#v", data)
		}
		if _, ok := data["app"]; ok {
			t.Fatalf("PATCH must omit app relationship: %#v", data)
		}
		relationships := taxCategoryWriteMap(t, data, "relationships")
		category := taxCategoryWriteMap(t, relationships, "category")
		categoryData := taxCategoryWriteMap(t, category, "data")
		if categoryData["type"] != "taxCategories" || categoryData["id"] != "cat-games" {
			t.Fatalf("category relationship = %#v", categoryData)
		}
		conditions := taxCategoryWriteMap(t, relationships, "enabledConditions")
		conditionData, ok := conditions["data"].([]any)
		if !ok || len(conditionData) != 2 {
			t.Fatalf("enabled conditions = %#v", conditions)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := testWebClient(server).SaveAppTaxCategory(context.Background(), "app-1", "cat-games", []string{"cond-1", "cond-2"}, true); err != nil {
		t.Fatalf("SaveAppTaxCategory() error = %v", err)
	}
}

func taxCategoryWriteMap(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", key, parent[key])
	}
	return value
}

func TestGetAppTaxCategoryRejectsUnexpectedResourceIdentity(t *testing.T) {
	for _, resource := range []struct{ name, id, kind string }{{"wrong app", "app-2", "appTaxCategories"}, {"wrong type", "app-1", "apps"}, {"missing type", "app-1", ""}} {
		t.Run(resource.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != http.MethodGet || r.URL.Path != "/appTaxCategories/app-1" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": resource.id, "type": resource.kind, "relationships": map[string]any{"category": map[string]any{"data": map[string]string{"id": "cat-games", "type": "taxCategories"}}}}}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			result, err := testWebClient(server).GetAppTaxCategory(context.Background(), "app-1")
			if err == nil || result != nil || requests != 1 {
				t.Fatalf("accepted unexpected tax resource: result=%+v error=%v requests=%d", result, err, requests)
			}
		})
	}
}
