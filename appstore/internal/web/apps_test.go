package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeCreateAttrsDefaults(t *testing.T) {
	attrs, err := normalizeCreateAttrs(AppCreateAttributes{
		Name:     "My App",
		BundleID: "com.example.app",
		SKU:      "SKU123",
	})
	if err != nil {
		t.Fatalf("normalizeCreateAttrs error: %v", err)
	}
	if attrs.PrimaryLocale != defaultPrimaryLocale {
		t.Fatalf("expected default locale %q, got %q", defaultPrimaryLocale, attrs.PrimaryLocale)
	}
	if attrs.Platform != defaultPlatform {
		t.Fatalf("expected default platform %q, got %q", defaultPlatform, attrs.Platform)
	}
	if attrs.VersionString != defaultVersion {
		t.Fatalf("expected default version %q, got %q", defaultVersion, attrs.VersionString)
	}
}

func TestNormalizeCreateAttrsRejectsInvalidPlatform(t *testing.T) {
	_, err := normalizeCreateAttrs(AppCreateAttributes{
		Name:     "My App",
		BundleID: "com.example.app",
		SKU:      "SKU123",
		Platform: "WATCH_OS",
	})
	if err == nil {
		t.Fatal("expected invalid platform error")
	}
}

func TestBuildAppCreateRequestUsesLocalizationForName(t *testing.T) {
	req := buildAppCreateRequest(AppCreateAttributes{
		Name:          "My App",
		BundleID:      "com.example.app",
		SKU:           "SKU123",
		PrimaryLocale: "en-US",
		Platform:      "IOS",
		VersionString: "1.0",
	})

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	payload := string(raw)

	if strings.Contains(payload, `"attributes":{"name":"My App","sku"`) {
		t.Fatalf("expected name not to be part of top-level app attributes, payload=%s", payload)
	}
	if !strings.Contains(payload, `"appInfoLocalizations"`) {
		t.Fatalf("expected appInfoLocalization relationship, payload=%s", payload)
	}
	if !strings.Contains(payload, `"name":"My App"`) {
		t.Fatalf("expected localized app name in payload, payload=%s", payload)
	}
}

func TestFindAppEscapesBundleIDQuery(t *testing.T) {
	var gotRawQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}
	_, err := client.FindApp(context.Background(), "com.example/app?x=1")
	if err != nil {
		t.Fatalf("FindApp error: %v", err)
	}
	if strings.Contains(gotRawQuery, "com.example/app?x=1") {
		t.Fatalf("expected escaped query value, got raw query %q", gotRawQuery)
	}
	if !strings.Contains(gotRawQuery, "filter%5BbundleId%5D=") && !strings.Contains(gotRawQuery, "filter[bundleId]=") {
		t.Fatalf("expected bundleId filter query, got %q", gotRawQuery)
	}
}

func TestDeleteAppSendsRemovedPatch(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"apps","id":"1234567890","attributes":{"name":"Throwaway","bundleId":"com.example.throwaway","removed":true}}}`))
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}
	app, err := client.DeleteApp(context.Background(), "1234567890")
	if err != nil {
		t.Fatalf("DeleteApp error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Fatalf("expected PATCH, got %s", gotMethod)
	}
	if gotPath != "/apps/1234567890" {
		t.Fatalf("expected /apps/1234567890, got %s", gotPath)
	}
	for _, want := range []string{
		`"type":"apps"`,
		`"id":"1234567890"`,
		`"removed":true`,
	} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("expected request body to contain %s, got %s", want, gotBody)
		}
	}
	if app == nil || app.Data.ID != "1234567890" {
		t.Fatalf("expected app response id, got %+v", app)
	}
}

func TestDeleteAppRequiresID(t *testing.T) {
	client := &Client{}
	if _, err := client.DeleteApp(context.Background(), "  "); err == nil {
		t.Fatal("expected missing app id error")
	}
}

func TestListRemovedAppsUsesRemovedFilterAndDecodesVersions(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/iris/v1/apps" {
			t.Fatalf("expected /iris/v1/apps, got %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		query := r.URL.Query()
		if query.Get("filter[removed]") != "true" {
			t.Fatalf("expected removed filter, got query %q", r.URL.RawQuery)
		}
		if query.Get("limit") != "48" {
			t.Fatalf("expected limit=48, got query %q", r.URL.RawQuery)
		}
		if query.Get("include") != "appStoreIcon,displayableVersions" {
			t.Fatalf("expected displayable versions include, got query %q", r.URL.RawQuery)
		}
		if !strings.Contains(query.Get("fields[apps]"), "removed") {
			t.Fatalf("expected removed app fields, got query %q", r.URL.RawQuery)
		}
		if query.Get("limit[displayableVersions]") != "20" {
			t.Fatalf("expected displayable version limit, got query %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [{
				"type": "apps",
				"id": "1234567890",
				"attributes": {
					"name": "Throwaway",
					"bundleId": "com.example.throwaway",
					"primaryLocale": "en-US",
					"sku": "THROWAWAY",
					"removed": true,
					"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
					"marketplace": "APP_STORE"
				},
				"relationships": {
					"displayableVersions": {
						"data": [{"type": "appStoreVersions", "id": "version-1"}]
					}
				}
			}],
			"included": [{
				"type": "appStoreVersions",
				"id": "version-1",
				"attributes": {
					"platform": "IOS",
					"versionString": "1.0",
					"appStoreState": "PREPARE_FOR_SUBMISSION",
					"createdDate": "2026-07-06T10:00:00Z",
					"isWatchOnly": false
				}
			}],
			"links": {"self": "https://appstoreconnect.apple.com/iris/v1/apps?filter[removed]=true"}
		}`))
	}))
	defer server.Close()

	client := &Client{
		httpClient: server.Client(),
		baseURL:    server.URL + "/iris/v1",
	}
	result, err := client.ListRemovedApps(context.Background(), RemovedAppsListOptions{Limit: 48})
	if err != nil {
		t.Fatalf("ListRemovedApps error: %v", err)
	}
	if !strings.Contains(gotQuery, "filter%5Bremoved%5D=true") && !strings.Contains(gotQuery, "filter[removed]=true") {
		t.Fatalf("expected removed filter in raw query, got %q", gotQuery)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 app, got %d", len(result.Data))
	}
	app := result.Data[0]
	if app.ID != "1234567890" || app.Name != "Throwaway" || app.BundleID != "com.example.throwaway" || !app.Removed {
		t.Fatalf("unexpected app result: %+v", app)
	}
	if app.Status != "PREPARE_FOR_SUBMISSION" {
		t.Fatalf("expected displayable version status, got %q", app.Status)
	}
	if len(app.DisplayableVersions) != 1 {
		t.Fatalf("expected 1 displayable version, got %d", len(app.DisplayableVersions))
	}
	version := app.DisplayableVersions[0]
	if version.ID != "version-1" || version.Platform != "IOS" || version.VersionString != "1.0" {
		t.Fatalf("unexpected version result: %+v", version)
	}
}

func TestListRemovedAppsPaginatesNextLinks(t *testing.T) {
	requests := 0
	serverURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch requests {
		case 1:
			if r.URL.Query().Get("filter[removed]") != "true" {
				t.Fatalf("expected first page to use removed filter, got query %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{
				"data": [{
					"type": "apps",
					"id": "app-1",
					"attributes": {"name": "First", "removed": true}
				}],
				"links": {"next": "` + serverURL + `/iris/v1/apps?filter[removed]=true&page=2"}
			}`))
		case 2:
			if r.URL.Path != "/iris/v1/apps" || r.URL.Query().Get("filter[removed]") != "true" || r.URL.Query().Get("page") != "2" {
				t.Fatalf("expected second request to follow next link, got %s?%s", r.URL.Path, r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{
				"data": [{
					"type": "apps",
					"id": "app-2",
					"attributes": {"name": "Second", "removed": true}
				}],
				"links": {}
			}`))
		default:
			t.Fatalf("unexpected request %d", requests)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := &Client{
		httpClient: server.Client(),
		baseURL:    server.URL + "/iris/v1",
	}
	result, err := client.ListRemovedApps(context.Background(), RemovedAppsListOptions{Limit: 48, Paginate: true})
	if err != nil {
		t.Fatalf("ListRemovedApps error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(result.Data))
	}
	if result.Data[0].ID != "app-1" || result.Data[1].ID != "app-2" {
		t.Fatalf("unexpected app ids: %+v", result.Data)
	}
	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}
}

func TestListRemovedAppsRejectsNextWithoutRemovedFilter(t *testing.T) {
	client := &Client{
		httpClient: http.DefaultClient,
		baseURL:    "https://appstoreconnect.apple.com/iris/v1",
	}
	_, err := client.ListRemovedApps(context.Background(), RemovedAppsListOptions{
		Next: "https://appstoreconnect.apple.com/iris/v1/apps?limit=48",
	})
	if err == nil {
		t.Fatal("expected missing removed filter error")
	}
	if !strings.Contains(err.Error(), "filter[removed]=true") {
		t.Fatalf("expected removed filter error, got %v", err)
	}
}
