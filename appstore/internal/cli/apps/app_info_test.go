package apps

import (
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestSelectLatestAppStoreVersion(t *testing.T) {
	versions := []asc.Resource[asc.AppStoreVersionAttributes]{
		{
			ID: "old",
			Attributes: asc.AppStoreVersionAttributes{
				CreatedDate: "2024-01-01T00:00:00Z",
			},
		},
		{
			ID: "new",
			Attributes: asc.AppStoreVersionAttributes{
				CreatedDate: "2025-01-01T00:00:00Z",
			},
		},
	}

	selected := selectLatestAppStoreVersion(versions)
	if selected.ID != "new" {
		t.Fatalf("expected latest version to be %q, got %q", "new", selected.ID)
	}
}

func TestSelectLatestAppStoreVersionFallsBackToFirst(t *testing.T) {
	versions := []asc.Resource[asc.AppStoreVersionAttributes]{
		{
			ID: "first",
			Attributes: asc.AppStoreVersionAttributes{
				CreatedDate: "invalid-date",
			},
		},
		{
			ID: "second",
			Attributes: asc.AppStoreVersionAttributes{
				CreatedDate: "",
			},
		},
	}

	selected := selectLatestAppStoreVersion(versions)
	if selected.ID != "first" {
		t.Fatalf("expected fallback to the first version, got %q", selected.ID)
	}
}
