package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/handlertest"
)

func TestReviewReadersFollowNextLinks(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		page1      func(next string) string
		page2      string
		wantIDs    []string
		collectIDs func(*Client) ([]string, error)
	}{
		{
			name: "submissions",
			path: "/apps/app-123/reviewSubmissions",
			page1: func(next string) string {
				return `{
					"data": [{"id":"sub-1","type":"reviewSubmissions","attributes":{"state":"COMPLETE"}}],
					"links": {"next": "` + next + `"}
				}`
			},
			page2:   `{"data": [{"id":"sub-2","type":"reviewSubmissions","attributes":{"state":"IN_REVIEW"}}]}`,
			wantIDs: []string{"sub-1", "sub-2"},
			collectIDs: func(client *Client) ([]string, error) {
				got, err := client.ListReviewSubmissions(context.Background(), "app-123")
				if err != nil {
					return nil, err
				}
				ids := make([]string, 0, len(got))
				for _, item := range got {
					ids = append(ids, item.ID)
				}
				return ids, nil
			},
		},
		{
			name: "threads",
			path: "/resolutionCenterThreads",
			page1: func(next string) string {
				return `{
					"data": [{"id":"thread-1","type":"resolutionCenterThreads","attributes":{"state":"OPEN"}}],
					"links": {"next": "` + next + `"}
				}`
			},
			page2:   `{"data": [{"id":"thread-2","type":"resolutionCenterThreads","attributes":{"state":"CLOSED"}}]}`,
			wantIDs: []string{"thread-1", "thread-2"},
			collectIDs: func(client *Client) ([]string, error) {
				got, err := client.ListResolutionCenterThreadsBySubmission(context.Background(), "sub-1")
				if err != nil {
					return nil, err
				}
				ids := make([]string, 0, len(got))
				for _, item := range got {
					ids = append(ids, item.ID)
				}
				return ids, nil
			},
		},
		{
			name: "messages",
			path: "/resolutionCenterThreads/thread-1/resolutionCenterMessages",
			page1: func(next string) string {
				return `{
					"data": [{"id":"m1","type":"resolutionCenterMessages","attributes":{"messageBody":"first"}}],
					"links": {"next": "` + next + `"}
				}`
			},
			page2:   `{"data": [{"id":"m2","type":"resolutionCenterMessages","attributes":{"messageBody":"second"}}]}`,
			wantIDs: []string{"m1", "m2"},
			collectIDs: func(client *Client) ([]string, error) {
				got, err := client.ListResolutionCenterMessages(context.Background(), "thread-1", false)
				if err != nil {
					return nil, err
				}
				ids := make([]string, 0, len(got))
				for _, item := range got {
					ids = append(ids, item.ID)
				}
				return ids, nil
			},
		},
		{
			name: "rejections",
			path: "/reviewRejections",
			page1: func(next string) string {
				return `{
					"data": [{"id":"rej-1","type":"reviewRejections","attributes":{"reasons":[{"reasonCode":"2.1.0"}]}}],
					"links": {"next": "` + next + `"}
				}`
			},
			page2:   `{"data": [{"id":"rej-2","type":"reviewRejections","attributes":{"reasons":[{"reasonCode":"4.0.0"}]}}]}`,
			wantIDs: []string{"rej-1", "rej-2"},
			collectIDs: func(client *Client) ([]string, error) {
				got, err := client.ListReviewRejections(context.Background(), "thread-1")
				if err != nil {
					return nil, err
				}
				ids := make([]string, 0, len(got))
				for _, item := range got {
					ids = append(ids, item.ID)
				}
				return ids, nil
			},
		},
		{
			name: "items",
			path: "/reviewSubmissions/sub-1/items",
			page1: func(next string) string {
				return `{
					"data": [{"id":"item-1","type":"reviewSubmissionItems"}],
					"links": {"next": "` + next + `"}
				}`
			},
			page2:   `{"data": [{"id":"item-2","type":"reviewSubmissionItems"}]}`,
			wantIDs: []string{"item-1", "item-2"},
			collectIDs: func(client *Client) ([]string, error) {
				got, err := client.ListReviewSubmissionItems(context.Background(), "sub-1")
				if err != nil {
					return nil, err
				}
				ids := make([]string, 0, len(got))
				for _, item := range got {
					ids = append(ids, item.ID)
				}
				return ids, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := handlertest.New(t)
			nextLink := ""
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					fixture.Respond(w, "unexpected path: %s", r.URL.Path)
					return
				}
				requests++
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Query().Get("cursor") == "page-2" {
					_, _ = w.Write([]byte(tt.page2))
					return
				}
				_, _ = w.Write([]byte(tt.page1(nextLink)))
			}))
			defer server.Close()
			nextLink = server.URL + tt.path + "?cursor=page-2"

			ids, err := tt.collectIDs(testWebClient(server))
			if err != nil {
				t.Fatalf("reader error = %v", err)
			}
			if requests != 2 {
				t.Fatalf("expected 2 page requests, got %d", requests)
			}
			if len(ids) != len(tt.wantIDs) {
				t.Fatalf("expected %d records across pages, got %#v", len(tt.wantIDs), ids)
			}
			for i, want := range tt.wantIDs {
				if ids[i] != want {
					t.Fatalf("record %d = %q, want %q (all=%#v)", i, ids[i], want, ids)
				}
			}
		})
	}
}

func TestReviewReadersTerminateSelfReferentialNext(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		bodyID     string
		collectErr func(*Client) error
	}{
		{
			name:   "submissions",
			path:   "/apps/app-123/reviewSubmissions",
			bodyID: "sub-1",
			collectErr: func(client *Client) error {
				_, err := client.ListReviewSubmissions(context.Background(), "app-123")
				return err
			},
		},
		{
			name:   "items",
			path:   "/reviewSubmissions/sub-1/items",
			bodyID: "item-1",
			collectErr: func(client *Client) error {
				_, err := client.ListReviewSubmissionItems(context.Background(), "sub-1")
				return err
			},
		},
		{
			name:   "threads",
			path:   "/resolutionCenterThreads",
			bodyID: "thread-1",
			collectErr: func(client *Client) error {
				_, err := client.ListResolutionCenterThreadsBySubmission(context.Background(), "sub-1")
				return err
			},
		},
		{
			name:   "messages",
			path:   "/resolutionCenterThreads/thread-1/resolutionCenterMessages",
			bodyID: "m1",
			collectErr: func(client *Client) error {
				_, err := client.ListResolutionCenterMessages(context.Background(), "thread-1", false)
				return err
			},
		},
		{
			name:   "rejections",
			path:   "/reviewRejections",
			bodyID: "rej-1",
			collectErr: func(client *Client) error {
				_, err := client.ListReviewRejections(context.Background(), "thread-1")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := handlertest.New(t)
			nextLink := ""
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					fixture.Respond(w, "unexpected path: %s", r.URL.Path)
					return
				}
				requests++
				if requests > 5 {
					fixture.Respond(w, "pagination loop escaped after %d requests", requests)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"data": [{"id":"` + tt.bodyID + `","type":"resources"}],
					"links": {"next": "` + nextLink + `"}
				}`))
			}))
			defer server.Close()
			nextLink = server.URL + tt.path

			err := tt.collectErr(testWebClient(server))
			if err == nil {
				t.Fatal("expected self-referential next link to terminate")
			}
			if !strings.Contains(err.Error(), "pagination loop detected") {
				t.Fatalf("expected pagination loop error, got %v", err)
			}
			if requests > 2 {
				t.Fatalf("expected loop detection within 2 requests, got %d", requests)
			}
		})
	}
}

func TestReviewReadersFollowQueryOnlyNextLinks(t *testing.T) {
	fixture := handlertest.New(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reviewRejections" {
			fixture.Respond(w, "query-only next lost collection path: %s", r.URL.Path)
			return
		}
		requests++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("cursor") == "page-2" {
			_, _ = w.Write([]byte(`{"data": [{"id":"rej-2","type":"reviewRejections"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{
			"data": [{"id":"rej-1","type":"reviewRejections"}],
			"links": {"next": "?cursor=page-2"}
		}`))
	}))
	defer server.Close()

	got, err := testWebClient(server).ListReviewRejections(context.Background(), "thread-1")
	if err != nil {
		t.Fatalf("ListReviewRejections() error = %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected 2 page requests, got %d", requests)
	}
	if len(got) != 2 || got[0].ID != "rej-1" || got[1].ID != "rej-2" {
		t.Fatalf("expected both query-only pages, got %#v", got)
	}
}
