package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCreateSubscriptionPlanPricesBuildsExpectedInlinePatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/iris/v1/subscriptions/sub-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			Data     jsonAPIResource   `json:"data"`
			Included []jsonAPIResource `json:"included"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		prices := parseRelationshipRefs(body.Data.Relationships["prices"].Data)
		if len(prices) != 2 || len(body.Included) != 2 {
			t.Fatalf("expected two inline prices, got refs=%#v included=%#v", prices, body.Included)
		}
		if got := stringAttr(body.Included[0].Attributes, "planType"); got != "UPFRONT" {
			t.Fatalf("expected UPFRONT first, got %q", got)
		}
		if got := stringAttr(body.Included[1].Attributes, "planType"); got != "MONTHLY" {
			t.Fatalf("expected MONTHLY second, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"subscriptions","id":"sub-1"}}`))
	}))
	t.Cleanup(server.Close)

	client := &Client{httpClient: server.Client(), baseURL: server.URL + "/iris/v1"}
	got, err := client.CreateSubscriptionPlanPrices(context.Background(), "sub-1", "upfront-point", "monthly-point")
	if err != nil {
		t.Fatalf("CreateSubscriptionPlanPrices() error = %v", err)
	}
	if got.SubscriptionID != "sub-1" || got.UpfrontPricePointID != "upfront-point" || got.MonthlyPricePointID != "monthly-point" {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestSetSubscriptionPlanPricesIncludesScheduleAttributes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Included []jsonAPIResource `json:"included"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(body.Included) != 2 {
			t.Fatalf("expected two included prices, got %d", len(body.Included))
		}
		for _, price := range body.Included {
			if got := stringAttr(price.Attributes, "startDate"); got != "2026-07-01" {
				t.Fatalf("startDate = %q", got)
			}
			if got, ok := price.Attributes["preserveCurrentPrice"].(bool); !ok || !got {
				t.Fatalf("preserveCurrentPrice = %#v", price.Attributes["preserveCurrentPrice"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"type":"subscriptions","id":"sub-1"}}`))
	}))
	t.Cleanup(server.Close)

	client := &Client{httpClient: server.Client(), baseURL: server.URL + "/iris/v1"}
	_, err := client.SetSubscriptionPlanPrices(context.Background(), "sub-1", []SubscriptionPlanPrice{
		{PlanType: "UPFRONT", PricePointID: "upfront", StartDate: "2026-07-01", PreserveCurrentPrice: true},
		{PlanType: "MONTHLY", PricePointID: "monthly", StartDate: "2026-07-01", PreserveCurrentPrice: true},
	})
	if err != nil {
		t.Fatalf("SetSubscriptionPlanPrices() error = %v", err)
	}
}

func TestListSubscriptionPricesBuildsExpectedRequest(t *testing.T) {
	var gotPath string
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{
			"type":"subscriptionPrices",
			"id":"price-1",
			"attributes":{"planType":"UPFRONT","startDate":"2026-07-01T00:00:00.000Z","preserved":true},
			"relationships":{
				"territory":{"data":{"type":"territories","id":"NOR"}},
				"subscriptionPricePoint":{"data":{"type":"subscriptionPricePoints","id":"upfront-point"}}
			}
		}]}`))
	}))
	t.Cleanup(server.Close)

	client := &Client{httpClient: server.Client(), baseURL: server.URL + "/iris/v1"}
	got, err := client.ListSubscriptionPrices(context.Background(), "sub-1", "nor")
	if err != nil {
		t.Fatalf("ListSubscriptionPrices() error = %v", err)
	}
	if gotPath != "/iris/v1/subscriptions/sub-1/prices" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.Contains(gotQuery, "filter%5Bterritory%5D=NOR") || !strings.Contains(gotQuery, "include=subscriptionPricePoint") {
		t.Fatalf("unexpected query: %q", gotQuery)
	}
	if len(got) != 1 || got[0].ID != "price-1" || got[0].PlanType != "UPFRONT" || got[0].PricePointID != "upfront-point" {
		t.Fatalf("unexpected prices: %#v", got)
	}
	if got[0].StartDate != "2026-07-01" || !got[0].Preserved || got[0].Territory != "NOR" {
		t.Fatalf("unexpected decoded schedule: %#v", got[0])
	}
}

func TestFindSubscriptionPriceMatchesPlanTerritoryPointAndStartDate(t *testing.T) {
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC)
	prices := []SubscriptionPrice{
		{ID: "current", PlanType: "UPFRONT", Territory: "NOR", PricePointID: "upfront-point"},
		{ID: "scheduled", PlanType: "UPFRONT", Territory: "NOR", PricePointID: "upfront-point", StartDate: "2026-07-01T00:00:00Z"},
		{ID: "monthly", PlanType: "MONTHLY", Territory: "NOR", PricePointID: "monthly-point", StartDate: "2026-07-01"},
	}
	got, ok := FindSubscriptionPrice(prices, "UPFRONT", "NOR", "upfront-point", "2026-07-01", now)
	if !ok || got.ID != "scheduled" {
		t.Fatalf("scheduled match = %#v ok=%t", got, ok)
	}
	if _, ok := FindSubscriptionPrice(prices, "UPFRONT", "NOR", "stale-point", "", now); ok {
		t.Fatal("stale price point should not match")
	}
}

func TestFindSubscriptionPriceSelectsLatestNonFutureRecordWhenStartDateOmitted(t *testing.T) {
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC)
	prices := []SubscriptionPrice{
		{ID: "obsolete-empty", PlanType: "UPFRONT", Territory: "NOR", PricePointID: "old-point"},
		{ID: "effective", PlanType: "UPFRONT", Territory: "NOR", PricePointID: "upfront-point", StartDate: "2026-08-01"},
		{ID: "future", PlanType: "UPFRONT", Territory: "NOR", PricePointID: "future-point", StartDate: "2026-12-01"},
	}
	got, ok := FindSubscriptionPrice(prices, "UPFRONT", "NOR", "upfront-point", "", now)
	if !ok || got.ID != "effective" {
		t.Fatalf("effective current match = %#v ok=%t", got, ok)
	}
	if _, ok := FindSubscriptionPrice(prices, "UPFRONT", "NOR", "old-point", "", now); ok {
		t.Fatal("obsolete empty-date record must not verify the current price")
	}
}

func TestFindSubscriptionPricePrefersNonPreservedOnEqualEffectiveDate(t *testing.T) {
	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC)
	preserved := SubscriptionPrice{ID: "preserved", PlanType: "UPFRONT", Territory: "NOR", PricePointID: "old-point", StartDate: "2026-08-01", Preserved: true}
	canonical := SubscriptionPrice{ID: "canonical", PlanType: "UPFRONT", Territory: "NOR", PricePointID: "upfront-point", StartDate: "2026-08-01", Preserved: false}
	for _, prices := range [][]SubscriptionPrice{
		{preserved, canonical},
		{canonical, preserved},
	} {
		got, ok := FindSubscriptionPrice(prices, "UPFRONT", "NOR", "upfront-point", "", now)
		if !ok || got.ID != "canonical" {
			t.Fatalf("canonical match = %#v ok=%t (order %#v)", got, ok, prices)
		}
		if _, ok := FindSubscriptionPrice(prices, "UPFRONT", "NOR", "old-point", "", now); ok {
			t.Fatal("same-day preserved record must not verify when a canonical price exists")
		}
	}
}

func TestListSubscriptionPricesFollowsNextPage(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.RawQuery, "cursor=page-2"):
			_, _ = w.Write([]byte(`{"data":[{
				"type":"subscriptionPrices","id":"price-2",
				"attributes":{"planType":"MONTHLY"},
				"relationships":{
					"territory":{"data":{"type":"territories","id":"NOR"}},
					"subscriptionPricePoint":{"data":{"type":"subscriptionPricePoints","id":"monthly-point"}}
				}
			}]}`))
		default:
			_, _ = w.Write([]byte(`{"data":[{
				"type":"subscriptionPrices","id":"price-1",
				"attributes":{"planType":"UPFRONT"},
				"relationships":{
					"territory":{"data":{"type":"territories","id":"NOR"}},
					"subscriptionPricePoint":{"data":{"type":"subscriptionPricePoints","id":"upfront-point"}}
				}
			}],"links":{"next":"/iris/v1/subscriptions/sub-1/prices?cursor=page-2"}}`))
		}
	}))
	t.Cleanup(server.Close)

	client := &Client{httpClient: server.Client(), baseURL: server.URL + "/iris/v1"}
	got, err := client.ListSubscriptionPrices(context.Background(), "sub-1", "NOR")
	if err != nil {
		t.Fatalf("ListSubscriptionPrices() error = %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("requests = %d, want 2 (%v)", len(paths), paths)
	}
	if len(got) != 2 || got[0].ID != "price-1" || got[1].ID != "price-2" {
		t.Fatalf("unexpected prices: %#v", got)
	}
}
