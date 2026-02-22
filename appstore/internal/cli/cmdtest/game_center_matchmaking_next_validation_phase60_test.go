package cmdtest

import "testing"

func TestGameCenterMatchmakingQueuesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "matchmaking", "queues", "list"},
		"game-center matchmaking queues list: --next",
	)
}

func TestGameCenterMatchmakingQueuesListPaginateFromNext(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterMatchmakingQueues?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterMatchmakingQueues?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterMatchmakingQueues","id":"gc-matchmaking-queue-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterMatchmakingQueues","id":"gc-matchmaking-queue-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "matchmaking", "queues", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-matchmaking-queue-next-1",
		"gc-matchmaking-queue-next-2",
	)
}

func TestGameCenterMatchmakingRuleSetsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "matchmaking", "rule-sets", "list"},
		"game-center matchmaking rule-sets list: --next",
	)
}

func TestGameCenterMatchmakingRuleSetsListPaginateFromNext(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterMatchmakingRuleSets?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterMatchmakingRuleSets?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterMatchmakingRuleSets","id":"gc-matchmaking-rule-set-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterMatchmakingRuleSets","id":"gc-matchmaking-rule-set-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "matchmaking", "rule-sets", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-matchmaking-rule-set-next-1",
		"gc-matchmaking-rule-set-next-2",
	)
}

func TestGameCenterMatchmakingRuleSetQueuesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "matchmaking", "rule-sets", "queues", "list"},
		"game-center matchmaking rule-sets queues list: --next",
	)
}

func TestGameCenterMatchmakingRuleSetQueuesListPaginateFromNextWithoutRuleSetID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterMatchmakingRuleSets/rule-set-1/matchmakingQueues?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterMatchmakingRuleSets/rule-set-1/matchmakingQueues?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterMatchmakingQueues","id":"gc-matchmaking-rule-set-queue-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterMatchmakingQueues","id":"gc-matchmaking-rule-set-queue-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "matchmaking", "rule-sets", "queues", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-matchmaking-rule-set-queue-next-1",
		"gc-matchmaking-rule-set-queue-next-2",
	)
}

func TestGameCenterMatchmakingRulesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "matchmaking", "rules", "list"},
		"game-center matchmaking rules list: --next",
	)
}

func TestGameCenterMatchmakingRulesListPaginateFromNextWithoutRuleSetID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterMatchmakingRuleSets/rule-set-1/rules?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterMatchmakingRuleSets/rule-set-1/rules?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterMatchmakingRules","id":"gc-matchmaking-rule-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterMatchmakingRules","id":"gc-matchmaking-rule-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "matchmaking", "rules", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-matchmaking-rule-next-1",
		"gc-matchmaking-rule-next-2",
	)
}

func TestGameCenterMatchmakingTeamsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "matchmaking", "teams", "list"},
		"game-center matchmaking teams list: --next",
	)
}

func TestGameCenterMatchmakingTeamsListPaginateFromNextWithoutRuleSetID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterMatchmakingRuleSets/rule-set-1/teams?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterMatchmakingRuleSets/rule-set-1/teams?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterMatchmakingTeams","id":"gc-matchmaking-team-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterMatchmakingTeams","id":"gc-matchmaking-team-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "matchmaking", "teams", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-matchmaking-team-next-1",
		"gc-matchmaking-team-next-2",
	)
}
