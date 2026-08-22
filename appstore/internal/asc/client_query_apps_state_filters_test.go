package asc

import (
	"context"
	"net/http"
	"testing"
)

func TestGetApps_WithVersionAndReviewSubmissionStateFilters(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/apps" {
			t.Fatalf("expected path /v1/apps, got %s", req.URL.Path)
		}
		values := req.URL.Query()
		if got := values.Get("filter[appStoreVersions.appVersionState]"); got != "IN_REVIEW,WAITING_FOR_REVIEW" {
			t.Fatalf("expected filter[appStoreVersions.appVersionState]=IN_REVIEW,WAITING_FOR_REVIEW, got %q", got)
		}
		if got := values.Get("filter[reviewSubmissions.state]"); got != "IN_REVIEW" {
			t.Fatalf("expected filter[reviewSubmissions.state]=IN_REVIEW, got %q", got)
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetApps(
		context.Background(),
		WithAppsVersionStates([]string{"in_review", " waiting_for_review "}),
		WithAppsReviewSubmissionStates([]string{"in_review"}),
	); err != nil {
		t.Fatalf("GetApps() error: %v", err)
	}
}

func TestGetApps_OmitsStateFiltersWhenUnset(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		values := req.URL.Query()
		if _, ok := values["filter[appStoreVersions.appVersionState]"]; ok {
			t.Fatalf("expected no appVersionState filter, got %v", values)
		}
		if _, ok := values["filter[reviewSubmissions.state]"]; ok {
			t.Fatalf("expected no reviewSubmissions state filter, got %v", values)
		}
	}, response)

	if _, err := client.GetApps(
		context.Background(),
		WithAppsVersionStates(nil),
		WithAppsReviewSubmissionStates([]string{"   "}),
	); err != nil {
		t.Fatalf("GetApps() error: %v", err)
	}
}
