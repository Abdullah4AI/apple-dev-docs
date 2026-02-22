package asc

// GameCenterDetailAttributes represents a Game Center detail resource.
type GameCenterDetailAttributes struct {
	ArcadeEnabled                      bool `json:"arcadeEnabled,omitempty"`
	ChallengeEnabled                   bool `json:"challengeEnabled,omitempty"`
	LeaderboardSetEnabled              bool `json:"leaderboardSetEnabled,omitempty"`
	LeaderboardEnabled                 bool `json:"leaderboardEnabled,omitempty"`
	AchievementEnabled                 bool `json:"achievementEnabled,omitempty"`
	MultiplayerSessionEnabled          bool `json:"multiplayerSessionEnabled,omitempty"`
	MultiplayerTurnBasedSessionEnabled bool `json:"multiplayerTurnBasedSessionEnabled,omitempty"`
}

// GameCenterDetailResponse is the response from Game Center detail endpoints.
type GameCenterDetailResponse = SingleResponse[GameCenterDetailAttributes]

// Valid leaderboard formatters.
var ValidLeaderboardFormatters = []string{
	"INTEGER",
	"DECIMAL_POINT_1_PLACE",
	"DECIMAL_POINT_2_PLACE",
	"DECIMAL_POINT_3_PLACE",
	"ELAPSED_TIME_MILLISECOND",
	"ELAPSED_TIME_SECOND",
	"ELAPSED_TIME_MINUTE",
	"MONEY_WHOLE",
	"MONEY_POINT_2_PLACE",
}

// Valid leaderboard score sort types.
var ValidScoreSortTypes = []string{
	"ASC",
	"DESC",
}

// Valid leaderboard submission types.
var ValidSubmissionTypes = []string{
	"BEST_SCORE",
	"MOST_RECENT_SCORE",
}

// GameCenterVersionState represents the state of a Game Center version.
type GameCenterVersionState string

const (
	GameCenterVersionStatePrepareForSubmission GameCenterVersionState = "PREPARE_FOR_SUBMISSION"
	GameCenterVersionStateReadyForReview       GameCenterVersionState = "READY_FOR_REVIEW"
	GameCenterVersionStateWaitingForReview     GameCenterVersionState = "WAITING_FOR_REVIEW"
	GameCenterVersionStateInReview             GameCenterVersionState = "IN_REVIEW"
	GameCenterVersionStateDeveloperRejected    GameCenterVersionState = "DEVELOPER_REJECTED"
	GameCenterVersionStateRejected             GameCenterVersionState = "REJECTED"
	GameCenterVersionStateAccepted             GameCenterVersionState = "ACCEPTED"
	GameCenterVersionStatePendingRelease       GameCenterVersionState = "PENDING_RELEASE"
	GameCenterVersionStateLive                 GameCenterVersionState = "LIVE"
	GameCenterVersionStateReplacedWithNew      GameCenterVersionState = "REPLACED_WITH_NEW_VERSION"
)
