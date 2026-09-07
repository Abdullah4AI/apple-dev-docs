package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func TestListDeveloperICloudContainersUsesCapturedModernProxyContract(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		t.Run("hidden="+strings.ToLower(strconv.FormatBool(hidden)), func(t *testing.T) {
			client := developerPortalTestClient(t, func(r *http.Request) (*http.Response, error) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == developerPortalTeamsPath:
					return developerPortalTestResponse(http.StatusOK, developerPortalTeamsFixture(), nil), nil
				case r.Method == http.MethodPost && r.URL.Path == developerServicesPath+"/cloudContainers":
					if got := r.Header.Get("X-HTTP-Method-Override"); got != http.MethodGet {
						t.Fatalf("method override = %q, want GET", got)
					}
					if got := r.URL.Query().Get("filter[AND][hidden]"); got != strconv.FormatBool(hidden) {
						t.Fatalf("hidden filter = %q, want %q", got, strconv.FormatBool(hidden))
					}
					request := decodeDeveloperPortalProxyReadRequest(t, r)
					if request.TeamID != "TEAM123456" {
						t.Fatalf("teamId = %q, want TEAM123456", request.TeamID)
					}
					if request.URLEncodedQueryParams != "limit=1000&offset=0&sort=name" {
						t.Fatalf("urlEncodedQueryParams = %q, want captured query", request.URLEncodedQueryParams)
					}
					query, err := url.ParseQuery(request.URLEncodedQueryParams)
					if err != nil {
						t.Fatalf("parse urlEncodedQueryParams: %v", err)
					}
					for key, want := range map[string]string{"limit": "1000", "offset": "0", "sort": "name"} {
						if got := query.Get(key); got != want {
							t.Errorf("query %s = %q, want %q", key, got, want)
						}
					}
					if query.Get("filter[AND][hidden]") != "" {
						t.Fatal("hidden filter must remain in the request URL, not the encoded body query")
					}
					return developerPortalTestResponse(http.StatusOK, `{
						"data":[{
							"type":"cloudContainers",
							"id":"cloud-1",
							"attributes":{
								"identifier":"iCloud.com.example.app",
								"hidden":false,
								"prefix":"TEAM123456",
								"canEdit":true,
								"name":"Example Container",
								"canDelete":false,
								"responseId":"response-1"
							},
							"links":{"self":"/cloudContainers/cloud-1"}
						}],
						"links":{"self":"/cloudContainers"},
						"meta":{"paging":{"total":20,"limit":1000}},
						"unknownTopLevel":{"keep":true}
					}`, nil), nil
				default:
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
					return nil, nil
				}
			})

			result, err := client.ListDeveloperICloudContainers(context.Background(), hidden)
			if err != nil {
				t.Fatalf("ListDeveloperICloudContainers() error: %v", err)
			}
			if len(result.Data) != 1 || result.Data[0].ID != "cloud-1" || result.Data[0].Type != "cloudContainers" {
				t.Fatalf("unexpected data: %+v", result.Data)
			}
			container := result.Data[0]
			if container.Attributes.Identifier != "iCloud.com.example.app" || container.Attributes.Name != "Example Container" {
				t.Fatalf("unexpected attributes: %+v", container.Attributes)
			}
			if !container.Attributes.CanEdit || container.Attributes.CanDelete {
				t.Fatalf("unexpected permissions: %+v", container.Attributes)
			}
			if result.Meta["paging"].(map[string]any)["total"] != float64(20) {
				t.Fatalf("unexpected paging metadata: %#v", result.Meta)
			}
			if !strings.Contains(string(result.Raw), `"unknownTopLevel":{"keep":true}`) {
				t.Fatalf("raw response omitted unknown top-level field: %s", result.Raw)
			}
			encoded, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("marshal result: %v", err)
			}
			assertCompactJSONEqual(t, encoded, result.Raw)
		})
	}
}

func TestListDeveloperICloudContainersRejectsMalformedEnvelope(t *testing.T) {
	client := developerPortalTestClient(t, func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == developerPortalTeamsPath:
			return developerPortalTestResponse(http.StatusOK, developerPortalTeamsFixture(), nil), nil
		case r.Method == http.MethodPost && r.URL.Path == developerServicesPath+"/cloudContainers":
			return developerPortalTestResponse(http.StatusOK, `{"data":[{"type":"wrong","id":"cloud-1"}]}`, nil), nil
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
			return nil, nil
		}
	})

	_, err := client.ListDeveloperICloudContainers(context.Background(), false)
	if err == nil || !strings.Contains(err.Error(), "cloud container resource") {
		t.Fatalf("error = %v, want invalid cloud container resource", err)
	}
}
