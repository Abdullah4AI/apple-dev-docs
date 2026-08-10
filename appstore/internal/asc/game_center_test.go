package asc

import (
	"slices"
	"testing"
)

func TestValidLeaderboardFormattersMatchOpenAPI(t *testing.T) {
	want := []string{
		"INTEGER",
		"DECIMAL_POINT_1_PLACE",
		"DECIMAL_POINT_2_PLACE",
		"DECIMAL_POINT_3_PLACE",
		"ELAPSED_TIME_CENTISECOND",
		"ELAPSED_TIME_MINUTE",
		"ELAPSED_TIME_SECOND",
		"MONEY_POUND_DECIMAL",
		"MONEY_POUND",
		"MONEY_DOLLAR_DECIMAL",
		"MONEY_DOLLAR",
		"MONEY_EURO_DECIMAL",
		"MONEY_EURO",
		"MONEY_FRANC_DECIMAL",
		"MONEY_FRANC",
		"MONEY_KRONER_DECIMAL",
		"MONEY_KRONER",
		"MONEY_YEN",
	}

	if !slices.Equal(ValidLeaderboardFormatters, want) {
		t.Fatalf("ValidLeaderboardFormatters = %v, want %v", ValidLeaderboardFormatters, want)
	}
}
