package asc

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestSubscriptionPlanAvailabilitySetSummaryRows(t *testing.T) {
	availableInNew := true
	result := &SubscriptionPlanAvailabilitySetResult{
		SubscriptionID:            "sub-1",
		PlanAvailabilityID:        "plan-1",
		PlanType:                  "UPFRONT",
		Changed:                   true,
		AvailableInNewTerritories: &availableInNew,
		AddedTerritories:          []string{"CAN"},
		RemovedTerritories:        []string{"DEU"},
		UnchangedTerritories:      []string{"USA"},
		AvailableTerritories:      []string{"CAN", "USA"},
	}

	headers, rows := subscriptionPlanAvailabilitySetSummaryRows(result)
	if len(headers) != 10 || len(rows) != 1 || len(rows[0]) != len(headers) {
		t.Fatalf("unexpected summary shape: headers=%v rows=%v", headers, rows)
	}
	if rows[0][3] != "true" || rows[0][4] != "false" || rows[0][5] != "true" {
		t.Fatalf("unexpected changed/created/availableInNewTerritories cells: %v", rows[0])
	}
	if rows[0][9] != "2" {
		t.Fatalf("expected 2 available territories, got %q", rows[0][9])
	}
}

func TestSubscriptionPlanAvailabilitySetTerritoryRowsSkipEmptyGroups(t *testing.T) {
	result := &SubscriptionPlanAvailabilitySetResult{
		AddedTerritories:    []string{"CAN"},
		ExcludedTerritories: []string{"USA"},
	}

	_, rows := subscriptionPlanAvailabilitySetTerritoryRows(result)
	if len(rows) != 2 {
		t.Fatalf("expected only the non-empty groups, got %v", rows)
	}
	if rows[0][0] != "added" || rows[1][0] != "excluded" {
		t.Fatalf("unexpected group ordering: %v", rows)
	}
}

func TestFormatSubscriptionPlanAvailabilityTerritoryCellSummarizesLongLists(t *testing.T) {
	territories := make([]string, 0, 25)
	for i := 0; i < 25; i++ {
		territories = append(territories, "T"+strings.Repeat("X", i%3))
	}

	cell := formatSubscriptionPlanAvailabilityTerritoryCell(territories)
	if !strings.HasSuffix(cell, "(+5 more)") {
		t.Fatalf("expected the remainder to be summarized, got %q", cell)
	}
	if got := strings.Count(cell, ","); got != 19 {
		t.Fatalf("expected 20 listed territories, got %d separators in %q", got, cell)
	}
	if short := formatSubscriptionPlanAvailabilityTerritoryCell([]string{"USA", "CAN"}); short != "USA,CAN" {
		t.Fatalf("expected short lists to render verbatim, got %q", short)
	}
}

func TestSubscriptionPlanAvailabilitiesRowsIncludesTerritories(t *testing.T) {
	availableInNew := true
	resp := &SubscriptionPlanAvailabilitiesResponse{
		Data: []Resource[SubscriptionPlanAvailabilityAttributes]{{
			ID: "plan-upfront",
			Attributes: SubscriptionPlanAvailabilityAttributes{
				PlanType:                  SubscriptionPlanTypeUpfront,
				AvailableInNewTerritories: &availableInNew,
			},
			Relationships: json.RawMessage(`{"availableTerritories":{"data":[{"type":"territories","id":"USA"}],"meta":{"paging":{"total":175,"limit":50}}}}`),
		}},
	}

	headers, rows := subscriptionPlanAvailabilityShowRows(resp)
	if len(headers) != 4 || headers[3] != "Territories" {
		t.Fatalf("expected a Territories column, got %v", headers)
	}
	if len(rows) != 1 || !strings.Contains(rows[0][3], "USA") || !strings.Contains(rows[0][3], "+174 more") {
		t.Fatalf("expected included territories plus the paging remainder, got %v", rows)
	}
	if strings.Count(rows[0][3], "more") != 1 {
		t.Fatalf("expected a single remainder suffix, got %q", rows[0][3])
	}
}

func TestSubscriptionPlanAvailabilitiesRowsOmitTerritoriesColumn(t *testing.T) {
	resp := &SubscriptionPlanAvailabilitiesResponse{
		Data: []Resource[SubscriptionPlanAvailabilityAttributes]{{
			ID: "plan-monthly",
			Attributes: SubscriptionPlanAvailabilityAttributes{
				PlanType: SubscriptionPlanTypeMonthly,
			},
		}},
	}

	headers, rows := subscriptionPlanAvailabilitiesRows(resp)
	if len(headers) != 3 {
		t.Fatalf("expected the shared list renderer to keep three columns, got %v", headers)
	}
	for _, header := range headers {
		if header == "Territories" {
			t.Fatalf("shared list renderer should not expose Territories, got %v", headers)
		}
	}
	if len(rows) != 1 || rows[0][0] != "plan-monthly" {
		t.Fatalf("unexpected list rows: %v", rows)
	}
}

func TestFormatPlanAvailabilityShowTerritoryCellUsesOneSuffix(t *testing.T) {
	ids := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		ids = append(ids, fmt.Sprintf("T%02d", i))
	}

	cell := formatPlanAvailabilityShowTerritoryCell(ids, 175, true)
	if strings.Count(cell, "more") != 1 {
		t.Fatalf("expected one remainder suffix, got %q", cell)
	}
	if !strings.HasSuffix(cell, "(+155 more)") {
		t.Fatalf("expected 175-20 omitted territories, got %q", cell)
	}
}
