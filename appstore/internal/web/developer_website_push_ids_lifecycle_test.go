package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

const websitePushIDDetailResponse = `{
	"data": {
		"type": "websitepushIds",
		"id": "5D2243QPXH",
		"attributes": {
			"identifier": "web.example.com",
			"name": "Example Website",
			"canEdit": true,
			"canDelete": true,
			"responseId": "response-1"
		},
		"relationships": {
			"websitepushIdCapabilities": {
				"meta": {"paging": {"total": 0, "limit": 2147483647}},
				"data": []
			}
		}
	},
	"links": {"self": "/websitepushIds/5D2243QPXH"},
	"providerExtension": {"keep": true}
}`

func TestGetDeveloperWebsitePushIDUsesCapturedDetailContract(t *testing.T) {
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2:
			if request.Method != http.MethodPost || request.URL.Path != developerServicesPath+"/websitepushIds/5D2243QPXH" {
				t.Fatalf("unexpected detail request %s %s", request.Method, request.URL.String())
			}
			if request.Header.Get("X-HTTP-Method-Override") != http.MethodGet {
				t.Fatalf("method override = %q, want GET", request.Header.Get("X-HTTP-Method-Override"))
			}
			if request.URL.Query().Get("include") != "websitepushIdCapabilities" {
				t.Fatalf("include = %q", request.URL.Query().Get("include"))
			}
			if got := mustReadBody(t, request); !bytes.Equal(bytes.TrimSpace(got), []byte(`{"teamId":"TEAM123456"}`)) {
				t.Fatalf("detail body = %s", got)
			}
			return developerPortalTestResponse(http.StatusOK, websitePushIDDetailResponse, nil), nil
		default:
			t.Fatalf("unexpected request %d: %s %s", requestNumber, request.Method, request.URL.String())
			return nil, nil
		}
	})

	result, err := client.GetDeveloperWebsitePushID(context.Background(), "5D2243QPXH")
	if err != nil {
		t.Fatalf("GetDeveloperWebsitePushID() error: %v", err)
	}
	if result.Data.ID != "5D2243QPXH" || result.Data.Type != "websitepushIds" {
		t.Fatalf("unexpected data: %+v", result.Data)
	}
	if result.Data.Attributes["identifier"] != "web.example.com" {
		t.Fatalf("unexpected attributes: %+v", result.Data.Attributes)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	if !strings.Contains(string(encoded), `"providerExtension":{"keep":true}`) {
		t.Fatalf("raw JSON envelope was not preserved: %s", encoded)
	}
}

func TestCreateDeveloperWebsitePushIDUsesCapturedContractAndVerifiesResource(t *testing.T) {
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2:
			if request.URL.Path != "/services-account/QH65B2/account/ios/identifiers/listWebsitePushIds.action" {
				t.Fatalf("unexpected preflight list request %s", request.URL.String())
			}
			return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[]}`, nil), nil
		case 3:
			if request.Method != http.MethodPost || request.URL.Path != developerServicesPath+"/capabilities" {
				t.Fatalf("unexpected capabilities request %s %s", request.Method, request.URL.String())
			}
			if request.Header.Get("X-HTTP-Method-Override") != http.MethodGet {
				t.Fatalf("capability method override = %q", request.Header.Get("X-HTTP-Method-Override"))
			}
			if request.URL.Query().Get("filter[referenceType]") != "websitepushId" {
				t.Fatalf("capability filter = %q", request.URL.Query().Get("filter[referenceType]"))
			}
			if got := mustReadBody(t, request); !bytes.Equal(bytes.TrimSpace(got), []byte(`{"teamId":"TEAM123456"}`)) {
				t.Fatalf("capability body = %s", got)
			}
			return developerPortalTestResponse(http.StatusOK, `{"data":[]}`, nil), nil
		case 4:
			if request.Method != http.MethodPost || request.URL.Path != developerServicesPath+"/websitepushIds" {
				t.Fatalf("unexpected create request %s %s", request.Method, request.URL.String())
			}
			if request.Header.Get("X-HTTP-Method-Override") != "" {
				t.Fatalf("create unexpectedly used method override %q", request.Header.Get("X-HTTP-Method-Override"))
			}
			var payload struct {
				Data struct {
					Type          string            `json:"type"`
					Attributes    map[string]string `json:"attributes"`
					Relationships map[string]struct {
						Data []json.RawMessage `json:"data"`
					} `json:"relationships"`
				} `json:"data"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			if payload.Data.Type != "websitepushIds" || payload.Data.Attributes["name"] != "Example Website" || payload.Data.Attributes["identifier"] != "web.example.com" || payload.Data.Attributes["teamId"] != "TEAM123456" {
				t.Fatalf("create payload = %+v", payload.Data)
			}
			relationship, ok := payload.Data.Relationships["websitepushIdCapabilities"]
			if !ok || len(relationship.Data) != 0 {
				t.Fatalf("create capability relationship = %+v", payload.Data.Relationships)
			}
			return developerPortalTestResponse(http.StatusCreated, `{"data":{"type":"websitepushIds","id":"5D2243QPXH"}}`, nil), nil
		case 5:
			if request.URL.Path != developerServicesPath+"/websitepushIds/5D2243QPXH" || request.Header.Get("X-HTTP-Method-Override") != http.MethodGet {
				t.Fatalf("unexpected verification detail request %s %s", request.Method, request.URL.String())
			}
			return developerPortalTestResponse(http.StatusOK, websitePushIDDetailResponse, nil), nil
		default:
			t.Fatalf("unexpected request %d: %s %s", requestNumber, request.Method, request.URL.String())
			return nil, nil
		}
	})

	result, err := client.CreateDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDCreateRequest{Name: "Example Website", Identifier: "web.example.com"})
	if err != nil {
		t.Fatalf("CreateDeveloperWebsitePushID() error: %v", err)
	}
	if result == nil || result.WebsitePushID != "5D2243QPXH" || result.Identifier != "web.example.com" || !result.Changed || !result.Verified || result.Status != "created" {
		t.Fatalf("unexpected create receipt: %+v", result)
	}
}

func TestCreateDeveloperWebsitePushIDRejectsIncompleteLegacyPage(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response string
	}{
		{"smaller full page", `{"resultCode":0,"pageNumber":1,"pageSize":1,"websitePushIdList":[{"websitePushId":"web.other.example","name":"Other"}]}`},
		{"missing page size", `{"resultCode":0,"pageNumber":1,"websitePushIdList":[]}`},
		{"missing page number", `{"resultCode":0,"pageSize":1000,"websitePushIdList":[]}`},
		{"later page", `{"resultCode":0,"pageNumber":2,"pageSize":1000,"websitePushIdList":[]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
				calls++
				if requestNumber == 1 {
					return assertDeveloperPortalBootstrap(t, request), nil
				}
				if requestNumber == 2 {
					return developerPortalTestResponse(http.StatusOK, tc.response, nil), nil
				}
				return nil, errors.New("unexpected request after incomplete list")
			})
			result, err := client.CreateDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDCreateRequest{Name: "Example Website", Identifier: "web.example.com"})
			if err == nil || !strings.Contains(err.Error(), "page") || result != nil || calls != 2 {
				t.Fatalf("create result=%+v error=%v requests=%d, want page refusal after bootstrap/list only", result, err, calls)
			}
		})
	}
}

func TestCreateDeveloperWebsitePushIDRejectsUnrecognizedCollisionRows(t *testing.T) {
	for _, row := range []string{`{"name":"Unknown"}`, `{"id":"RESOURCE123","name":"Unknown"}`, `{"websitePushId":123,"name":"Unknown"}`} {
		t.Run(row, func(t *testing.T) {
			calls := 0
			client := newDeveloperAppGroupsTestClient(t, func(n int, request *http.Request) (*http.Response, error) {
				calls++
				if n == 1 {
					return assertDeveloperPortalBootstrap(t, request), nil
				}
				if n == 2 {
					return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[`+row+`]}`, nil), nil
				}
				return nil, errors.New("unexpected request after unrecognized row")
			})
			result, err := client.CreateDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDCreateRequest{Name: "Example Website", Identifier: "web.example.com"})
			if err == nil || !strings.Contains(err.Error(), "identifier") || calls != 2 || result != nil {
				t.Fatalf("result=%+v error=%v requests=%d, want refusal before mutation", result, err, calls)
			}
		})
	}
}

func TestCreateDeveloperWebsitePushIDFailsClosedOnUnknownCapabilityGraph(t *testing.T) {
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2:
			return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[]}`, nil), nil
		case 3:
			return developerPortalTestResponse(http.StatusOK, `{"data":[{"type":"capabilities","id":"cap-1"}]}`, nil), nil
		default:
			t.Fatalf("capability graph refusal must not send request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.CreateDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDCreateRequest{Name: "Example Website", Identifier: "web.example.com"})
	if err == nil || !strings.Contains(err.Error(), "capability") {
		t.Fatalf("expected capability graph refusal, got %v", err)
	}
}

func TestCreateDeveloperWebsitePushIDSettlesEmptyCreateResponseFromLegacyList(t *testing.T) {
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2:
			return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[]}`, nil), nil
		case 3:
			return developerPortalTestResponse(http.StatusOK, `{"data":[]}`, nil), nil
		case 4:
			if request.URL.Path != developerServicesPath+"/websitepushIds" || request.Method != http.MethodPost {
				t.Fatalf("unexpected create request %s %s", request.Method, request.URL.String())
			}
			return developerPortalTestResponse(http.StatusCreated, "", nil), nil
		case 5:
			if request.URL.Path != "/services-account/QH65B2/account/ios/identifiers/listWebsitePushIds.action" {
				t.Fatalf("unexpected settlement list request %s", request.URL.String())
			}
			return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[{"id":"5D2243QPXH","websitePushId":"web.example.com","identifier":"web.example.com","name":"Example Website"}]}`, nil), nil
		case 6:
			if request.URL.Path != developerServicesPath+"/websitepushIds/5D2243QPXH" || request.Header.Get("X-HTTP-Method-Override") != http.MethodGet {
				t.Fatalf("unexpected verification detail request %s %s", request.Method, request.URL.String())
			}
			return developerPortalTestResponse(http.StatusOK, websitePushIDDetailResponse, nil), nil
		default:
			t.Fatalf("empty create response must not retry the POST; unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	result, err := client.CreateDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDCreateRequest{Name: "Example Website", Identifier: "web.example.com"})
	if err != nil {
		t.Fatalf("CreateDeveloperWebsitePushID() error: %v", err)
	}
	if result == nil || result.WebsitePushID != "5D2243QPXH" || !result.Verified {
		t.Fatalf("unexpected settled create receipt: %+v", result)
	}
}

func TestCreateDeveloperWebsitePushIDDoesNotPromoteLegacyIdentifierToResourceID(t *testing.T) {
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2:
			return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[]}`, nil), nil
		case 3:
			return developerPortalTestResponse(http.StatusOK, `{"data":[]}`, nil), nil
		case 4:
			return developerPortalTestResponse(http.StatusCreated, "", nil), nil
		case 5:
			return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[{"websitePushId":"web.example.com","name":"Example Website"}]}`, nil), nil
		default:
			t.Fatalf("legacy identifier must not be sent as a modern resource ID; unexpected request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.CreateDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDCreateRequest{Name: "Example Website", Identifier: "web.example.com"})
	var unverified *DeveloperWebsitePushIDUnverifiedError
	if !errors.As(err, &unverified) || !strings.Contains(err.Error(), "opaque resource ID") {
		t.Fatalf("expected unverified identifier-only settlement, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "may have succeeded") || !strings.Contains(err.Error(), "before retrying") {
		t.Fatalf("unverified create must explain the uncertain outcome and retry guidance: %v", err)
	}
}

func TestCreateDeveloperWebsitePushIDSettlesServerErrorWithoutRetrying(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusRequestTimeout} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
				switch requestNumber {
				case 1:
					return assertDeveloperPortalBootstrap(t, request), nil
				case 2:
					return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[]}`, nil), nil
				case 3:
					return developerPortalTestResponse(http.StatusOK, `{"data":[]}`, nil), nil
				case 4:
					return developerPortalTestResponse(status, `{"errors":[{"code":"TIMEOUT"}]}`, nil), nil
				case 5:
					return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[{"id":"5D2243QPXH","websitePushId":"web.example.com","identifier":"web.example.com","name":"Example Website"}]}`, nil), nil
				case 6:
					return developerPortalTestResponse(http.StatusOK, websitePushIDDetailResponse, nil), nil
				default:
					t.Fatalf("server-error create must settle with reads and never retry POST; request %d", requestNumber)
					return nil, nil
				}
			})

			result, err := client.CreateDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDCreateRequest{Name: "Example Website", Identifier: "web.example.com"})
			if err != nil {
				t.Fatalf("CreateDeveloperWebsitePushID() error: %v", err)
			}
			if result == nil || result.WebsitePushID != "5D2243QPXH" || !result.Verified {
				t.Fatalf("unexpected settled server-error create receipt: %+v", result)
			}
		})
	}
}

func TestCreateDeveloperWebsitePushIDRejectsWrongPostReadIdentity(t *testing.T) {
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2:
			return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[]}`, nil), nil
		case 3:
			return developerPortalTestResponse(http.StatusOK, `{"data":[]}`, nil), nil
		case 4:
			return developerPortalTestResponse(http.StatusCreated, `{"data":{"type":"websitepushIds","id":"5D2243QPXH"}}`, nil), nil
		case 5:
			wrongIdentity := strings.Replace(websitePushIDDetailResponse, `"identifier": "web.example.com"`, `"identifier": "web.other.example"`, 1)
			return developerPortalTestResponse(http.StatusOK, wrongIdentity, nil), nil
		default:
			t.Fatalf("unexpected request %d after wrong post-read identity", requestNumber)
			return nil, nil
		}
	})

	_, err := client.CreateDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDCreateRequest{Name: "Example Website", Identifier: "web.example.com"})
	var unverified *DeveloperWebsitePushIDUnverifiedError
	if !errors.As(err, &unverified) || !strings.Contains(err.Error(), "identifier") {
		t.Fatalf("expected unverified wrong-identity result, got %T: %v", err, err)
	}
}

func TestDeleteDeveloperWebsitePushIDUsesCapturedContractAndVerifiesAbsence(t *testing.T) {
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2:
			if request.URL.Path != developerServicesPath+"/websitepushIds/5D2243QPXH" || request.Header.Get("X-HTTP-Method-Override") != http.MethodGet {
				t.Fatalf("unexpected preflight detail request %s %s", request.Method, request.URL.String())
			}
			return developerPortalTestResponse(http.StatusOK, websitePushIDDetailResponse, nil), nil
		case 3:
			if request.Method != http.MethodPost || request.URL.Path != developerServicesPath+"/websitepushIds/5D2243QPXH" || request.Header.Get("X-HTTP-Method-Override") != http.MethodDelete {
				t.Fatalf("unexpected delete request %s %s override=%q", request.Method, request.URL.String(), request.Header.Get("X-HTTP-Method-Override"))
			}
			if got := mustReadBody(t, request); !bytes.Equal(bytes.TrimSpace(got), []byte(`{"teamId":"TEAM123456"}`)) {
				t.Fatalf("delete body = %s", got)
			}
			return developerPortalTestResponse(http.StatusNoContent, "", nil), nil
		case 4:
			if request.URL.Path != developerServicesPath+"/websitepushIds/5D2243QPXH" || request.Header.Get("X-HTTP-Method-Override") != http.MethodGet {
				t.Fatalf("unexpected post-delete detail request %s %s", request.Method, request.URL.String())
			}
			return developerPortalTestResponse(http.StatusNotFound, `{}`, nil), nil
		case 5:
			if request.URL.Path != "/services-account/QH65B2/account/ios/identifiers/listWebsitePushIds.action" {
				t.Fatalf("unexpected final list request %s", request.URL.String())
			}
			return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[]}`, nil), nil
		default:
			t.Fatalf("unexpected request %d: %s %s", requestNumber, request.Method, request.URL.String())
			return nil, nil
		}
	})

	result, err := client.DeleteDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDDeleteRequest{WebsitePushID: "5D2243QPXH"})
	if err != nil {
		t.Fatalf("DeleteDeveloperWebsitePushID() error: %v", err)
	}
	if result == nil || result.WebsitePushID != "5D2243QPXH" || result.Identifier != "web.example.com" || !result.Changed || !result.Verified || result.Status != "deleted" {
		t.Fatalf("unexpected delete receipt: %+v", result)
	}
}

func TestDeleteDeveloperWebsitePushIDRefusesNonEmptyCapabilityRelationship(t *testing.T) {
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2:
			return developerPortalTestResponse(http.StatusOK, strings.Replace(websitePushIDDetailResponse, `"data": []`, `"data": [{"type":"websitepushIdCapabilities","id":"cap-1"}]`, 1), nil), nil
		default:
			t.Fatalf("capability graph refusal must not send request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.DeleteDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDDeleteRequest{WebsitePushID: "5D2243QPXH"})
	if err == nil || !strings.Contains(err.Error(), "capability") {
		t.Fatalf("expected capability graph refusal, got %v", err)
	}
}

func TestDeleteDeveloperWebsitePushIDRejectsStillReadablePostcondition(t *testing.T) {
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2, 4:
			return developerPortalTestResponse(http.StatusOK, websitePushIDDetailResponse, nil), nil
		case 3:
			return developerPortalTestResponse(http.StatusNoContent, "", nil), nil
		default:
			t.Fatalf("still-readable delete must stop without retry/list request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.DeleteDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDDeleteRequest{WebsitePushID: "5D2243QPXH"})
	var unverified *DeveloperWebsitePushIDUnverifiedError
	if !errors.As(err, &unverified) || !strings.Contains(err.Error(), "still readable") {
		t.Fatalf("expected unverified still-readable result, got %T: %v", err, err)
	}
}

func TestDeleteDeveloperWebsitePushIDSettlesServerErrorAndPreservesName(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusRequestTimeout} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
				switch requestNumber {
				case 1:
					return assertDeveloperPortalBootstrap(t, request), nil
				case 2:
					return developerPortalTestResponse(http.StatusOK, websitePushIDDetailResponse, nil), nil
				case 3:
					return developerPortalTestResponse(status, `{"errors":[{"code":"TIMEOUT"}]}`, nil), nil
				case 4:
					return developerPortalTestResponse(http.StatusNotFound, `{}`, nil), nil
				case 5:
					return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[]}`, nil), nil
				default:
					t.Fatalf("server-error delete must settle with reads and never retry DELETE; request %d", requestNumber)
					return nil, nil
				}
			})

			result, err := client.DeleteDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDDeleteRequest{WebsitePushID: "5D2243QPXH"})
			if err != nil {
				t.Fatalf("DeleteDeveloperWebsitePushID() error: %v", err)
			}
			if result == nil || result.Name != "Example Website" || !result.Verified {
				t.Fatalf("unexpected settled server-error delete receipt: %+v", result)
			}
		})
	}
}

func TestWebsitePushIDMutationReceiptUsesRegisteredOutputRenderer(t *testing.T) {
	result := &asc.WebWebsitePushIDMutationResult{Operation: "create", WebsitePushID: "5D2243QPXH", Identifier: "web.example.com", Name: "Example Website", Changed: true, Verified: true, Status: "created"}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	if !strings.Contains(string(encoded), `"websitePushId":"5D2243QPXH"`) {
		t.Fatalf("receipt JSON = %s", encoded)
	}
}

func TestWebsitePushIDRequestValidation(t *testing.T) {
	for _, request := range []DeveloperWebsitePushIDCreateRequest{{}, {Name: "Example", Identifier: "web/example"}, {Name: "Example\tName", Identifier: "web.example.com"}} {
		if _, err := request.validate(); err == nil {
			t.Fatalf("request %+v unexpectedly validated", request)
		}
	}
	if _, err := (DeveloperWebsitePushIDDeleteRequest{}).validate(); err == nil {
		t.Fatal("empty delete request unexpectedly validated")
	}
	if _, err := (DeveloperWebsitePushIDCreateRequest{Name: "Example", Identifier: "web.example.com"}).validate(); err != nil {
		t.Fatalf("valid create request rejected: %v", err)
	}
	if _, err := (DeveloperWebsitePushIDDeleteRequest{WebsitePushID: "5D2243QPXH"}).validate(); err != nil {
		t.Fatalf("valid delete request rejected: %v", err)
	}
}

func TestDeleteDeveloperWebsitePushIDRejectsUnrecognizedPostListIdentifier(t *testing.T) {
	for _, row := range []string{`{"id":"OTHER_RESOURCE"}`, `{"id":"OTHER_RESOURCE","identifier":42}`, `{"id":"OTHER_RESOURCE","websitePushId":null}`} {
		t.Run(row, func(t *testing.T) {
			requests, deletes := 0, 0
			client := newDeveloperAppGroupsTestClient(t, func(number int, request *http.Request) (*http.Response, error) {
				requests++
				if request.Header.Get("X-HTTP-Method-Override") == http.MethodDelete {
					deletes++
				}
				switch number {
				case 1:
					return assertDeveloperPortalBootstrap(t, request), nil
				case 2:
					return developerPortalTestResponse(http.StatusOK, websitePushIDDetailResponse, nil), nil
				case 3:
					return developerPortalTestResponse(http.StatusNoContent, "", nil), nil
				case 4:
					return developerPortalTestResponse(http.StatusNotFound, `{}`, nil), nil
				case 5:
					return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[`+row+`]}`, nil), nil
				default:
					t.Fatalf("unexpected request %d", number)
					return nil, nil
				}
			})
			result, err := client.DeleteDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDDeleteRequest{WebsitePushID: "5D2243QPXH"})
			var unverified *DeveloperWebsitePushIDUnverifiedError
			if result != nil || !errors.As(err, &unverified) || requests != 5 || deletes != 1 {
				t.Fatalf("expected unverified delete with no retry: result=%+v error=%v requests=%d deletes=%d", result, err, requests, deletes)
			}
		})
	}
}

func TestWebsitePushIDMutationsRejectIncompleteCapabilityPages(t *testing.T) {
	for _, metadata := range []string{`"meta":{"paging":{"total":1}}`, `"meta":{"paging":{"total":"unknown"}}`, `"links":{"next":"/more-capabilities"}`} {
		for _, operation := range []string{"create", "delete"} {
			t.Run(operation+"/"+metadata, func(t *testing.T) {
				requests := 0
				client := newDeveloperAppGroupsTestClient(t, func(number int, request *http.Request) (*http.Response, error) {
					requests++
					if number == 1 {
						return assertDeveloperPortalBootstrap(t, request), nil
					}
					if operation == "create" && number == 2 {
						return developerPortalTestResponse(http.StatusOK, `{"resultCode":0,"pageNumber":1,"pageSize":1000,"websitePushIdList":[]}`, nil), nil
					}
					if operation == "create" && number == 3 {
						return developerPortalTestResponse(http.StatusOK, `{"data":[],`+metadata+`}`, nil), nil
					}
					if operation == "delete" && number == 2 {
						body := strings.Replace(websitePushIDDetailResponse, `"meta": {"paging": {"total": 0, "limit": 2147483647}}`, metadata, 1)
						if body == websitePushIDDetailResponse {
							t.Error("fixture replacement failed")
							return nil, errors.New("fixture replacement failed")
						}
						return developerPortalTestResponse(http.StatusOK, body, nil), nil
					}
					return nil, errors.New("unexpected mutation after incomplete capability page")
				})
				var result *asc.WebWebsitePushIDMutationResult
				var err error
				wantRequests := 2
				if operation == "create" {
					wantRequests = 3
					result, err = client.CreateDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDCreateRequest{Name: "Example Website", Identifier: "web.example.com"})
				} else {
					result, err = client.DeleteDeveloperWebsitePushID(context.Background(), DeveloperWebsitePushIDDeleteRequest{WebsitePushID: "5D2243QPXH"})
				}
				if err == nil || result != nil || requests != wantRequests {
					t.Fatalf("expected preflight refusal: result=%+v error=%v requests=%d want=%d", result, err, requests, wantRequests)
				}
			})
		}
	}
}
