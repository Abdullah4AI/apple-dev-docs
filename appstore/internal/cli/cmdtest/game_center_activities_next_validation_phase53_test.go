package cmdtest

import "testing"

func TestGameCenterActivitiesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "activities", "list"},
		"game-center activities list: --next",
	)
}

func TestGameCenterActivitiesListPaginateFromNextWithoutApp(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/gc-detail-1/gameCenterActivities?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/gc-detail-1/gameCenterActivities?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterActivities","id":"gc-activity-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterActivities","id":"gc-activity-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "activities", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-activity-next-1",
		"gc-activity-next-2",
	)
}

func TestGameCenterActivityVersionsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "activities", "versions", "list"},
		"game-center activities versions list: --next",
	)
}

func TestGameCenterActivityVersionsListPaginateFromNextWithoutActivityID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterActivities/activity-1/versions?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterActivities/activity-1/versions?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterActivityVersions","id":"gc-activity-version-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterActivityVersions","id":"gc-activity-version-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "activities", "versions", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-activity-version-next-1",
		"gc-activity-version-next-2",
	)
}

func TestGameCenterActivityLocalizationsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "activities", "localizations", "list"},
		"game-center activities localizations list: --next",
	)
}

func TestGameCenterActivityLocalizationsListPaginateFromNextWithoutVersionID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterActivityVersions/version-1/localizations?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterActivityVersions/version-1/localizations?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterActivityLocalizations","id":"gc-activity-localization-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterActivityLocalizations","id":"gc-activity-localization-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "activities", "localizations", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-activity-localization-next-1",
		"gc-activity-localization-next-2",
	)
}

func TestGameCenterActivityReleasesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "activities", "releases", "list"},
		"game-center activities releases list: --next",
	)
}

func TestGameCenterActivityReleasesListPaginateFromNextWithoutApp(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/gc-detail-1/activityReleases?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/gc-detail-1/activityReleases?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterActivityVersionReleases","id":"gc-activity-release-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterActivityVersionReleases","id":"gc-activity-release-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "activities", "releases", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-activity-release-next-1",
		"gc-activity-release-next-2",
	)
}
