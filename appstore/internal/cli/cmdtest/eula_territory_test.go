package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestEULACreateNormalizesTerritories(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/v1/endUserLicenseAgreements" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}

		var payload struct {
			Data struct {
				Relationships struct {
					Territories struct {
						Data []struct {
							ID string `json:"id"`
						} `json:"data"`
					} `json:"territories"`
				} `json:"relationships"`
			} `json:"data"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request payload: %v", err)
		}
		if got := payload.Data.Relationships.Territories.Data[0].ID; got != "USA" {
			t.Fatalf("expected first territory USA, got %q", got)
		}
		if got := payload.Data.Relationships.Territories.Data[1].ID; got != "FRA" {
			t.Fatalf("expected second territory FRA, got %q", got)
		}

		return jsonHTTPResponse(http.StatusCreated, `{"data":{"type":"endUserLicenseAgreements","id":"eula-1","attributes":{"agreementText":"Terms"}}}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	if err := root.Parse([]string{
		"eula", "create",
		"--app", "app-1",
		"--agreement-text", "Terms",
		"--territory", "US,France",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := root.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
}

func TestEULAUpdateNormalizesTerritories(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch || req.URL.Path != "/v1/endUserLicenseAgreements/eula-1" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}

		var payload struct {
			Data struct {
				Relationships struct {
					Territories struct {
						Data []struct {
							ID string `json:"id"`
						} `json:"data"`
					} `json:"territories"`
				} `json:"relationships"`
			} `json:"data"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request payload: %v", err)
		}
		if got := payload.Data.Relationships.Territories.Data[0].ID; got != "USA" {
			t.Fatalf("expected first territory USA, got %q", got)
		}
		if got := payload.Data.Relationships.Territories.Data[1].ID; got != "FRA" {
			t.Fatalf("expected second territory FRA, got %q", got)
		}

		return jsonHTTPResponse(http.StatusOK, `{"data":{"type":"endUserLicenseAgreements","id":"eula-1","attributes":{"agreementText":"Updated"}}}`), nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	if err := root.Parse([]string{
		"eula", "update",
		"--id", "eula-1",
		"--territory", "US,France",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := root.Run(context.Background()); err != nil {
		t.Fatalf("run error: %v", err)
	}
}
