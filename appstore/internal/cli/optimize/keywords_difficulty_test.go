package optimize

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

var difficultyReferenceTime = time.Date(2026, time.August, 19, 0, 0, 0, 0, time.UTC)

func daysAgo(days int) string {
	return difficultyReferenceTime.Add(-time.Duration(days) * 24 * time.Hour).Format(time.RFC3339)
}

func assertClose(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s = %.10f, want %.10f", label, got, want)
	}
}

// TestScoreCompetitorAppMatchesPublishedParityVector pins the documented
// per-app scoring vector so the formula cannot drift silently.
func TestScoreCompetitorAppMatchesPublishedParityVector(t *testing.T) {
	signals := scoreCompetitorApp(competitorApp{
		AppID:                     "1",
		Name:                      "Timer for Focus",
		PublisherName:             "Example Labs",
		AverageUserRating:         4.5,
		UserRatingCount:           1000,
		CurrentVersionReleaseDate: daysAgo(30),
		ReleaseDate:               daysAgo(400),
	}, "focus timer", difficultyReferenceTime)

	if signals.KeywordMatch != keywordMatchTitleAllWords {
		t.Fatalf("KeywordMatch = %q, want %q", signals.KeywordMatch, keywordMatchTitleAllWords)
	}
	assertClose(t, "normalizedRatingCount", signals.NormalizedRatingCount, 0.1)
	assertClose(t, "normalizedAverageRating", signals.NormalizedAverageRating, 0.75)
	assertClose(t, "normalizedAge", signals.NormalizedAge, 0.9178082191780822)
	assertClose(t, "ratingsPerDay", signals.RatingsPerDay, 2.5)
	assertClose(t, "normalizedRatingsPerDay", signals.NormalizedRatingsPerDay, 0.26136363636363635)
	assertClose(t, "appScore", signals.AppScore, 0.5051899128268991)
}

// TestComputeKeywordDifficultyMatchesPublishedParityVector pins the documented
// keyword-level aggregation vector.
func TestComputeKeywordDifficultyMatchesPublishedParityVector(t *testing.T) {
	result := computeKeywordDifficulty([]float64{0.8, 0.7, 0.6, 0.5, 0.4}, 120)

	if result.Fallback {
		t.Fatal("Fallback = true, want false")
	}
	assertClose(t, "averageAppScore", result.AverageAppScore, 0.6)
	assertClose(t, "minimumAppScore", result.MinimumAppScore, 0.4)
	assertClose(t, "normalizedAppCount", result.NormalizedAppCount, 0.5789473684210527)
	assertClose(t, "difficulty", result.Difficulty, 47.53036437246964)
	assertClose(t, "minDifficulty", result.MinDifficulty, 40)
}

func TestScoreCompetitorAppNormalizesEachSignal(t *testing.T) {
	tests := []struct {
		name string
		app  competitorApp
		want func(*testing.T, asc.KeywordScoreSignals)
	}{
		{
			name: "rating count clamps at ten thousand",
			app:  competitorApp{UserRatingCount: 25000, ReleaseDate: daysAgo(1000), CurrentVersionReleaseDate: daysAgo(1)},
			want: func(t *testing.T, signals asc.KeywordScoreSignals) {
				assertClose(t, "normalizedRatingCount", signals.NormalizedRatingCount, 1)
			},
		},
		{
			name: "average rating at or below three scores zero",
			app:  competitorApp{AverageUserRating: 3, UserRatingCount: 5000},
			want: func(t *testing.T, signals asc.KeywordScoreSignals) {
				assertClose(t, "normalizedAverageRating", signals.NormalizedAverageRating, 0)
			},
		},
		{
			name: "average rating is damped by thin rating counts",
			app:  competitorApp{AverageUserRating: 5, UserRatingCount: 10},
			want: func(t *testing.T, signals asc.KeywordScoreSignals) {
				assertClose(t, "normalizedAverageRating", signals.NormalizedAverageRating, 0.5)
			},
		},
		{
			name: "average rating clamps above five",
			app:  competitorApp{AverageUserRating: 6, UserRatingCount: 100},
			want: func(t *testing.T, signals asc.KeywordScoreSignals) {
				assertClose(t, "normalizedAverageRating", signals.NormalizedAverageRating, 1)
			},
		},
		{
			name: "stale releases lose all age credit",
			app:  competitorApp{CurrentVersionReleaseDate: daysAgo(400)},
			want: func(t *testing.T, signals asc.KeywordScoreSignals) {
				assertClose(t, "normalizedAge", signals.NormalizedAge, 0)
			},
		},
		{
			name: "missing dates degrade to a one year window",
			app:  competitorApp{UserRatingCount: 365},
			want: func(t *testing.T, signals asc.KeywordScoreSignals) {
				assertClose(t, "daysSinceFirstRelease", signals.DaysSinceFirstRelease, 365)
				assertClose(t, "daysSinceLastRelease", signals.DaysSinceLastRelease, 365)
				assertClose(t, "normalizedAge", signals.NormalizedAge, 0)
				assertClose(t, "ratingsPerDay", signals.RatingsPerDay, 1)
				assertClose(t, "normalizedRatingsPerDay", signals.NormalizedRatingsPerDay, 0.25)
			},
		},
		{
			name: "unparseable dates degrade to a one year window",
			app:  competitorApp{ReleaseDate: "not-a-date", CurrentVersionReleaseDate: "also-not-a-date", UserRatingCount: 365},
			want: func(t *testing.T, signals asc.KeywordScoreSignals) {
				assertClose(t, "daysSinceFirstRelease", signals.DaysSinceFirstRelease, 365)
				assertClose(t, "ratingsPerDay", signals.RatingsPerDay, 1)
			},
		},
		{
			name: "same day releases floor the day count at one",
			app:  competitorApp{ReleaseDate: daysAgo(0), CurrentVersionReleaseDate: daysAgo(0), UserRatingCount: 50},
			want: func(t *testing.T, signals asc.KeywordScoreSignals) {
				assertClose(t, "daysSinceFirstRelease", signals.DaysSinceFirstRelease, 1)
				assertClose(t, "ratingsPerDay", signals.RatingsPerDay, 50)
				assertClose(t, "normalizedRatingsPerDay", signals.NormalizedRatingsPerDay, 0.25+0.75*(49.0/99.0))
			},
		},
		{
			name: "ratings per day clamps at one hundred",
			app:  competitorApp{ReleaseDate: daysAgo(1), CurrentVersionReleaseDate: daysAgo(1), UserRatingCount: 100000},
			want: func(t *testing.T, signals asc.KeywordScoreSignals) {
				assertClose(t, "normalizedRatingsPerDay", signals.NormalizedRatingsPerDay, 1)
			},
		},
		{
			name: "no ratings score zero ratings per day",
			app:  competitorApp{ReleaseDate: daysAgo(100), CurrentVersionReleaseDate: daysAgo(100)},
			want: func(t *testing.T, signals asc.KeywordScoreSignals) {
				assertClose(t, "ratingsPerDay", signals.RatingsPerDay, 0)
				assertClose(t, "normalizedRatingsPerDay", signals.NormalizedRatingsPerDay, 0)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.want(t, scoreCompetitorApp(test.app, "unmatched keyword", difficultyReferenceTime))
		})
	}
}

func TestScoreCompetitorAppJSONPreservesMissingReleaseDates(t *testing.T) {
	signals := scoreCompetitorApp(competitorApp{}, "focus timer", difficultyReferenceTime)

	encoded, err := json.Marshal(signals)
	if err != nil {
		t.Fatalf("marshal signals: %v", err)
	}
	for _, want := range []string{`"releaseDate":""`, `"currentVersionReleaseDate":""`} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("JSON missing %s: %s", want, encoded)
		}
	}
}

func TestComputeKeywordDifficultyEdges(t *testing.T) {
	t.Run("falls back below five apps", func(t *testing.T) {
		result := computeKeywordDifficulty([]float64{0.9, 0.9, 0.9, 0.9}, 4)
		if !result.Fallback {
			t.Fatal("Fallback = false, want true")
		}
		assertClose(t, "difficulty", result.Difficulty, 1)
		assertClose(t, "minDifficulty", result.MinDifficulty, 1)
		assertClose(t, "averageAppScore", result.AverageAppScore, 0.9)
		assertClose(t, "minimumAppScore", result.MinimumAppScore, 0.9)
		assertClose(t, "normalizedAppCount", result.NormalizedAppCount, 0)
	})

	t.Run("clamps difficulty to at least one", func(t *testing.T) {
		result := computeKeywordDifficulty([]float64{0, 0, 0, 0, 0}, 10)
		assertClose(t, "normalizedAppCount", result.NormalizedAppCount, 0)
		assertClose(t, "difficulty", result.Difficulty, 1)
		assertClose(t, "minDifficulty", result.MinDifficulty, 0)
	})

	t.Run("saturates at one hundred", func(t *testing.T) {
		result := computeKeywordDifficulty([]float64{1, 1, 1, 1, 1}, 500)
		assertClose(t, "normalizedAppCount", result.NormalizedAppCount, 1)
		assertClose(t, "difficulty", result.Difficulty, 100)
		assertClose(t, "minDifficulty", result.MinDifficulty, 100)
	})

	t.Run("normalizes app count between the bounds", func(t *testing.T) {
		result := computeKeywordDifficulty([]float64{0.5, 0.5, 0.5, 0.5, 0.5}, 200)
		assertClose(t, "normalizedAppCount", result.NormalizedAppCount, 1)
	})
}

func TestDetectKeywordMatchWalksTheLadderInOrder(t *testing.T) {
	tests := []struct {
		name     string
		keyword  string
		title    string
		subtitle string
		want     string
	}{
		{
			name:    "title exact phrase",
			keyword: "focus timer",
			title:   "Focus Timer Pro",
			want:    keywordMatchTitleExactPhrase,
		},
		{
			name:    "title all words",
			keyword: "focus timer",
			title:   "Timer for Focus",
			want:    keywordMatchTitleAllWords,
		},
		{
			name:     "subtitle exact phrase",
			keyword:  "focus timer",
			title:    "Pomodoro",
			subtitle: "A calm focus timer",
			want:     keywordMatchSubtitleExactPhrase,
		},
		{
			name:     "combined phrase across title and subtitle",
			keyword:  "focus timer",
			title:    "Deep Focus",
			subtitle: "Timer and streaks",
			want:     keywordMatchCombinedPhrase,
		},
		{
			name:     "subtitle all words",
			keyword:  "focus timer",
			title:    "Pomodoro",
			subtitle: "Timer for deep focus",
			want:     keywordMatchSubtitleAllWords,
		},
		{
			name:     "no match",
			keyword:  "focus timer",
			title:    "Grocery List",
			subtitle: "Shopping made simple",
			want:     keywordMatchNone,
		},
		{
			name:    "punctuation normalizes",
			keyword: "cafe timer",
			title:   "Cafe — Timer!",
			want:    keywordMatchTitleExactPhrase,
		},
		{
			name: "decomposed and precomposed accents compare equal",
			// A combining acute (U+0301) against a precomposed U+00E9.
			keyword: "cafe\u0301 timer",
			title:   "caf\u00e9 Timer",
			want:    keywordMatchTitleExactPhrase,
		},
		{
			name: "accents are preserved rather than folded",
			// NFKC composes encodings of the same character but never strips
			// marks, so an unaccented keyword does not match an accented title.
			keyword: "cafe timer",
			title:   "caf\u00e9 Timer",
			want:    keywordMatchNone,
		},
		{
			name:    "compatibility forms normalize",
			keyword: "focus timer",
			title:   "ＦＯＣＵＳ ＴＩＭＥＲ",
			want:    keywordMatchTitleExactPhrase,
		},
		{
			name:    "title exact phrase wins over subtitle evidence",
			keyword: "focus timer",
			title:   "Focus Timer",
			// A subtitle match must never downgrade a title phrase match.
			subtitle: "Focus timer",
			want:     keywordMatchTitleExactPhrase,
		},
		{
			name:    "title substring is not a phrase",
			keyword: "ai",
			title:   "Mail",
			want:    keywordMatchNone,
		},
		{
			name:     "subtitle substring is not a phrase",
			keyword:  "app",
			title:    "Pomodoro",
			subtitle: "Happy Notes",
			want:     keywordMatchNone,
		},
		{
			name:     "combined substring is not a phrase",
			keyword:  "ail notes",
			title:    "Mail",
			subtitle: "Notes",
			want:     keywordMatchNone,
		},
		{
			name:    "empty keyword never matches",
			keyword: "   ",
			title:   "Focus Timer",
			want:    keywordMatchNone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := detectKeywordMatch(test.keyword, test.title, test.subtitle)
			if got != test.want {
				t.Fatalf("detectKeywordMatch = %q, want %q", got, test.want)
			}
		})
	}
}

func TestKeywordMatchScoreLadder(t *testing.T) {
	want := map[string]float64{
		keywordMatchTitleExactPhrase:    1.0,
		keywordMatchTitleAllWords:       0.8,
		keywordMatchSubtitleExactPhrase: 0.5,
		keywordMatchCombinedPhrase:      0.4,
		keywordMatchSubtitleAllWords:    0.4,
		keywordMatchNone:                0,
	}
	for kind, score := range want {
		if got := keywordMatchScore(kind); got != score {
			t.Fatalf("keywordMatchScore(%q) = %v, want %v", kind, got, score)
		}
	}
}

func TestIsBrandKeywordUsesLeaderPublisherThenPeerScale(t *testing.T) {
	leader := func(publisher string, ratings int64) competitorApp {
		return competitorApp{AppID: "1", Name: "Leader", PublisherName: publisher, UserRatingCount: ratings}
	}
	peer := func(id, publisher string, ratings int64) competitorApp {
		return competitorApp{AppID: id, Name: "Peer " + id, PublisherName: publisher, UserRatingCount: ratings}
	}

	tests := []struct {
		name    string
		keyword string
		apps    []competitorApp
		want    bool
	}{
		{
			name:    "keyword token missing from leader publisher",
			keyword: "focus timer",
			apps:    []competitorApp{leader("Focus Labs", 100000)},
			want:    false,
		},
		{
			name:    "leader publisher covers the keyword and carries scale",
			keyword: "duolingo",
			apps:    []competitorApp{leader("Duolingo", 1000)},
			want:    true,
		},
		{
			name:    "small leader with large independent peers",
			keyword: "acme notes",
			apps: []competitorApp{
				leader("Acme Notes Inc", 999),
				peer("2", "Other Co", 10000),
				peer("3", "Third Co", 20000),
			},
			want: true,
		},
		{
			name:    "small leader with small independent peers",
			keyword: "acme notes",
			apps: []competitorApp{
				leader("Acme Notes Inc", 999),
				peer("2", "Other Co", 100),
				peer("3", "Third Co", 200),
			},
			want: false,
		},
		{
			name:    "small leader with no independent peers",
			keyword: "acme notes",
			apps: []competitorApp{
				leader("Acme Notes Inc", 999),
				peer("2", "Acme Notes Inc", 500000),
			},
			want: false,
		},
		{
			name:    "no apps",
			keyword: "acme notes",
			apps:    nil,
			want:    false,
		},
		{
			name:    "empty keyword",
			keyword: "  ",
			apps:    []competitorApp{leader("Acme", 100000)},
			want:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isBrandKeyword(test.keyword, test.apps); got != test.want {
				t.Fatalf("isBrandKeyword = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIsBrandKeywordUsesTheMedianOfIndependentPeers(t *testing.T) {
	apps := []competitorApp{
		{AppID: "1", PublisherName: "Acme Notes Inc", UserRatingCount: 10},
		{AppID: "2", PublisherName: "Other Co", UserRatingCount: 1},
		{AppID: "3", PublisherName: "Third Co", UserRatingCount: 9000},
		{AppID: "4", PublisherName: "Fourth Co", UserRatingCount: 11000},
		{AppID: "5", PublisherName: "Fifth Co", UserRatingCount: 1000000},
	}
	// Independent peers are 1, 9000, 11000, 1000000; the median is 10000.
	if !isBrandKeyword("acme notes", apps) {
		t.Fatal("isBrandKeyword = false, want true at the median boundary")
	}

	apps[3].UserRatingCount = 10998
	// Independent peers become 1, 9000, 10998, 1000000 for a median of 9999.
	if isBrandKeyword("acme notes", apps) {
		t.Fatal("isBrandKeyword = true, want false just below the median boundary")
	}
}
