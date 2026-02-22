package cmdtest

import "testing"

func TestGameCenterGroupsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "groups", "list"},
		"game-center groups list: --next",
	)
}

func TestGameCenterGroupsListPaginateFromNextWithoutApp(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterGroups","id":"gc-group-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterGroups","id":"gc-group-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "groups", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-group-next-1",
		"gc-group-next-2",
	)
}

func TestGameCenterGroupAchievementsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "groups", "achievements", "list"},
		"game-center groups achievements list: --next",
	)
}

func TestGameCenterGroupAchievementsListPaginateFromNextWithoutGroupID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterAchievements?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterAchievements?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterAchievements","id":"gc-group-achievement-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterAchievements","id":"gc-group-achievement-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "groups", "achievements", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-group-achievement-next-1",
		"gc-group-achievement-next-2",
	)
}

func TestGameCenterGroupLeaderboardsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "groups", "leaderboards", "list"},
		"game-center groups leaderboards list: --next",
	)
}

func TestGameCenterGroupLeaderboardsListPaginateFromNextWithoutGroupID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterLeaderboards?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterLeaderboards?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-group-leaderboard-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboards","id":"gc-group-leaderboard-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "groups", "leaderboards", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-group-leaderboard-next-1",
		"gc-group-leaderboard-next-2",
	)
}

func TestGameCenterGroupLeaderboardSetsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "groups", "leaderboard-sets", "list"},
		"game-center groups leaderboard-sets list: --next",
	)
}

func TestGameCenterGroupLeaderboardSetsListPaginateFromNextWithoutGroupID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterLeaderboardSets?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterLeaderboardSets?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterLeaderboardSets","id":"gc-group-leaderboard-set-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterLeaderboardSets","id":"gc-group-leaderboard-set-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "groups", "leaderboard-sets", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-group-leaderboard-set-next-1",
		"gc-group-leaderboard-set-next-2",
	)
}

func TestGameCenterGroupActivitiesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "groups", "activities", "list"},
		"game-center groups activities list: --next",
	)
}

func TestGameCenterGroupActivitiesListPaginateFromNextWithoutGroupID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterActivities?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterActivities?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterActivities","id":"gc-group-activity-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterActivities","id":"gc-group-activity-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "groups", "activities", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-group-activity-next-1",
		"gc-group-activity-next-2",
	)
}

func TestGameCenterGroupChallengesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "groups", "challenges", "list"},
		"game-center groups challenges list: --next",
	)
}

func TestGameCenterGroupChallengesListPaginateFromNextWithoutGroupID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterChallenges?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterChallenges?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterChallenges","id":"gc-group-challenge-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterChallenges","id":"gc-group-challenge-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "groups", "challenges", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-group-challenge-next-1",
		"gc-group-challenge-next-2",
	)
}

func TestGameCenterGroupDetailsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "groups", "details", "list"},
		"game-center groups details list: --next",
	)
}

func TestGameCenterGroupDetailsListPaginateFromNextWithoutGroupID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterDetails?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterGroups/group-1/gameCenterDetails?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterDetails","id":"gc-group-detail-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterDetails","id":"gc-group-detail-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "groups", "details", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-group-detail-next-1",
		"gc-group-detail-next-2",
	)
}
