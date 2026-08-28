package asc

import (
	"net/url"
	"testing"
)

func TestBuildBetaTestersQueryEmitsSortIncludeAndInviteType(t *testing.T) {
	query := &betaTestersQuery{}
	opts := []BetaTestersOption{
		WithBetaTestersSort("-lastName"),
		WithBetaTestersInclude([]string{"betaGroups", " apps "}),
		WithBetaTestersInviteTypes([]string{"EMAIL", " PUBLIC_LINK "}),
	}
	for _, opt := range opts {
		opt(query)
	}

	qs, err := buildBetaTestersQuery("APP_ID", query)
	if err != nil {
		t.Fatalf("buildBetaTestersQuery() error: %v", err)
	}
	values, parseErr := url.ParseQuery(qs)
	if parseErr != nil {
		t.Fatalf("parse query: %v", parseErr)
	}

	if got := values.Get("sort"); got != "-lastName" {
		t.Fatalf("expected sort=-lastName, got %q", got)
	}
	if got := values.Get("include"); got != "betaGroups,apps" {
		t.Fatalf("expected include=betaGroups,apps, got %q", got)
	}
	if got := values.Get("filter[inviteType]"); got != "EMAIL,PUBLIC_LINK" {
		t.Fatalf("expected filter[inviteType]=EMAIL,PUBLIC_LINK, got %q", got)
	}
	if got := values.Get("filter[apps]"); got != "APP_ID" {
		t.Fatalf("expected filter[apps]=APP_ID, got %q", got)
	}
}

func TestBuildBetaTestersQueryOmitsUnsetSortIncludeAndInviteType(t *testing.T) {
	query := &betaTestersQuery{}
	opts := []BetaTestersOption{
		WithBetaTestersSort("   "),
		WithBetaTestersInclude(nil),
		WithBetaTestersInviteTypes([]string{" "}),
	}
	for _, opt := range opts {
		opt(query)
	}

	qs, err := buildBetaTestersQuery("APP_ID", query)
	if err != nil {
		t.Fatalf("buildBetaTestersQuery() error: %v", err)
	}
	values, parseErr := url.ParseQuery(qs)
	if parseErr != nil {
		t.Fatalf("parse query: %v", parseErr)
	}

	for _, key := range []string{"sort", "include", "filter[inviteType]"} {
		if _, present := values[key]; present {
			t.Fatalf("expected %s to be omitted, got %q", key, values.Get(key))
		}
	}
}

func TestBuildBetaTestersQueryKeepsInviteTypeWithGroupFilter(t *testing.T) {
	query := &betaTestersQuery{}
	opts := []BetaTestersOption{
		WithBetaTestersGroupIDs([]string{"group-1"}),
		WithBetaTestersInviteTypes([]string{"PUBLIC_LINK"}),
		WithBetaTestersInclude([]string{"betaGroups"}),
	}
	for _, opt := range opts {
		opt(query)
	}

	qs, err := buildBetaTestersQuery("APP_ID", query)
	if err != nil {
		t.Fatalf("buildBetaTestersQuery() error: %v", err)
	}
	values, parseErr := url.ParseQuery(qs)
	if parseErr != nil {
		t.Fatalf("parse query: %v", parseErr)
	}

	if got := values.Get("filter[betaGroups]"); got != "group-1" {
		t.Fatalf("expected filter[betaGroups]=group-1, got %q", got)
	}
	if got := values.Get("filter[inviteType]"); got != "PUBLIC_LINK" {
		t.Fatalf("expected filter[inviteType]=PUBLIC_LINK, got %q", got)
	}
	if got := values.Get("include"); got != "betaGroups" {
		t.Fatalf("expected include=betaGroups, got %q", got)
	}
}
