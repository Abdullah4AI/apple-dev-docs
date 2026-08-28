package validation

import (
	"fmt"
	"strings"
	"testing"
)

func validScreenshots(count int) []Screenshot {
	shots := make([]Screenshot, 0, count)
	for i := 1; i <= count; i++ {
		shots = append(shots, Screenshot{
			ID:       fmt.Sprintf("shot-%d", i),
			FileName: fmt.Sprintf("shot-%d.png", i),
			Width:    1242,
			Height:   2688,
		})
	}
	return shots
}

func TestScreenshotChecks_Mismatch(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-1",
			DisplayType: "APP_IPHONE_65",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-1", FileName: "shot.png", Width: 100, Height: 100},
			},
		},
	}

	checks := screenshotChecks("IOS", sets)
	if !hasCheckID(checks, "screenshots.dimension_mismatch") {
		t.Fatalf("expected dimension mismatch check")
	}
}

func TestScreenshotChecks_Pass(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-1",
			DisplayType: "APP_IPHONE_65",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-1", FileName: "shot.png", Width: 1242, Height: 2688},
			},
		},
	}

	checks := screenshotChecks("IOS", sets)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d", len(checks))
	}
}

func TestScreenshotChecks_PassIPhone65ConsolidatedSlot(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-1",
			DisplayType: "APP_IPHONE_65",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-1", FileName: "shot-1.png", Width: 1242, Height: 2688},
				{ID: "shot-2", FileName: "shot-2.png", Width: 1284, Height: 2778},
			},
		},
	}

	checks := screenshotChecks("IOS", sets)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestScreenshotChecks_PassLatestLargeIPhoneSizes(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-1",
			DisplayType: "APP_IPHONE_67",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-1", FileName: "shot-1.png", Width: 1260, Height: 2736},
				{ID: "shot-2", FileName: "shot-2.png", Width: 1320, Height: 2868},
			},
		},
	}

	checks := screenshotChecks("IOS", sets)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestScreenshotChecks_PassLatestIPhone61Size(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-1",
			DisplayType: "APP_IPHONE_61",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-1", FileName: "shot-1.png", Width: 1206, Height: 2622},
			},
		},
	}

	checks := screenshotChecks("IOS", sets)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestScreenshotChecks_PassLatestIPhone58And65AndIPad11Sizes(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-58",
			DisplayType: "APP_IPHONE_58",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-58", FileName: "shot-58.png", Width: 1170, Height: 2532},
			},
		},
		{
			ID:          "set-65",
			DisplayType: "APP_IPHONE_65",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-65", FileName: "shot-65.png", Width: 1284, Height: 2778},
			},
		},
		{
			ID:          "set-ipad11",
			DisplayType: "APP_IPAD_PRO_3GEN_11",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-ipad11", FileName: "shot-ipad11.png", Width: 1488, Height: 2266},
			},
		},
	}

	checks := screenshotChecks("IOS", sets)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestScreenshotChecks_PassIPadPro129M5Size(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-1",
			DisplayType: "APP_IPAD_PRO_3GEN_129",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-1", FileName: "shot-1.png", Width: 2064, Height: 2752},
				{ID: "shot-2", FileName: "shot-2.png", Width: 2752, Height: 2064},
			},
		},
	}

	checks := screenshotChecks("IOS", sets)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestScreenshotChecks_PassDesktopAndWatchUltraNewestSizes(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-mac",
			DisplayType: "APP_DESKTOP",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-mac", FileName: "mac.png", Width: 2880, Height: 1800},
			},
		},
		{
			ID:          "set-watch",
			DisplayType: "APP_WATCH_ULTRA",
			Locale:      "en-US",
			Screenshots: []Screenshot{
				{ID: "shot-watch", FileName: "watch.png", Width: 422, Height: 514},
			},
		},
	}

	checks := screenshotChecks("IOS", sets)
	if len(checks) != 1 {
		t.Fatalf("expected one platform mismatch check for APP_DESKTOP under IOS, got %d (%v)", len(checks), checks)
	}
	if checks[0].ID != "screenshots.display_type_platform_mismatch" {
		t.Fatalf("expected platform mismatch check, got %s", checks[0].ID)
	}

	iosOnly := screenshotChecks("IOS", []ScreenshotSet{sets[1]})
	if len(iosOnly) != 0 {
		t.Fatalf("expected no checks for watch ultra IOS set, got %d (%v)", len(iosOnly), iosOnly)
	}

	macOnly := screenshotChecks("MAC_OS", []ScreenshotSet{sets[0]})
	if len(macOnly) != 0 {
		t.Fatalf("expected no checks for desktop MAC_OS set, got %d (%v)", len(macOnly), macOnly)
	}
}

func TestScreenshotChecks_ExceedsMaxScreenshots(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-1",
			DisplayType: "APP_IPHONE_65",
			Locale:      "en-US",
			Screenshots: validScreenshots(LimitScreenshotsPerSet + 1),
		},
	}

	checks := screenshotChecks("IOS", sets)
	if len(checks) != 1 {
		t.Fatalf("expected exactly one check, got %d (%v)", len(checks), checks)
	}

	check := checks[0]
	if check.ID != "screenshots.count.exceeds_max" {
		t.Fatalf("expected screenshots.count.exceeds_max, got %s", check.ID)
	}
	if check.Severity != SeverityError {
		t.Fatalf("expected error severity, got %s", check.Severity)
	}
	if check.Locale != "en-US" {
		t.Fatalf("expected locale en-US, got %q", check.Locale)
	}
	if check.ResourceType != "appScreenshotSet" || check.ResourceID != "set-1" {
		t.Fatalf("expected appScreenshotSet set-1, got %s %s", check.ResourceType, check.ResourceID)
	}
	if !strings.Contains(check.Message, "APP_IPHONE_65") || !strings.Contains(check.Message, "11") {
		t.Fatalf("expected message to name the display type and count, got %q", check.Message)
	}
	for _, want := range []string{"en-US", "APP_IPHONE_65", "11", "10"} {
		if !strings.Contains(check.Remediation, want) {
			t.Fatalf("expected remediation to contain %q, got %q", want, check.Remediation)
		}
	}
}

func TestScreenshotChecks_PassAtMaxScreenshots(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-1",
			DisplayType: "APP_IPHONE_65",
			Locale:      "en-US",
			Screenshots: validScreenshots(LimitScreenshotsPerSet),
		},
	}

	checks := screenshotChecks("IOS", sets)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}

func TestScreenshotChecks_ExceedsMaxAlongsideDimensionMismatch(t *testing.T) {
	shots := validScreenshots(LimitScreenshotsPerSet + 1)
	shots[3] = Screenshot{ID: "shot-bad", FileName: "bad.png", Width: 100, Height: 100}

	sets := []ScreenshotSet{
		{
			ID:          "set-1",
			DisplayType: "APP_IPHONE_65",
			Locale:      "en-US",
			Screenshots: shots,
		},
	}

	checks := screenshotChecks("IOS", sets)
	if len(checks) != 2 {
		t.Fatalf("expected two checks, got %d (%v)", len(checks), checks)
	}
	if !hasCheckID(checks, "screenshots.count.exceeds_max") {
		t.Fatalf("expected screenshots.count.exceeds_max check, got %v", checks)
	}
	if !hasCheckID(checks, "screenshots.dimension_mismatch") {
		t.Fatalf("expected screenshots.dimension_mismatch check, got %v", checks)
	}
}

func TestScreenshotChecks_ExceedsMaxForUnknownDisplayType(t *testing.T) {
	sets := []ScreenshotSet{
		{
			ID:          "set-1",
			DisplayType: "APP_MYSTERY_DEVICE",
			Locale:      "fr-FR",
			Screenshots: validScreenshots(LimitScreenshotsPerSet + 2),
		},
	}

	checks := screenshotChecks("IOS", sets)
	if !hasCheckID(checks, "screenshots.count.exceeds_max") {
		t.Fatalf("expected screenshots.count.exceeds_max check for unknown display type, got %v", checks)
	}
	if !hasCheckID(checks, "screenshots.display_type_unknown") {
		t.Fatalf("expected screenshots.display_type_unknown check, got %v", checks)
	}
}

func TestScreenshotPresenceChecks_NoSets(t *testing.T) {
	versionLocs := []VersionLocalization{
		{ID: "ver-loc-1", Locale: "en-US"},
	}

	checks := screenshotPresenceChecks("en-US", versionLocs, nil)
	if !hasCheckID(checks, "screenshots.required.any") {
		t.Fatalf("expected screenshots.required.any check")
	}
}

func TestScreenshotPresenceChecks_MissingSetsForLocalization(t *testing.T) {
	versionLocs := []VersionLocalization{
		{ID: "ver-loc-en", Locale: "en-US"},
		{ID: "ver-loc-fr", Locale: "fr-FR"},
	}
	sets := []ScreenshotSet{
		{
			ID:             "set-fr-1",
			DisplayType:    "APP_IPHONE_65",
			Locale:         "fr-FR",
			LocalizationID: "ver-loc-fr",
			Screenshots: []Screenshot{
				{ID: "shot-1", FileName: "shot.png", Width: 1242, Height: 2688},
			},
		},
	}

	checks := screenshotPresenceChecks("en-US", versionLocs, sets)
	if !hasCheckID(checks, "screenshots.required.localization_missing_sets") {
		t.Fatalf("expected screenshots.required.localization_missing_sets check")
	}

	foundEN := false
	for _, c := range checks {
		if c.ID == "screenshots.required.localization_missing_sets" && c.Locale == "en-US" {
			foundEN = true
			break
		}
	}
	if !foundEN {
		t.Fatalf("expected missing-sets check for en-US, got %v", checks)
	}
}

func TestScreenshotPresenceChecks_EmptySet(t *testing.T) {
	versionLocs := []VersionLocalization{
		{ID: "ver-loc-1", Locale: "en-US"},
	}
	sets := []ScreenshotSet{
		{
			ID:             "set-1",
			DisplayType:    "APP_IPHONE_65",
			Locale:         "en-US",
			LocalizationID: "ver-loc-1",
			Screenshots:    nil,
		},
	}

	checks := screenshotPresenceChecks("en-US", versionLocs, sets)
	if !hasCheckID(checks, "screenshots.required.set_nonempty") {
		t.Fatalf("expected screenshots.required.set_nonempty check")
	}
}

func TestScreenshotPresenceChecks_Pass(t *testing.T) {
	versionLocs := []VersionLocalization{
		{ID: "ver-loc-1", Locale: "en-US"},
	}
	sets := []ScreenshotSet{
		{
			ID:             "set-1",
			DisplayType:    "APP_IPHONE_65",
			Locale:         "en-US",
			LocalizationID: "ver-loc-1",
			Screenshots: []Screenshot{
				{ID: "shot-1", FileName: "shot.png", Width: 1242, Height: 2688},
			},
		},
	}

	checks := screenshotPresenceChecks("en-US", versionLocs, sets)
	if len(checks) != 0 {
		t.Fatalf("expected no checks, got %d (%v)", len(checks), checks)
	}
}
