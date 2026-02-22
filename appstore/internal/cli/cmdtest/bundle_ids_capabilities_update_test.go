package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleIDCapabilitiesUpdateMissingID(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"bundle-ids", "capabilities", "update", "--settings", `[{"key":"K"}]`}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--id is required") {
		t.Fatalf("expected --id is required error, got %q", stderr)
	}
}

func TestBundleIDCapabilitiesUpdateNoUpdateFields(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"bundle-ids", "capabilities", "update", "--id", "cap1"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "at least one update field is required") {
		t.Fatalf("expected update field required error, got %q", stderr)
	}
}

func TestBundleIDCapabilitiesUpdateEmptySettingsArrayNoUpdateFields(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"bundle-ids", "capabilities", "update", "--id", "cap1", "--settings", "[]"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "at least one update field is required") {
		t.Fatalf("expected update field required error, got %q", stderr)
	}
}

func TestBundleIDCapabilitiesUpdateInvalidSettingsJSON(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"bundle-ids", "capabilities", "update", "--id", "cap1", "--settings", "not-json"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		err := root.Run(context.Background())
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--settings must be valid JSON array") {
		t.Fatalf("expected invalid JSON error, got %q", stderr)
	}
}

func TestBundleIDCapabilitiesUpdateSuccessOutput(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", req.Method)
		}
		if req.URL.Path != "/v1/bundleIdCapabilities/cap1" {
			t.Fatalf("expected path /v1/bundleIdCapabilities/cap1, got %s", req.URL.Path)
		}
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body error: %v", err)
		}
		var body map[string]any
		if err := json.Unmarshal(payload, &body); err != nil {
			t.Fatalf("decode body error: %v", err)
		}
		data, ok := body["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data object in body")
		}
		if data["type"] != "bundleIdCapabilities" {
			t.Fatalf("expected type bundleIdCapabilities, got %v", data["type"])
		}
		if data["id"] != "cap1" {
			t.Fatalf("expected id cap1, got %v", data["id"])
		}
		attrs, ok := data["attributes"].(map[string]any)
		if !ok {
			t.Fatalf("expected attributes object in body")
		}
		settings, ok := attrs["settings"].([]any)
		if !ok || len(settings) != 1 {
			t.Fatalf("expected 1 setting, got %v", attrs["settings"])
		}

		respBody := `{"data":{"type":"bundleIdCapabilities","id":"cap1","attributes":{"capabilityType":"ICLOUD","settings":[{"key":"ICLOUD_VERSION","options":[{"key":"XCODE_13","enabled":true}]}]}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(respBody)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"bundle-ids", "capabilities", "update", "--id", "cap1", "--settings", `[{"key":"ICLOUD_VERSION","options":[{"key":"XCODE_13","enabled":true}]}]`}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"cap1"`) {
		t.Fatalf("expected capability id in output, got %q", stdout)
	}
	if !strings.Contains(stdout, `"capabilityType":"ICLOUD"`) {
		t.Fatalf("expected capabilityType in output, got %q", stdout)
	}
}

func TestBundleIDCapabilitiesUpdateWithCapabilityType(t *testing.T) {
	setupAuth(t)
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", req.Method)
		}
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("read body error: %v", err)
		}
		if !strings.Contains(string(payload), `"capabilityType":"PUSH_NOTIFICATIONS"`) {
			t.Fatalf("expected capabilityType in body, got %s", string(payload))
		}

		respBody := `{"data":{"type":"bundleIdCapabilities","id":"cap1","attributes":{"capabilityType":"PUSH_NOTIFICATIONS"}}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(respBody)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"bundle-ids", "capabilities", "update", "--id", "cap1", "--capability", "PUSH_NOTIFICATIONS"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, `"id":"cap1"`) {
		t.Fatalf("expected capability id in output, got %q", stdout)
	}
}
