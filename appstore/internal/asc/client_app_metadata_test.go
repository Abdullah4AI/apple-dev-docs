package asc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestGetAppSearchKeywords_SendsRequestWithFilters(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[{"type":"appKeywords","id":"keyword-1"}]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/apps/app-1/searchKeywords" {
			t.Fatalf("expected path /v1/apps/app-1/searchKeywords, got %s", req.URL.Path)
		}
		values := req.URL.Query()
		if values.Get("filter[platform]") != "IOS,MAC_OS" {
			t.Fatalf("expected filter[platform]=IOS,MAC_OS, got %q", values.Get("filter[platform]"))
		}
		if values.Get("filter[locale]") != "en-US,ja" {
			t.Fatalf("expected filter[locale]=en-US,ja, got %q", values.Get("filter[locale]"))
		}
		if values.Get("limit") != "5" {
			t.Fatalf("expected limit=5, got %q", values.Get("limit"))
		}
		assertAuthorized(t, req)
	}, response)

	_, err := client.GetAppSearchKeywords(
		context.Background(),
		"app-1",
		WithAppSearchKeywordsPlatforms([]string{"ios", "MAC_OS"}),
		WithAppSearchKeywordsLocales([]string{"en-US", "ja"}),
		WithAppSearchKeywordsLimit(5),
	)
	if err != nil {
		t.Fatalf("GetAppSearchKeywords() error: %v", err)
	}
}

func TestGetAppSearchKeywords_UsesNextURL(t *testing.T) {
	next := "https://api.appstoreconnect.apple.com/v1/apps/app-1/searchKeywords?cursor=abc"
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.URL.String() != next {
			t.Fatalf("expected next URL %q, got %q", next, req.URL.String())
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppSearchKeywords(context.Background(), "", WithAppSearchKeywordsNextURL(next)); err != nil {
		t.Fatalf("GetAppSearchKeywords() error: %v", err)
	}
}

func TestGetAppSearchKeywords_RequiresAppID(t *testing.T) {
	client := newTestClient(t, nil, jsonResponse(http.StatusOK, `{"data":[]}`))
	if _, err := client.GetAppSearchKeywords(context.Background(), ""); err == nil {
		t.Fatal("expected error for missing app ID")
	}
}

func TestSetAppSearchKeywords_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusNoContent, `{}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", req.Method)
		}
		if req.URL.Path != "/v1/apps/app-1/relationships/searchKeywords" {
			t.Fatalf("expected path /v1/apps/app-1/relationships/searchKeywords, got %s", req.URL.Path)
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var payload RelationshipRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		if len(payload.Data) != 2 {
			t.Fatalf("expected 2 keywords, got %d", len(payload.Data))
		}
		if payload.Data[0].Type != ResourceTypeAppKeywords {
			t.Fatalf("expected appKeywords type, got %q", payload.Data[0].Type)
		}
		assertAuthorized(t, req)
	}, response)

	if err := client.SetAppSearchKeywords(context.Background(), "app-1", []string{"kw-1", "kw-2"}); err != nil {
		t.Fatalf("SetAppSearchKeywords() error: %v", err)
	}
}

func TestSetAppSearchKeywords_ValidationErrors(t *testing.T) {
	client := newTestClient(t, nil, jsonResponse(http.StatusNoContent, `{}`))
	if err := client.SetAppSearchKeywords(context.Background(), "", []string{"kw-1"}); err == nil {
		t.Fatal("expected error for missing app ID")
	}
	if err := client.SetAppSearchKeywords(context.Background(), "app-1", nil); err == nil {
		t.Fatal("expected error for missing keywords")
	}
}

func TestGetAppStoreVersionLocalizationSearchKeywords_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[{"type":"appKeywords","id":"keyword-1"}]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appStoreVersionLocalizations/loc-1/searchKeywords" {
			t.Fatalf("expected path /v1/appStoreVersionLocalizations/loc-1/searchKeywords, got %s", req.URL.Path)
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppStoreVersionLocalizationSearchKeywords(context.Background(), "loc-1"); err != nil {
		t.Fatalf("GetAppStoreVersionLocalizationSearchKeywords() error: %v", err)
	}
}

func TestAddAppStoreVersionLocalizationSearchKeywords_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusNoContent, `{}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appStoreVersionLocalizations/loc-1/relationships/searchKeywords" {
			t.Fatalf("expected path /v1/appStoreVersionLocalizations/loc-1/relationships/searchKeywords, got %s", req.URL.Path)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var payload RelationshipRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		if len(payload.Data) != 2 {
			t.Fatalf("expected 2 keywords, got %d", len(payload.Data))
		}
		assertAuthorized(t, req)
	}, response)

	if err := client.AddAppStoreVersionLocalizationSearchKeywords(context.Background(), "loc-1", []string{"kw-1", "kw-2"}); err != nil {
		t.Fatalf("AddAppStoreVersionLocalizationSearchKeywords() error: %v", err)
	}
}

func TestDeleteAppStoreVersionLocalizationSearchKeywords_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusNoContent, `{}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appStoreVersionLocalizations/loc-1/relationships/searchKeywords" {
			t.Fatalf("expected path /v1/appStoreVersionLocalizations/loc-1/relationships/searchKeywords, got %s", req.URL.Path)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		var payload RelationshipRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		if len(payload.Data) != 1 {
			t.Fatalf("expected 1 keyword, got %d", len(payload.Data))
		}
		assertAuthorized(t, req)
	}, response)

	if err := client.DeleteAppStoreVersionLocalizationSearchKeywords(context.Background(), "loc-1", []string{"kw-1"}); err != nil {
		t.Fatalf("DeleteAppStoreVersionLocalizationSearchKeywords() error: %v", err)
	}
}

func TestGetAppStoreVersionLocalizationPreviewSets_SendsRequestWithLimit(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appStoreVersionLocalizations/loc-1/appPreviewSets" {
			t.Fatalf("expected path /v1/appStoreVersionLocalizations/loc-1/appPreviewSets, got %s", req.URL.Path)
		}
		if req.URL.Query().Get("limit") != "10" {
			t.Fatalf("expected limit=10, got %q", req.URL.Query().Get("limit"))
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppStoreVersionLocalizationPreviewSets(context.Background(), "loc-1", WithAppStoreVersionLocalizationPreviewSetsLimit(10)); err != nil {
		t.Fatalf("GetAppStoreVersionLocalizationPreviewSets() error: %v", err)
	}
}

func TestGetAppStoreVersionLocalizationPreviewSetsRelationships_UsesNextURL(t *testing.T) {
	next := "https://api.appstoreconnect.apple.com/v1/appStoreVersionLocalizations/loc-1/relationships/appPreviewSets?cursor=abc"
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.URL.String() != next {
			t.Fatalf("expected next URL %q, got %q", next, req.URL.String())
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppStoreVersionLocalizationPreviewSetsRelationships(context.Background(), "", WithLinkagesNextURL(next)); err != nil {
		t.Fatalf("GetAppStoreVersionLocalizationPreviewSetsRelationships() error: %v", err)
	}
}

func TestGetAppStoreVersionLocalizationScreenshotSets_SendsRequestWithLimit(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appStoreVersionLocalizations/loc-1/appScreenshotSets" {
			t.Fatalf("expected path /v1/appStoreVersionLocalizations/loc-1/appScreenshotSets, got %s", req.URL.Path)
		}
		if req.URL.Query().Get("limit") != "5" {
			t.Fatalf("expected limit=5, got %q", req.URL.Query().Get("limit"))
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppStoreVersionLocalizationScreenshotSets(context.Background(), "loc-1", WithAppStoreVersionLocalizationScreenshotSetsLimit(5)); err != nil {
		t.Fatalf("GetAppStoreVersionLocalizationScreenshotSets() error: %v", err)
	}
}

func TestGetAppStoreVersionLocalizationScreenshotSetsRelationships_UsesNextURL(t *testing.T) {
	next := "https://api.appstoreconnect.apple.com/v1/appStoreVersionLocalizations/loc-1/relationships/appScreenshotSets?cursor=abc"
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.URL.String() != next {
			t.Fatalf("expected next URL %q, got %q", next, req.URL.String())
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppStoreVersionLocalizationScreenshotSetsRelationships(context.Background(), "", WithLinkagesNextURL(next)); err != nil {
		t.Fatalf("GetAppStoreVersionLocalizationScreenshotSetsRelationships() error: %v", err)
	}
}

func TestGetAppStoreVersionRelationships_SendsRequest(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context, client *Client) error
		path string
	}{
		{
			name: "age rating",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppStoreVersionAgeRatingDeclarationRelationship(ctx, "version-1")
				return err
			},
			path: "/v1/appStoreVersions/version-1/relationships/ageRatingDeclaration",
		},
		{
			name: "review detail",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppStoreVersionReviewDetailRelationship(ctx, "version-1")
				return err
			},
			path: "/v1/appStoreVersions/version-1/relationships/appStoreReviewDetail",
		},
		{
			name: "app clip default experience",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppStoreVersionAppClipDefaultExperienceRelationship(ctx, "version-1")
				return err
			},
			path: "/v1/appStoreVersions/version-1/relationships/appClipDefaultExperience",
		},
		{
			name: "submission",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppStoreVersionSubmissionRelationship(ctx, "version-1")
				return err
			},
			path: "/v1/appStoreVersions/version-1/relationships/appStoreVersionSubmission",
		},
		{
			name: "routing app coverage",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppStoreVersionRoutingAppCoverageRelationship(ctx, "version-1")
				return err
			},
			path: "/v1/appStoreVersions/version-1/relationships/routingAppCoverage",
		},
		{
			name: "alternative distribution package",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppStoreVersionAlternativeDistributionPackageRelationship(ctx, "version-1")
				return err
			},
			path: "/v1/appStoreVersions/version-1/relationships/alternativeDistributionPackage",
		},
		{
			name: "game center app version",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppStoreVersionGameCenterAppVersionRelationship(ctx, "version-1")
				return err
			},
			path: "/v1/appStoreVersions/version-1/relationships/gameCenterAppVersion",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := jsonResponse(http.StatusOK, `{"data":{"type":"apps","id":"rel-1"}}`)
			client := newTestClient(t, func(req *http.Request) {
				if req.Method != http.MethodGet {
					t.Fatalf("expected GET, got %s", req.Method)
				}
				if req.URL.Path != test.path {
					t.Fatalf("expected path %s, got %s", test.path, req.URL.Path)
				}
				assertAuthorized(t, req)
			}, response)

			if err := test.call(context.Background(), client); err != nil {
				t.Fatalf("call error: %v", err)
			}
		})
	}
}

func TestGetAppStoreVersionRelationshipLists_SendsRequestWithLimit(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context, client *Client) error
		path string
	}{
		{
			name: "experiments v1",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppStoreVersionExperimentsRelationships(ctx, "version-1", WithLinkagesLimit(20))
				return err
			},
			path: "/v1/appStoreVersions/version-1/relationships/appStoreVersionExperiments",
		},
		{
			name: "experiments v2",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppStoreVersionExperimentsV2Relationships(ctx, "version-1", WithLinkagesLimit(20))
				return err
			},
			path: "/v1/appStoreVersions/version-1/relationships/appStoreVersionExperimentsV2",
		},
		{
			name: "customer reviews",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppStoreVersionCustomerReviewsRelationships(ctx, "version-1", WithLinkagesLimit(20))
				return err
			},
			path: "/v1/appStoreVersions/version-1/relationships/customerReviews",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := jsonResponse(http.StatusOK, `{"data":[{"type":"apps","id":"rel-1"}]}`)
			client := newTestClient(t, func(req *http.Request) {
				if req.Method != http.MethodGet {
					t.Fatalf("expected GET, got %s", req.Method)
				}
				if req.URL.Path != test.path {
					t.Fatalf("expected path %s, got %s", test.path, req.URL.Path)
				}
				if req.URL.Query().Get("limit") != "20" {
					t.Fatalf("expected limit=20, got %q", req.URL.Query().Get("limit"))
				}
				assertAuthorized(t, req)
			}, response)

			if err := test.call(context.Background(), client); err != nil {
				t.Fatalf("call error: %v", err)
			}
		})
	}
}

func TestGetAppStoreVersionAppClipDefaultExperience_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":{"type":"appClipDefaultExperiences","id":"exp-1"}}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appStoreVersions/version-1/appClipDefaultExperience" {
			t.Fatalf("expected path /v1/appStoreVersions/version-1/appClipDefaultExperience, got %s", req.URL.Path)
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppStoreVersionAppClipDefaultExperience(context.Background(), "version-1"); err != nil {
		t.Fatalf("GetAppStoreVersionAppClipDefaultExperience() error: %v", err)
	}
}

func TestGetAppStoreVersionExperimentsV2ForVersion_WithLimit(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appStoreVersions/version-1/appStoreVersionExperimentsV2" {
			t.Fatalf("expected path /v1/appStoreVersions/version-1/appStoreVersionExperimentsV2, got %s", req.URL.Path)
		}
		if req.URL.Query().Get("limit") != "12" {
			t.Fatalf("expected limit=12, got %q", req.URL.Query().Get("limit"))
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppStoreVersionExperimentsV2ForVersion(context.Background(), "version-1", WithAppStoreVersionExperimentsV2Limit(12)); err != nil {
		t.Fatalf("GetAppStoreVersionExperimentsV2ForVersion() error: %v", err)
	}
}

func TestGetAppStoreVersionCustomerReviews_WithLimit(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appStoreVersions/version-1/customerReviews" {
			t.Fatalf("expected path /v1/appStoreVersions/version-1/customerReviews, got %s", req.URL.Path)
		}
		if req.URL.Query().Get("limit") != "5" {
			t.Fatalf("expected limit=5, got %q", req.URL.Query().Get("limit"))
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppStoreVersionCustomerReviews(context.Background(), "version-1", WithLimit(5)); err != nil {
		t.Fatalf("GetAppStoreVersionCustomerReviews() error: %v", err)
	}
}

func TestGetAppCategory_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":{"type":"appCategories","id":"GAMES","attributes":{"platforms":["IOS"]}}}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appCategories/GAMES" {
			t.Fatalf("expected path /v1/appCategories/GAMES, got %s", req.URL.Path)
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppCategory(context.Background(), "GAMES"); err != nil {
		t.Fatalf("GetAppCategory() error: %v", err)
	}
}

func TestGetAppCategoryParent_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":{"type":"appCategories","id":"PARENT"}}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appCategories/GAMES/parent" {
			t.Fatalf("expected path /v1/appCategories/GAMES/parent, got %s", req.URL.Path)
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppCategoryParent(context.Background(), "GAMES"); err != nil {
		t.Fatalf("GetAppCategoryParent() error: %v", err)
	}
}

func TestGetAppCategorySubcategories_SendsRequestWithLimit(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appCategories/GAMES/subcategories" {
			t.Fatalf("expected path /v1/appCategories/GAMES/subcategories, got %s", req.URL.Path)
		}
		if req.URL.Query().Get("limit") != "50" {
			t.Fatalf("expected limit=50, got %q", req.URL.Query().Get("limit"))
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppCategorySubcategories(context.Background(), "GAMES", WithAppCategoriesLimit(50)); err != nil {
		t.Fatalf("GetAppCategorySubcategories() error: %v", err)
	}
}

func TestGetAppCategorySubcategories_UsesNextURL(t *testing.T) {
	next := "https://api.appstoreconnect.apple.com/v1/appCategories/GAMES/subcategories?cursor=abc"
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.URL.String() != next {
			t.Fatalf("expected next URL %q, got %q", next, req.URL.String())
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppCategorySubcategories(context.Background(), "", WithAppCategoriesNextURL(next)); err != nil {
		t.Fatalf("GetAppCategorySubcategories() error: %v", err)
	}
}

func TestGetAlternativeDistributionPackageForVersion_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":{"type":"alternativeDistributionPackages","id":"pkg-1"}}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appStoreVersions/version-1/alternativeDistributionPackage" {
			t.Fatalf("expected path /v1/appStoreVersions/version-1/alternativeDistributionPackage, got %s", req.URL.Path)
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAlternativeDistributionPackageForVersion(context.Background(), "version-1"); err != nil {
		t.Fatalf("GetAlternativeDistributionPackageForVersion() error: %v", err)
	}
}

func TestGetAppInfo_SendsRequestWithInclude(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":{"type":"appInfos","id":"info-1","attributes":{"state":"READY"}}}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appInfos/info-1" {
			t.Fatalf("expected path /v1/appInfos/info-1, got %s", req.URL.Path)
		}
		if req.URL.Query().Get("include") != "ageRatingDeclaration" {
			t.Fatalf("expected include=ageRatingDeclaration, got %q", req.URL.Query().Get("include"))
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppInfo(context.Background(), "info-1", WithAppInfoInclude([]string{"ageRatingDeclaration"})); err != nil {
		t.Fatalf("GetAppInfo() error: %v", err)
	}
}

func TestGetAppInfoRelationships_SendsRequest(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context, client *Client) error
		path string
	}{
		{
			name: "age rating",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppInfoAgeRatingDeclarationRelationship(ctx, "info-1")
				return err
			},
			path: "/v1/appInfos/info-1/relationships/ageRatingDeclaration",
		},
		{
			name: "primary category",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppInfoPrimaryCategoryRelationship(ctx, "info-1")
				return err
			},
			path: "/v1/appInfos/info-1/relationships/primaryCategory",
		},
		{
			name: "secondary subcategory two",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppInfoSecondarySubcategoryTwoRelationship(ctx, "info-1")
				return err
			},
			path: "/v1/appInfos/info-1/relationships/secondarySubcategoryTwo",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := jsonResponse(http.StatusOK, `{"data":{"type":"appCategories","id":"cat-1"}}`)
			client := newTestClient(t, func(req *http.Request) {
				if req.Method != http.MethodGet {
					t.Fatalf("expected GET, got %s", req.Method)
				}
				if req.URL.Path != test.path {
					t.Fatalf("expected path %s, got %s", test.path, req.URL.Path)
				}
				assertAuthorized(t, req)
			}, response)

			if err := test.call(context.Background(), client); err != nil {
				t.Fatalf("call error: %v", err)
			}
		})
	}
}

func TestGetAppInfoCategoryRelatedResources_SendsRequest(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context, client *Client) error
		path string
	}{
		{
			name: "primary category",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppInfoPrimaryCategory(ctx, "info-1")
				return err
			},
			path: "/v1/appInfos/info-1/primaryCategory",
		},
		{
			name: "primary subcategory one",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppInfoPrimarySubcategoryOne(ctx, "info-1")
				return err
			},
			path: "/v1/appInfos/info-1/primarySubcategoryOne",
		},
		{
			name: "primary subcategory two",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppInfoPrimarySubcategoryTwo(ctx, "info-1")
				return err
			},
			path: "/v1/appInfos/info-1/primarySubcategoryTwo",
		},
		{
			name: "secondary category",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppInfoSecondaryCategory(ctx, "info-1")
				return err
			},
			path: "/v1/appInfos/info-1/secondaryCategory",
		},
		{
			name: "secondary subcategory one",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppInfoSecondarySubcategoryOne(ctx, "info-1")
				return err
			},
			path: "/v1/appInfos/info-1/secondarySubcategoryOne",
		},
		{
			name: "secondary subcategory two",
			call: func(ctx context.Context, client *Client) error {
				_, err := client.GetAppInfoSecondarySubcategoryTwo(ctx, "info-1")
				return err
			},
			path: "/v1/appInfos/info-1/secondarySubcategoryTwo",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := jsonResponse(http.StatusOK, `{"data":{"type":"appCategories","id":"cat-1"}}`)
			client := newTestClient(t, func(req *http.Request) {
				if req.Method != http.MethodGet {
					t.Fatalf("expected GET, got %s", req.Method)
				}
				if req.URL.Path != test.path {
					t.Fatalf("expected path %s, got %s", test.path, req.URL.Path)
				}
				assertAuthorized(t, req)
			}, response)

			if err := test.call(context.Background(), client); err != nil {
				t.Fatalf("call error: %v", err)
			}
		})
	}
}

func TestGetAppInfoTerritoryAgeRatingsRelationships_UsesNextURL(t *testing.T) {
	next := "https://api.appstoreconnect.apple.com/v1/appInfos/info-1/relationships/territoryAgeRatings?cursor=abc"
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.URL.String() != next {
			t.Fatalf("expected next URL %q, got %q", next, req.URL.String())
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppInfoTerritoryAgeRatingsRelationships(context.Background(), "", WithLinkagesNextURL(next)); err != nil {
		t.Fatalf("GetAppInfoTerritoryAgeRatingsRelationships() error: %v", err)
	}
}

func TestGetAppInfoTerritoryAgeRatings_SendsRequest(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/appInfos/info-1/territoryAgeRatings" {
			t.Fatalf("expected path /v1/appInfos/info-1/territoryAgeRatings, got %s", req.URL.Path)
		}
		values := req.URL.Query()
		if got := values.Get("fields[territoryAgeRatings]"); got != "appStoreAgeRating,territory" {
			t.Fatalf("expected fields[territoryAgeRatings]=appStoreAgeRating,territory, got %q", got)
		}
		if got := values.Get("fields[territories]"); got != "currency" {
			t.Fatalf("expected fields[territories]=currency, got %q", got)
		}
		if got := values.Get("include"); got != "territory" {
			t.Fatalf("expected include=territory, got %q", got)
		}
		if got := values.Get("limit"); got != "10" {
			t.Fatalf("expected limit=10, got %q", got)
		}
		assertAuthorized(t, req)
	}, response)

	_, err := client.GetAppInfoTerritoryAgeRatings(context.Background(), "info-1",
		WithTerritoryAgeRatingsFields([]string{"appStoreAgeRating", "territory"}),
		WithTerritoryAgeRatingsTerritoryFields([]string{"currency"}),
		WithTerritoryAgeRatingsInclude([]string{"territory"}),
		WithTerritoryAgeRatingsLimit(10),
	)
	if err != nil {
		t.Fatalf("GetAppInfoTerritoryAgeRatings() error: %v", err)
	}
}

func TestGetAppInfoTerritoryAgeRatings_UsesNextURL(t *testing.T) {
	next := "https://api.appstoreconnect.apple.com/v1/appInfos/info-1/territoryAgeRatings?cursor=abc"
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.URL.String() != next {
			t.Fatalf("expected next URL %q, got %q", next, req.URL.String())
		}
		assertAuthorized(t, req)
	}, response)

	if _, err := client.GetAppInfoTerritoryAgeRatings(context.Background(), "", WithTerritoryAgeRatingsNextURL(next)); err != nil {
		t.Fatalf("GetAppInfoTerritoryAgeRatings() error: %v", err)
	}
}
