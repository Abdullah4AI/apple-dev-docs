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

func TestNormalizeAppAvailabilityCreateAttributes(t *testing.T) {
	t.Run("normalizes and deduplicates territory ids", func(t *testing.T) {
		attrs, err := normalizeAppAvailabilityCreateAttributes(AppAvailabilityCreateAttributes{
			AppID:                " app-123 ",
			AvailableTerritories: []string{" usa ", "gbr", "USA", ""},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if attrs.AppID != "app-123" {
			t.Fatalf("expected trimmed app id, got %q", attrs.AppID)
		}
		if got := strings.Join(attrs.AvailableTerritories, ","); got != "GBR,USA" {
			t.Fatalf("expected normalized territories GBR,USA, got %q", got)
		}
	})

	t.Run("requires app id", func(t *testing.T) {
		_, err := normalizeAppAvailabilityCreateAttributes(AppAvailabilityCreateAttributes{
			AvailableTerritories: []string{"USA"},
		})
		if err == nil || !strings.Contains(err.Error(), "app id is required") {
			t.Fatalf("expected missing app id error, got %v", err)
		}
	})

	t.Run("requires at least one territory", func(t *testing.T) {
		_, err := normalizeAppAvailabilityCreateAttributes(AppAvailabilityCreateAttributes{
			AppID:                "app-123",
			AvailableTerritories: []string{"", "   "},
		})
		if err == nil || !strings.Contains(err.Error(), "at least one available territory is required") {
			t.Fatalf("expected missing territory error, got %v", err)
		}
	})
}

func TestCreateAppAvailabilityBuildsExpectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/appAvailabilities" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		data := body["data"].(map[string]any)
		attributes := data["attributes"].(map[string]any)
		if got := attributes["availableInNewTerritories"]; got != false {
			t.Fatalf("expected availableInNewTerritories=false, got %#v", got)
		}
		relationships := data["relationships"].(map[string]any)
		app := relationships["app"].(map[string]any)["data"].(map[string]any)
		if app["id"] != "app-123" || app["type"] != "apps" {
			t.Fatalf("unexpected app relationship: %#v", app)
		}
		territories := relationships["availableTerritories"].(map[string]any)["data"].([]any)
		if len(territories) != 2 {
			t.Fatalf("expected 2 territories, got %d", len(territories))
		}
		first := territories[0].(map[string]any)
		second := territories[1].(map[string]any)
		if first["id"] != "GBR" || second["id"] != "USA" {
			t.Fatalf("expected sorted territory ids GBR/USA, got %#v %#v", first, second)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"id": "avail-123",
				"type": "appAvailabilities",
				"attributes": {"availableInNewTerritories": false},
				"relationships": {
					"availableTerritories": {
						"data": [
							{"type": "territories", "id": "GBR"},
							{"type": "territories", "id": "USA"}
						]
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client := testWebClient(server)
	created, err := client.CreateAppAvailability(context.Background(), AppAvailabilityCreateAttributes{
		AppID:                     "app-123",
		AvailableInNewTerritories: false,
		AvailableTerritories:      []string{"usa", "gbr"},
	})
	if err != nil {
		t.Fatalf("CreateAppAvailability() error = %v", err)
	}
	if created == nil {
		t.Fatal("expected created app availability")
		return
	}
	if created.ID != "avail-123" {
		t.Fatalf("expected id avail-123, got %q", created.ID)
	}
	if created.AvailableInNewTerritories {
		t.Fatal("expected availableInNewTerritories=false")
	}
	if got := strings.Join(created.AvailableTerritories, ","); got != "GBR,USA" {
		t.Fatalf("expected decoded territories GBR,USA, got %q", got)
	}
}

func TestGetAppAvailabilityBuildsExpectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apps/app-123/appAvailabilityV2" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"id": "avail-123",
				"type": "appAvailabilities",
				"attributes": {"availableInNewTerritories": true},
				"relationships": {
					"availableTerritories": {
						"data": [
							{"type": "territories", "id": "USA"},
							{"type": "territories", "id": "GBR"}
						]
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client := testWebClient(server)
	got, err := client.GetAppAvailability(context.Background(), "app-123")
	if err != nil {
		t.Fatalf("GetAppAvailability() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected app availability")
		return
	}
	if got.ID != "avail-123" || !got.AvailableInNewTerritories {
		t.Fatalf("unexpected app availability payload: %#v", got)
	}
	if joined := strings.Join(got.AvailableTerritories, ","); joined != "GBR,USA" {
		t.Fatalf("expected sorted territories GBR,USA, got %q", joined)
	}
	if !got.AvailableTerritoriesLoaded {
		t.Fatal("expected availableTerritories relationship to be marked loaded")
	}
}

func TestGetAppAvailabilityMarksEmptyTerritoriesLoaded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"id": "avail-123",
				"type": "appAvailabilities",
				"attributes": {"availableInNewTerritories": false},
				"relationships": {
					"availableTerritories": {"data": []}
				}
			}
		}`))
	}))
	defer server.Close()

	client := testWebClient(server)
	got, err := client.GetAppAvailability(context.Background(), "app-123")
	if err != nil {
		t.Fatalf("GetAppAvailability() error = %v", err)
	}
	if !got.AvailableTerritoriesLoaded {
		t.Fatal("expected empty availableTerritories.data to count as loaded")
	}
	if !got.AvailableInNewTerritoriesKnown {
		t.Fatal("expected availableInNewTerritories=false to count as known")
	}
	if len(got.AvailableTerritories) != 0 {
		t.Fatalf("expected no territories, got %#v", got.AvailableTerritories)
	}
}

func TestGetAppAvailabilityLeavesNewTerritoriesUnknownWhenAttributeOmitted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"id": "avail-123",
				"type": "appAvailabilities",
				"attributes": {},
				"relationships": {
					"availableTerritories": {"data": []}
				}
			}
		}`))
	}))
	defer server.Close()

	client := testWebClient(server)
	got, err := client.GetAppAvailability(context.Background(), "app-123")
	if err != nil {
		t.Fatalf("GetAppAvailability() error = %v", err)
	}
	if !got.AvailableTerritoriesLoaded {
		t.Fatal("expected empty availableTerritories.data to count as loaded")
	}
	if got.AvailableInNewTerritoriesKnown {
		t.Fatal("omitted availableInNewTerritories must not count as known")
	}
}

func TestGetAppAvailabilityReadsPaginatedTerritoryAvailabilities(t *testing.T) {
	fixture := handlertest.New(t)
	const appID = "6759231657"
	availabilityPath := "/apps/" + appID + "/appAvailabilityV2"
	territoriesPath := "/iris/v2/appAvailabilities/" + appID + "/territoryAvailabilities"

	nextLink := ""
	var territoryPages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case availabilityPath:
			if r.Method != http.MethodGet {
				fixture.Respond(w, "unexpected method: %s", r.Method)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {
					"type": "appAvailabilities",
					"id": "6759231657",
					"attributes": {"availableInNewTerritories": false},
					"relationships": {
						"territoryAvailabilities": {
							"links": {
								"self": "https://appstoreconnect.apple.com/iris/v2/appAvailabilities/6759231657/relationships/territoryAvailabilities",
								"related": "https://appstoreconnect.apple.com/iris/v2/appAvailabilities/6759231657/territoryAvailabilities"
							}
						}
					}
				}
			}`))
		case territoriesPath:
			if r.Method != http.MethodGet {
				fixture.Respond(w, "unexpected method: %s", r.Method)
				return
			}
			query := r.URL.Query()
			if query.Get("cursor") == "" {
				if got := query.Get("include"); got != "territory" {
					fixture.Respond(w, "expected include=territory, got %q", got)
					return
				}
				if got := query.Get("limit"); got != "200" {
					fixture.Respond(w, "expected limit=200, got %q", got)
					return
				}
			}
			territoryPages = append(territoryPages, r.URL.RawQuery)
			w.Header().Set("Content-Type", "application/json")
			if query.Get("cursor") == "BQ" {
				_, _ = w.Write([]byte(`{
					"data": [
						{
							"type": "territoryAvailabilities",
							"id": "eyJzIjoiNjc1OTIzMTY1NyIsInQiOiJHQlIifQ",
							"attributes": {"available": true},
							"relationships": {"territory": {"data": {"type": "territories", "id": "GBR"}}}
						},
						{
							"type": "territoryAvailabilities",
							"id": "eyJzIjoiNjc1OTIzMTY1NyIsInQiOiJDQU4ifQ",
							"attributes": {"available": false},
							"relationships": {"territory": {"data": {"type": "territories", "id": "CAN"}}}
						}
					],
					"links": {
						"self": "https://appstoreconnect.apple.com/iris/v2/appAvailabilities/6759231657/territoryAvailabilities?cursor=BQ&include=territory&limit=200"
					}
				}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"type": "territoryAvailabilities",
						"id": "eyJzIjoiNjc1OTIzMTY1NyIsInQiOiJVU0EifQ",
						"attributes": {"available": true},
						"relationships": {"territory": {"data": {"type": "territories", "id": "USA"}}}
					},
					{
						"type": "territoryAvailabilities",
						"id": "eyJzIjoiNjc1OTIzMTY1NyIsInQiOiJGUkEifQ",
						"attributes": {"available": false},
						"relationships": {"territory": {"data": {"type": "territories", "id": "FRA"}}}
					}
				],
				"links": {
					"self": "https://appstoreconnect.apple.com/iris/v2/appAvailabilities/6759231657/territoryAvailabilities",
					"next": "` + nextLink + `"
				}
			}`))
		default:
			fixture.Respond(w, "unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	nextLink = server.URL + territoriesPath + "?cursor=BQ&include=territory&limit=200"

	client := testWebClient(server)
	got, err := client.GetAppAvailability(context.Background(), appID)
	if err != nil {
		t.Fatalf("GetAppAvailability() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected app availability")
	}
	if !got.AvailableTerritoriesLoaded {
		t.Fatal("expected territoryAvailabilities pagination to mark territories loaded")
	}
	if !got.AvailableInNewTerritoriesKnown || got.AvailableInNewTerritories {
		t.Fatalf("expected availableInNewTerritories=false and known, got %#v", got)
	}
	if joined := strings.Join(got.AvailableTerritories, ","); joined != "GBR,USA" {
		t.Fatalf("expected sorted available territories GBR,USA, got %q", joined)
	}
	if len(territoryPages) != 2 {
		t.Fatalf("expected two territoryAvailabilities pages, got %d (%v)", len(territoryPages), territoryPages)
	}
}

func TestGetAppAvailabilityFailsWhenTerritoryAvailabilitiesUnreadable(t *testing.T) {
	fixture := handlertest.New(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apps/app-123/appAvailabilityV2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "app-123",
					"type": "appAvailabilities",
					"attributes": {"availableInNewTerritories": false},
					"relationships": {
						"territoryAvailabilities": {
							"links": {
								"related": "https://appstoreconnect.apple.com/iris/v2/appAvailabilities/app-123/territoryAvailabilities"
							}
						}
					}
				}
			}`))
		case "/iris/v2/appAvailabilities/app-123/territoryAvailabilities":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"status":"500","code":"UNEXPECTED","title":"boom"}]}`))
		default:
			fixture.Respond(w, "unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	_, err := testWebClient(server).GetAppAvailability(context.Background(), "app-123")
	if err == nil {
		t.Fatal("expected territoryAvailabilities read to fail closed")
	}
	if !strings.Contains(err.Error(), "territoryAvailabilities") {
		t.Fatalf("expected error to name territoryAvailabilities, got %v", err)
	}
}

func TestGetAppAvailabilityFailsWhenTerritoryAvailabilityOmitsAvailable(t *testing.T) {
	fixture := handlertest.New(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apps/app-123/appAvailabilityV2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "app-123",
					"type": "appAvailabilities",
					"attributes": {"availableInNewTerritories": false}
				}
			}`))
		case "/iris/v2/appAvailabilities/app-123/territoryAvailabilities":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": [{
					"type": "territoryAvailabilities",
					"id": "ta-usa",
					"attributes": {},
					"relationships": {"territory": {"data": {"type": "territories", "id": "USA"}}}
				}],
				"links": {"self": "https://appstoreconnect.apple.com/iris/v2/appAvailabilities/app-123/territoryAvailabilities"}
			}`))
		default:
			fixture.Respond(w, "unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	_, err := testWebClient(server).GetAppAvailability(context.Background(), "app-123")
	if err == nil {
		t.Fatal("expected missing available boolean to fail closed")
	}
	if !strings.Contains(err.Error(), "available") {
		t.Fatalf("expected error to name the available attribute, got %v", err)
	}
}

func TestGetAppAvailabilityFailsWhenTerritoryAvailabilitiesDataMissing(t *testing.T) {
	tests := map[string]string{
		"field omitted": `{}`,
		"null":          `{"data":null}`,
	}
	for name, relatedResponse := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := handlertest.New(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/apps/app-123/appAvailabilityV2":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{
						"data": {
							"id": "avail-123",
							"type": "appAvailabilities",
							"attributes": {"availableInNewTerritories": false},
							"relationships": {
								"territoryAvailabilities": {
									"links": {"related": "https://appstoreconnect.apple.com/iris/v2/appAvailabilities/avail-123/territoryAvailabilities"}
								}
							}
						}
					}`))
				case "/iris/v2/appAvailabilities/avail-123/territoryAvailabilities":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(relatedResponse))
				default:
					fixture.Respond(w, "unexpected path: %s", r.URL.Path)
				}
			}))
			defer server.Close()

			_, err := testWebClient(server).GetAppAvailability(context.Background(), "app-123")
			if err == nil {
				t.Fatal("expected missing territoryAvailabilities data to fail closed")
			}
			if !strings.Contains(err.Error(), "data") {
				t.Fatalf("expected error to name missing data, got %v", err)
			}
		})
	}
}

func TestGetAppAvailabilityFailsWhenTerritoryAvailabilitiesLinksMissing(t *testing.T) {
	tests := map[string]string{
		"field omitted": `{"data": [{"id":"ta-usa","type":"territoryAvailabilities","attributes":{"available":true},"relationships":{"territory":{"data":{"type":"territories","id":"USA"}}}}]}`,
		"null":          `{"data": [{"id":"ta-usa","type":"territoryAvailabilities","attributes":{"available":true},"relationships":{"territory":{"data":{"type":"territories","id":"USA"}}}}],"links":null}`,
		"self omitted":  `{"data": [{"id":"ta-usa","type":"territoryAvailabilities","attributes":{"available":true},"relationships":{"territory":{"data":{"type":"territories","id":"USA"}}}}],"links":{}}`,
	}
	for name, relatedResponse := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := handlertest.New(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/apps/app-123/appAvailabilityV2":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{
						"data": {
							"id": "avail-123",
							"type": "appAvailabilities",
							"attributes": {"availableInNewTerritories": false},
							"relationships": {
								"territoryAvailabilities": {
									"links": {"related": "https://appstoreconnect.apple.com/iris/v2/appAvailabilities/avail-123/territoryAvailabilities"}
								}
							}
						}
					}`))
				case "/iris/v2/appAvailabilities/avail-123/territoryAvailabilities":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(relatedResponse))
				default:
					fixture.Respond(w, "unexpected path: %s", r.URL.Path)
				}
			}))
			defer server.Close()

			_, err := testWebClient(server).GetAppAvailability(context.Background(), "app-123")
			if err == nil {
				t.Fatal("expected missing territoryAvailabilities links to fail closed")
			}
			if !strings.Contains(err.Error(), "links") {
				t.Fatalf("expected error to name missing links, got %v", err)
			}
		})
	}
}

func TestGetAppAvailabilityDoesNotTreatRelatedNotFoundAsMissing(t *testing.T) {
	fixture := handlertest.New(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apps/app-123/appAvailabilityV2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": {
					"id": "avail-123",
					"type": "appAvailabilities",
					"attributes": {"availableInNewTerritories": false},
					"relationships": {
						"territoryAvailabilities": {
							"links": {"related": "https://appstoreconnect.apple.com/iris/v2/appAvailabilities/avail-123/territoryAvailabilities"}
						}
					}
				}
			}`))
		case "/iris/v2/appAvailabilities/avail-123/territoryAvailabilities":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"status":"404","code":"NOT_FOUND","title":"not found"}]}`))
		default:
			fixture.Respond(w, "unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	_, err := testWebClient(server).GetAppAvailability(context.Background(), "app-123")
	if err == nil {
		t.Fatal("expected related collection read to fail")
	}
	if IsNotFound(err) {
		t.Fatalf("related collection 404 must not be treated as missing app availability: %v", err)
	}
}

func TestGetAppAvailabilityPrimaryNotFoundRemainsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apps/app-123/appAvailabilityV2" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"status":"404","code":"NOT_FOUND","title":"not found"}]}`))
	}))
	defer server.Close()

	_, err := testWebClient(server).GetAppAvailability(context.Background(), "app-123")
	if err == nil {
		t.Fatal("expected primary availability read to fail")
	}
	if !IsNotFound(err) {
		t.Fatalf("primary availability 404 must remain not found: %v", err)
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(&APIError{Status: http.StatusNotFound}) {
		t.Fatal("expected 404 APIError to be treated as not found")
	}
	if IsNotFound(&APIError{Status: http.StatusConflict}) {
		t.Fatal("did not expect 409 APIError to be treated as not found")
	}
}
