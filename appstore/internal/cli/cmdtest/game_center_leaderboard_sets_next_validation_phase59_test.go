package cmdtest

import "testing"

func TestGameCenterLeaderboardSetsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboard-sets", "list"},
		"game-center leaderboard-sets list: --next",
	)
}

func TestGameCenterLeaderboardSetsListPaginateFromNextWithoutApp(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboardSets?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboardSets?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardSets","id":"gc-leaderboard-set-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardSets","id":"gc-leaderboard-set-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboard-sets", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-set-next-1",
		"gc-leaderboard-set-next-2",
	)
}

func TestGameCenterLeaderboardSetLocalizationsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboard-sets", "localizations", "list"},
		"game-center leaderboard-sets localizations list: --next",
	)
}

func TestGameCenterLeaderboardSetLocalizationsListPaginateFromNextWithoutSetID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboardSets/set-1/localizations?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboardSets/set-1/localizations?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardSetLocalizations","id":"gc-leaderboard-set-localization-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardSetLocalizations","id":"gc-leaderboard-set-localization-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboard-sets", "localizations", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-set-localization-next-1",
		"gc-leaderboard-set-localization-next-2",
	)
}

func TestGameCenterLeaderboardSetMemberLocalizationsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboard-sets", "member-localizations", "list"},
		"game-center leaderboard-sets member-localizations list: --next",
	)
}

func TestGameCenterLeaderboardSetMemberLocalizationsListPaginateFromNextWithoutSetOrLeaderboardID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboardSetMemberLocalizations?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboardSetMemberLocalizations?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardSetMemberLocalizations","id":"gc-leaderboard-set-member-localization-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardSetMemberLocalizations","id":"gc-leaderboard-set-member-localization-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboard-sets", "member-localizations", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-set-member-localization-next-1",
		"gc-leaderboard-set-member-localization-next-2",
	)
}

func TestGameCenterLeaderboardSetMembersListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboard-sets", "members", "list"},
		"game-center leaderboard-sets members list: --next",
	)
}

func TestGameCenterLeaderboardSetMembersListPaginateFromNextWithoutSetID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboardSets/set-1/gameCenterLeaderboards?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboardSets/set-1/gameCenterLeaderboards?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-leaderboard-set-member-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-leaderboard-set-member-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboard-sets", "members", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-set-member-next-1",
		"gc-leaderboard-set-member-next-2",
	)
}

func TestGameCenterLeaderboardSetReleasesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboard-sets", "releases", "list"},
		"game-center leaderboard-sets releases list: --next",
	)
}

func TestGameCenterLeaderboardSetReleasesListPaginateFromNextWithoutSetID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboardSets/set-1/releases?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterLeaderboardSets/set-1/releases?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardSetReleases","id":"gc-leaderboard-set-release-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardSetReleases","id":"gc-leaderboard-set-release-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboard-sets", "releases", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-set-release-next-1",
		"gc-leaderboard-set-release-next-2",
	)
}

func TestGameCenterLeaderboardSetsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboard-sets", "v2", "list"},
		"game-center leaderboard-sets v2 list: --next",
	)
}

func TestGameCenterLeaderboardSetsV2ListPaginateFromNextWithoutAppOrGroup(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboardSetsV2?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/detail-1/gameCenterLeaderboardSetsV2?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardSets","id":"gc-leaderboard-set-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardSets","id":"gc-leaderboard-set-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboard-sets", "v2", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-set-v2-next-1",
		"gc-leaderboard-set-v2-next-2",
	)
}

func TestGameCenterLeaderboardSetLocalizationsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboard-sets", "v2", "localizations", "list"},
		"game-center leaderboard-sets v2 localizations list: --next",
	)
}

func TestGameCenterLeaderboardSetLocalizationsV2ListPaginateFromNextWithoutVersionID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v2/gameCenterLeaderboardSetVersions/ver-1/localizations?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v2/gameCenterLeaderboardSetVersions/ver-1/localizations?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardSetLocalizations","id":"gc-leaderboard-set-localization-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardSetLocalizations","id":"gc-leaderboard-set-localization-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboard-sets", "v2", "localizations", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-set-localization-v2-next-1",
		"gc-leaderboard-set-localization-v2-next-2",
	)
}

func TestGameCenterLeaderboardSetMembersV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboard-sets", "v2", "members", "list"},
		"game-center leaderboard-sets v2 members list: --next",
	)
}

func TestGameCenterLeaderboardSetMembersV2ListPaginateFromNextWithoutSetID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v2/gameCenterLeaderboardSets/set-1/gameCenterLeaderboards?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v2/gameCenterLeaderboardSets/set-1/gameCenterLeaderboards?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-leaderboard-set-member-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-leaderboard-set-member-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboard-sets", "v2", "members", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-set-member-v2-next-1",
		"gc-leaderboard-set-member-v2-next-2",
	)
}

func TestGameCenterLeaderboardSetVersionsV2ListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "leaderboard-sets", "v2", "versions", "list"},
		"game-center leaderboard-sets v2 versions list: --next",
	)
}

func TestGameCenterLeaderboardSetVersionsV2ListPaginateFromNextWithoutSetID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v2/gameCenterLeaderboardSets/set-1/versions?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v2/gameCenterLeaderboardSets/set-1/versions?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardSetVersions","id":"gc-leaderboard-set-version-v2-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardSetVersions","id":"gc-leaderboard-set-version-v2-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "leaderboard-sets", "v2", "versions", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-leaderboard-set-version-v2-next-1",
		"gc-leaderboard-set-version-v2-next-2",
	)
}
