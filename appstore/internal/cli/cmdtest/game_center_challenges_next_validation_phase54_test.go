package cmdtest

import "testing"

func TestGameCenterChallengesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "challenges", "list"},
		"game-center challenges list: --next",
	)
}

func TestGameCenterChallengesListPaginateFromNextWithoutApp(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/gc-detail-1/gameCenterChallenges?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/gc-detail-1/gameCenterChallenges?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterChallenges","id":"gc-challenge-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterChallenges","id":"gc-challenge-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "challenges", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-challenge-next-1",
		"gc-challenge-next-2",
	)
}

func TestGameCenterChallengeVersionsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "challenges", "versions", "list"},
		"game-center challenges versions list: --next",
	)
}

func TestGameCenterChallengeVersionsListPaginateFromNextWithoutChallengeID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterChallenges/challenge-1/versions?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterChallenges/challenge-1/versions?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterChallengeVersions","id":"gc-challenge-version-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterChallengeVersions","id":"gc-challenge-version-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "challenges", "versions", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-challenge-version-next-1",
		"gc-challenge-version-next-2",
	)
}

func TestGameCenterChallengeLocalizationsListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "challenges", "localizations", "list"},
		"game-center challenges localizations list: --next",
	)
}

func TestGameCenterChallengeLocalizationsListPaginateFromNextWithoutVersionID(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterChallengeVersions/version-1/localizations?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterChallengeVersions/version-1/localizations?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterChallengeLocalizations","id":"gc-challenge-localization-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterChallengeLocalizations","id":"gc-challenge-localization-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "challenges", "localizations", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-challenge-localization-next-1",
		"gc-challenge-localization-next-2",
	)
}

func TestGameCenterChallengeReleasesListRejectsInvalidNextURL(t *testing.T) {
	runGameCenterAchievementsInvalidNextURLCases(
		t,
		[]string{"game-center", "challenges", "releases", "list"},
		"game-center challenges releases list: --next",
	)
}

func TestGameCenterChallengeReleasesListPaginateFromNextWithoutApp(t *testing.T) {
	const firstURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/gc-detail-1/challengeReleases?cursor=AQ&limit=200"
	const secondURL = "https://api.appstoreconnect.apple.com/v1/gameCenterDetails/gc-detail-1/challengeReleases?cursor=BQ&limit=200"

	firstBody := `{"data":[{"type":"gameCenterChallengeVersionReleases","id":"gc-challenge-release-next-1"}],"links":{"next":"` + secondURL + `"}}`
	secondBody := `{"data":[{"type":"gameCenterChallengeVersionReleases","id":"gc-challenge-release-next-2"}],"links":{"next":""}}`

	runGameCenterAchievementsPaginateFromNext(
		t,
		[]string{"game-center", "challenges", "releases", "list"},
		firstURL,
		secondURL,
		firstBody,
		secondBody,
		"gc-challenge-release-next-1",
		"gc-challenge-release-next-2",
	)
}
