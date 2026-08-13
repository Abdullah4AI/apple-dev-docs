package testflight

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBetaGroupsListBuildIDHelpMatchesOpenAPISnapshot pins the --build-id help
// text to what docs/openapi/latest.json actually documents: there is no GET on
// the build-side betaGroups relationship, but GET /v1/builds/{id} does accept
// include=betaGroups, so the help must not claim that no build-side read exists.
func TestBetaGroupsListBuildIDHelpMatchesOpenAPISnapshot(t *testing.T) {
	pathsIndex := readOpenAPIPathsIndex(t)
	if strings.Contains(pathsIndex, "GET /v1/builds/{id}/relationships/betaGroups") {
		t.Fatal("OpenAPI snapshot now documents a build-side betaGroups relationship GET; revisit the --build-id lookup design and this help text")
	}

	help := betaGroupsListLongHelp(t)

	if strings.Contains(help, "does not expose a build-side GET for beta groups") {
		t.Fatalf("help still claims App Store Connect exposes no build-side GET for beta groups, but the snapshot documents include=betaGroups on GET /v1/builds/{id}: %q", help)
	}
	for _, want := range []string{
		"GET /v1/builds/{id}/relationships/betaGroups",
		"include=betaGroups",
		"limit[betaGroups]",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("expected --build-id help to mention %q, got %q", want, help)
		}
	}
}

func betaGroupsListLongHelp(t *testing.T) string {
	t.Helper()

	for _, subcommand := range BetaGroupsCommand().Subcommands {
		if subcommand.Name == "list" {
			return subcommand.LongHelp
		}
	}
	t.Fatal("expected beta-groups list subcommand")
	return ""
}

func readOpenAPIPathsIndex(t *testing.T) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "openapi", "paths.txt"))
	if err != nil {
		t.Fatalf("read OpenAPI path index: %v", err)
	}
	return string(data)
}
