package cmdtest

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSandboxListNormalizesTerritoryFilter(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", req.Method)
		}
		if req.URL.Path != "/v2/sandboxTesters" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{
			"data":[
				{"type":"sandboxTesters","id":"tester-usa","attributes":{"email":"usa@example.com","territory":"USA"}},
				{"type":"sandboxTesters","id":"tester-fra","attributes":{"email":"fra@example.com","territory":"FRA"}}
			]
		}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"sandbox", "list", "--territory", "United States"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	assertEmptyStderr(t, stderr)
	if !strings.Contains(stdout, `"id":"tester-usa"`) {
		t.Fatalf("expected USA tester in output, got %q", stdout)
	}
	if strings.Contains(stdout, `"id":"tester-fra"`) {
		t.Fatalf("did not expect FRA tester in output, got %q", stdout)
	}
}

func TestSandboxListAllowsEmptyTerritoryFilter(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", req.Method)
		}
		if req.URL.Path != "/v2/sandboxTesters" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{
			"data":[
				{"type":"sandboxTesters","id":"tester-usa","attributes":{"territory":"USA"}},
				{"type":"sandboxTesters","id":"tester-fra","attributes":{"territory":"FRA"}}
			]
		}`)
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"sandbox", "list", "--territory="}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	assertEmptyStderr(t, stderr)
	if !strings.Contains(stdout, `"id":"tester-usa"`) || !strings.Contains(stdout, `"id":"tester-fra"`) {
		t.Fatalf("expected unfiltered tester output, got %q", stdout)
	}
}

func TestSandboxListKeepsTerritoryFilterAcrossPagination(t *testing.T) {
	stdout, stderr, requests := runSandboxListFilterPagination(
		t,
		[]string{"--territory", "United States"},
		"",
		`{"type":"sandboxTesters","id":"tester-usa","attributes":{"territory":"USA"}}`,
		`{"type":"sandboxTesters","id":"tester-fra","attributes":{"territory":"FRA"}}`,
	)

	assertEmptyStderr(t, stderr)
	if len(requests) != 2 {
		t.Fatalf("expected two paginated requests, got %d (%v)", len(requests), requests)
	}
	if !strings.Contains(stdout, `"id":"tester-usa"`) {
		t.Fatalf("expected USA tester in output, got %q", stdout)
	}
	if strings.Contains(stdout, `"id":"tester-fra"`) {
		t.Fatalf("did not expect FRA tester in output, got %q", stdout)
	}
}

func TestSandboxListKeepsEmailFilterAcrossPagination(t *testing.T) {
	stdout, stderr, requests := runSandboxListFilterPagination(
		t,
		[]string{"--email", "usa@example.com"},
		"",
		`{"type":"sandboxTesters","id":"tester-usa","attributes":{"email":"usa@example.com"}}`,
		`{"type":"sandboxTesters","id":"tester-other","attributes":{"email":"other@example.com"}}`,
	)

	assertEmptyStderr(t, stderr)
	if len(requests) != 2 {
		t.Fatalf("expected two paginated requests, got %d (%v)", len(requests), requests)
	}
	if !strings.Contains(stdout, `"id":"tester-usa"`) {
		t.Fatalf("expected matching tester in output, got %q", stdout)
	}
	if strings.Contains(stdout, `"id":"tester-other"`) {
		t.Fatalf("did not expect nonmatching tester in output, got %q", stdout)
	}
}

func TestSandboxListKeepsFilterAcrossNextPagination(t *testing.T) {
	firstURL := "https://api.appstoreconnect.apple.com/v2/sandboxTesters?cursor=page-1"
	stdout, stderr, requests := runSandboxListFilterPagination(
		t,
		[]string{"--territory", "USA"},
		firstURL,
		`{"type":"sandboxTesters","id":"tester-usa","attributes":{"territory":"USA"}}`,
		`{"type":"sandboxTesters","id":"tester-fra","attributes":{"territory":"FRA"}}`,
	)

	assertEmptyStderr(t, stderr)
	if len(requests) != 2 {
		t.Fatalf("expected two paginated requests, got %d (%v)", len(requests), requests)
	}
	if requests[0] != firstURL {
		t.Fatalf("expected initial request to use --next URL %q, got %q", firstURL, requests[0])
	}
	if !strings.Contains(stdout, `"id":"tester-usa"`) {
		t.Fatalf("expected USA tester in output, got %q", stdout)
	}
	if strings.Contains(stdout, `"id":"tester-fra"`) {
		t.Fatalf("did not expect FRA tester in output, got %q", stdout)
	}
}

func runSandboxListFilterPagination(t *testing.T, filterArgs []string, initialNext, firstData, secondData string) (string, string, []string) {
	t.Helper()
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	secondURL := "https://api.appstoreconnect.apple.com/v2/sandboxTesters?cursor=page-2"
	requests := []string{}
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", req.Method)
		}
		if req.URL.Path != "/v2/sandboxTesters" {
			t.Fatalf("unexpected path %q", req.URL.Path)
		}

		requests = append(requests, req.URL.String())
		switch len(requests) {
		case 1:
			if initialNext == "" && req.URL.Query().Get("limit") != "200" {
				t.Fatalf("expected limit=200 for --paginate, got %q", req.URL.Query().Get("limit"))
			}
			return jsonResponse(http.StatusOK, `{"data":[`+firstData+`],"links":{"next":"`+secondURL+`"}}`)
		case 2:
			if req.URL.String() != secondURL {
				t.Fatalf("expected continuation URL %q, got %q", secondURL, req.URL.String())
			}
			return jsonResponse(http.StatusOK, `{"data":[`+secondData+`]}`)
		default:
			t.Fatalf("unexpected request %d: %s", len(requests), req.URL)
			return nil, nil
		}
	})

	args := append([]string{"sandbox", "list"}, filterArgs...)
	args = append(args, "--paginate", "--output", "json")
	if initialNext != "" {
		args = append(args, "--next", initialNext)
	}

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

	return stdout, stderr, requests
}
