package asc

import (
	"fmt"
	"strings"
)

// SubscriptionPlanAvailabilitySetResult is the stable computed output contract
// for subscriptions pricing plan-availability set.
type SubscriptionPlanAvailabilitySetResult struct {
	SubscriptionID            string   `json:"subscriptionId"`
	PlanAvailabilityID        string   `json:"planAvailabilityId"`
	PlanType                  string   `json:"planType"`
	Changed                   bool     `json:"changed"`
	Created                   bool     `json:"created"`
	AvailableInNewTerritories *bool    `json:"availableInNewTerritories,omitempty"`
	AddedTerritories          []string `json:"addedTerritories"`
	RemovedTerritories        []string `json:"removedTerritories"`
	UnchangedTerritories      []string `json:"unchangedTerritories"`
	ExcludedTerritories       []string `json:"excludedTerritories,omitempty"`
	AvailableTerritories      []string `json:"availableTerritories"`
}

func subscriptionPlanAvailabilitySetSummaryRows(result *SubscriptionPlanAvailabilitySetResult) ([]string, [][]string) {
	headers := []string{
		"Subscription", "Plan Availability", "Plan Type", "Changed", "Created",
		"Available In New Territories", "Added", "Removed", "Unchanged", "Territories",
	}
	rows := [][]string{{
		result.SubscriptionID,
		result.PlanAvailabilityID,
		result.PlanType,
		fmt.Sprintf("%t", result.Changed),
		fmt.Sprintf("%t", result.Created),
		formatOptionalSubscriptionBool(result.AvailableInNewTerritories),
		fmt.Sprintf("%d", len(result.AddedTerritories)),
		fmt.Sprintf("%d", len(result.RemovedTerritories)),
		fmt.Sprintf("%d", len(result.UnchangedTerritories)),
		fmt.Sprintf("%d", len(result.AvailableTerritories)),
	}}
	return headers, rows
}

func subscriptionPlanAvailabilitySetTerritoryRows(result *SubscriptionPlanAvailabilitySetResult) ([]string, [][]string) {
	headers := []string{"Action", "Territories"}
	rows := make([][]string, 0, 4)
	for _, group := range []struct {
		action string
		values []string
	}{
		{action: "added", values: result.AddedTerritories},
		{action: "removed", values: result.RemovedTerritories},
		{action: "unchanged", values: result.UnchangedTerritories},
		{action: "excluded", values: result.ExcludedTerritories},
	} {
		if len(group.values) == 0 {
			continue
		}
		rows = append(rows, []string{group.action, formatSubscriptionPlanAvailabilityTerritoryCell(group.values)})
	}
	return headers, rows
}

// subscriptionPlanAvailabilityTerritoryCellLimit caps how many territory IDs a
// table or Markdown cell lists before summarizing the remainder. JSON output
// always carries the complete lists.
const subscriptionPlanAvailabilityTerritoryCellLimit = 20

func formatSubscriptionPlanAvailabilityTerritoryCell(territoryIDs []string) string {
	if len(territoryIDs) <= subscriptionPlanAvailabilityTerritoryCellLimit {
		return strings.Join(territoryIDs, ",")
	}
	shown := territoryIDs[:subscriptionPlanAvailabilityTerritoryCellLimit]
	return fmt.Sprintf(
		"%s (+%d more)",
		strings.Join(shown, ","),
		len(territoryIDs)-subscriptionPlanAvailabilityTerritoryCellLimit,
	)
}
