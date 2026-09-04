package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAppDistributionBuildsExpectedRequest(t *testing.T) {
	var gotPath string
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "apps",
				"id": "app-123",
				"attributes": {
					"name": "Example",
					"bundleId": "com.example.app",
					"distributionType": "CUSTOM",
					"educationDiscountType": "NOT_APPLICABLE"
				}
			}
		}`))
	}))
	defer server.Close()

	got, err := testWebClient(server).GetAppDistribution(context.Background(), "app-123")
	if err != nil {
		t.Fatalf("GetAppDistribution() error = %v", err)
	}
	if gotPath != "/apps/app-123" {
		t.Fatalf("path = %q, want /apps/app-123", gotPath)
	}
	if gotQuery != "" {
		t.Fatalf("query = %q, want empty", gotQuery)
	}
	if got.AppID != "app-123" {
		t.Fatalf("appID = %q, want app-123", got.AppID)
	}
	if got.DistributionType != "CUSTOM" {
		t.Fatalf("distributionType = %q, want CUSTOM", got.DistributionType)
	}
	if got.EducationDiscountType != "NOT_APPLICABLE" {
		t.Fatalf("educationDiscountType = %q, want NOT_APPLICABLE", got.EducationDiscountType)
	}
	if got.BundleID != "com.example.app" {
		t.Fatalf("bundleId = %q, want com.example.app", got.BundleID)
	}
}

func TestGetAppDistributionOmitsMissingAttributes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": {"type": "apps", "id": "app-456", "attributes": {"name": "Example"}}}`))
	}))
	defer server.Close()

	got, err := testWebClient(server).GetAppDistribution(context.Background(), "app-456")
	if err != nil {
		t.Fatalf("GetAppDistribution() error = %v", err)
	}
	if got.DistributionType != "" || got.EducationDiscountType != "" {
		t.Fatalf("expected empty distribution attributes, got %+v", got)
	}
	if got.AppID != "app-456" {
		t.Fatalf("appID = %q, want app-456", got.AppID)
	}
}

func TestGetAppDistributionRequiresAppID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL.Path)
	}))
	defer server.Close()

	if _, err := testWebClient(server).GetAppDistribution(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty app id")
	}
}

func TestGetAppDistributionPropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":[{"status":"403","code":"FORBIDDEN_ERROR","title":"Access denied","detail":"no access"}]}`))
	}))
	defer server.Close()

	if _, err := testWebClient(server).GetAppDistribution(context.Background(), "app-123"); err == nil {
		t.Fatal("expected error for 403 response")
	}
}
