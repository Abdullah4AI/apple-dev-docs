package cmdtest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	appclipscli "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/appclips"
)

func TestAppClipsInvocationsCreateSupportsInlineLocalization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/betaAppClipInvocations" {
			t.Fatalf("expected path /v1/betaAppClipInvocations, got %s", r.URL.Path)
		}

		var payload asc.BetaAppClipInvocationCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Data.Relationships == nil || payload.Data.Relationships.BuildBundle == nil {
			t.Fatal("expected build bundle relationship")
		}
		if payload.Data.Relationships.BuildBundle.Data.ID != "bundle-1" {
			t.Fatalf("expected build bundle id bundle-1, got %s", payload.Data.Relationships.BuildBundle.Data.ID)
		}
		if payload.Data.Relationships.BetaAppClipInvocationLocalizations == nil {
			t.Fatal("expected localization relationships")
		}
		if len(payload.Data.Relationships.BetaAppClipInvocationLocalizations.Data) != 1 {
			t.Fatalf("expected one localization relationship, got %d", len(payload.Data.Relationships.BetaAppClipInvocationLocalizations.Data))
		}
		if payload.Data.Relationships.BetaAppClipInvocationLocalizations.Data[0].ID != "${localization-1}" {
			t.Fatalf("expected placeholder localization id, got %s", payload.Data.Relationships.BetaAppClipInvocationLocalizations.Data[0].ID)
		}
		if len(payload.Included) != 1 {
			t.Fatalf("expected one included localization, got %d", len(payload.Included))
		}
		if payload.Included[0].Attributes.Locale != "en-US" {
			t.Fatalf("expected locale en-US, got %s", payload.Included[0].Attributes.Locale)
		}
		if payload.Included[0].Attributes.Title != "Try it" {
			t.Fatalf("expected title Try it, got %s", payload.Included[0].Attributes.Title)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"type":"betaAppClipInvocations","id":"inv-1","attributes":{"url":"https://example.com/clip"}},"links":{}}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	keyPath := t.TempDir() + "/key.p8"
	writeECDSAPEM(t, keyPath)

	transport := appClipsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = serverURL.Scheme
		cloned.URL.Host = serverURL.Host
		return server.Client().Transport.RoundTrip(cloned)
	})
	client, err := asc.NewClientWithHTTPClient("TEST_KEY", "TEST_ISSUER", keyPath, &http.Client{Transport: transport})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	restore := appclipscli.SetClientFactory(func() (*asc.Client, error) {
		return client, nil
	})
	t.Cleanup(restore)

	root := RootCommand("1.2.3")
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"app-clips", "invocations", "create",
			"--build-bundle-id", "bundle-1",
			"--url", "https://example.com/clip",
			"--locale", "en-US",
			"--title", "Try it",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response asc.BetaAppClipInvocationResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout=%q", err, stdout)
	}
	if response.Data.ID != "inv-1" {
		t.Fatalf("expected invocation id inv-1, got %s", response.Data.ID)
	}
}

func TestAppClipsInvocationsCreateSupportsExistingLocalizationIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/betaAppClipInvocations" {
			t.Fatalf("expected path /v1/betaAppClipInvocations, got %s", r.URL.Path)
		}

		var payload asc.BetaAppClipInvocationCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Data.Relationships == nil || payload.Data.Relationships.BetaAppClipInvocationLocalizations == nil {
			t.Fatal("expected localization relationships")
		}
		if len(payload.Data.Relationships.BetaAppClipInvocationLocalizations.Data) != 1 {
			t.Fatalf("expected one localization relationship, got %d", len(payload.Data.Relationships.BetaAppClipInvocationLocalizations.Data))
		}
		if payload.Data.Relationships.BetaAppClipInvocationLocalizations.Data[0].ID != "loc-1" {
			t.Fatalf("expected localization id loc-1, got %s", payload.Data.Relationships.BetaAppClipInvocationLocalizations.Data[0].ID)
		}
		if len(payload.Included) != 0 {
			t.Fatalf("expected no included localizations, got %d", len(payload.Included))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"type":"betaAppClipInvocations","id":"inv-1","attributes":{"url":"https://example.com/clip"}},"links":{}}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}

	keyPath := t.TempDir() + "/key.p8"
	writeECDSAPEM(t, keyPath)

	transport := appClipsRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = serverURL.Scheme
		cloned.URL.Host = serverURL.Host
		return server.Client().Transport.RoundTrip(cloned)
	})
	client, err := asc.NewClientWithHTTPClient("TEST_KEY", "TEST_ISSUER", keyPath, &http.Client{Transport: transport})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	restore := appclipscli.SetClientFactory(func() (*asc.Client, error) {
		return client, nil
	})
	t.Cleanup(restore)

	root := RootCommand("1.2.3")
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"app-clips", "invocations", "create",
			"--build-bundle-id", "bundle-1",
			"--url", "https://example.com/clip",
			"--localization-id", "loc-1",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response asc.BetaAppClipInvocationResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("unmarshal stdout: %v\nstdout=%q", err, stdout)
	}
	if response.Data.ID != "inv-1" {
		t.Fatalf("expected invocation id inv-1, got %s", response.Data.ID)
	}
}

type appClipsRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn appClipsRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
