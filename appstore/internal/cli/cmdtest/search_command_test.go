package cmdtest

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strings"
	"testing"

	rootcmd "github.com/Abdullah4AI/apple-developer-toolkit/appstore/cmd"
)

type searchResponse struct {
	Query   string         `json:"query"`
	Count   int            `json:"count"`
	Results []searchResult `json:"results"`
}

type searchResult struct {
	Command  string   `json:"command"`
	Summary  string   `json:"summary"`
	Usage    string   `json:"usage"`
	Score    int      `json:"score"`
	Matched  []string `json:"matched"`
	Examples []string `json:"examples"`
}

func TestSearchFindsCommandsFromTaskWordsAsJSON(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"search", "--output", "json", "external", "testers"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}

	if response.Query != "external testers" {
		t.Fatalf("expected normalized query, got %q", response.Query)
	}
	if response.Count == 0 || len(response.Results) == 0 {
		t.Fatalf("expected search results, got %#v", response)
	}
	if !searchResultsContain(response.Results, "asc testflight testers") {
		t.Fatalf("expected TestFlight tester command in results, got %#v", response.Results)
	}
	for _, result := range response.Results {
		if strings.TrimSpace(result.Command) == "" {
			t.Fatalf("expected command path in result: %#v", result)
		}
		if strings.TrimSpace(result.Summary) == "" {
			t.Fatalf("expected summary in result: %#v", result)
		}
		if result.Score <= 0 {
			t.Fatalf("expected positive score in result: %#v", result)
		}
		if len(result.Matched) == 0 {
			t.Fatalf("expected match reasons in result: %#v", result)
		}
	}
}

func TestSearchUsesAliasesForAgentVocabulary(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"search", "--output", "json", "ship", "app", "review"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}

	if !searchResultsContain(response.Results, "asc publish appstore") && !searchResultsContain(response.Results, "asc review submit") {
		t.Fatalf("expected publish or review submission command for ship app review, got %#v", response.Results)
	}
	if !searchResultsMention(response.Results, "alias:ship") {
		t.Fatalf("expected alias match reason for ship query, got %#v", response.Results)
	}
}

func TestSearchPrioritizesCanonicalPublishWorkflowForNaturalLanguage(t *testing.T) {
	tests := []struct {
		name     string
		query    []string
		expected string
	}{
		{name: "ship app", query: []string{"ship", "app"}, expected: "asc publish appstore"},
		{name: "release app", query: []string{"release", "app"}, expected: "asc publish appstore"},
		{name: "upload appstore status", query: []string{"upload", "appstore", "status"}, expected: "asc publish appstore"},
		{name: "ship beta", query: []string{"ship", "beta"}, expected: "asc publish testflight"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var code int
			stdout, stderr := captureOutput(t, func() {
				args := []string{"search", "--output", "json", "--limit", "5"}
				args = append(args, test.query...)
				code = rootcmd.Run(args, "1.2.3")
			})

			if code != 0 {
				t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var response searchResponse
			if err := json.Unmarshal([]byte(stdout), &response); err != nil {
				t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
			}
			if len(response.Results) == 0 {
				t.Fatalf("expected search results, got %#v", response)
			}
			if response.Results[0].Command != test.expected {
				t.Fatalf("expected canonical publish workflow %q first, got %#v", test.expected, response.Results)
			}
		})
	}
}

func TestSearchPrioritizesReleaseDashboardForCrossSurfaceStatusQuery(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{
			"search",
			"--output",
			"json",
			"verify",
			"TestFlight",
			"build",
			"upload",
			"and",
			"App",
			"Review",
			"status",
		}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if len(response.Results) == 0 {
		t.Fatalf("expected search results, got %#v", response)
	}
	if response.Results[0].Command != "asc status" {
		t.Fatalf("expected release dashboard first, got %#v", response.Results)
	}
}

func TestSearchKeepsAppReviewStatusAheadOfAggregateDashboard(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "app", "review", "status"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if len(response.Results) == 0 {
		t.Fatalf("expected search results, got %#v", response)
	}
	if response.Results[0].Command != "asc review status" {
		t.Fatalf("expected App Review status first, got %#v", response.Results)
	}
}

func TestSearchDoesNotPrioritizeWorkflowDuplicateForOtherXcodeCloudResources(t *testing.T) {
	for _, test := range []struct {
		name     string
		query    []string
		expected string
	}{
		{
			name:     "duplicate artifact",
			query:    []string{"duplicate", "Xcode", "Cloud", "artifact"},
			expected: "asc xcode-cloud artifacts",
		},
		{
			name:  "duplicate build run status",
			query: []string{"duplicate", "Xcode", "Cloud", "build", "run", "status"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var code int
			stdout, stderr := captureOutput(t, func() {
				args := []string{"search", "--output", "json", "--limit", "5"}
				args = append(args, test.query...)
				code = rootcmd.Run(args, "1.2.3")
			})

			if code != 0 {
				t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var response searchResponse
			if err := json.Unmarshal([]byte(stdout), &response); err != nil {
				t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
			}
			if len(response.Results) == 0 {
				t.Fatalf("expected search results, got %#v", response)
			}
			if response.Results[0].Command == "asc xcode-cloud workflows duplicate" {
				t.Fatalf("expected a non-workflow Xcode Cloud result first, got %#v", response.Results)
			}
			if test.expected != "" && response.Results[0].Command != test.expected {
				t.Fatalf("expected %q first, got %#v", test.expected, response.Results)
			}
		})
	}
}

func TestSearchPrioritizesScopedReleaseStatusQueries(t *testing.T) {
	tests := []struct {
		name     string
		query    []string
		expected string
	}{
		{
			name:     "phased release status",
			query:    []string{"phased", "release", "status"},
			expected: "asc versions phased-release view",
		},
		{
			name:     "phased release overview",
			query:    []string{"phased", "release", "overview"},
			expected: "asc versions phased-release view",
		},
		{
			name:     "beta review status",
			query:    []string{"beta", "review", "status"},
			expected: "asc builds beta-app-review-submission view",
		},
		{
			name:     "update phased release status",
			query:    []string{"update", "phased", "release", "status"},
			expected: "asc versions phased-release update",
		},
		{
			name:     "TestFlight App Review status",
			query:    []string{"TestFlight", "App", "Review", "status"},
			expected: "asc testflight review view",
		},
		{
			name:     "App Review attachment upload status",
			query:    []string{"App", "Review", "attachment", "upload", "status"},
			expected: "asc review attachments-upload",
		},
		{
			name:     "list App Review attachment uploads",
			query:    []string{"list", "App", "Review", "attachment", "uploads"},
			expected: "asc review attachments-list",
		},
		{
			name:     "get App Review attachment upload",
			query:    []string{"get", "App", "Review", "attachment", "upload"},
			expected: "asc review attachments-get",
		},
		{
			name:     "delete App Review attachment upload",
			query:    []string{"delete", "App", "Review", "attachment", "upload"},
			expected: "asc review attachments-delete",
		},
		{
			name:     "TestFlight beta review submission status",
			query:    []string{"TestFlight", "beta", "review", "submission", "status"},
			expected: "asc testflight review submissions view",
		},
		{
			name:     "list beta review submissions status",
			query:    []string{"list", "beta", "review", "submissions", "status"},
			expected: "asc testflight review submissions list",
		},
		{
			name:     "list TestFlight beta review submissions status",
			query:    []string{"list", "TestFlight", "beta", "review", "submissions", "status"},
			expected: "asc testflight review submissions list",
		},
		{
			name:     "list TestFlight App Review submissions for build status",
			query:    []string{"list", "TestFlight", "App", "Review", "submissions", "for", "build", "status"},
			expected: "asc testflight review submissions list",
		},
		{
			name:     "get TestFlight review submission build status",
			query:    []string{"get", "TestFlight", "review", "submission", "build", "status"},
			expected: "asc testflight review submissions build",
		},
		{
			name:     "update release pipeline",
			query:    []string{"update", "release", "pipeline"},
			expected: "asc versions phased-release update",
		},
		{
			name:     "App Store Connect and TestFlight system status",
			query:    []string{"App", "Store", "Connect", "and", "TestFlight", "system", "status"},
			expected: "asc system-status",
		},
		{
			name:     "TestFlight and App Store account status",
			query:    []string{"TestFlight", "and", "App", "Store", "account", "status"},
			expected: "asc account status",
		},
		{
			name:     "TestFlight App Store telemetry status",
			query:    []string{"TestFlight", "App", "Store", "telemetry", "status"},
			expected: "asc telemetry status",
		},
		{
			name:     "disable telemetry status",
			query:    []string{"disable", "telemetry", "status"},
			expected: "asc telemetry disable",
		},
		{
			name:     "enable telemetry status",
			query:    []string{"enable", "telemetry", "status"},
			expected: "asc telemetry enable",
		},
		{
			name:     "telemetry reset-id status",
			query:    []string{"telemetry", "reset-id", "status"},
			expected: "asc telemetry reset-id",
		},
		{
			name:     "TestFlight and App Store auth status",
			query:    []string{"TestFlight", "and", "App", "Store", "auth", "status"},
			expected: "asc auth status",
		},
		{
			name:     "StoreKit auth status",
			query:    []string{"StoreKit", "auth", "status"},
			expected: "asc storekit auth status",
		},
		{
			name:     "login StoreKit auth status",
			query:    []string{"login", "StoreKit", "auth", "status"},
			expected: "asc storekit auth login",
		},
		{
			name:     "Apple Ads auth status",
			query:    []string{"Apple", "Ads", "auth", "status"},
			expected: "asc ads auth status",
		},
		{
			name:     "discover Apple Ads auth status",
			query:    []string{"discover", "Apple", "Ads", "auth", "status"},
			expected: "asc ads auth discover",
		},
		{
			name:     "Apple Ads account auth status",
			query:    []string{"Apple", "Ads", "account", "auth", "status"},
			expected: "asc ads auth status",
		},
		{
			name:     "web auth status",
			query:    []string{"web", "auth", "status"},
			expected: "asc web auth status",
		},
		{
			name:     "web auth capabilities status",
			query:    []string{"web", "auth", "capabilities", "status"},
			expected: "asc web auth capabilities",
		},
		{
			name:     "auth doctor status",
			query:    []string{"auth", "doctor", "status"},
			expected: "asc auth doctor",
		},
		{
			name:     "TestFlight and App Store agreement status",
			query:    []string{"TestFlight", "and", "App", "Store", "agreement", "status"},
			expected: "asc web agreements status",
		},
		{
			name:     "Xcode Cloud build and App Review status",
			query:    []string{"Xcode", "Cloud", "build", "and", "App", "Review", "status"},
			expected: "asc xcode-cloud status",
		},
		{
			name:     "trigger Xcode Cloud workflow status",
			query:    []string{"trigger", "Xcode", "Cloud", "workflow", "status"},
			expected: "asc xcode-cloud run",
		},
		{
			name:     "doctor Xcode Cloud workflow status",
			query:    []string{"doctor", "Xcode", "Cloud", "workflow", "status"},
			expected: "asc xcode-cloud doctor",
		},
		{
			name:     "list Xcode Cloud workflows status",
			query:    []string{"list", "Xcode", "Cloud", "workflows", "status"},
			expected: "asc xcode-cloud workflows list",
		},
		{
			name:     "view Xcode Cloud artifact status",
			query:    []string{"view", "Xcode", "Cloud", "artifact", "status"},
			expected: "asc xcode-cloud artifacts view",
		},
		{
			name:     "view TestFlight review app status",
			query:    []string{"view", "TestFlight", "review", "app", "status"},
			expected: "asc testflight review app view",
		},
		{
			name:     "TestFlight App Store notarization status",
			query:    []string{"TestFlight", "App", "Store", "notarization", "status"},
			expected: "asc notarization status",
		},
		{
			name:     "log TestFlight App Store notarization status",
			query:    []string{"log", "TestFlight", "App", "Store", "notarization", "status"},
			expected: "asc notarization log",
		},
		{
			name:     "list TestFlight App Store notarization status",
			query:    []string{"list", "TestFlight", "App", "Store", "notarization", "status"},
			expected: "asc notarization list",
		},
		{
			name:     "submit TestFlight App Store notarization status",
			query:    []string{"submit", "TestFlight", "App", "Store", "notarization", "status"},
			expected: "asc notarization submit",
		},
		{
			name:     "TestFlight App Review release dashboard",
			query:    []string{"TestFlight", "App", "Review", "release", "dashboard"},
			expected: "asc status",
		},
		{
			name:     "TestFlight and App Review dashboard",
			query:    []string{"TestFlight", "and", "App", "Review", "dashboard"},
			expected: "asc status",
		},
		{
			name:     "App Store App Clip build domain status",
			query:    []string{"App", "Store", "App", "Clip", "build", "domain", "status"},
			expected: "asc app-clips domain-status",
		},
		{
			name:     "App Clip domain cache status",
			query:    []string{"App", "Clip", "domain", "cache", "status"},
			expected: "asc app-clips domain-status cache",
		},
		{
			name:     "App Clip domain debug status",
			query:    []string{"App", "Clip", "domain", "debug", "status"},
			expected: "asc app-clips domain-status debug",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var code int
			stdout, stderr := captureOutput(t, func() {
				args := []string{"search", "--output", "json", "--limit", "5"}
				args = append(args, test.query...)
				code = rootcmd.Run(args, "1.2.3")
			})

			if code != 0 {
				t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var response searchResponse
			if err := json.Unmarshal([]byte(stdout), &response); err != nil {
				t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
			}
			if len(response.Results) == 0 {
				t.Fatalf("expected search results, got %#v", response)
			}
			if response.Results[0].Command != test.expected {
				t.Fatalf("expected scoped status command %q first, got %#v", test.expected, response.Results)
			}
		})
	}
}

func TestSearchKeepsAnalyticsOverviewAheadOfReleaseDashboard(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "analytics", "overview"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if len(response.Results) == 0 {
		t.Fatalf("expected search results, got %#v", response)
	}
	if response.Results[0].Command != "asc web analytics overview" {
		t.Fatalf("expected Analytics overview first, got %#v", response.Results)
	}
}

func TestSearchPrioritizesExplicitReleaseDashboardOverScopedTerms(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{
			"search",
			"--output",
			"json",
			"TestFlight",
			"and",
			"phased",
			"App",
			"Store",
			"release",
			"dashboard",
		}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if len(response.Results) == 0 {
		t.Fatalf("expected search results, got %#v", response)
	}
	if response.Results[0].Command != "asc status" {
		t.Fatalf("expected aggregate release dashboard first, got %#v", response.Results)
	}
}

func TestSearchPrioritizesReleasePipelineDashboard(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "release", "pipeline"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if len(response.Results) == 0 {
		t.Fatalf("expected search results, got %#v", response)
	}
	if response.Results[0].Command != "asc status" {
		t.Fatalf("expected release pipeline dashboard first, got %#v", response.Results)
	}
}

func TestSearchPrioritizesPreciseCommandPathsForNaturalLanguage(t *testing.T) {
	tests := []struct {
		name     string
		query    []string
		expected string
	}{
		{name: "create app", query: []string{"create", "app"}, expected: "asc web apps create"},
		{name: "build status", query: []string{"build", "status"}, expected: "asc xcode-cloud status"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var code int
			stdout, stderr := captureOutput(t, func() {
				args := []string{"search", "--output", "json", "--limit", "5"}
				args = append(args, test.query...)
				code = rootcmd.Run(args, "1.2.3")
			})

			if code != 0 {
				t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var response searchResponse
			if err := json.Unmarshal([]byte(stdout), &response); err != nil {
				t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
			}
			if len(response.Results) == 0 {
				t.Fatalf("expected search results, got %#v", response)
			}
			if response.Results[0].Command != test.expected {
				t.Fatalf("expected precise command %q first, got %#v", test.expected, response.Results)
			}
		})
	}
}

func TestSearchKeepsBuildDownloadsAheadOfGenericDownloads(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "--limit", "5", "download", "build"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if len(response.Results) == 0 {
		t.Fatalf("expected search results, got %#v", response)
	}
	if response.Results[0].Command != "asc builds dsyms" {
		t.Fatalf("expected build dSYM download command first, got %#v", response.Results)
	}
}

func TestSearchKeepsExactBuildUploadAheadOfBroaderPublishWorkflows(t *testing.T) {
	for _, query := range [][]string{
		{"upload", "build"},
		{"upload", "build", "app"},
	} {
		t.Run(strings.Join(query, " "), func(t *testing.T) {
			var code int
			stdout, stderr := captureOutput(t, func() {
				args := []string{"search", "--output", "json", "--limit", "5"}
				args = append(args, query...)
				code = rootcmd.Run(args, "1.2.3")
			})

			if code != 0 {
				t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var response searchResponse
			if err := json.Unmarshal([]byte(stdout), &response); err != nil {
				t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
			}
			if len(response.Results) == 0 {
				t.Fatalf("expected search results, got %#v", response)
			}
			if response.Results[0].Command != "asc builds upload" {
				t.Fatalf("expected exact build upload command first, got %#v", response.Results)
			}
		})
	}
}

func TestSearchDoesNotRouteGenericAppReviewToPublishWorkflow(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "--limit", "5", "app", "review"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if len(response.Results) == 0 {
		t.Fatalf("expected search results, got %#v", response)
	}
	if response.Results[0].Command == "asc publish appstore" {
		t.Fatalf("expected a review-specific command ahead of the publish workflow, got %#v", response.Results)
	}
}

func TestSearchUsesTypoToleranceAsFallback(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"search", "--output", "json", "testfligth", "testers"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}

	if !searchResultsContain(response.Results, "asc testflight testers") {
		t.Fatalf("expected TestFlight tester command for typo query, got %#v", response.Results)
	}
	if !searchResultsMention(response.Results, "fuzzy:testflight") {
		t.Fatalf("expected fuzzy match reason for testfligth typo, got %#v", response.Results)
	}
}

func TestSearchSupportsTableOutput(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"search", "--output", "table", "build", "upload"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "command") || !strings.Contains(stdout, "summary") {
		t.Fatalf("expected table headers, got %q", stdout)
	}
	if !strings.Contains(stdout, "asc builds upload") {
		t.Fatalf("expected build upload command in table output, got %q", stdout)
	}
}

func TestSearchReturnsEmptyResultsForNoMatches(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"search", "--output", "json", "zzzz-not-real"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if response.Count != 0 || len(response.Results) != 0 {
		t.Fatalf("expected empty result set, got %#v", response)
	}
}

func TestSearchSupportsLimitFlag(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "--limit", "2", "build"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if response.Count > 2 || len(response.Results) > 2 {
		t.Fatalf("expected at most 2 results, got %#v", response)
	}
}

func TestSearchDoesNotLeakFlagsAcrossRepeatedRuns(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"search", "--output", "json", "--limit", "1", "build"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var limited searchResponse
	if err := json.Unmarshal([]byte(stdout), &limited); err != nil {
		t.Fatalf("failed to unmarshal limited search JSON: %v\nstdout=%s", err, stdout)
	}
	if limited.Count != 1 || len(limited.Results) != 1 {
		t.Fatalf("expected one limited result, got %#v", limited)
	}

	stdout, stderr = captureOutput(t, func() {
		if err := root.Parse([]string{"search", "--output", "json", "build"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var defaultLimit searchResponse
	if err := json.Unmarshal([]byte(stdout), &defaultLimit); err != nil {
		t.Fatalf("failed to unmarshal default-limit search JSON: %v\nstdout=%s", err, stdout)
	}
	if defaultLimit.Count <= 1 || len(defaultLimit.Results) <= 1 {
		t.Fatalf("expected default limit after repeated run, got %#v", defaultLimit)
	}
}

func TestSearchRejectsNonPositiveLimit(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--limit", "0", "build"}, "1.2.3")
	})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "--limit must be greater than 0") {
		t.Fatalf("expected limit validation error, got %q", stderr)
	}
}

func TestSearchAcceptsRootFlagsBeforeSubcommand(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"--profile", "ci", "search", "--output", "json", "--limit", "1", "build"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if response.Count != 1 || len(response.Results) != 1 {
		t.Fatalf("expected one limited result, got %#v", response)
	}
}

func TestSearchFlagValueCanMatchSubcommandName(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "--limit", "3", "completion"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if response.Query != "completion" {
		t.Fatalf("expected completion query, got %q", response.Query)
	}
	if !searchResultsContain(response.Results, "asc completion") {
		t.Fatalf("expected completion command in results, got %#v", response.Results)
	}
}

func TestSearchFindsFirstClassXcodeBuildCommand(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "--limit", "10", "asc", "xcode", "build"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if !searchResultsContain(response.Results, "asc xcode build") {
		t.Fatalf("expected xcode build command in results, got %#v", response.Results)
	}
}

func TestSearchSupportsMixedFlagOrder(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "build", "--limit", "1"}, "1.2.3")
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if response.Query != "build" {
		t.Fatalf("expected build query, got %q", response.Query)
	}
	if response.Count != 1 || len(response.Results) != 1 {
		t.Fatalf("expected one limited result, got %#v", response)
	}
	if !searchResultsContain(response.Results, "asc build") {
		t.Fatalf("expected build command in results, got %#v", response.Results)
	}
}

func TestSearchPreservesRootShapedQueryTerms(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "logging", "--debug"}, "1.2.3")
	})

	if code != rootcmd.ExitSuccess {
		t.Fatalf("exit code = %d, want %d with stderr %q", code, rootcmd.ExitSuccess, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	var response searchResponse
	if err := json.Unmarshal([]byte(stdout), &response); err != nil {
		t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
	}
	if response.Query != "logging --debug" {
		t.Fatalf("query = %q, want %q", response.Query, "logging --debug")
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{"search"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", runErr)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "search query is required") {
		t.Fatalf("expected missing query error, got %q", stderr)
	}
}

func TestSearchRejectsBlankQuery(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "json", "   "}, "1.2.3")
	})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "search query is required") {
		t.Fatalf("expected missing query error, got %q", stderr)
	}
}

func TestSearchInvalidOutputExitsWithUsageCode(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "--output", "yaml", "builds"}, "1.2.3")
	})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, `(got "yaml")`) {
		t.Fatalf("expected unsupported format error, got %q", stderr)
	}
}

func TestSearchInvalidMixedOrderOutputExitsWithUsageCode(t *testing.T) {
	var code int
	stdout, stderr := captureOutput(t, func() {
		code = rootcmd.Run([]string{"search", "builds", "--output", "yaml"}, "1.2.3")
	})

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, `(got "yaml")`) {
		t.Fatalf("expected unsupported format error, got %q", stderr)
	}
}

func searchResultsContain(results []searchResult, commandPrefix string) bool {
	for _, result := range results {
		if strings.HasPrefix(result.Command, commandPrefix) {
			return true
		}
	}
	return false
}

func searchResultsMention(results []searchResult, match string) bool {
	for _, result := range results {
		for _, item := range result.Matched {
			if item == match {
				return true
			}
		}
	}
	return false
}

func TestSearchRoutesExplicitActionsAheadOfStatusQueries(t *testing.T) {
	tests := []struct {
		name     string
		query    []string
		expected string
	}{
		{
			name:     "accept Apple Developer agreement status",
			query:    []string{"accept", "Apple", "Developer", "agreement", "status"},
			expected: "asc web agreements accept",
		},
		{
			name:     "download Apple Developer agreement status",
			query:    []string{"download", "Apple", "Developer", "agreement", "status"},
			expected: "asc web agreements download",
		},
		{
			name:     "download TestFlight beta license agreement",
			query:    []string{"download", "TestFlight", "beta", "license", "agreement"},
			expected: "asc testflight agreements",
		},
		{
			name:     "agreement status without acceptance",
			query:    []string{"Apple", "Developer", "agreement", "status"},
			expected: "asc web agreements status",
		},
		{
			name:     "split telemetry reset id status",
			query:    []string{"reset", "telemetry", "id", "status"},
			expected: "asc telemetry reset-id",
		},
		{
			name:     "hyphenated telemetry reset-id status",
			query:    []string{"telemetry", "reset-id", "status"},
			expected: "asc telemetry reset-id",
		},
		{
			name:     "split auth issuer id status",
			query:    []string{"auth", "issuer", "id", "status"},
			expected: "asc auth issuer-id",
		},
		{
			name:     "hyphenated auth issuer-id status",
			query:    []string{"auth", "issuer-id", "status"},
			expected: "asc auth issuer-id",
		},
		{
			name:     "view beta review app status",
			query:    []string{"view", "beta", "review", "app", "status"},
			expected: "asc testflight review app view",
		},
		{
			name:     "TestFlight App Store metadata status",
			query:    []string{"TestFlight", "App", "Store", "metadata", "status"},
			expected: "asc metadata status",
		},
		{
			name:     "cancel TestFlight App Store submission status",
			query:    []string{"cancel", "TestFlight", "App", "Store", "submission", "status"},
			expected: "asc submit cancel",
		},
		{
			name:     "cancel App Store submission status",
			query:    []string{"cancel", "App", "Store", "submission", "status"},
			expected: "asc submit cancel",
		},
		{
			name:     "cancel TestFlight review submission status",
			query:    []string{"cancel", "TestFlight", "review", "submission", "status"},
			expected: "asc testflight review submissions view",
		},
		{
			name:     "cancel beta review submission status",
			query:    []string{"cancel", "beta", "review", "submission", "status"},
			expected: "asc testflight review submissions view",
		},
		{
			name:     "cancel beta app review submission status",
			query:    []string{"cancel", "beta", "app", "review", "submission", "status"},
			expected: "asc testflight review submissions view",
		},
		{
			name:     "cancel TestFlight and App Review submission status",
			query:    []string{"cancel", "TestFlight", "and", "App", "Review", "submission", "status"},
			expected: "asc submit cancel",
		},
		{
			name:     "cancel TestFlight review submission for build status",
			query:    []string{"cancel", "TestFlight", "review", "submission", "for", "build", "status"},
			expected: "asc testflight review submissions build",
		},
		{
			name:     "view Xcode Cloud build run",
			query:    []string{"view", "Xcode", "Cloud", "build", "run"},
			expected: "asc xcode-cloud build-runs view",
		},
		{
			name:     "doctor Xcode Cloud build run status",
			query:    []string{"doctor", "Xcode", "Cloud", "build", "run", "status"},
			expected: "asc xcode-cloud doctor",
		},
		{
			name:     "duplicate Xcode Cloud workflow status",
			query:    []string{"duplicate", "Xcode", "Cloud", "workflow", "status"},
			expected: "asc xcode-cloud workflows duplicate",
		},
		{
			name:     "approve metadata status",
			query:    []string{"approve", "metadata", "status"},
			expected: "asc metadata approve",
		},
		{
			name:     "validate metadata status",
			query:    []string{"validate", "metadata", "status"},
			expected: "asc metadata validate",
		},
		{
			name:     "metadata keywords plan status",
			query:    []string{"metadata", "keywords", "plan", "status"},
			expected: "asc metadata keywords plan",
		},
		{
			name:     "metadata keywords apply status",
			query:    []string{"metadata", "keywords", "apply", "status"},
			expected: "asc metadata keywords apply",
		},
		{
			name:     "metadata keywords push status",
			query:    []string{"metadata", "keywords", "push", "status"},
			expected: "asc metadata keywords push",
		},
		{
			name:     "metadata keywords audit status",
			query:    []string{"metadata", "keywords", "audit", "status"},
			expected: "asc metadata keywords audit",
		},
		{
			name:     "subscriptions analytics dashboard",
			query:    []string{"subscriptions", "analytics", "dashboard"},
			expected: "asc web analytics subscriptions",
		},
		{
			name:     "product pages analytics dashboard",
			query:    []string{"product", "pages", "analytics", "dashboard"},
			expected: "asc web analytics product-pages",
		},
		{
			name:     "in app events analytics dashboard",
			query:    []string{"in", "app", "events", "analytics", "dashboard"},
			expected: "asc web analytics in-app-events",
		},
		{
			name:     "app clips analytics dashboard",
			query:    []string{"app", "clips", "analytics", "dashboard"},
			expected: "asc web analytics app-clips",
		},
		{
			name:     "TestFlight App Store analytics overview",
			query:    []string{"TestFlight", "App", "Store", "analytics", "overview"},
			expected: "asc web analytics overview",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var code int
			stdout, stderr := captureOutput(t, func() {
				args := []string{"search", "--output", "json", "--limit", "5"}
				args = append(args, test.query...)
				code = rootcmd.Run(args, "1.2.3")
			})

			if code != 0 {
				t.Fatalf("expected exit code 0, got %d with stderr %q", code, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			var response searchResponse
			if err := json.Unmarshal([]byte(stdout), &response); err != nil {
				t.Fatalf("failed to unmarshal search JSON: %v\nstdout=%s", err, stdout)
			}
			if len(response.Results) == 0 {
				t.Fatalf("expected search results, got %#v", response)
			}
			if response.Results[0].Command != test.expected {
				t.Fatalf("expected %q first, got %#v", test.expected, response.Results)
			}
		})
	}
}
