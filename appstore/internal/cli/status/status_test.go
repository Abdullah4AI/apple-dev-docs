package status

import (
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestParseInclude_DefaultsToAllSections(t *testing.T) {
	includes, err := parseInclude("")
	if err != nil {
		t.Fatalf("parseInclude error: %v", err)
	}

	if !includes.app || !includes.builds || !includes.testflight || !includes.appstore || !includes.submission || !includes.review || !includes.phasedRelease || !includes.links {
		t.Fatalf("expected all sections enabled by default, got %+v", includes)
	}
}

func TestParseInclude_RejectsUnknownSection(t *testing.T) {
	_, err := parseInclude("builds,unknown")
	if err == nil {
		t.Fatal("expected error for unknown include section")
	}
}

func TestParseInclude_AppOnly(t *testing.T) {
	includes, err := parseInclude("app")
	if err != nil {
		t.Fatalf("parseInclude error: %v", err)
	}
	if !includes.app {
		t.Fatal("expected app include enabled")
	}
	if includes.builds || includes.testflight || includes.appstore || includes.submission || includes.review || includes.phasedRelease || includes.links {
		t.Fatalf("expected only app include enabled, got %+v", includes)
	}
}

func TestSelectLatestAppStoreVersion_DeterministicTieBreak(t *testing.T) {
	versions := []asc.Resource[asc.AppStoreVersionAttributes]{
		{
			ID: "ver-1",
			Attributes: asc.AppStoreVersionAttributes{
				CreatedDate: "2026-02-20T00:00:00Z",
			},
		},
		{
			ID: "ver-2",
			Attributes: asc.AppStoreVersionAttributes{
				CreatedDate: "2026-02-20T00:00:00Z",
			},
		},
	}

	selected := selectLatestAppStoreVersion(versions)
	if selected == nil {
		t.Fatal("expected selected version, got nil")
	}
	if selected.ID != "ver-2" {
		t.Fatalf("expected deterministic tie-break to choose ver-2, got %q", selected.ID)
	}
}

func TestSelectLatestAppStoreVersion_ParsesRFC3339Offsets(t *testing.T) {
	versions := []asc.Resource[asc.AppStoreVersionAttributes]{
		{
			ID: "ver-older",
			Attributes: asc.AppStoreVersionAttributes{
				CreatedDate: "2026-02-20T01:00:00+01:00",
			},
		},
		{
			ID: "ver-newer",
			Attributes: asc.AppStoreVersionAttributes{
				CreatedDate: "2026-02-20T00:30:00Z",
			},
		},
	}

	selected := selectLatestAppStoreVersion(versions)
	if selected == nil {
		t.Fatal("expected selected version, got nil")
	}
	if selected.ID != "ver-newer" {
		t.Fatalf("expected ver-newer to be selected, got %q", selected.ID)
	}
}

func TestSelectLatestReviewSubmission_DeterministicTieBreak(t *testing.T) {
	submissions := []asc.ReviewSubmissionResource{
		{
			ID: "sub-1",
			Attributes: asc.ReviewSubmissionAttributes{
				SubmittedDate: "2026-02-20T00:00:00Z",
			},
		},
		{
			ID: "sub-2",
			Attributes: asc.ReviewSubmissionAttributes{
				SubmittedDate: "2026-02-20T00:00:00Z",
			},
		},
	}

	selected := selectLatestReviewSubmission(submissions)
	if selected == nil {
		t.Fatal("expected selected submission, got nil")
	}
	if selected.ID != "sub-2" {
		t.Fatalf("expected deterministic tie-break to choose sub-2, got %q", selected.ID)
	}
}

func TestSelectLatestReviewSubmission_ParsesRFC3339Offsets(t *testing.T) {
	submissions := []asc.ReviewSubmissionResource{
		{
			ID: "sub-older",
			Attributes: asc.ReviewSubmissionAttributes{
				SubmittedDate: "2026-02-20T01:00:00+01:00",
			},
		},
		{
			ID: "sub-newer",
			Attributes: asc.ReviewSubmissionAttributes{
				SubmittedDate: "2026-02-20T00:30:00Z",
			},
		},
	}

	selected := selectLatestReviewSubmission(submissions)
	if selected == nil {
		t.Fatal("expected selected submission, got nil")
	}
	if selected.ID != "sub-newer" {
		t.Fatalf("expected sub-newer to be selected, got %q", selected.ID)
	}
}

func TestSelectLatestBetaReviewSubmission_ParsesRFC3339Offsets(t *testing.T) {
	submissions := []asc.Resource[asc.BetaAppReviewSubmissionAttributes]{
		{
			ID: "beta-sub-older",
			Attributes: asc.BetaAppReviewSubmissionAttributes{
				SubmittedDate: "2026-02-20T01:00:00+01:00",
			},
		},
		{
			ID: "beta-sub-newer",
			Attributes: asc.BetaAppReviewSubmissionAttributes{
				SubmittedDate: "2026-02-20T00:30:00Z",
			},
		},
	}

	selected := selectLatestBetaReviewSubmission(submissions)
	if selected == nil {
		t.Fatal("expected selected submission, got nil")
	}
	if selected.ID != "beta-sub-newer" {
		t.Fatalf("expected beta-sub-newer to be selected, got %q", selected.ID)
	}
}

func TestBuildStatusSummary_RedWhenBlockingIssuesExist(t *testing.T) {
	resp := &dashboardResponse{
		Submission: &submissionSection{
			InFlight:       true,
			BlockingIssues: []string{"submission abc has unresolved issues"},
		},
	}

	summary := buildStatusSummary(resp)
	if summary.Health != "red" {
		t.Fatalf("expected health=red, got %q", summary.Health)
	}
	if summary.NextAction == "" {
		t.Fatal("expected next action")
	}
	if len(summary.Blockers) == 0 {
		t.Fatal("expected blockers")
	}
}

func TestBuildStatusSummary_YellowWhenReviewInFlight(t *testing.T) {
	resp := &dashboardResponse{
		Review: &reviewSection{
			State: "WAITING_FOR_REVIEW",
		},
	}

	summary := buildStatusSummary(resp)
	if summary.Health != "yellow" {
		t.Fatalf("expected health=yellow, got %q", summary.Health)
	}
}

func TestBuildStatusSummary_GreenWhenReadyForSale(t *testing.T) {
	resp := &dashboardResponse{
		AppStore: &appStoreSection{
			State: "READY_FOR_SALE",
		},
		Builds: &buildsSection{
			Latest: &latestBuild{ID: "build-1"},
		},
	}

	summary := buildStatusSummary(resp)
	if summary.Health != "green" {
		t.Fatalf("expected health=green, got %q", summary.Health)
	}
	if summary.NextAction != "No action needed." {
		t.Fatalf("expected no action needed, got %q", summary.NextAction)
	}
}

func TestPhasedReleaseProgressBar(t *testing.T) {
	bar := phasedReleaseProgressBar(&phasedReleaseSection{
		Configured:       true,
		CurrentDayNumber: 3,
	})
	if bar == "" {
		t.Fatal("expected progress bar")
	}
	if bar != "[####------] 3/7" {
		t.Fatalf("expected deterministic bar, got %q", bar)
	}
}

func TestBuildExternalStatesByBuildID_AvoidsAmbiguousPositionalFallback(t *testing.T) {
	buildIDs := []string{"build-2", "build-1"}
	betaDetails := &asc.BuildBetaDetailsResponse{
		Data: []asc.Resource[asc.BuildBetaDetailAttributes]{
			{
				ID: "detail-1",
				Attributes: asc.BuildBetaDetailAttributes{
					ExternalBuildState: "IN_BETA_TESTING",
				},
			},
			{
				ID: "detail-2",
				Attributes: asc.BuildBetaDetailAttributes{
					ExternalBuildState: "READY_FOR_TESTING",
				},
			},
		},
	}

	statesByBuildID := buildExternalStatesByBuildID(buildIDs, betaDetails)
	if len(statesByBuildID) != 0 {
		t.Fatalf("expected no mapping without build relationships for multiple builds, got %+v", statesByBuildID)
	}
}

func TestBuildExternalStatesByBuildID_UsesSingleItemPositionalFallback(t *testing.T) {
	buildIDs := []string{"build-1"}
	betaDetails := &asc.BuildBetaDetailsResponse{
		Data: []asc.Resource[asc.BuildBetaDetailAttributes]{
			{
				ID: "detail-1",
				Attributes: asc.BuildBetaDetailAttributes{
					ExternalBuildState: "IN_BETA_TESTING",
				},
			},
		},
	}

	statesByBuildID := buildExternalStatesByBuildID(buildIDs, betaDetails)
	if statesByBuildID["build-1"] != "IN_BETA_TESTING" {
		t.Fatalf("expected build-1 to map to IN_BETA_TESTING, got %q", statesByBuildID["build-1"])
	}
}

func TestStateSymbolClassification(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "READY_FOR_SALE", want: "[+]"},
		{value: "IN_REVIEW", want: "[~]"},
		{value: "READY_FOR_REVIEW", want: "[~]"},
		{value: "UNRESOLVED_ISSUES", want: "[x]"},
		{value: "", want: "[-]"},
	}
	for _, test := range tests {
		if got := stateSymbol(test.value); got != test.want {
			t.Fatalf("stateSymbol(%q) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestFormatDateWithRelative(t *testing.T) {
	originalNow := statusNow
	statusNow = func() time.Time {
		return time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		statusNow = originalNow
	})

	got := formatDateWithRelative("2026-02-19T12:00:00Z")
	if got != "2026-02-19T12:00:00Z (1d ago)" {
		t.Fatalf("unexpected relative time output %q", got)
	}
}
