package asc

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestResolveCurrentAppInfoID(t *testing.T) {
	tests := []struct {
		name     string
		appInfos []Resource[AppInfoAttributes]
		wantID   string
		wantErr  string
	}{
		{
			name: "selects the only non-historical app info",
			appInfos: []Resource[AppInfoAttributes]{
				{ID: "info-old", Attributes: AppInfoAttributes{"state": "REPLACED_WITH_NEW_INFO"}},
				{ID: "info-current", Attributes: AppInfoAttributes{"state": "READY_FOR_DISTRIBUTION"}},
			},
			wantID: "info-current",
		},
		{
			name: "rejects ambiguous current app infos",
			appInfos: []Resource[AppInfoAttributes]{
				{ID: "info-one", Attributes: AppInfoAttributes{"state": "READY_FOR_REVIEW"}},
				{ID: "info-two", Attributes: AppInfoAttributes{"state": "PREPARE_FOR_SUBMISSION"}},
			},
			wantErr: "multiple current app infos found",
		},
		{
			name: "rejects all-historical app infos",
			appInfos: []Resource[AppInfoAttributes]{
				{ID: "info-old", Attributes: AppInfoAttributes{"state": "REPLACED_WITH_NEW_INFO"}},
			},
			wantErr: "no current app info found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveCurrentAppInfoID("app-1", tt.appInfos)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("resolveCurrentAppInfoID() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveCurrentAppInfoID() error = %v", err)
			}
			if got != tt.wantID {
				t.Fatalf("resolveCurrentAppInfoID() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestResolveCurrentAppInfoIDForAppFollowsEveryPage(t *testing.T) {
	tests := []struct {
		name       string
		firstPage  string
		secondPage string
		wantID     string
		wantErr    string
	}{
		{
			name:       "finds current app info on the second page",
			firstPage:  `{"data":[{"type":"appInfos","id":"info-old","attributes":{"state":"REPLACED_WITH_NEW_INFO"}}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/apps/app-1/appInfos?cursor=next"}}`,
			secondPage: `{"data":[{"type":"appInfos","id":"info-current","attributes":{"state":"READY_FOR_DISTRIBUTION"}}],"links":{}}`,
			wantID:     "info-current",
		},
		{
			name:       "detects current candidates split across pages",
			firstPage:  `{"data":[{"type":"appInfos","id":"info-one","attributes":{"state":"READY_FOR_REVIEW"}}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/apps/app-1/appInfos?cursor=next"}}`,
			secondPage: `{"data":[{"type":"appInfos","id":"info-two","attributes":{"state":"PREPARE_FOR_SUBMISSION"}}],"links":{}}`,
			wantErr:    "multiple current app infos found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests := make([]string, 0, 2)
			client := newTestClient(t, func(req *http.Request) {
				requests = append(requests, req.URL.String())
			}, jsonResponse(http.StatusOK, test.firstPage), jsonResponse(http.StatusOK, test.secondPage))

			got, err := client.ResolveCurrentAppInfoIDForApp(context.Background(), "app-1")
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("ResolveCurrentAppInfoIDForApp() error = %v, want containing %q", err, test.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("ResolveCurrentAppInfoIDForApp() error = %v", err)
				}
				if got != test.wantID {
					t.Fatalf("ResolveCurrentAppInfoIDForApp() = %q, want %q", got, test.wantID)
				}
			}
			if len(requests) != 2 {
				t.Fatalf("requests = %v, want two pages", requests)
			}
			if !strings.Contains(requests[0], "limit=200") || !strings.Contains(requests[0], "fields%5BappInfos%5D=state") {
				t.Fatalf("first request = %q, want maximum page size and state field", requests[0])
			}
			if !strings.Contains(requests[1], "cursor=next") {
				t.Fatalf("second request = %q, want continuation URL", requests[1])
			}
		})
	}
}

func TestAutoResolveAppInfoIDByVersionState(t *testing.T) {
	tests := []struct {
		name         string
		versionState string
		candidates   []AppInfoCandidate
		wantID       string
		wantOK       bool
	}{
		{
			name:         "matches exact shared state",
			versionState: "WAITING_FOR_REVIEW",
			candidates: []AppInfoCandidate{
				{ID: "info-1", State: "PREPARE_FOR_SUBMISSION"},
				{ID: "info-2", State: "WAITING_FOR_REVIEW"},
			},
			wantID: "info-2",
			wantOK: true,
		},
		{
			name:         "maps pending developer release to pending release",
			versionState: "PENDING_DEVELOPER_RELEASE",
			candidates: []AppInfoCandidate{
				{ID: "info-1", State: "PREPARE_FOR_SUBMISSION"},
				{ID: "info-2", State: "PENDING_RELEASE"},
			},
			wantID: "info-2",
			wantOK: true,
		},
		{
			name:         "maps pending apple release to pending release",
			versionState: "PENDING_APPLE_RELEASE",
			candidates: []AppInfoCandidate{
				{ID: "info-1", State: "WAITING_FOR_REVIEW"},
				{ID: "info-2", State: "PENDING_RELEASE"},
			},
			wantID: "info-2",
			wantOK: true,
		},
		{
			name:         "maps replaced with new version to replaced with new info",
			versionState: "REPLACED_WITH_NEW_VERSION",
			candidates: []AppInfoCandidate{
				{ID: "info-1", State: "READY_FOR_REVIEW"},
				{ID: "info-2", State: "REPLACED_WITH_NEW_INFO"},
			},
			wantID: "info-2",
			wantOK: true,
		},
		{
			name:         "maps ready for sale fallback to ready for distribution",
			versionState: "READY_FOR_SALE",
			candidates: []AppInfoCandidate{
				{ID: "info-1", State: "READY_FOR_REVIEW"},
				{ID: "info-2", State: "READY_FOR_DISTRIBUTION"},
			},
			wantID: "info-2",
			wantOK: true,
		},
		{
			name:         "maps preorder ready for sale to ready for distribution",
			versionState: "PREORDER_READY_FOR_SALE",
			candidates: []AppInfoCandidate{
				{ID: "info-1", State: "READY_FOR_REVIEW"},
				{ID: "info-2", State: "READY_FOR_DISTRIBUTION"},
			},
			wantID: "info-2",
			wantOK: true,
		},
		{
			name:         "returns false when alias remains ambiguous",
			versionState: "PENDING_DEVELOPER_RELEASE",
			candidates: []AppInfoCandidate{
				{ID: "info-1", State: "PENDING_RELEASE"},
				{ID: "info-2", State: "PENDING_RELEASE"},
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := AutoResolveAppInfoIDByVersionState(tt.candidates, tt.versionState)
			if gotOK != tt.wantOK {
				t.Fatalf("AutoResolveAppInfoIDByVersionState() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotID != tt.wantID {
				t.Fatalf("AutoResolveAppInfoIDByVersionState() id = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}

func TestResolveAppStoreVersionStatePrefersAppVersionState(t *testing.T) {
	attrs := AppStoreVersionAttributes{
		AppVersionState: "PREORDER_READY_FOR_SALE",
		AppStoreState:   "READY_FOR_SALE",
	}

	got := ResolveAppStoreVersionState(attrs)
	if got != "PREORDER_READY_FOR_SALE" {
		t.Fatalf("ResolveAppStoreVersionState() = %q, want %q", got, "PREORDER_READY_FOR_SALE")
	}
}
