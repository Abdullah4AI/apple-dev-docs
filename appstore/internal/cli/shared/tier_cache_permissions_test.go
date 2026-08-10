//go:build !windows

package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTierCacheSaveUsesPrivatePermissions(t *testing.T) {
	for _, tc := range []struct {
		name         string
		existingMode os.FileMode
	}{
		{name: "new cache"},
		{name: "existing restrictive cache", existingMode: 0o600},
		{name: "existing permissive cache", existingMode: 0o644},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cacheDir := useTierCacheDirForTest(t)

			cachePath, err := tierCachePath("app123", "USA")
			if err != nil {
				t.Fatalf("tierCachePath() error: %v", err)
			}
			if got := filepath.Dir(cachePath); got != cacheDir {
				t.Fatalf("cache dir = %q, want %q", got, cacheDir)
			}
			if tc.existingMode != 0 {
				if err := os.WriteFile(cachePath, []byte("existing cache"), tc.existingMode); err != nil {
					t.Fatalf("write cache with mode %o: %v", tc.existingMode, err)
				}
			}

			if err := SaveTierCache("app123", "USA", []TierEntry{{
				Tier:          1,
				PricePointID:  "pp-1",
				CustomerPrice: "0.99",
			}}); err != nil {
				t.Fatalf("SaveTierCache() error: %v", err)
			}

			info, err := os.Stat(cachePath)
			if err != nil {
				t.Fatalf("stat cache: %v", err)
			}
			if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
				t.Fatalf("cache mode = %o, want %o", got, want)
			}
		})
	}
}
