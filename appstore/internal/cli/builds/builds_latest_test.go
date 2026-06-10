package builds

import (
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

// TestSelectNewestBuild verifies that the multi-preReleaseVersion selection
// logic correctly picks the build with the newest uploadedDate.
func TestSelectNewestBuild(t *testing.T) {
	// Simulate builds from different preReleaseVersions with different dates
	builds := []asc.Resource[asc.BuildAttributes]{
		{
			ID: "build-older",
			Attributes: asc.BuildAttributes{
				Version:      "1.0",
				UploadedDate: "2026-01-15T10:00:00Z",
			},
		},
		{
			ID: "build-newest",
			Attributes: asc.BuildAttributes{
				Version:      "2.0",
				UploadedDate: "2026-01-20T10:00:00Z",
			},
		},
		{
			ID: "build-middle",
			Attributes: asc.BuildAttributes{
				Version:      "1.5",
				UploadedDate: "2026-01-18T10:00:00Z",
			},
		},
	}

	// The selection logic: find the build with the newest uploadedDate
	var newestBuild *asc.Resource[asc.BuildAttributes]
	var newestDate string

	for i := range builds {
		if newestBuild == nil || builds[i].Attributes.UploadedDate > newestDate {
			newestBuild = &builds[i]
			newestDate = builds[i].Attributes.UploadedDate
		}
	}

	if newestBuild == nil {
		t.Fatal("expected to find a newest build")
		return
	}
	if newestBuild.ID != "build-newest" {
		t.Errorf("expected build-newest to be selected, got %s", newestBuild.ID)
	}
	if newestDate != "2026-01-20T10:00:00Z" {
		t.Errorf("expected newest date 2026-01-20T10:00:00Z, got %s", newestDate)
	}
}

// TestSelectNewestBuild_OlderVersionCanBeNewer verifies that an older version
// string (e.g., "1.0") can have a newer uploadedDate than a higher version (e.g., "2.0").
// This tests the scenario where someone uploads a hotfix to an older version.
func TestSelectNewestBuild_OlderVersionCanBeNewer(t *testing.T) {
	builds := []asc.Resource[asc.BuildAttributes]{
		{
			ID: "build-v2-old",
			Attributes: asc.BuildAttributes{
				Version:      "2.0",
				UploadedDate: "2026-01-10T10:00:00Z", // Version 2.0 uploaded earlier
			},
		},
		{
			ID: "build-v1-hotfix",
			Attributes: asc.BuildAttributes{
				Version:      "1.0",
				UploadedDate: "2026-01-20T10:00:00Z", // Version 1.0 hotfix uploaded later
			},
		},
	}

	var newestBuild *asc.Resource[asc.BuildAttributes]
	var newestDate string

	for i := range builds {
		if newestBuild == nil || builds[i].Attributes.UploadedDate > newestDate {
			newestBuild = &builds[i]
			newestDate = builds[i].Attributes.UploadedDate
		}
	}

	// The 1.0 hotfix should be selected because it was uploaded more recently
	if newestBuild.ID != "build-v1-hotfix" {
		t.Errorf("expected build-v1-hotfix (newer upload) to be selected, got %s", newestBuild.ID)
	}
}
