package cmdtest

import "testing"

func TestGameCenterLeaderboardsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboards", "list"},
		"game-center leaderboards list: --next",
	)
}

func TestGameCenterLeaderboardsListPaginateFromNextWithoutApp(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboards?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboards?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-leaderboard-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-leaderboard-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboards", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-next-1",
		"gc-leaderboard-next-2",
	)
}

func TestGameCenterLeaderboardLocalizationsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboards", "localizations", "list"},
		"game-center leaderboards localizations list: --next",
	)
}

func TestGameCenterLeaderboardLocalizationsListPaginateFromNextWithoutLeaderboardID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboards/lb-1/localizations?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboards/lb-1/localizations?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardLocalizations","id":"gc-leaderboard-localization-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardLocalizations","id":"gc-leaderboard-localization-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboards", "localizations", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-localization-next-1",
		"gc-leaderboard-localization-next-2",
	)
}

func TestGameCenterLeaderboardReleasesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboards", "releases", "list"},
		"game-center leaderboards releases list: --next",
	)
}

func TestGameCenterLeaderboardReleasesListPaginateFromNextWithoutID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboards/lb-1/releases?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboards/lb-1/releases?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardReleases","id":"gc-leaderboard-release-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardReleases","id":"gc-leaderboard-release-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboards", "releases", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-release-next-1",
		"gc-leaderboard-release-next-2",
	)
}

func TestGameCenterLeaderboardsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboards", "v2", "list"},
		"game-center leaderboards v2 list: --next",
	)
}

func TestGameCenterLeaderboardsV2ListPaginateFromNextWithoutAppOrGroup(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboardsV2?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboardsV2?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-leaderboard-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-leaderboard-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboards", "v2", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-v2-next-1",
		"gc-leaderboard-v2-next-2",
	)
}

func TestGameCenterLeaderboardVersionsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboards", "v2", "versions", "list"},
		"game-center leaderboards v2 versions list: --next",
	)
}

func TestGameCenterLeaderboardVersionsV2ListPaginateFromNextWithoutLeaderboardID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v2/gameCenterLeaderboards/lb-1/versions?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v2/gameCenterLeaderboards/lb-1/versions?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardVersions","id":"gc-leaderboard-version-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardVersions","id":"gc-leaderboard-version-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboards", "v2", "versions", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-version-v2-next-1",
		"gc-leaderboard-version-v2-next-2",
	)
}

func TestGameCenterLeaderboardLocalizationsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboards", "v2", "localizations", "list"},
		"game-center leaderboards v2 localizations list: --next",
	)
}

func TestGameCenterLeaderboardLocalizationsV2ListPaginateFromNextWithoutVersionID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v2/gameCenterLeaderboardVersions/ver-1/localizations?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v2/gameCenterLeaderboardVersions/ver-1/localizations?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardLocalizations","id":"gc-leaderboard-localization-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardLocalizations","id":"gc-leaderboard-localization-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboards", "v2", "localizations", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-localization-v2-next-1",
		"gc-leaderboard-localization-v2-next-2",
	)
}
