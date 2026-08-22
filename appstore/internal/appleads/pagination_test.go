package appleads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestPaginateAllPlatformGeoUsesPageSizeAndAggregates(t *testing.T) {
	spec, ok := PlatformEndpointByCommandPath("geo", "search")
	if !ok {
		t.Fatal("missing geo search endpoint")
	}
	requests := []url.Values{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests = append(requests, req.URL.Query())
		switch req.URL.Query().Get("offset") {
		case "5":
			_, _ = w.Write([]byte(`{"result":[{"id":"geo-6"},{"id":"geo-7"}],"pagination":{"totalCount":9,"offset":5,"pageSize":2}}`))
		case "7":
			_, _ = w.Write([]byte(`{"result":[{"id":"geo-8"},{"id":"geo-9"}],"pagination":{"totalCount":9,"offset":7,"pageSize":2}}`))
		default:
			t.Errorf("unexpected offset %q", req.URL.Query().Get("offset"))
			http.Error(w, "unexpected offset", http.StatusBadRequest)
		}
	}))
	defer server.Close()
	client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "account"}, WithPlatformBaseURL(server.URL+"/v1/"))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	raw, err := client.PaginateAll(context.Background(), spec, nil, url.Values{"supplySource": {"APPSTORE"}, "query": {"San Francisco"}}, 5, 2, nil)
	if err != nil {
		t.Fatalf("PaginateAll() error: %v", err)
	}
	var got platformPaginatedEnvelope
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	if len(got.Result) != 4 {
		t.Fatalf("result count = %d, want 4", len(got.Result))
	}
	if got.Pagination.TotalCount == nil || *got.Pagination.TotalCount != 9 {
		t.Fatalf("totalCount = %v, want 9", got.Pagination.TotalCount)
	}
	if got.Pagination.Offset != 5 || got.Pagination.PageSize != 2 {
		t.Fatalf("pagination = %+v, want offset 5 pageSize 2", got.Pagination)
	}
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(requests))
	}
	for index, wantOffset := range []string{"5", "7"} {
		if got := requests[index].Get("offset"); got != wantOffset {
			t.Errorf("request[%d] offset = %q, want %q", index, got, wantOffset)
		}
		if got := requests[index].Get("pageSize"); got != "2" {
			t.Errorf("request[%d] pageSize = %q, want 2", index, got)
		}
		if _, present := requests[index]["limit"]; present {
			t.Errorf("request[%d] unexpectedly set legacy limit: %v", index, requests[index])
		}
	}
}

func TestPaginateAllPlatformGeoStopsOnEmptyPageWithoutTotalCount(t *testing.T) {
	spec, ok := PlatformEndpointByCommandPath("geo", "search")
	if !ok {
		t.Fatal("missing geo search endpoint")
	}
	offsets := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		offset := req.URL.Query().Get("offset")
		offsets = append(offsets, offset)
		switch offset {
		case "0":
			_, _ = w.Write([]byte(`{"result":[{"id":"geo-1"},{"id":"geo-2"}],"pagination":{"offset":0,"pageSize":2}}`))
		case "2":
			_, _ = w.Write([]byte(`{"result":[],"pagination":{"offset":2,"pageSize":2}}`))
		default:
			t.Errorf("unexpected offset %q", offset)
			http.Error(w, "unexpected offset", http.StatusBadRequest)
		}
	}))
	defer server.Close()
	client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "account"}, WithPlatformBaseURL(server.URL+"/v1/"))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	raw, err := client.PaginateAll(context.Background(), spec, nil, url.Values{"supplySource": {"APPSTORE"}}, 0, 2, nil)
	if err != nil {
		t.Fatalf("PaginateAll() error: %v", err)
	}
	var got platformPaginatedEnvelope
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	if len(got.Result) != 2 {
		t.Fatalf("result count = %d, want 2", len(got.Result))
	}
	if got.Pagination.TotalCount != nil {
		t.Fatalf("totalCount = %v, want omitted", got.Pagination.TotalCount)
	}
	if !reflect.DeepEqual(offsets, []string{"0", "2"}) {
		t.Fatalf("offsets = %v, want [0 2]", offsets)
	}
}

func TestPaginateAllPlatformGETCapsUnboundedPages(t *testing.T) {
	spec, ok := PlatformEndpointByCommandPath("geo", "search")
	if !ok {
		t.Fatal("missing geo search endpoint")
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"result":[{"id":"geo"}],"pagination":{"offset":` + req.URL.Query().Get("offset") + `,"pageSize":1}}`))
	}))
	defer server.Close()
	client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "account"}, WithPlatformBaseURL(server.URL+"/v1/"))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.PaginateAll(context.Background(), spec, nil, url.Values{"supplySource": {"APPSTORE"}}, 0, 1, nil)
	if err == nil {
		t.Fatal("PaginateAll() unexpectedly succeeded for an unbounded result")
	}
	if got, want := requests, MaxPlatformPaginationPages; got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}
	for _, want := range []string{"1000-page safety limit", "narrow your query", "--offset"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want it to contain %q", err, want)
		}
	}
}

func TestPaginateAllPlatformGETStopsWhenContextCanceled(t *testing.T) {
	spec, ok := PlatformEndpointByCommandPath("geo", "search")
	if !ok {
		t.Fatal("missing geo search endpoint")
	}
	requests := 0
	ctx, cancel := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		if requests != 1 {
			t.Errorf("request count = %d, want cancellation before a second request", requests)
		}
		_, _ = w.Write([]byte(`{"result":[{"id":"geo"}],"pagination":{"offset":0,"pageSize":1}}`))
		cancel()
	}))
	defer server.Close()
	client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "account"}, WithPlatformBaseURL(server.URL+"/v1/"))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.PaginateAll(ctx, spec, nil, url.Values{"supplySource": {"APPSTORE"}}, 0, 1, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PaginateAll() error = %v, want context.Canceled", err)
	}
	if requests != 1 {
		t.Fatalf("request count = %d, want 1", requests)
	}
}

func TestPaginateAllPlatformChangeHistoryMergesNestedChanges(t *testing.T) {
	spec, ok := PlatformEndpointByCommandPath("change-history", "view")
	if !ok {
		t.Fatal("missing change-history view endpoint")
	}
	var requests []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests = append(requests, req.URL.Query())
		switch req.URL.Query().Get("offset") {
		case "0":
			_, _ = w.Write([]byte(`{"dataType":"ChangeDetail","pagination":{"totalCount":3,"offset":0,"pageSize":2},"result":[{"detailId":"Campaign.1.txn","details":[{"transactionId":"txn","changes":[{"field":"name"},{"field":"status"}]}]}]}`))
		case "2":
			_, _ = w.Write([]byte(`{"dataType":"ChangeDetail","pagination":{"totalCount":3,"offset":2,"pageSize":2},"result":[{"detailId":"Campaign.1.txn","details":[{"transactionId":"txn","changes":[{"field":"dailyBudget"}]}]}]}`))
		default:
			t.Fatalf("unexpected offset %q", req.URL.Query().Get("offset"))
		}
	}))
	defer server.Close()
	client, err := NewClient(Credentials{AccessToken: "ACCESS", AdAccountID: "account"}, WithPlatformBaseURL(server.URL+"/v1/"))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	raw, err := client.PaginateAll(context.Background(), spec, map[string]string{"detailId": "Campaign.1.txn"}, nil, 0, 2, nil)
	if err != nil {
		t.Fatalf("PaginateAll() error: %v", err)
	}
	var got struct {
		DataType   string             `json:"dataType"`
		Pagination platformPageDetail `json:"pagination"`
		Result     []struct {
			DetailID string `json:"detailId"`
			Details  []struct {
				TransactionID string            `json:"transactionId"`
				Changes       []json.RawMessage `json:"changes"`
			} `json:"details"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}
	if got.DataType != "ChangeDetail" || len(got.Result) != 1 || len(got.Result[0].Details) != 1 || len(got.Result[0].Details[0].Changes) != 3 {
		t.Fatalf("aggregated response = %+v", got)
	}
	if got.Pagination.TotalCount == nil || *got.Pagination.TotalCount != 3 || got.Pagination.Offset != 0 || got.Pagination.PageSize != 2 {
		t.Fatalf("pagination = %+v", got.Pagination)
	}
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(requests))
	}
	for index, wantOffset := range []string{"0", "2"} {
		if got := requests[index].Get("offset"); got != wantOffset {
			t.Errorf("request[%d] offset = %q, want %q", index, got, wantOffset)
		}
		if got := requests[index].Get("limit"); got != "2" {
			t.Errorf("request[%d] limit = %q, want 2", index, got)
		}
	}
}
