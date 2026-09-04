package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/handlertest"
)

func TestListReviewRejectionsRetainsIncludedContext(t *testing.T) {
	fixture := handlertest.New(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reviewRejections" {
			fixture.Respond(w, "unexpected path: %s", r.URL.Path)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{
				"id": "rej-1",
				"type": "reviewRejections",
				"attributes": {
					"reasons": [{
						"reasonSection": "2.1",
						"reasonDescription": "App crashed during review",
						"reasonCode": "2.1.0"
					}]
				},
				"relationships": {
					"appStoreVersion": {"data": {"type":"appStoreVersions","id":"v1"}},
					"build": {"data": {"type":"builds","id":"b1"}},
					"gameCenterAchievementVersions": {"data": {"type":"gameCenterAchievementVersions","id":"gc1"}},
					"rejectionAttachments": {"data": [{"type":"rejectionAttachments","id":"ratt-1"}]}
				}
			}],
			"included": [
				{"id":"v1","type":"appStoreVersions","attributes":{"versionString":"1.2.3","platform":"IOS"}},
				{"id":"b1","type":"builds","attributes":{"version":"45","uploadedDate":"2026-02-01T00:00:00Z"}},
				{"id":"gc1","type":"gameCenterAchievementVersions","attributes":{"version":3}},
				{"id":"ratt-1","type":"rejectionAttachments","attributes":{"fileName":"Crash.png","downloadUrl":"https://example.invalid/signed?token=secret"}}
			]
		}`))
	}))
	defer server.Close()

	got, err := testWebClient(server).ListReviewRejections(context.Background(), "thread-1")
	if err != nil {
		t.Fatalf("ListReviewRejections() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one rejection, got %#v", got)
	}
	rejection := got[0]
	if rejection.ID != "rej-1" {
		t.Fatalf("unexpected rejection id: %#v", rejection)
	}
	if len(rejection.AttachmentIDs) != 1 || rejection.AttachmentIDs[0] != "ratt-1" {
		t.Fatalf("expected existing attachmentIds to stay, got %#v", rejection.AttachmentIDs)
	}

	relatedByType := map[string]ReviewRelatedResource{}
	for _, related := range rejection.Related {
		relatedByType[related.Type] = related
	}
	version, ok := relatedByType["appStoreVersions"]
	if !ok || version.ID != "v1" || version.Label != "1.2.3" {
		t.Fatalf("expected included app version context, got %#v", rejection.Related)
	}
	build, ok := relatedByType["builds"]
	if !ok || build.ID != "b1" || build.Label != "45" {
		t.Fatalf("expected included build context, got %#v", rejection.Related)
	}
	achievement, ok := relatedByType["gameCenterAchievementVersions"]
	if !ok || achievement.ID != "gc1" || achievement.Label != "3" {
		t.Fatalf("expected numeric Game Center version label, got %#v", rejection.Related)
	}
	if _, ok := relatedByType["rejectionAttachments"]; ok {
		t.Fatalf("did not expect attachment resources in related context: %#v", rejection.Related)
	}
	if len(rejection.Related) != 3 {
		t.Fatalf("expected three related artifacts, got %#v", rejection.Related)
	}
	if rejection.Related[0].Relationship != "appStoreVersion" || rejection.Related[1].Relationship != "build" || rejection.Related[2].Relationship != "gameCenterAchievementVersions" {
		t.Fatalf("expected stable related order by relationship name, got %#v", rejection.Related)
	}

	encoded, err := json.Marshal(rejection)
	if err != nil {
		t.Fatalf("marshal rejection: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("unmarshal rejection json: %v", err)
	}
	for _, key := range []string{"id", "reasons", "attachmentIds", "related"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("expected json key %q, got %s", key, encoded)
		}
	}
	if strings.Contains(string(encoded), "downloadUrl") || strings.Contains(string(encoded), "example.invalid/signed") {
		t.Fatalf("signed attachment URL leaked into rejection json: %s", encoded)
	}
}

func TestListReviewSubmissionItemsOmitsRejectedExperimentV2Include(t *testing.T) {
	fixture := handlertest.New(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reviewSubmissions/sub-1/items" {
			fixture.Respond(w, "unexpected path: %s", r.URL.Path)
			return
		}
		include := r.URL.Query().Get("include")
		tokens := map[string]bool{}
		for _, token := range strings.Split(include, ",") {
			tokens[strings.TrimSpace(token)] = true
		}
		// iris GET /reviewSubmissions/{id}/items rejects this name with
		// PARAMETER_ERROR.INVALID (verified live 2026-09-03) even though the
		// public OpenAPI lists it.
		if tokens["appStoreVersionExperimentV2"] {
			fixture.Respond(w, "include must not contain appStoreVersionExperimentV2 (iris rejects it): %q", include)
			return
		}
		for _, want := range []string{
			"appStoreVersion",
			"appCustomProductPageVersion",
			"appStoreVersionExperiment",
			"appEvent",
			"backgroundAssetVersion",
			"gameCenterAchievementVersion",
			"gameCenterActivityVersion",
			"gameCenterChallengeVersion",
			"gameCenterLeaderboardSetVersion",
			"gameCenterLeaderboardVersion",
			"inAppPurchaseVersion",
			"subscriptionVersion",
			"subscriptionGroupVersion",
		} {
			if !tokens[want] {
				fixture.Respond(w, "missing include %s in %q", want, include)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	items, err := testWebClient(server).ListReviewSubmissionItems(context.Background(), "sub-1")
	if err != nil {
		t.Fatalf("ListReviewSubmissionItems() error = %v", err)
	}
	if items == nil {
		t.Fatal("expected empty items slice, got nil")
	}
}

func TestListReviewSubmissionItemsRetainsIncludedContext(t *testing.T) {
	fixture := handlertest.New(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reviewSubmissions/sub-1/items" {
			fixture.Respond(w, "unexpected path: %s", r.URL.Path)
			return
		}
		include := r.URL.Query().Get("include")
		for _, want := range []string{
			"inAppPurchaseVersion",
			"subscriptionVersion",
			"subscriptionGroupVersion",
		} {
			if !strings.Contains(include, want) {
				fixture.Respond(w, "missing include %s in %q", want, include)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [{
				"id": "item-1",
				"type": "reviewSubmissionItems",
				"relationships": {
					"appStoreVersion": {"data": {"type":"appStoreVersions","id":"v1"}},
					"appEvent": {"data": {"type":"appEvents","id":"e1"}},
					"inAppPurchaseVersion": {"data": {"type":"inAppPurchaseVersions","id":"iapv1"}}
				}
			}],
			"included": [
				{"id":"v1","type":"appStoreVersions","attributes":{"versionString":"2.0.0","platform":"IOS"}},
				{"id":"e1","type":"appEvents","attributes":{"name":"Spring Sale","referenceName":"spring-sale"}},
				{"id":"iapv1","type":"inAppPurchaseVersions","attributes":{"versionString":"3.1"}}
			]
		}`))
	}))
	defer server.Close()

	items, err := testWebClient(server).ListReviewSubmissionItems(context.Background(), "sub-1")
	if err != nil {
		t.Fatalf("ListReviewSubmissionItems() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item, got %#v", items)
	}

	relatedByType := map[string]ReviewSubmissionItemRelation{}
	for _, related := range items[0].Related {
		if related.Type == "" || related.ID == "" || related.Relationship == "" {
			t.Fatalf("expected existing relationship keys, got %#v", related)
		}
		relatedByType[related.Type] = related
	}
	if relatedByType["appStoreVersions"].Label != "2.0.0" {
		t.Fatalf("expected version label, got %#v", items[0].Related)
	}
	if relatedByType["appEvents"].Label != "Spring Sale" {
		t.Fatalf("expected event name label, got %#v", items[0].Related)
	}
	if relatedByType["inAppPurchaseVersions"].Label != "3.1" {
		t.Fatalf("expected IAP version label, got %#v", items[0].Related)
	}

	encoded, err := json.Marshal(items[0])
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("unmarshal item json: %v", err)
	}
	for _, key := range []string{"id", "type", "related"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("expected json key %q, got %s", key, encoded)
		}
	}
	related, _ := raw["related"].([]any)
	if len(related) == 0 {
		t.Fatalf("expected related array, got %s", encoded)
	}
	first, _ := related[0].(map[string]any)
	for _, key := range []string{"relationship", "type", "id"} {
		if _, ok := first[key]; !ok {
			t.Fatalf("expected related key %q to stay, got %s", key, encoded)
		}
	}
}
