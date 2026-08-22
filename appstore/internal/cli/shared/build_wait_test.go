package shared

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

type buildWaitRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn buildWaitRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type transientBuildWaitError struct{}

func (transientBuildWaitError) Error() string   { return "temporary network failure" }
func (transientBuildWaitError) Timeout() bool   { return true }
func (transientBuildWaitError) Temporary() bool { return true }

func newBuildWaitTestClient(t *testing.T, transport buildWaitRoundTripFunc) *asc.Client {
	t.Helper()

	keyPath := filepath.Join(t.TempDir(), "key.p8")
	writeECDSAPEM(t, keyPath)

	httpClient := &http.Client{Transport: transport}
	client, err := asc.NewClientWithHTTPClient("KEY123", "ISS456", keyPath, httpClient)
	if err != nil {
		t.Fatalf("NewClientWithHTTPClient() error: %v", err)
	}
	return client
}

func buildWaitJSONResponse(body string) (*http.Response, error) {
	return buildWaitJSONStatusResponse(http.StatusOK, body)
}

func buildWaitJSONStatusResponse(statusCode int, body string) (*http.Response, error) {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func TestWaitForBuildByNumberOrUploadFailureRejectsStaleBuildFromDifferentUpload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	uploadCalls := 0
	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}

		switch req.URL.Path {
		case "/v1/buildUploads/upload-current":
			uploadCalls++
			return buildWaitJSONResponse(`{
				"data": {
					"type": "buildUploads",
					"id": "upload-current",
					"attributes": {
						"cfBundleShortVersionString": "1.2.3",
						"cfBundleVersion": "42",
						"platform": "IOS",
						"state": {
							"state": "PROCESSING"
						}
					}
				}
			}`)
		case "/v1/preReleaseVersions":
			return buildWaitJSONResponse(`{
				"data": [
					{
						"type": "preReleaseVersions",
						"id": "prv-1",
						"attributes": {
							"version": "1.2.3",
							"platform": "IOS"
						}
					}
				],
				"links": {}
			}`)
		case "/v1/builds":
			if got := req.URL.Query().Get("include"); got != "buildUpload" {
				t.Fatalf("expected include=buildUpload when upload ID is known, got %q", got)
			}
			cancel()
			return buildWaitJSONResponse(`{
				"data": [
					{
						"type": "builds",
						"id": "stale-build",
						"attributes": {
							"version": "42",
							"uploadedDate": "2026-03-16T12:00:05Z"
						},
						"relationships": {
							"buildUpload": {
								"data": {
									"type": "buildUploads",
									"id": "stale-upload"
								}
							}
						}
					}
				],
				"links": {}
			}`)
		default:
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
	})

	_, err := WaitForBuildByNumberOrUploadFailure(ctx, client, "app-1", "upload-current", "1.2.3", "42", "IOS", time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation after rejecting stale build, got %v", err)
	}
	if uploadCalls == 0 {
		t.Fatal("expected build upload lookup before accepting a discovered build")
	}
}

func TestWaitForBuildByNumberOrUploadFailureReturnsBuildLinkedFromUpload(t *testing.T) {
	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}

		switch req.URL.Path {
		case "/v1/buildUploads/upload-current":
			return buildWaitJSONResponse(`{
				"data": {
					"type": "buildUploads",
					"id": "upload-current",
					"attributes": {
						"cfBundleShortVersionString": "1.2.3",
						"cfBundleVersion": "42",
						"platform": "IOS"
					},
					"relationships": {
						"build": {
							"data": {
								"type": "builds",
								"id": "build-123"
							}
						}
					}
				}
			}`)
		case "/v1/builds/build-123":
			return buildWaitJSONResponse(`{
				"data": {
					"type": "builds",
					"id": "build-123",
					"attributes": {
						"version": "42",
						"processingState": "PROCESSING"
					}
				}
			}`)
		case "/v1/preReleaseVersions", "/v1/builds":
			t.Fatalf("did not expect build discovery list request once upload links a build: %s", req.URL.Path)
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
	})

	buildResp, err := WaitForBuildByNumberOrUploadFailure(context.Background(), client, "app-1", "upload-current", "1.2.3", "42", "IOS", time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForBuildByNumberOrUploadFailure() error: %v", err)
	}
	if buildResp == nil {
		t.Fatal("expected linked build response")
		return
	}
	if buildResp.Data.ID != "build-123" {
		t.Fatalf("expected linked build ID build-123, got %q", buildResp.Data.ID)
	}
}

func TestWaitForBuildByNumberOrUploadFailureIncludesProcessingDiagnostics(t *testing.T) {
	restoreDiagnostics := SetBuildUploadFailureDiagnosticsForTesting(func(context.Context, *asc.Client, string, *asc.BuildUploadResponse) (string, error) {
		return `Invalid Siri Support. App Intent description "Searches Apple Music" cannot contain "apple"`, nil
	})
	t.Cleanup(restoreDiagnostics)

	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/buildUploads/upload-current" {
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
		return buildWaitJSONResponse(`{
			"data": {
				"type": "buildUploads",
				"id": "upload-current",
				"attributes": {
					"cfBundleShortVersionString": "1.2.3",
					"cfBundleVersion": "42",
					"platform": "IOS",
					"state": {
						"state": "FAILED",
						"errors": [
							{"code": "90626"}
						]
					}
				}
			}
		}`)
	})

	_, err := WaitForBuildByNumberOrUploadFailure(context.Background(), client, "app-1", "upload-current", "1.2.3", "42", "IOS", time.Millisecond)
	if err == nil {
		t.Fatal("expected build upload failure, got nil")
	}
	if !strings.Contains(err.Error(), `build upload "upload-current" failed with state FAILED: 90626`) {
		t.Fatalf("expected original upload failure details, got %v", err)
	}
	if !strings.Contains(err.Error(), `Invalid Siri Support. App Intent description "Searches Apple Music" cannot contain "apple"`) {
		t.Fatalf("expected enriched processing details, got %v", err)
	}
}

func TestWaitForBuildByNumberOrUploadFailureFallsBackWhenUploadLookupFails(t *testing.T) {
	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}

		switch req.URL.Path {
		case "/v1/buildUploads/upload-current":
			return buildWaitJSONStatusResponse(http.StatusNotFound, `{
				"errors": [
					{"status": "404", "code": "NOT_FOUND", "title": "not found"}
				]
			}`)
		case "/v1/preReleaseVersions":
			return buildWaitJSONResponse(`{
				"data": [
					{
						"type": "preReleaseVersions",
						"id": "prv-1",
						"attributes": {
							"version": "1.2.3",
							"platform": "IOS"
						}
					}
				],
				"links": {}
			}`)
		case "/v1/builds":
			return buildWaitJSONResponse(`{
				"data": [
					{
						"type": "builds",
						"id": "build-123",
						"attributes": {
							"version": "42",
							"processingState": "PROCESSING"
						},
						"relationships": {
							"buildUpload": {
								"data": {
									"type": "buildUploads",
									"id": "upload-current"
								}
							}
						}
					}
				],
				"links": {}
			}`)
		default:
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
	})

	buildResp, err := WaitForBuildByNumberOrUploadFailure(context.Background(), client, "app-1", "upload-current", "1.2.3", "42", "IOS", time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForBuildByNumberOrUploadFailure() error: %v", err)
	}
	if buildResp == nil {
		t.Fatal("expected build response after falling back to build discovery")
		return
	}
	if buildResp.Data.ID != "build-123" {
		t.Fatalf("expected build ID build-123, got %q", buildResp.Data.ID)
	}
}

func TestWaitForBuildByNumberOrUploadFailureFallsBackWhenLinkedBuildLookupFails(t *testing.T) {
	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}

		switch req.URL.Path {
		case "/v1/buildUploads/upload-current":
			return buildWaitJSONResponse(`{
				"data": {
					"type": "buildUploads",
					"id": "upload-current",
					"attributes": {
						"cfBundleShortVersionString": "1.2.3",
						"cfBundleVersion": "42",
						"platform": "IOS",
						"state": {
							"state": "PROCESSING"
						}
					},
					"relationships": {
						"build": {
							"data": {
								"type": "builds",
								"id": "build-123"
							}
						}
					}
				}
			}`)
		case "/v1/builds/build-123":
			return buildWaitJSONStatusResponse(http.StatusNotFound, `{
				"errors": [
					{"status": "404", "code": "NOT_FOUND", "title": "not found"}
				]
			}`)
		case "/v1/preReleaseVersions":
			return buildWaitJSONResponse(`{
				"data": [
					{
						"type": "preReleaseVersions",
						"id": "prv-1",
						"attributes": {
							"version": "1.2.3",
							"platform": "IOS"
						}
					}
				],
				"links": {}
			}`)
		case "/v1/builds":
			return buildWaitJSONResponse(`{
				"data": [
					{
						"type": "builds",
						"id": "build-123",
						"attributes": {
							"version": "42",
							"processingState": "PROCESSING"
						},
						"relationships": {
							"buildUpload": {
								"data": {
									"type": "buildUploads",
									"id": "upload-current"
								}
							}
						}
					}
				],
				"links": {}
			}`)
		default:
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
	})

	buildResp, err := WaitForBuildByNumberOrUploadFailure(context.Background(), client, "app-1", "upload-current", "1.2.3", "42", "IOS", time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForBuildByNumberOrUploadFailure() error: %v", err)
	}
	if buildResp == nil {
		t.Fatal("expected build response after falling back from linked build lookup")
		return
	}
	if buildResp.Data.ID != "build-123" {
		t.Fatalf("expected build ID build-123, got %q", buildResp.Data.ID)
	}
}

func TestWaitForBuildByNumberOrUploadFailureReturnsUploadLookupErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}

		switch req.URL.Path {
		case "/v1/buildUploads/upload-current":
			return buildWaitJSONStatusResponse(http.StatusUnauthorized, `{
				"errors": [
					{"status": "401", "code": "UNAUTHORIZED", "title": "unauthorized"}
				]
			}`)
		case "/v1/preReleaseVersions":
			cancel()
			return buildWaitJSONResponse(`{"data":[]}`)
		default:
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
	})

	_, err := WaitForBuildByNumberOrUploadFailure(ctx, client, "app-1", "upload-current", "1.2.3", "42", "IOS", time.Millisecond)
	if err == nil {
		t.Fatal("expected upload lookup error, got nil")
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("expected unauthorized upload lookup error, got %v", err)
	}
}

func TestWaitForBuildByNumberOrUploadFailureReturnsMalformedUploadRelationships(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}

		switch req.URL.Path {
		case "/v1/buildUploads/upload-current":
			return buildWaitJSONResponse(`{
				"data": {
					"type": "buildUploads",
					"id": "upload-current",
					"attributes": {
						"cfBundleShortVersionString": "1.2.3",
						"cfBundleVersion": "42",
						"platform": "IOS",
						"state": {
							"state": "PROCESSING"
						}
					},
					"relationships": {
						"build": "bad-shape"
					}
				}
			}`)
		case "/v1/preReleaseVersions":
			cancel()
			return buildWaitJSONResponse(`{"data":[]}`)
		default:
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
	})

	_, err := WaitForBuildByNumberOrUploadFailure(ctx, client, "app-1", "upload-current", "1.2.3", "42", "IOS", time.Millisecond)
	if err == nil {
		t.Fatal("expected malformed relationship error, got nil")
	}
	if !strings.Contains(err.Error(), `parse build upload "upload-current" relationships`) {
		t.Fatalf("expected malformed relationship error, got %v", err)
	}
}

func TestWaitForBuildByNumberOrUploadFailureToleratesTransientLookupFailures(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "0")

	preReleaseCalls := 0
	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}

		switch req.URL.Path {
		case "/v1/preReleaseVersions":
			preReleaseCalls++
			if preReleaseCalls <= 2 {
				return buildWaitJSONStatusResponse(http.StatusServiceUnavailable, `{
					"errors": [
						{"status": "503", "code": "SERVICE_UNAVAILABLE", "title": "unavailable"}
					]
				}`)
			}
			return buildWaitJSONResponse(`{
				"data": [
					{
						"type": "preReleaseVersions",
						"id": "prv-1",
						"attributes": {
							"version": "1.2.3",
							"platform": "IOS"
						}
					}
				],
				"links": {}
			}`)
		case "/v1/builds":
			return buildWaitJSONResponse(`{
				"data": [
					{
						"type": "builds",
						"id": "build-123",
						"attributes": {
							"version": "42",
							"processingState": "PROCESSING"
						}
					}
				],
				"links": {}
			}`)
		default:
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
	})

	var buildResp *asc.BuildResponse
	var err error
	stderr := captureStderr(t, func() {
		buildResp, err = WaitForBuildByNumberOrUploadFailure(context.Background(), client, "app-1", "", "1.2.3", "42", "IOS", time.Millisecond)
	})
	if err != nil {
		t.Fatalf("WaitForBuildByNumberOrUploadFailure() error: %v", err)
	}
	if buildResp == nil {
		t.Fatal("expected build response after tolerating transient failures")
		return
	}
	if buildResp.Data.ID != "build-123" {
		t.Fatalf("expected build ID build-123, got %q", buildResp.Data.ID)
	}
	for _, want := range []string{
		"transient App Store Connect error while waiting (1/5)",
		"transient App Store Connect error while waiting (2/5)",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("expected stderr to contain %q, got %q", want, stderr)
		}
	}
}

func TestWaitForBuildByNumberOrUploadFailureFailsAfterConsecutiveTransientLimit(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "0")

	preReleaseCalls := 0
	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/preReleaseVersions" {
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
		preReleaseCalls++
		return buildWaitJSONStatusResponse(http.StatusServiceUnavailable, `{
			"errors": [
				{"status": "503", "code": "SERVICE_UNAVAILABLE", "title": "unavailable"}
			]
		}`)
	})

	var err error
	captureStderr(t, func() {
		_, err = WaitForBuildByNumberOrUploadFailure(context.Background(), client, "app-1", "", "1.2.3", "42", "IOS", time.Millisecond)
	})
	if err == nil {
		t.Fatal("expected error once transient failures exceed the ceiling, got nil")
	}
	if !strings.Contains(err.Error(), "giving up after 6 consecutive transient App Store Connect errors") {
		t.Fatalf("expected consecutive transient failure error, got %v", err)
	}
	if preReleaseCalls != asc.DefaultMaxConsecutivePollFailures+1 {
		t.Fatalf("expected %d lookups, got %d", asc.DefaultMaxConsecutivePollFailures+1, preReleaseCalls)
	}
}

func TestVerifyBuildUploadAfterCommitIgnoresRetryableLookupErrorsUntilBuildLinks(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "0")
	lookupCalls := 0
	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/buildUploads/upload-current" {
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
		lookupCalls++
		if lookupCalls == 1 {
			return nil, transientBuildWaitError{}
		}
		return buildWaitJSONResponse(`{
			"data": {
				"type": "buildUploads",
				"id": "upload-current",
				"attributes": {
					"cfBundleShortVersionString": "1.2.3",
					"cfBundleVersion": "42",
					"platform": "IOS",
					"state": {"state": "UPLOADED"}
				},
				"relationships": {
					"build": {
						"data": {
							"type": "builds",
							"id": "build-123"
						}
					}
				}
			}
		}`)
	})

	err := VerifyBuildUploadAfterCommit(context.Background(), client, "app-1", "upload-current", time.Millisecond, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("VerifyBuildUploadAfterCommit() error: %v", err)
	}
	if lookupCalls < 2 {
		t.Fatalf("expected retryable lookup error to be retried, got %d lookup(s)", lookupCalls)
	}
}

func TestVerifyBuildUploadAfterCommitIgnoresRetryDelayBeyondVerificationBudget(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "5s")
	asc.ResetConfigCacheForTest()
	t.Cleanup(asc.ResetConfigCacheForTest)

	lookupCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/buildUploads/upload-current" {
			t.Errorf("unexpected path: %s", req.URL.Path)
		}
		lookupCalls++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{
			"errors": [{"status": "429", "code": "RATE_LIMIT_EXCEEDED", "title": "Too many requests"}]
		}`)
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		redirected := req.Clone(req.Context())
		redirected.URL.Scheme = serverURL.Scheme
		redirected.URL.Host = serverURL.Host
		redirected.Host = serverURL.Host
		return server.Client().Do(redirected)
	})

	verifyTimeout := 30 * time.Millisecond
	err = VerifyBuildUploadAfterCommit(context.Background(), client, "app-1", "upload-current", time.Millisecond, verifyTimeout)
	if err != nil {
		t.Fatalf("VerifyBuildUploadAfterCommit() error: %v", err)
	}
	if lookupCalls != 1 {
		t.Fatalf("expected one best-effort upload lookup before honoring Retry-After, got %d", lookupCalls)
	}
}

func TestResolveBuildStatusBundleIDReturnsAppBundleIDWhenSupported(t *testing.T) {
	previous := buildStatusBundleIDSupportedFn
	buildStatusBundleIDSupportedFn = func(context.Context) bool { return true }
	t.Cleanup(func() {
		buildStatusBundleIDSupportedFn = previous
	})

	client := newBuildWaitTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			return nil, fmt.Errorf("expected GET, got %s", req.Method)
		}
		if req.URL.Path != "/v1/apps/app-1" {
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
		return buildWaitJSONResponse(`{
			"data": {
				"type": "apps",
				"id": "app-1",
				"attributes": {
					"name": "Demo",
					"bundleId": "com.example.demo",
					"sku": "demo"
				}
			}
		}`)
	})

	bundleID := resolveBuildStatusBundleID(context.Background(), client, "app-1")
	if bundleID != "com.example.demo" {
		t.Fatalf("expected resolved bundle ID com.example.demo, got %q", bundleID)
	}
}

func TestBuildStatusPrivateKeyPathFallsBackToStoredPEMWhenPathMissing(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "AuthKey.p8")
	writeECDSAPEM(t, keyPath)

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	CleanupTempPrivateKeys()
	t.Cleanup(CleanupTempPrivateKeys)

	resolvedPath, err := buildStatusPrivateKeyPath(ResolvedAuthCredentials{
		KeyPath: filepath.Join(tempDir, "missing.p8"),
		KeyPEM:  string(keyData),
	})
	if err != nil {
		t.Fatalf("buildStatusPrivateKeyPath() error: %v", err)
	}
	if resolvedPath == filepath.Join(tempDir, "missing.p8") {
		t.Fatalf("expected fallback temp path, got missing configured path %q", resolvedPath)
	}
	if _, err := os.Stat(resolvedPath); err != nil {
		t.Fatalf("Stat(%q) error: %v", resolvedPath, err)
	}
	if _, err := asc.NewClient("KEY123", "ISS456", resolvedPath); err != nil {
		t.Fatalf("expected fallback private key path to be usable, got %v", err)
	}
}

func TestBuildStatusPrivateKeyPathPrefersStoredPEMOverExistingKeyPath(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "AuthKey-stale.p8")
	if err := os.WriteFile(keyPath, []byte("stale-key"), 0o600); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	validKeyPath := filepath.Join(tempDir, "AuthKey-valid.p8")
	writeECDSAPEM(t, validKeyPath)

	keyData, err := os.ReadFile(validKeyPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	CleanupTempPrivateKeys()
	t.Cleanup(CleanupTempPrivateKeys)

	resolvedPath, err := buildStatusPrivateKeyPath(ResolvedAuthCredentials{
		KeyPath: keyPath,
		KeyPEM:  string(keyData),
	})
	if err != nil {
		t.Fatalf("buildStatusPrivateKeyPath() error: %v", err)
	}
	if resolvedPath == keyPath {
		t.Fatalf("expected PEM-backed temp path instead of configured key path %q", resolvedPath)
	}
	if _, err := asc.NewClient("KEY123", "ISS456", resolvedPath); err != nil {
		t.Fatalf("expected PEM-backed fallback path to be usable, got %v", err)
	}
}

func TestBuildStatusPrivateKeyPathDecodesStoredBase64PEM(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "AuthKey-valid.p8")
	writeECDSAPEM(t, keyPath)

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	CleanupTempPrivateKeys()
	t.Cleanup(CleanupTempPrivateKeys)

	resolvedPath, err := buildStatusPrivateKeyPath(ResolvedAuthCredentials{
		KeyPEM: base64.StdEncoding.EncodeToString(keyData),
	})
	if err != nil {
		t.Fatalf("buildStatusPrivateKeyPath() error: %v", err)
	}
	if resolvedPath == "" {
		t.Fatal("expected decoded temp key path, got empty path")
	}
	resolvedData, err := os.ReadFile(resolvedPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", resolvedPath, err)
	}
	if string(resolvedData) != string(keyData) {
		t.Fatalf("expected decoded PEM data, got %q", string(resolvedData))
	}
	if _, err := asc.NewClient("KEY123", "ISS456", resolvedPath); err != nil {
		t.Fatalf("expected base64-decoded private key path to be usable, got %v", err)
	}
}
