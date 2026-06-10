package asc

import "fmt"

func subscriptionPlanAvailabilitiesRows(resp *SubscriptionPlanAvailabilitiesResponse) ([]string, [][]string) {
	headers := []string{"ID", "Plan Type", "Available In New Territories"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{
			item.ID,
			string(item.Attributes.PlanType),
			formatOptionalSubscriptionBool(item.Attributes.AvailableInNewTerritories),
		})
	}
	return headers, rows
}

func formatOptionalSubscriptionBool(value *bool) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%t", *value)
}
