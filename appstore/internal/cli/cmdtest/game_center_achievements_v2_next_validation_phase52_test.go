package cmdtest

import "testing"

func TestGameCenterAchievementsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "achievements", "v2", "list"},
		"game-center achievements v2 list: --next",
	)
}

func TestGameCenterAchievementsV2ListPaginateFromNextWithoutAppOrGroup(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/gc-detail-1/gameCenterAchievementsV2?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/gc-detail-1/gameCenterAchievementsV2?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterAchievements","id":"gc-achievement-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterAchievements","id":"gc-achievement-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "achievements", "v2", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-achievement-v2-next-1",
		"gc-achievement-v2-next-2",
	)
}

func TestGameCenterAchievementVersionsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "achievements", "v2", "versions", "list"},
		"game-center achievements v2 versions list: --next",
	)
}

func TestGameCenterAchievementVersionsV2ListPaginateFromNextWithoutAchievementID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v2/gameCenterAchievements/ach-v2-1/versions?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v2/gameCenterAchievements/ach-v2-1/versions?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterAchievementVersions","id":"gc-achievement-version-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterAchievementVersions","id":"gc-achievement-version-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "achievements", "v2", "versions", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-achievement-version-v2-next-1",
		"gc-achievement-version-v2-next-2",
	)
}

func TestGameCenterAchievementLocalizationsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "achievements", "v2", "localizations", "list"},
		"game-center achievements v2 localizations list: --next",
	)
}

func TestGameCenterAchievementLocalizationsV2ListPaginateFromNextWithoutVersionID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v2/gameCenterAchievementVersions/ver-v2-1/localizations?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v2/gameCenterAchievementVersions/ver-v2-1/localizations?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterAchievementLocalizations","id":"gc-achievement-localization-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterAchievementLocalizations","id":"gc-achievement-localization-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "achievements", "v2", "localizations", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-achievement-localization-v2-next-1",
		"gc-achievement-localization-v2-next-2",
	)
}
