package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestPreOrdersEndWithConfirmPostsExpectedPayload(t *testing.T) {
	setupAuth(t)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCount++
		if req.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", req.Method)
			http.Error(w, "unexpected method", http.StatusBadRequest)
			return
		}
		if req.URL.Path != "/v1/endAppAvailabilityPreOrders" {
			t.Errorf("expected /v1/endAppAvailabilityPreOrders, got %s", req.URL.Path)
			http.Error(w, "unexpected path", http.StatusBadRequest)
			return
		}

		var payload struct {
			Data struct {
				Type          string `json:"type"`
				Relationships struct {
					TerritoryAvailabilities struct {
						Data []struct {
							Type string `json:"type"`
							ID   string `json:"id"`
						} `json:"data"`
					} `json:"territoryAvailabilities"`
				} `json:"relationships"`
			} `json:"data"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if payload.Data.Type != "endAppAvailabilityPreOrders" {
			t.Errorf("unexpected data type %q", payload.Data.Type)
			http.Error(w, "unexpected data type", http.StatusBadRequest)
			return
		}
		linkages := payload.Data.Relationships.TerritoryAvailabilities.Data
		if len(linkages) != 2 || linkages[0].Type != "territoryAvailabilities" || linkages[0].ID != "ta-1" || linkages[1].Type != "territoryAvailabilities" || linkages[1].ID != "ta-2" {
			t.Errorf("unexpected territory availability linkages: %+v", linkages)
			http.Error(w, "unexpected linkages", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":{"type":"endAppAvailabilityPreOrders","id":"end-1"},"links":{"self":"https://api.appstoreconnect.apple.com/v1/endAppAvailabilityPreOrders/end-1"}}`)
	}))
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	installDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = serverURL.Scheme
		cloned.URL.Host = serverURL.Host
		return server.Client().Transport.RoundTrip(cloned)
	}))

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	if err := root.Parse([]string{
		"pre-orders", "end",
		"--territory-availability", "ta-1,ta-2",
		"--confirm",
		"--output", "json",
	}); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	stdout, stderr := captureOutput(t, func() {
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if requestCount != 1 {
		t.Fatalf("expected one request, got %d", requestCount)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var output struct {
		Data struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("decode stdout: %v; stdout=%q", err, stdout)
	}
	if output.Data.Type != "endAppAvailabilityPreOrders" || output.Data.ID != "end-1" {
		t.Fatalf("unexpected output: %+v", output.Data)
	}
}
