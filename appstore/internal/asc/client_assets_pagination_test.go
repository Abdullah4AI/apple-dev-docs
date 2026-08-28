package asc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestGetAllAppScreenshotSetsFollowsNextURL(t *testing.T) {
	const next = BaseURL + "/v1/appStoreVersionLocalizations/loc-1/appScreenshotSets?cursor=sets-2&limit=17"
	requestCount := 0
	client := newTestClient(
		t, func(req *http.Request) {
			switch requestCount {
			case 0:
				if req.URL.Path != "/v1/appStoreVersionLocalizations/loc-1/appScreenshotSets" {
					t.Fatalf("first request path = %q", req.URL.Path)
				}
				if got := req.URL.Query().Get("limit"); got != "1" {
					t.Fatalf("first request limit = %q, want 1", got)
				}
			case 1:
				if got := req.URL.String(); got != next {
					t.Fatalf("continuation URL = %q, want %q", got, next)
				}
			default:
				t.Fatalf("unexpected request %d: %s", requestCount+1, req.URL)
			}
			requestCount++
		},
		jsonResponse(http.StatusOK, `{"data":[{"type":"appScreenshotSets","id":"set-1","attributes":{"screenshotDisplayType":"APP_IPHONE_65"}}],"links":{"next":"`+next+`"}}`),
		jsonResponse(http.StatusOK, `{"data":[{"type":"appScreenshotSets","id":"set-2","attributes":{"screenshotDisplayType":"APP_IPAD_PRO_129"}}],"links":{}}`),
	)

	response, err := client.GetAllAppScreenshotSets(context.Background(), "loc-1", WithAppScreenshotSetsLimit(1))
	if err != nil {
		t.Fatalf("GetAllAppScreenshotSets() error: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want 2", requestCount)
	}
	if len(response.Data) != 2 || response.Data[0].ID != "set-1" || response.Data[1].ID != "set-2" {
		t.Fatalf("unexpected screenshot sets: %#v", response.Data)
	}
}

func TestGetAllAppScreenshotsFollowsNextURL(t *testing.T) {
	const next = BaseURL + "/v1/appScreenshotSets/set-1/appScreenshots?cursor=screenshots-2"
	requestCount := 0
	client := newTestClient(
		t, func(req *http.Request) {
			switch requestCount {
			case 0:
				if req.URL.Path != "/v1/appScreenshotSets/set-1/appScreenshots" {
					t.Fatalf("first request path = %q", req.URL.Path)
				}
			case 1:
				if got := req.URL.String(); got != next {
					t.Fatalf("continuation URL = %q, want %q", got, next)
				}
			default:
				t.Fatalf("unexpected request %d: %s", requestCount+1, req.URL)
			}
			requestCount++
		},
		jsonResponse(http.StatusOK, `{"data":[{"type":"appScreenshots","id":"shot-1","attributes":{"fileName":"01-home.png"}}],"links":{"next":"`+next+`"}}`),
		jsonResponse(http.StatusOK, `{"data":[{"type":"appScreenshots","id":"shot-2","attributes":{"fileName":"02-settings.png"}}],"links":{}}`),
	)

	response, err := client.GetAllAppScreenshots(context.Background(), "set-1")
	if err != nil {
		t.Fatalf("GetAllAppScreenshots() error: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want 2", requestCount)
	}
	if len(response.Data) != 2 || response.Data[0].ID != "shot-1" || response.Data[1].ID != "shot-2" {
		t.Fatalf("unexpected screenshots: %#v", response.Data)
	}
}

func TestGetAllAppScreenshotsRejectsRepeatedNextURL(t *testing.T) {
	const next = BaseURL + "/v1/appScreenshotSets/set-1/appScreenshots?cursor=repeat"
	client := newTestClient(
		t, nil,
		jsonResponse(http.StatusOK, `{"data":[{"type":"appScreenshots","id":"shot-1"}],"links":{"next":"`+next+`"}}`),
		jsonResponse(http.StatusOK, `{"data":[{"type":"appScreenshots","id":"shot-2"}],"links":{"next":"`+next+`"}}`),
	)

	_, err := client.GetAllAppScreenshots(context.Background(), "set-1")
	if !errors.Is(err, ErrRepeatedPaginationURL) {
		t.Fatalf("error = %v, want ErrRepeatedPaginationURL", err)
	}
}

func TestGetAllAppScreenshotsUsesFreshRequestContextPerPage(t *testing.T) {
	const next = BaseURL + "/v1/appScreenshotSets/set-1/appScreenshots?cursor=slow-2"
	requestCount := 0
	requestContextCalls := 0
	requestParents := make([]context.Context, 0, 2)
	requestContexts := make([]context.Context, 0, 2)
	var firstRequestContext context.Context
	client := newTestClient(
		t, func(req *http.Request) {
			switch requestCount {
			case 0:
				firstRequestContext = req.Context()
				if err := firstRequestContext.Err(); err != nil {
					t.Fatalf("first request context is already done: %v", err)
				}
			case 1:
				if firstRequestContext == nil {
					t.Fatal("continuation request arrived before first request")
				}
				select {
				case <-firstRequestContext.Done():
				default:
					t.Fatal("first request context was not canceled before continuation")
				}
				if err := req.Context().Err(); err != nil {
					t.Fatalf("continuation request context is already done: %v", err)
				}
			default:
				t.Fatalf("unexpected request %d: %s", requestCount+1, req.URL)
			}
			requestCount++
		},
		jsonResponse(http.StatusOK, `{"data":[{"type":"appScreenshots","id":"shot-1"}],"links":{"next":"`+next+`"}}`),
		jsonResponse(http.StatusOK, `{"data":[{"type":"appScreenshots","id":"shot-2"}],"links":{}}`),
	)

	parent := context.Background()
	requestContext := func(parent context.Context) (context.Context, context.CancelFunc) {
		requestContextCalls++
		requestParents = append(requestParents, parent)
		requestCtx, cancel := context.WithCancel(parent)
		requestContexts = append(requestContexts, requestCtx)
		return requestCtx, cancel
	}

	response, err := client.GetAllAppScreenshots(parent, "set-1", WithAppScreenshotsRequestContext(requestContext))
	if err != nil {
		t.Fatalf("GetAllAppScreenshots() error: %v", err)
	}
	if requestCount != 2 || requestContextCalls != 2 || len(requestContexts) != 2 || len(response.Data) != 2 {
		t.Fatalf("request count/context calls/contexts/data = %d/%d/%d/%d, want 2/2/2/2", requestCount, requestContextCalls, len(requestContexts), len(response.Data))
	}
	for index, requestParent := range requestParents {
		if requestParent != parent {
			t.Fatalf("request %d factory parent = %v, want operation parent", index+1, requestParent)
		}
	}
	if requestContexts[0] == requestContexts[1] {
		t.Fatal("continuation reused the first request context")
	}
}

func TestScreenshotCollectionRejectsOutOfRangeLimitsBeforeRequest(t *testing.T) {
	tests := []struct {
		name    string
		request func(*Client, int) error
		wantErr string
	}{
		{
			name: "screenshot sets",
			request: func(client *Client, limit int) error {
				_, err := client.GetAppScreenshotSets(context.Background(), "loc-1", WithAppScreenshotSetsLimit(limit))
				return err
			},
			wantErr: "appScreenshotSets: limit must be between 1 and 200",
		},
		{
			name: "screenshots",
			request: func(client *Client, limit int) error {
				_, err := client.GetAppScreenshots(context.Background(), "set-1", WithAppScreenshotsLimit(limit))
				return err
			},
			wantErr: "appScreenshots: limit must be between 1 and 200",
		},
	}

	for _, test := range tests {
		for _, limit := range []int{0, -1, appScreenshotCollectionLimitMax + 1} {
			t.Run(fmt.Sprintf("%s/%d", test.name, limit), func(t *testing.T) {
				requestCount := 0
				client := newTestClient(t, func(*http.Request) { requestCount++ }, jsonResponse(http.StatusOK, `{"data":[]}`))
				err := test.request(client, limit)
				if err == nil || err.Error() != test.wantErr {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				if requestCount != 0 {
					t.Fatalf("request count = %d, want 0", requestCount)
				}
			})
		}
	}
}
