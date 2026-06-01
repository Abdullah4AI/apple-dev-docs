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

func TestDevicesRegisterReusesExistingDeviceWithNormalizedUDID(t *testing.T) {
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
			if req.Method != http.MethodGet {
				t.Fatalf("expected GET before create, got %s", req.Method)
			}
			if req.URL.Path != "/v1/devices" {
				t.Fatalf("expected devices list path, got %s", req.URL.Path)
			}
			query := req.URL.Query()
			if got := query.Get("filter[platform]"); got != "IOS" {
				t.Fatalf("expected filter[platform]=IOS, got %q", got)
			}
			body := `{"data":[{"type":"devices","id":"device-existing","attributes":{"name":"Existing iPhone","platform":"IOS","udid":"ABC-123-DEF","status":"ENABLED"}}],"links":{"next":""}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("expected register to reuse existing device without POST, got request %d: %s %s", requestCount, req.Method, req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"devices", "register", "--name", "Existing iPhone", "--udid", "ABC123DEF", "--platform", "IOS"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"device-existing"`) {
		t.Fatalf("expected existing device output, got %q", stdout)
	}
}

func TestDevicesRegisterStopsScanningAfterExistingUDIDMatch(t *testing.T) {
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
			if req.Method != http.MethodGet {
				t.Fatalf("expected GET before create, got %s", req.Method)
			}
			if req.URL.Path != "/v1/devices" {
				t.Fatalf("expected devices list path, got %s", req.URL.Path)
			}
			body := `{"data":[{"type":"devices","id":"device-existing","attributes":{"name":"Existing iPhone","platform":"IOS","udid":"ABC-123-DEF","status":"ENABLED"}}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/devices?cursor=next"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("expected register to stop after first matching page, got request %d: %s %s", requestCount, req.Method, req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"devices", "register", "--name", "Existing iPhone", "--udid", "ABC123DEF", "--platform", "IOS"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"device-existing"`) {
		t.Fatalf("expected existing device output, got %q", stdout)
	}
	if requestCount != 1 {
		t.Fatalf("expected one preflight request, got %d", requestCount)
	}
}

func TestDevicesRegisterCreatesDeviceWhenNoExistingUDIDMatches(t *testing.T) {
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
			if req.Method != http.MethodGet {
				t.Fatalf("expected GET before create, got %s", req.Method)
			}
			if req.URL.Path != "/v1/devices" {
				t.Fatalf("expected devices list path, got %s", req.URL.Path)
			}
			if got := req.URL.Query().Get("filter[platform]"); got != "IOS" {
				t.Fatalf("expected filter[platform]=IOS, got %q", got)
			}
			body := `{"data":[],"links":{"next":""}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		case 2:
			if req.Method != http.MethodPost {
				t.Fatalf("expected POST after no existing match, got %s", req.Method)
			}
			if req.URL.Path != "/v1/devices" {
				t.Fatalf("expected create path, got %s", req.URL.Path)
			}
			var payload struct {
				Data struct {
					Type       string `json:"type"`
					Attributes struct {
						Name     string `json:"name"`
						UDID     string `json:"udid"`
						Platform string `json:"platform"`
					} `json:"attributes"`
				} `json:"data"`
			}
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode create payload: %v", err)
			}
			if payload.Data.Type != "devices" {
				t.Fatalf("expected type devices, got %q", payload.Data.Type)
			}
			if payload.Data.Attributes.Name != "New iPhone" {
				t.Fatalf("expected name New iPhone, got %q", payload.Data.Attributes.Name)
			}
			if payload.Data.Attributes.UDID != "NEW-UDID" {
				t.Fatalf("expected UDID NEW-UDID, got %q", payload.Data.Attributes.UDID)
			}
			if payload.Data.Attributes.Platform != "IOS" {
				t.Fatalf("expected platform IOS, got %q", payload.Data.Attributes.Platform)
			}
			body := `{"data":{"type":"devices","id":"device-new","attributes":{"name":"New iPhone","platform":"IOS","udid":"NEW-UDID","status":"ENABLED"}}}`
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		default:
			t.Fatalf("unexpected request %d: %s %s", requestCount, req.Method, req.URL.String())
			return nil, nil
		}
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"devices", "register", "--name", "New iPhone", "--udid", "NEW-UDID", "--platform", "IOS"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"device-new"`) {
		t.Fatalf("expected created device output, got %q", stdout)
	}
	if requestCount != 2 {
		t.Fatalf("expected GET and POST requests, got %d", requestCount)
	}
}
