package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

const resolveTestersMetricsBody = `{
  "data": [
    {
      "dataPoints": [{"start": "2026-08-01", "end": "2026-08-07", "values": {"crashCount": 0, "sessionCount": 3, "feedbackCount": 1}}],
      "dimensions": {"betaTesters": {"data": "tester-1"}}
    },
    {
      "dataPoints": [{"start": "2026-08-01", "end": "2026-08-07", "values": {"crashCount": 1, "sessionCount": 5, "feedbackCount": 0}}],
      "dimensions": {"betaTesters": {"data": "tester-2"}}
    }
  ],
  "included": [
    {"type": "betaTesters", "id": "tester-1", "attributes": {"firstName": "Ada", "lastName": "Lovelace", "email": "ada@example.com", "inviteType": "EMAIL", "state": "INSTALLED"}}
  ],
  "links": {"next": ""}
}`

func setResolveTestersEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ASC_APP_ID", "")

	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "key.p8")
	writeECDSAPEM(t, keyPath)
	t.Setenv("ASC_KEY_ID", "TEST_KEY")
	t.Setenv("ASC_ISSUER_ID", "TEST_ISSUER")
	t.Setenv("ASC_PRIVATE_KEY_PATH", keyPath)
}

func TestTestFlightMetricsBetaTesterUsagesResolveTesters(t *testing.T) {
	setResolveTestersEnv(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	callCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		switch callCount {
		case 1:
			if req.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", req.Method)
			}
			if req.URL.Path != "/v1/apps/app-123/metrics/betaTesterUsages" {
				t.Fatalf("expected metrics path, got %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(resolveTestersMetricsBody)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 2:
			if req.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", req.Method)
			}
			if req.URL.Path != "/v1/betaTesters" {
				t.Fatalf("expected path /v1/betaTesters, got %s", req.URL.Path)
			}
			values := req.URL.Query()
			// tester-1 is satisfied from the metrics page's included resources,
			// so only tester-2 needs a follow-up lookup.
			if got := values.Get("filter[id]"); got != "tester-2" {
				t.Fatalf("expected filter[id]=tester-2, got %q", got)
			}
			if got := values.Get("limit"); got != "200" {
				t.Fatalf("expected limit=200, got %q", got)
			}
			body := `{"data":[{"type":"betaTesters","id":"tester-2","attributes":{"firstName":"Grace","lastName":"Hopper","email":"grace@example.com","inviteType":"PUBLIC_LINK","state":"INVITED"}}],"links":{"next":""}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("unexpected request %d to %s", callCount, req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"testflight", "metrics", "app-testers", "--app", "app-123", "--resolve-testers"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 requests, got %d", callCount)
	}

	var printed struct {
		Data    []json.RawMessage `json:"data"`
		Testers map[string]struct {
			ID         string `json:"id"`
			Email      string `json:"email"`
			FirstName  string `json:"firstName"`
			LastName   string `json:"lastName"`
			State      string `json:"state"`
			InviteType string `json:"inviteType"`
		} `json:"testers"`
	}
	if err := json.Unmarshal([]byte(stdout), &printed); err != nil {
		t.Fatalf("failed to parse printed JSON: %v\noutput: %q", err, stdout)
	}
	if len(printed.Data) != 2 {
		t.Fatalf("expected 2 data rows, got %d", len(printed.Data))
	}
	if len(printed.Testers) != 2 {
		t.Fatalf("expected 2 resolved testers, got %d", len(printed.Testers))
	}

	included, ok := printed.Testers["tester-1"]
	if !ok {
		t.Fatalf("expected tester-1 in testers sidecar, got %q", stdout)
	}
	if included.ID != "tester-1" || included.Email != "ada@example.com" || included.FirstName != "Ada" || included.LastName != "Lovelace" || included.State != "INSTALLED" || included.InviteType != "EMAIL" {
		t.Fatalf("unexpected tester-1 sidecar entry: %+v", included)
	}

	fetched, ok := printed.Testers["tester-2"]
	if !ok {
		t.Fatalf("expected tester-2 in testers sidecar, got %q", stdout)
	}
	if fetched.ID != "tester-2" || fetched.Email != "grace@example.com" || fetched.FirstName != "Grace" || fetched.LastName != "Hopper" || fetched.State != "INVITED" || fetched.InviteType != "PUBLIC_LINK" {
		t.Fatalf("unexpected tester-2 sidecar entry: %+v", fetched)
	}
}

func TestTestFlightMetricsBetaTesterUsagesWithoutResolveTestersOmitsSidecar(t *testing.T) {
	setResolveTestersEnv(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	callCount := 0
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		if callCount > 1 {
			t.Fatalf("unexpected request %d to %s", callCount, req.URL.String())
		}
		if req.URL.Path != "/v1/apps/app-123/metrics/betaTesterUsages" {
			t.Fatalf("expected metrics path, got %s", req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(resolveTestersMetricsBody)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"testflight", "metrics", "app-testers", "--app", "app-123"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "\"tester-1\"") || !strings.Contains(stdout, "\"tester-2\"") {
		t.Fatalf("expected raw usage rows in output, got %q", stdout)
	}

	var printed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &printed); err != nil {
		t.Fatalf("failed to parse printed JSON: %v\noutput: %q", err, stdout)
	}
	if _, ok := printed["testers"]; ok {
		t.Fatalf("expected no testers key without --resolve-testers, got %q", stdout)
	}
}
