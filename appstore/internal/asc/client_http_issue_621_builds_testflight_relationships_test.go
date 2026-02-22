package asc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestIssue621_BuildsTestFlightRelationshipEndpoints_GET(t *testing.T) {
	ctx := context.Background()

	const (
		linkagesOK = `{"data":[{"type":"apps","id":"1"}],"links":{}}`
		toOneOK    = `{"data":{"type":"apps","id":"1"},"links":{}}`
	)

	tests := []struct {
		name     string
		wantPath string
		body     string
		call     func(*Client) error
	}{
		{
			name:     "GetBetaAppLocalizationAppRelationship",
			wantPath: "/v1/betaAppLocalizations/loc-1/relationships/app",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBetaAppLocalizationAppRelationship(ctx, "loc-1")
				return err
			},
		},
		{
			name:     "GetBetaAppReviewDetailAppRelationship",
			wantPath: "/v1/betaAppReviewDetails/detail-1/relationships/app",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBetaAppReviewDetailAppRelationship(ctx, "detail-1")
				return err
			},
		},
		{
			name:     "GetBetaAppReviewSubmissionBuildRelationship",
			wantPath: "/v1/betaAppReviewSubmissions/sub-1/relationships/build",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBetaAppReviewSubmissionBuildRelationship(ctx, "sub-1")
				return err
			},
		},
		{
			name:     "GetBetaBuildLocalizationBuildRelationship",
			wantPath: "/v1/betaBuildLocalizations/bbl-1/relationships/build",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBetaBuildLocalizationBuildRelationship(ctx, "bbl-1")
				return err
			},
		},
		{
			name:     "GetBetaFeedbackCrashSubmissionCrashLogRelationship",
			wantPath: "/v1/betaFeedbackCrashSubmissions/crash-1/relationships/crashLog",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBetaFeedbackCrashSubmissionCrashLogRelationship(ctx, "crash-1")
				return err
			},
		},
		{
			name:     "GetBetaGroupAppRelationship",
			wantPath: "/v1/betaGroups/group-1/relationships/app",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBetaGroupAppRelationship(ctx, "group-1")
				return err
			},
		},
		{
			name:     "GetBetaGroupBetaRecruitmentCriteriaRelationship",
			wantPath: "/v1/betaGroups/group-1/relationships/betaRecruitmentCriteria",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBetaGroupBetaRecruitmentCriteriaRelationship(ctx, "group-1")
				return err
			},
		},
		{
			name:     "GetBetaGroupBetaRecruitmentCriterionCompatibleBuildCheckRelationship",
			wantPath: "/v1/betaGroups/group-1/relationships/betaRecruitmentCriterionCompatibleBuildCheck",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBetaGroupBetaRecruitmentCriterionCompatibleBuildCheckRelationship(ctx, "group-1")
				return err
			},
		},
		{
			name:     "GetBuildBetaDetailBuildRelationship",
			wantPath: "/v1/buildBetaDetails/detail-1/relationships/build",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBuildBetaDetailBuildRelationship(ctx, "detail-1")
				return err
			},
		},
		{
			name:     "GetBuildBundleAppClipDomainCacheStatusRelationship",
			wantPath: "/v1/buildBundles/bb-1/relationships/appClipDomainCacheStatus",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBuildBundleAppClipDomainCacheStatusRelationship(ctx, "bb-1")
				return err
			},
		},
		{
			name:     "GetBuildBundleAppClipDomainDebugStatusRelationship",
			wantPath: "/v1/buildBundles/bb-1/relationships/appClipDomainDebugStatus",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBuildBundleAppClipDomainDebugStatusRelationship(ctx, "bb-1")
				return err
			},
		},
		{
			name:     "GetBuildBundleBetaAppClipInvocationsRelationships",
			wantPath: "/v1/buildBundles/bb-1/relationships/betaAppClipInvocations",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetBuildBundleBetaAppClipInvocationsRelationships(ctx, "bb-1")
				return err
			},
		},
		{
			name:     "GetBuildBundleBuildBundleFileSizesRelationships",
			wantPath: "/v1/buildBundles/bb-1/relationships/buildBundleFileSizes",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetBuildBundleBuildBundleFileSizesRelationships(ctx, "bb-1")
				return err
			},
		},
		{
			name:     "GetBuildUploadBuildUploadFilesRelationships",
			wantPath: "/v1/buildUploads/upl-1/relationships/buildUploadFiles",
			body:     linkagesOK,
			call: func(client *Client) error {
				_, err := client.GetBuildUploadBuildUploadFilesRelationships(ctx, "upl-1")
				return err
			},
		},
		{
			name:     "GetBuildAppEncryptionDeclarationRelationship",
			wantPath: "/v1/builds/build-1/relationships/appEncryptionDeclaration",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBuildAppEncryptionDeclarationRelationship(ctx, "build-1")
				return err
			},
		},
		{
			name:     "GetBuildBetaAppReviewSubmissionRelationship",
			wantPath: "/v1/builds/build-1/relationships/betaAppReviewSubmission",
			body:     toOneOK,
			call: func(client *Client) error {
				_, err := client.GetBuildBetaAppReviewSubmissionRelationship(ctx, "build-1")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(req *http.Request) {
				if req.Method != http.MethodGet {
					t.Fatalf("expected GET, got %s", req.Method)
				}
				if req.URL.Path != tt.wantPath {
					t.Fatalf("expected path %s, got %s", tt.wantPath, req.URL.Path)
				}
				assertAuthorized(t, req)
			}, jsonResponse(http.StatusOK, tt.body))

			if err := tt.call(client); err != nil {
				t.Fatalf("request error: %v", err)
			}
		})
	}
}

func TestIssue621_BuildsTestFlightRelationshipEndpoints_Mutations(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		wantMethod string
		wantPath   string
		wantBodyFn func(*testing.T, []byte)
		call       func(*Client) error
	}{
		{
			name:       "AddBuildsToBetaGroup (POST /v1/betaGroups/{id}/relationships/builds)",
			wantMethod: http.MethodPost,
			wantPath:   "/v1/betaGroups/group-1/relationships/builds",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 2 {
					t.Fatalf("expected 2 relationship items, got %d", len(got.Data))
				}
				if got.Data[0].Type != ResourceTypeBuilds || got.Data[0].ID != "build-1" {
					t.Fatalf("unexpected first item: %#v", got.Data[0])
				}
				if got.Data[1].Type != ResourceTypeBuilds || got.Data[1].ID != "build-2" {
					t.Fatalf("unexpected second item: %#v", got.Data[1])
				}
			},
			call: func(client *Client) error {
				return client.AddBuildsToBetaGroup(ctx, "group-1", []string{"build-1", "build-2"})
			},
		},
		{
			name:       "RemoveBuildsFromBetaGroup (DELETE /v1/betaGroups/{id}/relationships/builds)",
			wantMethod: http.MethodDelete,
			wantPath:   "/v1/betaGroups/group-1/relationships/builds",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got RelationshipRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if len(got.Data) != 1 {
					t.Fatalf("expected 1 relationship item, got %d", len(got.Data))
				}
				if got.Data[0].Type != ResourceTypeBuilds || got.Data[0].ID != "build-1" {
					t.Fatalf("unexpected item: %#v", got.Data[0])
				}
			},
			call: func(client *Client) error {
				return client.RemoveBuildsFromBetaGroup(ctx, "group-1", []string{"build-1"})
			},
		},
		{
			name:       "UpdateBuildAppEncryptionDeclarationRelationship (PATCH /v1/builds/{id}/relationships/appEncryptionDeclaration)",
			wantMethod: http.MethodPatch,
			wantPath:   "/v1/builds/build-1/relationships/appEncryptionDeclaration",
			wantBodyFn: func(t *testing.T, body []byte) {
				t.Helper()
				var got BuildAppEncryptionDeclarationRelationshipUpdateRequest
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unmarshal body: %v", err)
				}
				if got.Data.Type != ResourceTypeAppEncryptionDeclarations {
					t.Fatalf("expected type %q, got %q", ResourceTypeAppEncryptionDeclarations, got.Data.Type)
				}
				if got.Data.ID != "decl-1" {
					t.Fatalf("expected id %q, got %q", "decl-1", got.Data.ID)
				}
			},
			call: func(client *Client) error {
				return client.UpdateBuildAppEncryptionDeclarationRelationship(ctx, "build-1", "decl-1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(req *http.Request) {
				if req.Method != tt.wantMethod {
					t.Fatalf("expected %s, got %s", tt.wantMethod, req.Method)
				}
				if req.URL.Path != tt.wantPath {
					t.Fatalf("expected path %s, got %s", tt.wantPath, req.URL.Path)
				}

				body, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				tt.wantBodyFn(t, body)

				assertAuthorized(t, req)
			}, jsonResponse(http.StatusNoContent, ""))

			if err := tt.call(client); err != nil {
				t.Fatalf("request error: %v", err)
			}
		})
	}
}
