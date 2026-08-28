package asc

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetUsers_WithQueryParityOptions(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		if req.URL.Path != "/v1/users" {
			t.Fatalf("path = %q, want /v1/users", req.URL.Path)
		}
		values := req.URL.Query()
		want := map[string]string{
			"filter[visibleApps]": "app-1,app-2",
			"sort":                "-lastName",
			"fields[users]":       "username,lastName,visibleApps",
			"fields[apps]":        "name,bundleId",
			"include":             "visibleApps",
			"limit[visibleApps]":  "25",
		}
		for key, expected := range want {
			if got := values.Get(key); got != expected {
				t.Errorf("%s = %q, want %q", key, got, expected)
			}
		}
	}, response)

	if _, err := client.GetUsers(
		context.Background(),
		WithUsersVisibleAppIDs([]string{"app-1", " app-2 "}),
		WithUsersSort("-lastName"),
		WithUsersFields([]string{"username", "lastName"}),
		WithUsersAppFields([]string{"name", "bundleId"}),
		WithUsersInclude([]string{"visibleApps"}),
		WithUsersVisibleAppsLimit(25),
	); err != nil {
		t.Fatalf("GetUsers() error: %v", err)
	}
}

func TestGetUsers_IncludeVisibleAppsDeduplicatesPrimaryField(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		want := "fields%5Busers%5D=username%2CvisibleApps&include=visibleApps"
		if got := req.URL.RawQuery; got != want {
			t.Fatalf("raw query = %q, want %q", got, want)
		}
	}, response)

	if _, err := client.GetUsers(
		context.Background(),
		WithUsersFields([]string{"username", "visibleApps", "username"}),
		WithUsersInclude([]string{"visibleApps", "visibleApps"}),
	); err != nil {
		t.Fatalf("GetUsers() error: %v", err)
	}
}

func TestGetUsers_FieldOnlyPreservesSparseFieldset(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[]}`)
	client := newTestClient(t, func(req *http.Request) {
		want := "fields%5Busers%5D=username%2ClastName"
		if got := req.URL.RawQuery; got != want {
			t.Fatalf("raw query = %q, want %q", got, want)
		}
	}, response)

	if _, err := client.GetUsers(
		context.Background(),
		WithUsersFields([]string{"username", "lastName"}),
	); err != nil {
		t.Fatalf("GetUsers() error: %v", err)
	}
}

func TestGetUsers_SparseAttributesPreserveIncludedResources(t *testing.T) {
	response := jsonResponse(http.StatusOK, `{"data":[{"type":"users","id":"user-1","attributes":{"username":"user@example.com","lastName":"Doe"},"relationships":{"visibleApps":{"data":[{"type":"apps","id":"app-1"}]}}}],"included":[{"type":"apps","id":"app-1","attributes":{"name":"Example"}}],"links":{"next":""}}`)
	client := newTestClient(t, nil, response)

	users, err := client.GetUsers(
		context.Background(),
		WithUsersFields([]string{"username", "lastName"}),
		WithUsersInclude([]string{"visibleApps"}),
	)
	if err != nil {
		t.Fatalf("GetUsers() error: %v", err)
	}

	encoded, err := json.Marshal(users)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	var got struct {
		Data []struct {
			Attributes    map[string]json.RawMessage `json:"attributes"`
			Relationships json.RawMessage            `json:"relationships"`
		} `json:"data"`
		Included json.RawMessage `json:"included"`
	}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if len(got.Data) != 1 {
		t.Fatalf("data length = %d, want 1", len(got.Data))
	}
	if len(got.Data[0].Attributes) != 2 {
		t.Fatalf("sparse attributes = %s, want only username and lastName", got.Data[0].Attributes)
	}
	if string(got.Data[0].Attributes["username"]) != `"user@example.com"` {
		t.Fatalf("username = %s, want user@example.com", got.Data[0].Attributes["username"])
	}
	if string(got.Data[0].Attributes["lastName"]) != `"Doe"` {
		t.Fatalf("lastName = %s, want Doe", got.Data[0].Attributes["lastName"])
	}
	if string(got.Data[0].Relationships) != `{"visibleApps":{"data":[{"type":"apps","id":"app-1"}]}}` {
		t.Fatalf("relationships = %s, want visibleApps linkage", got.Data[0].Relationships)
	}
	if string(got.Included) != `[{"type":"apps","id":"app-1","attributes":{"name":"Example"}}]` {
		t.Fatalf("included = %s, want included app", got.Included)
	}
}
