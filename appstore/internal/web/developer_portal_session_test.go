package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const (
	developerPortalCSRFSecret   = "csrf-secret-DO-NOT-LEAK-abc123"
	developerPortalCookieSecret = "session-cookie-DO-NOT-LEAK-xyz789"
)

func TestEnsureDeveloperPortalSessionMultiTeamRequiresSelectorBeforeMutation(t *testing.T) {
	var mutationHits int
	client := developerPortalTestClient(t, func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case developerPortalTeamsPath:
			return developerPortalSecretTeamsResponse(developerPortalMultiTeamFixture()), nil
		default:
			mutationHits++
			t.Errorf("mutation request %s %s must not run without a selected team", r.Method, r.URL.Path)
			return developerPortalTestResponse(http.StatusInternalServerError, `{}`, nil), nil
		}
	})
	client.publicProviderID = "ASC-PROVIDER-UNRELATED"
	client.providerName = "Unrelated Provider"

	_, err := client.EnableDeveloperBundleIDCapability(context.Background(), DeveloperBundleIDCapabilityEnableRequest{
		BundleID:   "bundle-1",
		Capability: "PRIVATE_CLOUD_COMPUTE",
	})
	if err == nil {
		t.Fatal("expected ambiguous Developer Portal team error")
	}
	if !strings.Contains(err.Error(), "--developer-team") {
		t.Fatalf("error %q does not mention --developer-team", err)
	}
	if mutationHits != 0 {
		t.Fatalf("mutation requests = %d, want 0", mutationHits)
	}
	assertDeveloperPortalErrorHasNoSecrets(t, err)
}

func TestEnsureDeveloperPortalSessionSelectsTeamByID(t *testing.T) {
	var selectedTeamID string
	client := developerPortalTestClient(t, developerPortalCapabilityEnableRoundTrip(t, "TeAmTwO456", &selectedTeamID))
	client.SetDeveloperTeamSelector("TeAmTwO456")

	result, err := client.EnableDeveloperBundleIDCapability(context.Background(), DeveloperBundleIDCapabilityEnableRequest{
		BundleID:   "bundle-1",
		Capability: "PRIVATE_CLOUD_COMPUTE",
	})
	if err != nil {
		t.Fatalf("EnableDeveloperBundleIDCapability() error: %v", err)
	}
	if result == nil || !result.Changed {
		t.Fatalf("result = %+v", result)
	}
	if selectedTeamID != "TEAMTWO456" {
		t.Fatalf("selected teamId = %q, want TEAMTWO456", selectedTeamID)
	}
}

func TestEnsureDeveloperPortalSessionSelectsTeamByExactName(t *testing.T) {
	var selectedTeamID string
	client := developerPortalTestClient(t, developerPortalCapabilityEnableRoundTrip(t, "Example Two", &selectedTeamID))
	client.SetDeveloperTeamSelector("example two")

	_, err := client.EnableDeveloperBundleIDCapability(context.Background(), DeveloperBundleIDCapabilityEnableRequest{
		BundleID:   "bundle-1",
		Capability: "PRIVATE_CLOUD_COMPUTE",
	})
	if err != nil {
		t.Fatalf("EnableDeveloperBundleIDCapability() error: %v", err)
	}
	if selectedTeamID != "TEAMTWO456" {
		t.Fatalf("selected teamId = %q, want TEAMTWO456", selectedTeamID)
	}
}

func TestEnsureDeveloperPortalSessionUnknownSelectorListsTeams(t *testing.T) {
	var mutationHits int
	client := developerPortalTestClient(t, func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case developerPortalTeamsPath:
			return developerPortalSecretTeamsResponse(developerPortalMultiTeamFixture()), nil
		default:
			mutationHits++
			t.Errorf("mutation request %s %s must not run for an unknown selector", r.Method, r.URL.Path)
			return developerPortalTestResponse(http.StatusInternalServerError, `{}`, nil), nil
		}
	})
	client.SetDeveloperTeamSelector("NOPE")

	_, err := client.EnableDeveloperBundleIDCapability(context.Background(), DeveloperBundleIDCapabilityEnableRequest{
		BundleID:   "bundle-1",
		Capability: "PRIVATE_CLOUD_COMPUTE",
	})
	if err == nil {
		t.Fatal("expected unknown Developer Portal team error")
	}
	message := err.Error()
	for _, want := range []string{"NOPE", "TEAMONE123", "Example One", "TEAMTWO456", "Example Two", "--developer-team"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not contain %q", message, want)
		}
	}
	if mutationHits != 0 {
		t.Fatalf("mutation requests = %d, want 0", mutationHits)
	}
	assertDeveloperPortalErrorHasNoSecrets(t, err)
}

func TestEnsureDeveloperPortalSessionAmbiguousExactNameFailsBeforeMutation(t *testing.T) {
	var mutationHits int
	client := developerPortalTestClient(t, func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case developerPortalTeamsPath:
			return developerPortalSecretTeamsResponse(`{"teams":[{"teamId":"TEAMONE123","name":"Shared Name"},{"teamId":"TEAMTWO456","name":"Shared Name"}]}`), nil
		default:
			mutationHits++
			t.Errorf("mutation request %s %s must not run for an ambiguous team name", r.Method, r.URL.Path)
			return developerPortalTestResponse(http.StatusInternalServerError, `{}`, nil), nil
		}
	})
	client.SetDeveloperTeamSelector("shared name")

	_, err := client.EnableDeveloperBundleIDCapability(context.Background(), DeveloperBundleIDCapabilityEnableRequest{
		BundleID:   "bundle-1",
		Capability: "PRIVATE_CLOUD_COMPUTE",
	})
	if err == nil {
		t.Fatal("expected ambiguous Developer Portal team name error")
	}
	message := err.Error()
	for _, want := range []string{"shared name", "TEAMONE123", "TEAMTWO456", "--developer-team"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not contain %q", message, want)
		}
	}
	if mutationHits != 0 {
		t.Fatalf("mutation requests = %d, want 0", mutationHits)
	}
	assertDeveloperPortalErrorHasNoSecrets(t, err)
}

func TestEnsureDeveloperPortalSessionSelectsSingleTeam(t *testing.T) {
	var selectedTeamID string
	client := developerPortalTestClient(t, func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Path == developerPortalTeamsPath:
			return developerPortalTestResponse(http.StatusOK, developerPortalTeamsFixture(), nil), nil
		case strings.HasPrefix(r.URL.Path, developerServicesPath+"/"):
			var payload developerPortalProxyReadRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err == nil && payload.TeamID != "" {
				selectedTeamID = payload.TeamID
			}
			if strings.HasSuffix(r.URL.Path, "/capabilities") {
				return developerPortalTestResponse(http.StatusOK, developerCapabilityMetadata(true), http.Header{"csrf": []string{"token"}, "csrf_ts": []string{"time"}}), nil
			}
			if r.Method == http.MethodPatch {
				return developerPortalTestResponse(http.StatusOK, `{"data":{"id":"bundle-1","type":"bundleIds"}}`, nil), nil
			}
			return developerPortalTestResponse(http.StatusOK, developerBundleResponse(false), nil), nil
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			return developerPortalTestResponse(http.StatusNotFound, `{}`, nil), nil
		}
	})
	client.publicProviderID = "UNRELATED"
	client.providerName = "Different Provider"

	_, err := client.EnableDeveloperBundleIDCapability(context.Background(), DeveloperBundleIDCapabilityEnableRequest{
		BundleID:   "bundle-1",
		Capability: "PRIVATE_CLOUD_COMPUTE",
	})
	if err != nil {
		t.Fatalf("EnableDeveloperBundleIDCapability() error: %v", err)
	}
	if selectedTeamID != "TEAM123456" {
		t.Fatalf("selected teamId = %q, want TEAM123456", selectedTeamID)
	}
}

func TestEnsureDeveloperPortalSessionReusesPersistedTeam(t *testing.T) {
	var selectedTeamID string
	client := developerPortalTestClient(t, developerPortalCapabilityEnableRoundTrip(t, "TEAMTWO456", &selectedTeamID))
	client.developerTeamID = "TEAMTWO456"
	client.publicProviderID = "TEAMONE123"
	client.providerName = "Example One"

	_, err := client.EnableDeveloperBundleIDCapability(context.Background(), DeveloperBundleIDCapabilityEnableRequest{
		BundleID:   "bundle-1",
		Capability: "PRIVATE_CLOUD_COMPUTE",
	})
	if err != nil {
		t.Fatalf("EnableDeveloperBundleIDCapability() error: %v", err)
	}
	if selectedTeamID != "TEAMTWO456" {
		t.Fatalf("persisted team should win over provider match, got %q", selectedTeamID)
	}
}

func TestSelectDeveloperPortalTeamPrefixRequiresUniqueMatch(t *testing.T) {
	t.Run("unique prefix match", func(t *testing.T) {
		teams := []developerPortalTeam{
			{TeamID: "ACME123", Name: "Acme"},
			{TeamID: "OTHER456", Name: "Other"},
		}
		team, err := selectDeveloperPortalTeam(teams, "", "Acme Inc")
		if err != nil {
			t.Fatalf("selectDeveloperPortalTeam() error: %v", err)
		}
		if team.TeamID != "ACME123" {
			t.Fatalf("team = %+v", team)
		}
	})

	t.Run("ambiguous prefix match fails closed", func(t *testing.T) {
		teams := []developerPortalTeam{
			{TeamID: "TEAMONE123", Name: "Example"},
			{TeamID: "TEAMTWO456", Name: "Example Company"},
		}
		_, err := selectDeveloperPortalTeam(teams, "", "Example Company (App Store Connect)")
		if err == nil {
			t.Fatal("expected ambiguous prefix match to fail closed")
		}
		if !strings.Contains(err.Error(), "--developer-team") {
			t.Fatalf("error %q does not mention --developer-team", err)
		}
	})
}

func developerPortalMultiTeamFixture() string {
	return `{"teams":[{"teamId":"TEAMONE123","name":"Example One","status":"active"},{"teamId":"TEAMTWO456","name":"Example Two","status":"active"}]}`
}

func developerPortalSecretTeamsResponse(body string) *http.Response {
	headers := make(http.Header)
	headers.Set("csrf", developerPortalCSRFSecret)
	headers.Set("csrf_ts", "csrf-ts-"+developerPortalCSRFSecret)
	headers.Add("Set-Cookie", "myacinfo="+developerPortalCookieSecret+"; Path=/; Secure")
	return developerPortalTestResponse(http.StatusOK, body, headers)
}

func developerPortalCapabilityEnableRoundTrip(t *testing.T, wantSelector string, selectedTeamID *string) roundTripFunc {
	t.Helper()
	return func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Path == developerPortalTeamsPath:
			return developerPortalSecretTeamsResponse(developerPortalMultiTeamFixture()), nil
		case strings.HasPrefix(r.URL.Path, developerServicesPath+"/"):
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
				return developerPortalTestResponse(http.StatusInternalServerError, `{}`, nil), nil
			}
			if r.Header.Get("X-HTTP-Method-Override") == http.MethodGet {
				var payload developerPortalProxyReadRequest
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Errorf("decode payload: %v body=%s", err, body)
				}
				if payload.TeamID != "" {
					*selectedTeamID = payload.TeamID
				}
				if wantSelector != "" && payload.TeamID != "TEAMTWO456" {
					t.Errorf("teamId = %q, want TEAMTWO456 for selector %q", payload.TeamID, wantSelector)
				}
			}
			switch {
			case strings.HasSuffix(r.URL.Path, "/capabilities"):
				return developerPortalTestResponse(http.StatusOK, developerCapabilityMetadata(true), http.Header{"csrf": []string{"token"}, "csrf_ts": []string{"time"}}), nil
			case r.Method == http.MethodPatch:
				return developerPortalTestResponse(http.StatusOK, `{"data":{"id":"bundle-1","type":"bundleIds"}}`, nil), nil
			default:
				return developerPortalTestResponse(http.StatusOK, developerBundleResponse(false), nil), nil
			}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			return developerPortalTestResponse(http.StatusNotFound, `{}`, nil), nil
		}
	}
}

func assertDeveloperPortalErrorHasNoSecrets(t *testing.T, err error) {
	t.Helper()
	message := err.Error()
	for _, secret := range []string{developerPortalCSRFSecret, developerPortalCookieSecret} {
		if strings.Contains(message, secret) {
			t.Fatalf("error %q leaked secret %q", message, secret)
		}
	}
}
