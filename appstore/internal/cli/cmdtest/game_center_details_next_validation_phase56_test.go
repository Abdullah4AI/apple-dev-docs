package cmdtest

import "testing"

func TestGameCenterDetailsAppVersionsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "details", "app-versions", "list"},
		"game-center details app-versions list: --next",
	)
}

func TestGameCenterDetailsAppVersionsListPaginateFromNextWithoutID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterAppVersions?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterAppVersions?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterAppVersions","id":"gc-detail-app-version-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterAppVersions","id":"gc-detail-app-version-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "details", "app-versions", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-detail-app-version-next-1",
		"gc-detail-app-version-next-2",
	)
}

func TestGameCenterDetailsAchievementsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "details", "achievements-v2", "list"},
		"game-center details achievements-v2 list: --next",
	)
}

func TestGameCenterDetailsAchievementsV2ListPaginateFromNextWithoutID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterAchievementsV2?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterAchievementsV2?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterAchievements","id":"gc-detail-achievement-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterAchievements","id":"gc-detail-achievement-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "details", "achievements-v2", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-detail-achievement-v2-next-1",
		"gc-detail-achievement-v2-next-2",
	)
}

func TestGameCenterDetailsLeaderboardsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "details", "leaderboards-v2", "list"},
		"game-center details leaderboards-v2 list: --next",
	)
}

func TestGameCenterDetailsLeaderboardsV2ListPaginateFromNextWithoutID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboardsV2?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboardsV2?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-detail-leaderboard-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-detail-leaderboard-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "details", "leaderboards-v2", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-detail-leaderboard-v2-next-1",
		"gc-detail-leaderboard-v2-next-2",
	)
}

func TestGameCenterDetailsLeaderboardSetsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "details", "leaderboard-sets-v2", "list"},
		"game-center details leaderboard-sets-v2 list: --next",
	)
}

func TestGameCenterDetailsLeaderboardSetsV2ListPaginateFromNextWithoutID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboardSetsV2?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboardSetsV2?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardSets","id":"gc-detail-leaderboard-set-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardSets","id":"gc-detail-leaderboard-set-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "details", "leaderboard-sets-v2", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-detail-leaderboard-set-v2-next-1",
		"gc-detail-leaderboard-set-v2-next-2",
	)
}

func TestGameCenterDetailsAchievementReleasesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "details", "achievement-releases", "list"},
		"game-center details achievement-releases list: --next",
	)
}

func TestGameCenterDetailsAchievementReleasesListPaginateFromNextWithoutID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/achievementReleases?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/achievementReleases?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterAchievementReleases","id":"gc-detail-achievement-release-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterAchievementReleases","id":"gc-detail-achievement-release-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "details", "achievement-releases", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-detail-achievement-release-next-1",
		"gc-detail-achievement-release-next-2",
	)
}

func TestGameCenterDetailsLeaderboardReleasesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "details", "leaderboard-releases", "list"},
		"game-center details leaderboard-releases list: --next",
	)
}

func TestGameCenterDetailsLeaderboardReleasesListPaginateFromNextWithoutID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/leaderboardReleases?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/leaderboardReleases?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardReleases","id":"gc-detail-leaderboard-release-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardReleases","id":"gc-detail-leaderboard-release-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "details", "leaderboard-releases", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-detail-leaderboard-release-next-1",
		"gc-detail-leaderboard-release-next-2",
	)
}

func TestGameCenterDetailsLeaderboardSetReleasesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "details", "leaderboard-set-releases", "list"},
		"game-center details leaderboard-set-releases list: --next",
	)
}

func TestGameCenterDetailsLeaderboardSetReleasesListPaginateFromNextWithoutID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/leaderboardSetReleases?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/leaderboardSetReleases?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardSetReleases","id":"gc-detail-leaderboard-set-release-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardSetReleases","id":"gc-detail-leaderboard-set-release-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "details", "leaderboard-set-releases", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-detail-leaderboard-set-release-next-1",
		"gc-detail-leaderboard-set-release-next-2",
	)
}
