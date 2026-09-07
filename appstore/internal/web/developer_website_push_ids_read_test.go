package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestListDeveloperWebsitePushIDsUsesCapturedPortalFormEndpoint(t *testing.T) {
	const body = `{
		"creationTimestamp":123,
		"resultCode":0,
		"userLocale":"en_US",
		"protocolVersion":"1.0",
		"requestUrl":"/services-account/QH65B2/account/ios/identifiers/listWebsitePushIds.action",
		"responseId":"response-1",
		"isAdmin":true,
		"isMember":true,
		"isAgent":false,
		"pageNumber":1,
		"pageSize":1000,
		"mapleId":"maple-1",
		"mapleIdList":[],
		"websitePushIdList":[{"websitePushId":"web.example.com","name":"Example Website","providerExtension":{"keep":true}}],
		"providerExtension":{"keep":true}
	}`
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2:
			if request.Method != http.MethodPost || request.URL.Path != "/services-account/QH65B2/account/ios/identifiers/listWebsitePushIds.action" {
				t.Fatalf("unexpected Website Push ID list request %s %s", request.Method, request.URL.String())
			}
			assertDeveloperPortalForm(t, request, url.Values{
				"onlyCountLists": {"true"},
				"pageSize":       {"1000"},
				"pageNumber":     {"1"},
				"sort":           {"name=asc"},
				"teamId":         {"TEAM123456"},
			})
			for _, key := range []string{"sidx", "search"} {
				if _, ok := request.PostForm[key]; ok {
					t.Fatalf("form unexpectedly contains %s: %v", key, request.PostForm)
				}
			}
			return developerPortalTestResponse(http.StatusOK, body, nil), nil
		default:
			t.Fatalf("unexpected request %d: %s %s", requestNumber, request.Method, request.URL.String())
			return nil, nil
		}
	})

	result, err := client.ListDeveloperWebsitePushIDs(context.Background())
	if err != nil {
		t.Fatalf("ListDeveloperWebsitePushIDs() error: %v", err)
	}
	if result == nil || result.ResultCode == nil || *result.ResultCode != 0 {
		t.Fatalf("unexpected result code: %+v", result)
	}
	if result.PageNumber == nil || *result.PageNumber != 1 || result.PageSize != 1000 {
		t.Fatalf("unexpected page metadata: %+v", result)
	}
	if len(result.WebsitePushIDList) != 1 || result.WebsitePushIDList[0]["websitePushId"] != "web.example.com" {
		t.Fatalf("unexpected Website Push ID list: %+v", result.WebsitePushIDList)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		t.Fatalf("decode JSON output %q: %v", encoded, err)
	}
	for _, member := range []string{"creationTimestamp", "mapleIdList", "providerExtension"} {
		if _, ok := envelope[member]; !ok {
			t.Fatalf("JSON output omitted provider field %q: %s", member, encoded)
		}
	}
	if got := string(envelope["providerExtension"]); got != `{"keep":true}` {
		t.Fatalf("providerExtension = %s, want {\"keep\":true}", got)
	}
}

func TestListDeveloperWebsitePushIDsRejectsMalformedCollection(t *testing.T) {
	for name, body := range map[string]string{
		"missing": `{"resultCode":0}`,
		"null":    `{"resultCode":0,"websitePushIdList":null}`,
		"object":  `{"resultCode":0,"websitePushIdList":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
				switch requestNumber {
				case 1:
					return assertDeveloperPortalBootstrap(t, request), nil
				case 2:
					return developerPortalTestResponse(http.StatusOK, body, nil), nil
				default:
					t.Fatalf("malformed Website Push ID response must not lead to request %d", requestNumber)
					return nil, nil
				}
			})
			_, err := client.ListDeveloperWebsitePushIDs(context.Background())
			if err == nil || !strings.Contains(err.Error(), "websitePushIdList") {
				t.Fatalf("expected websitePushIdList parse error, got %v", err)
			}
		})
	}
}

func TestListDeveloperWebsitePushIDsRejectsFailureResultCode(t *testing.T) {
	client := newDeveloperAppGroupsTestClient(t, func(requestNumber int, request *http.Request) (*http.Response, error) {
		switch requestNumber {
		case 1:
			return assertDeveloperPortalBootstrap(t, request), nil
		case 2:
			return developerPortalTestResponse(http.StatusOK, `{"resultCode":42,"requestId":"request-42","userString":"Team access denied","websitePushIdList":[]}`, nil), nil
		default:
			t.Fatalf("failure response must not lead to request %d", requestNumber)
			return nil, nil
		}
	})

	_, err := client.ListDeveloperWebsitePushIDs(context.Background())
	if err == nil || !strings.Contains(err.Error(), "result code 42") || !strings.Contains(err.Error(), "Team access denied") {
		t.Fatalf("expected result-code error, got %v", err)
	}
}
