package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	cmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
	webcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/web"
	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

func TestWebAppsDeleteRunRejectsAppStillOnSaleWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": false,
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"id": "avail-1",
							"type": "appAvailabilities",
							"attributes": {"availableInNewTerritories": false},
							"relationships": {
								"availableTerritories": {
									"data": [{"type": "territories", "id": "USA"}]
								}
							}
						}
					}`), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/iris/v1/apps/1234567890":
					patchCalls++
					t.Fatal("did not expect PATCH when the app is still available for sale")
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitError {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitError, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "still available") || !strings.Contains(stderr, "USA") {
		t.Fatalf("expected stderr to name the on-sale territory blocker, got %q", stderr)
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func TestWebAppsDeleteRunRejectsBlockingAppVersionStateWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": false,
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {
									"data": [{"type": "appStoreVersions", "id": "version-1"}]
								}
							}
						},
						"included": [{
							"type": "appStoreVersions",
							"id": "version-1",
							"attributes": {
								"platform": "IOS",
								"versionString": "1.0",
								"appStoreState": "READY_FOR_SALE",
								"appVersionState": "WAITING_FOR_REVIEW"
							}
						}]
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"id": "avail-1",
							"type": "appAvailabilities",
							"attributes": {"availableInNewTerritories": false},
							"relationships": {
								"availableTerritories": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/iris/v1/apps/1234567890":
					patchCalls++
					t.Fatal("did not expect PATCH when a displayable version is WAITING_FOR_REVIEW")
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitError {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitError, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "WAITING_FOR_REVIEW") {
		t.Fatalf("expected stderr to name the blocking appVersionState, got %q", stderr)
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func TestWebAppsDeleteRunRejectsOmittedTerritoryLinkageWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": false,
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"id": "avail-1",
							"type": "appAvailabilities",
							"attributes": {"availableInNewTerritories": false},
							"relationships": {
								"territoryAvailabilities": {
									"links": {
										"related": "https://appstoreconnect.apple.com/iris/v2/appAvailabilities/avail-1/territoryAvailabilities"
									}
								}
							}
						}
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v2/appAvailabilities/avail-1/territoryAvailabilities":
					return webAppsDeleteJSONResponse(`{
						"data": [{
							"type": "territoryAvailabilities",
							"id": "ta-usa",
							"attributes": {},
							"relationships": {"territory": {"data": {"type": "territories", "id": "USA"}}}
						}]
					}`), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/iris/v1/apps/1234567890":
					patchCalls++
					t.Fatal("did not expect PATCH when territoryAvailabilities could not be read")
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitError {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitError, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "could not read availability") || !strings.Contains(stderr, "territoryAvailabilities") {
		t.Fatalf("expected stderr to name unreadable territoryAvailabilities, got %q", stderr)
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func TestWebAppsDeleteRunReReadsAndFailsWhenPatchDoesNotRemove(t *testing.T) {
	var patchCalls, postPatchGets int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					if patchCalls > 0 {
						postPatchGets++
					}
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": false,
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"id": "avail-1",
							"type": "appAvailabilities",
							"attributes": {"availableInNewTerritories": false},
							"relationships": {
								"availableTerritories": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/iris/v1/apps/1234567890":
					patchCalls++
					return webAppsDeleteJSONResponse(`{}`), nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitError {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitError, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout when removal is unverified, got %q", stdout)
	}
	if !strings.Contains(stderr, "did not confirm") || !strings.Contains(stderr, "removed") {
		t.Fatalf("expected stderr to say Apple did not confirm removal, got %q", stderr)
	}
	if patchCalls != 1 {
		t.Fatalf("expected one PATCH, got %d", patchCalls)
	}
	if postPatchGets < 1 {
		t.Fatalf("expected a post-PATCH re-read, got %d", postPatchGets)
	}
}

func TestWebAppsDeleteRunDryRunReportsEligibleWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": false,
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"id": "avail-1",
							"type": "appAvailabilities",
							"attributes": {"availableInNewTerritories": false},
							"relationships": {
								"availableTerritories": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodPatch:
					patchCalls++
					t.Fatalf("dry-run must not PATCH: %s %s", req.Method, req.URL.String())
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--dry-run",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var payload struct {
		AppID    string `json:"appId"`
		Name     string `json:"name"`
		BundleID string `json:"bundleId"`
		Removed  bool   `json:"removed"`
		DryRun   bool   `json:"dryRun"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error: %v; stdout=%q", err, stdout)
	}
	if payload.AppID != "1234567890" || payload.Name != "Throwaway" || payload.BundleID != "com.example.throwaway" {
		t.Fatalf("unexpected dry-run identity: %+v", payload)
	}
	if payload.Removed {
		t.Fatal("dry-run must report current server removed=false")
	}
	if !payload.DryRun {
		t.Fatal("expected dryRun=true")
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func TestWebAppsDeleteRunSuccessReceiptUsesRereadState(t *testing.T) {
	var patchCalls, postPatchGets int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					removed := patchCalls > 0
					if removed {
						postPatchGets++
					}
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": ` + boolJSON(removed) + `,
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"id": "avail-1",
							"type": "appAvailabilities",
							"attributes": {"availableInNewTerritories": false},
							"relationships": {
								"availableTerritories": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/iris/v1/apps/1234567890":
					patchCalls++
					return webAppsDeleteJSONResponse(`{}`), nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	var payload struct {
		AppID    string `json:"appId"`
		Name     string `json:"name"`
		BundleID string `json:"bundleId"`
		Removed  bool   `json:"removed"`
		DryRun   bool   `json:"dryRun"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error: %v; stdout=%q", err, stdout)
	}
	if payload.AppID != "1234567890" || payload.Name != "Throwaway" || payload.BundleID != "com.example.throwaway" {
		t.Fatalf("unexpected success identity: %+v", payload)
	}
	if !payload.Removed {
		t.Fatal("expected verified removed=true from the post-PATCH re-read")
	}
	if payload.DryRun {
		t.Fatal("did not expect dryRun on the mutation receipt")
	}
	if patchCalls != 1 {
		t.Fatalf("expected one PATCH, got %d", patchCalls)
	}
	if postPatchGets < 1 {
		t.Fatalf("expected a post-PATCH re-read, got %d", postPatchGets)
	}
}

func TestWebAppsDeleteRunRequiresConfirmUnlessDryRun(t *testing.T) {
	root := RootCommand("1.0.0")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	_, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"web", "apps", "delete", "--app", "1234567890"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	if runErr == nil {
		t.Fatal("expected usage error")
	}
	if !strings.Contains(stderr, "--confirm is required unless --dry-run is set") {
		t.Fatalf("expected confirm-unless-dry-run stderr, got %q", stderr)
	}
}

func TestWebAppsDeleteRunRejectsOmittedDisplayableVersionsWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": false,
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							}
						}
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					t.Fatal("did not expect availability read when displayableVersions were omitted")
					return nil, nil
				case req.Method == http.MethodPatch && req.URL.Path == "/iris/v1/apps/1234567890":
					patchCalls++
					t.Fatal("did not expect PATCH when displayableVersions were omitted")
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitError {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitError, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "could not confirm") || !strings.Contains(stderr, "displayableVersions") {
		t.Fatalf("expected stderr to name missing version linkage, got %q", stderr)
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func TestWebAppsDeleteRunRejectsUnknownNewTerritoriesWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": false,
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"id": "avail-1",
							"type": "appAvailabilities",
							"attributes": {},
							"relationships": {
								"availableTerritories": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodPatch && req.URL.Path == "/iris/v1/apps/1234567890":
					patchCalls++
					t.Fatal("did not expect PATCH when availableInNewTerritories was unknown")
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitError {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitError, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "could not confirm") || !strings.Contains(stderr, "availableInNewTerritories") {
		t.Fatalf("expected stderr to name missing new-territory setting, got %q", stderr)
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func TestWebAppsDeleteRunAlreadyRemovedSkipsEligibilityWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": true,
								"appStoreLegacyStatus": "WAITING_FOR_REVIEW",
								"marketplace": "ALT_MARKETPLACE"
							}
						}
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					t.Fatal("already-removed apps must not read availability")
					return nil, nil
				case req.Method == http.MethodPatch && req.URL.Path == "/iris/v1/apps/1234567890":
					patchCalls++
					t.Fatal("already-removed apps must not PATCH")
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitSuccess {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitSuccess, stdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"removed":true`) {
		t.Fatalf("expected success receipt with removed=true, got %q", stdout)
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func TestWebAppsDeleteRunRejectsVersionsWithoutDecodedStateWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": false,
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {
									"data": [{"type": "appStoreVersions", "id": "version-1"}]
								}
							}
						},
						"included": [{
							"type": "appStoreVersions",
							"id": "version-1",
							"attributes": {
								"platform": "IOS",
								"versionString": "1.0"
							}
						}]
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					t.Fatal("did not expect availability read when version state was undecoded")
					return nil, nil
				case req.Method == http.MethodPatch && req.URL.Path == "/iris/v1/apps/1234567890":
					patchCalls++
					t.Fatal("did not expect PATCH when version state was undecoded")
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitError {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitError, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "could not confirm") || !strings.Contains(stderr, "displayableVersions") {
		t.Fatalf("expected stderr to name missing version linkage, got %q", stderr)
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func TestWebAppsDeleteRunRejectsMismatchedAppIDWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "9999999999",
							"attributes": {
								"name": "Other",
								"bundleId": "com.example.other",
								"removed": false,
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodPatch:
					patchCalls++
					t.Fatal("did not expect PATCH when GET returned a different app id")
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitError {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitError, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "1234567890") || !strings.Contains(stderr, "9999999999") {
		t.Fatalf("expected stderr to name both app ids, got %q", stderr)
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func TestWebAppsDeleteRunRejectsUnknownReviewStatusWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"removed": false,
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890/appAvailabilityV2":
					t.Fatal("did not expect availability read when review status was unknown")
					return nil, nil
				case req.Method == http.MethodPatch && req.URL.Path == "/iris/v1/apps/1234567890":
					patchCalls++
					t.Fatal("did not expect PATCH when review status was unknown")
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitError {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitError, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "could not confirm") || !strings.Contains(stderr, "appStoreLegacyStatus") {
		t.Fatalf("expected stderr to name missing app-level review status, got %q", stderr)
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func TestWebAppsDeleteRunRejectsUnknownRemovedWithoutPatch(t *testing.T) {
	var patchCalls int
	restoreSession := webcmd.SetResolveWebSession(func(ctx context.Context, appleID, password, twoFactorCode string) (*webcore.AuthSession, string, error) {
		return &webcore.AuthSession{
			Client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				switch {
				case req.Method == http.MethodGet && req.URL.Path == "/iris/v1/apps/1234567890":
					return webAppsDeleteJSONResponse(`{
						"data": {
							"type": "apps",
							"id": "1234567890",
							"attributes": {
								"name": "Throwaway",
								"bundleId": "com.example.throwaway",
								"appStoreLegacyStatus": "PREPARE_FOR_SUBMISSION",
								"marketplace": "APP_STORE"
							},
							"relationships": {
								"displayableVersions": {"data": []}
							}
						}
					}`), nil
				case req.Method == http.MethodPatch:
					patchCalls++
					t.Fatal("did not expect PATCH when removed was unknown")
					return nil, nil
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return nil, nil
				}
			})},
		}, "cache", nil
	})
	t.Cleanup(restoreSession)

	var code int
	stdout, stderr := captureOutput(t, func() {
		code = cmd.Run([]string{
			"web", "apps", "delete",
			"--app", "1234567890",
			"--confirm",
			"--output", "json",
		}, "1.0.0")
	})
	if code != cmd.ExitError {
		t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, cmd.ExitError, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "could not confirm") || !strings.Contains(stderr, "removed") {
		t.Fatalf("expected stderr to name missing removed attribute, got %q", stderr)
	}
	if patchCalls != 0 {
		t.Fatalf("expected no PATCH, got %d", patchCalls)
	}
}

func webAppsDeleteJSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func boolJSON(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
