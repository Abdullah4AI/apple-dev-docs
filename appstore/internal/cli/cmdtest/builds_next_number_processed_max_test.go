package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

const (
	processedBuildsQuery       = "filter%5Bapp%5D=100000001&limit=200&sort=-uploadedDate"
	processedBuildUploadsQuery = "filter%5Bstate%5D=AWAITING_UPLOAD%2CPROCESSING%2CCOMPLETE&limit=200"
)

type processedMaximumRequestStep struct {
	path         string
	rawQuery     string
	responseBody string
}

type processedMaximumRequestRecorder struct {
	t     *testing.T
	mu    sync.Mutex
	steps []processedMaximumRequestStep
	next  int
}

func (r *processedMaximumRequestRecorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.next >= len(r.steps) {
		r.t.Errorf("unexpected extra request: %s %s", req.Method, req.URL.RequestURI())
		http.Error(w, "unexpected request", http.StatusInternalServerError)
		return
	}

	step := r.steps[r.next]
	r.next++
	if req.Method != http.MethodGet {
		r.t.Errorf("request %d method = %s, want GET", r.next, req.Method)
	}
	if req.URL.EscapedPath() != step.path {
		r.t.Errorf("request %d path = %q, want %q", r.next, req.URL.EscapedPath(), step.path)
	}
	if req.URL.RawQuery != step.rawQuery {
		r.t.Errorf("request %d query = %q, want %q", r.next, req.URL.RawQuery, step.rawQuery)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, step.responseBody)
}

func (r *processedMaximumRequestRecorder) requireComplete() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.next != len(r.steps) {
		r.t.Errorf("requests = %d, want %d in exact order", r.next, len(r.steps))
	}
}

func setProcessedMaximumTestClient(t *testing.T, steps []processedMaximumRequestStep) {
	t.Helper()
	setupAuth(t)

	recorder := &processedMaximumRequestRecorder{t: t, steps: steps}
	server := httptest.NewServer(recorder)
	t.Cleanup(func() {
		server.Close()
		recorder.requireComplete()
	})

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.Scheme + "://" + req.URL.Host; got != asc.BaseURL {
			t.Fatalf("request origin = %s, want %s", got, asc.BaseURL)
		}
		authorization := req.Header.Get("Authorization")
		token, ok := strings.CutPrefix(authorization, "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			t.Fatalf("Authorization = %q, want nonempty Bearer token", authorization)
		}
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = serverURL.Scheme
		cloned.URL.Host = serverURL.Host
		return server.Client().Transport.RoundTrip(cloned)
	})
	client, err := asc.NewClientWithHTTPClient(
		"TEST_KEY",
		"TEST_ISSUER",
		os.Getenv("ASC_PRIVATE_KEY_PATH"),
		&http.Client{Transport: transport},
	)
	if err != nil {
		t.Fatalf("create test client: %v", err)
	}
	t.Cleanup(shared.SetASCClientFactoryForTesting(func() (*asc.Client, error) {
		return client, nil
	}))
}

type processedMaximumOutput struct {
	LatestProcessedBuildNumber *string  `json:"latestProcessedBuildNumber"`
	LatestUploadBuildNumber    *string  `json:"latestUploadBuildNumber"`
	LatestObservedBuildNumber  *string  `json:"latestObservedBuildNumber"`
	NextBuildNumber            string   `json:"nextBuildNumber"`
	SourcesConsidered          []string `json:"sourcesConsidered"`
}

func runProcessedMaximumCommand(t *testing.T) (string, string, error) {
	t.Helper()

	root := RootCommand("test")
	root.FlagSet.SetOutput(io.Discard)
	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"builds", "next-build-number", "--app", "100000001", "--output", "json"}); err != nil {
			t.Fatalf("parse: %v", err)
		}
		runErr = root.Run(context.Background())
	})
	return stdout, stderr, runErr
}

func decodeProcessedMaximumOutput(t *testing.T, stdout string) processedMaximumOutput {
	t.Helper()

	var output processedMaximumOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode output: %v; stdout=%s", err, stdout)
	}
	return output
}

func requireProcessedMaximumValue(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s = nil, want %q", label, want)
	}
	if *got != want {
		t.Fatalf("%s = %q, want %q", label, *got, want)
	}
}

func TestBuildsNextBuildNumberUsesHighestProcessedNumber(t *testing.T) {
	tests := []struct {
		name              string
		chronologicalBody string
		maximumBody       string
		wantLatest        string
		wantObserved      string
		wantNext          string
	}{
		{
			name: "older numeric maximum",
			chronologicalBody: `{"data":[
				{"type":"builds","id":"build-new-50","attributes":{"version":"50","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-51","attributes":{"version":"51","uploadedDate":"2026-02-02T00:00:00Z"}},
				{"type":"builds","id":"build-old-100","attributes":{"version":"100","uploadedDate":"2026-02-01T00:00:00Z"}}
			]}`,
			wantLatest:   "50",
			wantObserved: "100",
			wantNext:     "101",
		},
		{
			name: "dotted numeric maximum",
			chronologicalBody: `{"data":[
				{"type":"builds","id":"build-new-1-9","attributes":{"version":"1.9","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-1-10","attributes":{"version":"1.10","uploadedDate":"2026-02-02T00:00:00Z"}}
			]}`,
			wantLatest:   "1.9",
			wantObserved: "1.10",
			wantNext:     "1.11",
		},
		{
			name:              "chronological seed survives lower second read",
			chronologicalBody: `{"data":[{"type":"builds","id":"build-new-100","attributes":{"version":"100","uploadedDate":"2026-02-03T00:00:00Z"}}]}`,
			maximumBody:       `{"data":[{"type":"builds","id":"build-old-50","attributes":{"version":"50","uploadedDate":"2026-02-02T00:00:00Z"}}]}`,
			wantLatest:        "100",
			wantObserved:      "100",
			wantNext:          "101",
		},
		{
			name: "latest zero placeholder",
			chronologicalBody: `{"data":[
				{"type":"builds","id":"build-new-zero","attributes":{"version":"0","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-100","attributes":{"version":"100","uploadedDate":"2026-02-02T00:00:00Z"}}
			]}`,
			wantObserved: "100",
			wantNext:     "101",
		},
		{
			name: "latest zero-style placeholder",
			chronologicalBody: `{"data":[
				{"type":"builds","id":"build-new-zero-dot","attributes":{"version":"0.1","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-100","attributes":{"version":"100","uploadedDate":"2026-02-02T00:00:00Z"}}
			]}`,
			wantObserved: "100",
			wantNext:     "101",
		},
		{
			name: "only non-positive placeholders",
			chronologicalBody: `{"data":[
				{"type":"builds","id":"build-new-zero","attributes":{"version":"0","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-zero-dot","attributes":{"version":"0.1","uploadedDate":"2026-02-02T00:00:00Z"}}
			]}`,
			wantNext: "1",
		},
		{
			name: "older non-positive placeholders",
			chronologicalBody: `{"data":[
				{"type":"builds","id":"build-new-50","attributes":{"version":"50","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-zero","attributes":{"version":"0","uploadedDate":"2026-02-02T00:00:00Z"}},
				{"type":"builds","id":"build-old-zero-dot","attributes":{"version":"0.1","uploadedDate":"2026-02-01T00:00:00Z"}}
			]}`,
			wantLatest:   "50",
			wantObserved: "50",
			wantNext:     "51",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			maximumBody := test.maximumBody
			if maximumBody == "" {
				maximumBody = test.chronologicalBody
			}
			setProcessedMaximumTestClient(t, []processedMaximumRequestStep{
				{path: "/v1/builds", rawQuery: processedBuildsQuery, responseBody: test.chronologicalBody},
				{path: "/v1/builds", rawQuery: processedBuildsQuery, responseBody: maximumBody},
				{
					path:         "/v1/apps/100000001/buildUploads",
					rawQuery:     processedBuildUploadsQuery,
					responseBody: `{"data":[],"links":{"next":""}}`,
				},
			})

			stdout, stderr, runErr := runProcessedMaximumCommand(t)
			if runErr != nil {
				t.Fatalf("run: %v", runErr)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			output := decodeProcessedMaximumOutput(t, stdout)
			if test.wantLatest == "" {
				if output.LatestProcessedBuildNumber != nil {
					t.Fatalf("latestProcessedBuildNumber = %q, want nil for non-positive placeholder", *output.LatestProcessedBuildNumber)
				}
			} else {
				requireProcessedMaximumValue(t, "latestProcessedBuildNumber", output.LatestProcessedBuildNumber, test.wantLatest)
			}
			if test.wantObserved == "" {
				if output.LatestObservedBuildNumber != nil {
					t.Fatalf("latestObservedBuildNumber = %q, want nil without a positive build number", *output.LatestObservedBuildNumber)
				}
			} else {
				requireProcessedMaximumValue(t, "latestObservedBuildNumber", output.LatestObservedBuildNumber, test.wantObserved)
			}
			if output.NextBuildNumber != test.wantNext {
				t.Fatalf("nextBuildNumber = %q, want %q", output.NextBuildNumber, test.wantNext)
			}
			if test.wantObserved == "" {
				if len(output.SourcesConsidered) != 0 {
					t.Fatalf("sourcesConsidered = %v, want empty without a positive build number", output.SourcesConsidered)
				}
			} else if len(output.SourcesConsidered) != 1 || output.SourcesConsidered[0] != "processed_builds" {
				t.Fatalf("sourcesConsidered = %v, want [processed_builds]", output.SourcesConsidered)
			}
		})
	}
}

func TestBuildsNextBuildNumberScansEveryProcessedPageForMaximum(t *testing.T) {
	const page2 = "https://api.appstoreconnect.apple.com/v1/builds?cursor=page-2"
	const page3 = "https://api.appstoreconnect.apple.com/v1/builds?cursor=page-3"
	firstPageBody := `{"data":[{"type":"builds","id":"build-new-50","attributes":{"version":"50","uploadedDate":"2026-02-03T00:00:00Z"}}],"links":{"next":"` + page2 + `"}}`
	secondPageBody := `{"data":[{"type":"builds","id":"build-old-40","attributes":{"version":"40","uploadedDate":"2026-02-02T00:00:00Z"}}],"links":{"next":"` + page3 + `"}}`
	thirdPageBody := `{"data":[{"type":"builds","id":"build-old-100","attributes":{"version":"100","uploadedDate":"2026-02-01T00:00:00Z"}}],"links":{"next":""}}`
	setProcessedMaximumTestClient(t, []processedMaximumRequestStep{
		{path: "/v1/builds", rawQuery: processedBuildsQuery, responseBody: firstPageBody},
		{path: "/v1/builds", rawQuery: "cursor=page-2", responseBody: secondPageBody},
		{path: "/v1/builds", rawQuery: processedBuildsQuery, responseBody: firstPageBody},
		{path: "/v1/builds", rawQuery: "cursor=page-2", responseBody: secondPageBody},
		{path: "/v1/builds", rawQuery: "cursor=page-3", responseBody: thirdPageBody},
		{
			path:         "/v1/apps/100000001/buildUploads",
			rawQuery:     processedBuildUploadsQuery,
			responseBody: `{"data":[],"links":{"next":""}}`,
		},
	})

	stdout, stderr, runErr := runProcessedMaximumCommand(t)
	if runErr != nil {
		t.Fatalf("run: %v", runErr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	output := decodeProcessedMaximumOutput(t, stdout)
	requireProcessedMaximumValue(t, "latestProcessedBuildNumber", output.LatestProcessedBuildNumber, "50")
	requireProcessedMaximumValue(t, "latestObservedBuildNumber", output.LatestObservedBuildNumber, "100")
	if output.NextBuildNumber != "101" {
		t.Fatalf("nextBuildNumber = %q, want 101", output.NextBuildNumber)
	}
}

func TestBuildsNextBuildNumberSkipsMalformedOlderProcessedNumber(t *testing.T) {
	responseBody := `{"data":[
				{"type":"builds","id":"build-new-50","attributes":{"version":"50","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-bad","attributes":{"version":"not-a-number","uploadedDate":"2026-02-02T00:00:00Z"}}
			]}`
	setProcessedMaximumTestClient(t, []processedMaximumRequestStep{
		{path: "/v1/builds", rawQuery: processedBuildsQuery, responseBody: responseBody},
		{path: "/v1/builds", rawQuery: processedBuildsQuery, responseBody: responseBody},
		{
			path:         "/v1/apps/100000001/buildUploads",
			rawQuery:     processedBuildUploadsQuery,
			responseBody: `{"data":[],"links":{"next":""}}`,
		},
	})

	stdout, stderr, runErr := runProcessedMaximumCommand(t)
	if runErr != nil {
		t.Fatalf("run: %v", runErr)
	}
	output := decodeProcessedMaximumOutput(t, stdout)
	requireProcessedMaximumValue(t, "latestProcessedBuildNumber", output.LatestProcessedBuildNumber, "50")
	requireProcessedMaximumValue(t, "latestObservedBuildNumber", output.LatestObservedBuildNumber, "50")
	if output.NextBuildNumber != "51" {
		t.Fatalf("nextBuildNumber = %q, want 51", output.NextBuildNumber)
	}
	requireBuildHistoryScanWarnings(t, stderr, []string{
		`Warning: skipping processed build build-old-bad: build number "not-a-number" is not a positive integer`,
	})
}

func TestBuildsNextBuildNumberSkipsMalformedLatestProcessedNumber(t *testing.T) {
	responseBody := `{"data":[
				{"type":"builds","id":"build-new-bad","attributes":{"version":"not-a-number","uploadedDate":"2026-02-03T00:00:00Z"}},
				{"type":"builds","id":"build-old-50","attributes":{"version":"50","uploadedDate":"2026-02-02T00:00:00Z"}}
			]}`
	setProcessedMaximumTestClient(t, []processedMaximumRequestStep{
		{path: "/v1/builds", rawQuery: processedBuildsQuery, responseBody: responseBody},
		{path: "/v1/builds", rawQuery: processedBuildsQuery, responseBody: responseBody},
		{
			path:         "/v1/apps/100000001/buildUploads",
			rawQuery:     processedBuildUploadsQuery,
			responseBody: `{"data":[],"links":{"next":""}}`,
		},
	})

	stdout, stderr, runErr := runProcessedMaximumCommand(t)
	if runErr != nil {
		t.Fatalf("run: %v", runErr)
	}
	output := decodeProcessedMaximumOutput(t, stdout)
	if output.LatestProcessedBuildNumber != nil {
		t.Fatalf("latestProcessedBuildNumber = %q, want nil for an unusable build number", *output.LatestProcessedBuildNumber)
	}
	requireProcessedMaximumValue(t, "latestObservedBuildNumber", output.LatestObservedBuildNumber, "50")
	if output.NextBuildNumber != "51" {
		t.Fatalf("nextBuildNumber = %q, want 51", output.NextBuildNumber)
	}
	requireBuildHistoryScanWarnings(t, stderr, []string{
		`Warning: skipping processed build build-new-bad: build number "not-a-number" is not a positive integer`,
	})
}
