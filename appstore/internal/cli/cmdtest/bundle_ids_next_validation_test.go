package cmdtest

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func runBundleIDsInvalidNextURLCases(
	t *testing.T,
	argsPrefix []string,
	wantErrPrefix string,
) {
	t.Helper()

	tests := []struct {
		name    string
		next    string
		wantErr string
	}{
		{
			name:    "invalid scheme",
			next:    "http://api.appstoreconnect.apple.com/v1/bundleIds?cursor=AQ",
			wantErr: wantErrPrefix + " must be an App Store Connect URL",
		},
		{
			name:    "malformed URL",
			next:    "https://api.appstoreconnect.apple.com/%zz",
			wantErr: wantErrPrefix + " must be a valid URL:",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append(append([]string{}, argsPrefix...), "--next", test.next)

			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			var runErr error
			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				runErr = root.Run(context.Background())
			})

			if runErr == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(runErr.Error(), test.wantErr) {
				t.Fatalf("expected error %q, got %v", test.wantErr, runErr)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
		})
	}
}

func runBundleIDsPaginateFromNext(
	t *testing.T,
	argsPrefix []string,
	firstURL string,
	secondURL string,
	firstBody string,
	secondBody string,
	wantIDs ...string,
) {
	t.Helper()

	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	requestCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestCount++
		switch requestCount {
		case 1:
			if req.Method != http.MethodGet || req.URL.String() != firstURL {
				t.Fatalf("unexpected first request: %s %s", req.Method, req.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(firstBody)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 2:
			if req.Method != http.MethodGet || req.URL.String() != secondURL {
				t.Fatalf("unexpected second request: %s %s", req.Method, req.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(secondBody)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("unexpected extra request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	args := append(append([]string{}, argsPrefix...), "--paginate", "--next", firstURL)

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse(args); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	for _, id := range wantIDs {
		needle := `"id":"` + id + `"`
		if !strings.Contains(stdout, needle) {
			t.Fatalf("expected output to contain %q, got %q", needle, stdout)
		}
	}
}

func TestBundleIDsListRejectsInvalidNextURL(t *testing.T) {
	runBundleIDsInvalidNextURLCases(
		t,
		[]string{"bundle-ids", "list"},
		"bundle-ids list: --next",
	)
}

func TestBundleIDsListPaginateFromNext(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/bundleIds?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/bundleIds?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"bundleIds","id":"bundle-id-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"bundleIds","id":"bundle-id-next-2"}],"links":{"next":""}}`

	runBundleIDsPaginateFromNext(
		t,
		[]string{"bundle-ids", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"bundle-id-next-1",
		"bundle-id-next-2",
	)
}

func TestBundleIDCapabilitiesListRejectsInvalidNextURL(t *testing.T) {
	runBundleIDsInvalidNextURLCases(
		t,
		[]string{"bundle-ids", "capabilities", "list"},
		"bundle-ids capabilities list: --next",
	)
}

func TestBundleIDCapabilitiesListPaginateFromNextWithoutBundle(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/bundleIds/bundle-1/bundleIdCapabilities?cursor=AQ"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/bundleIds/bundle-1/bundleIdCapabilities?cursor=BQ"

	firstBody := `{"data":[{"type":"bundleIdCapabilities","id":"bundle-capability-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"bundleIdCapabilities","id":"bundle-capability-next-2"}],"links":{"next":""}}`

	runBundleIDsPaginateFromNext(
		t,
		[]string{"bundle-ids", "capabilities", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"bundle-capability-next-1",
		"bundle-capability-next-2",
	)
}

func TestBundleIDProfilesListRejectsInvalidNextURL(t *testing.T) {
	runBundleIDsInvalidNextURLCases(
		t,
		[]string{"bundle-ids", "profiles", "list"},
		"bundle-ids profiles list: --next",
	)
}

func TestBundleIDProfilesListPaginateFromNextWithoutID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/bundleIds/bundle-1/profiles?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/bundleIds/bundle-1/profiles?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"profiles","id":"bundle-profile-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"profiles","id":"bundle-profile-next-2"}],"links":{"next":""}}`

	runBundleIDsPaginateFromNext(
		t,
		[]string{"bundle-ids", "profiles", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"bundle-profile-next-1",
		"bundle-profile-next-2",
	)
}
