package asc

import (
	"encoding/json"
	"fmt"
)

// SubscriptionGroupDeleteResult represents CLI output for group deletions.
type SubscriptionGroupDeleteResult struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// SubscriptionDeleteResult represents CLI output for subscription deletions.
type SubscriptionDeleteResult struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// SubscriptionPriceDeleteResult represents CLI output for subscription price deletions.
type SubscriptionPriceDeleteResult struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// SubscriptionIntroductoryOfferCreateSummary describes a single- or
// all-territories introductory-offer create operation.
type SubscriptionIntroductoryOfferCreateSummary struct {
	SubscriptionID  string                                              `json:"subscriptionId"`
	AvailabilityID  string                                              `json:"availabilityId,omitempty"`
	Territory       string                                              `json:"territory,omitempty"`
	AllTerritories  bool                                                `json:"allTerritories"`
	DryRun          bool                                                `json:"dryRun"`
	ContinueOnError bool                                                `json:"continueOnError"`
	Total           int                                                 `json:"total"`
	Created         int                                                 `json:"created"`
	Skipped         int                                                 `json:"skipped"`
	Failed          int                                                 `json:"failed"`
	Skips           []SubscriptionIntroductoryOfferCreateSummarySkip    `json:"skips,omitempty"`
	Failures        []SubscriptionIntroductoryOfferCreateSummaryFailure `json:"failures,omitempty"`
}

// SubscriptionIntroductoryOfferCreateSummarySkip records an existing offer
// that was not recreated in an all-territories operation.
type SubscriptionIntroductoryOfferCreateSummarySkip struct {
	Territory string `json:"territory"`
	Reason    string `json:"reason"`
}

// SubscriptionIntroductoryOfferCreateSummaryFailure records a territory that
// could not be processed.
type SubscriptionIntroductoryOfferCreateSummaryFailure struct {
	Territory string `json:"territory"`
	Error     string `json:"error"`
}

func subscriptionGroupsRows(resp *SubscriptionGroupsResponse) ([]string, [][]string) {
	headers := []string{"ID", "Reference Name"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{
			item.ID,
			compactWhitespace(item.Attributes.ReferenceName),
		})
	}
	return headers, rows
}

func subscriptionGroupVersionsRows(resp *SubscriptionGroupVersionsResponse) ([]string, [][]string) {
	headers := []string{"ID", "Version", "State"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{
			item.ID,
			fmt.Sprintf("%d", item.Attributes.Version),
			item.Attributes.State,
		})
	}
	return headers, rows
}

func subscriptionGroupLocalizationsV2Rows(resp *SubscriptionGroupLocalizationsV2Response) ([]string, [][]string) {
	headers := []string{"ID", "Locale", "Name", "Custom App Name"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{
			item.ID,
			item.Attributes.Locale,
			compactWhitespace(item.Attributes.Name),
			compactWhitespace(item.Attributes.CustomAppName),
		})
	}
	return headers, rows
}

func subscriptionsRows(resp *SubscriptionsResponse) ([]string, [][]string) {
	headers := []string{"ID", "Name", "Product ID", "Period", "State"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{
			item.ID,
			compactWhitespace(item.Attributes.Name),
			item.Attributes.ProductID,
			item.Attributes.SubscriptionPeriod,
			item.Attributes.State,
		})
	}
	return headers, rows
}

func subscriptionVersionsRows(resp *SubscriptionVersionsResponse) ([]string, [][]string) {
	headers := []string{"ID", "Version", "State"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{item.ID, fmt.Sprintf("%d", item.Attributes.Version), string(item.Attributes.State)})
	}
	return headers, rows
}

func subscriptionLocalizationsV2Rows(resp *SubscriptionLocalizationsV2Response) ([]string, [][]string) {
	headers := []string{"ID", "Locale", "Name", "Description"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		rows = append(rows, []string{
			item.ID,
			item.Attributes.Locale,
			compactWhitespace(item.Attributes.Name),
			compactWhitespace(item.Attributes.Description),
		})
	}
	return headers, rows
}

func subscriptionImagesV2Rows(resp *SubscriptionImagesV2Response) ([]string, [][]string) {
	headers := []string{"ID", "File Name", "File Size", "State"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		state := ""
		if item.Attributes.AssetDeliveryState != nil && item.Attributes.AssetDeliveryState.State != nil {
			state = *item.Attributes.AssetDeliveryState.State
		}
		rows = append(rows, []string{item.ID, item.Attributes.FileName, fmt.Sprintf("%d", item.Attributes.FileSize), state})
	}
	return headers, rows
}

func subscriptionPriceRows(resp *SubscriptionPriceResponse) ([]string, [][]string) {
	headers := []string{"ID", "Start Date", "Preserved"}
	rows := [][]string{{
		resp.Data.ID,
		resp.Data.Attributes.StartDate,
		fmt.Sprintf("%t", resp.Data.Attributes.Preserved),
	}}
	return headers, rows
}

func subscriptionPricesRows(resp *SubscriptionPricesResponse) ([]string, [][]string, error) {
	headers := []string{"ID", "Territory", "Price Point", "Start Date", "Preserved"}
	rows := make([][]string, 0, len(resp.Data))
	for _, item := range resp.Data {
		territoryID, pricePointID, err := subscriptionPriceRelationshipIDs(item.Relationships)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, []string{
			item.ID,
			territoryID,
			pricePointID,
			item.Attributes.StartDate,
			fmt.Sprintf("%t", item.Attributes.Preserved),
		})
	}
	return headers, rows, nil
}

func subscriptionAvailabilityRows(resp *SubscriptionAvailabilityResponse) ([]string, [][]string) {
	headers := []string{"ID", "Available In New Territories"}
	rows := [][]string{{
		resp.Data.ID,
		fmt.Sprintf("%t", resp.Data.Attributes.AvailableInNewTerritories),
	}}
	return headers, rows
}

func subscriptionGracePeriodRows(resp *SubscriptionGracePeriodResponse) ([]string, [][]string) {
	headers := []string{"ID", "Opt In", "Sandbox Opt In", "Duration", "Renewal Type"}
	rows := [][]string{{
		resp.Data.ID,
		fmt.Sprintf("%t", resp.Data.Attributes.OptIn),
		fmt.Sprintf("%t", resp.Data.Attributes.SandboxOptIn),
		resp.Data.Attributes.Duration,
		resp.Data.Attributes.RenewalType,
	}}
	return headers, rows
}

func subscriptionGroupDeleteResultRows(result *SubscriptionGroupDeleteResult) ([]string, [][]string) {
	headers := []string{"ID", "Deleted"}
	rows := [][]string{{result.ID, fmt.Sprintf("%t", result.Deleted)}}
	return headers, rows
}

func subscriptionDeleteResultRows(result *SubscriptionDeleteResult) ([]string, [][]string) {
	headers := []string{"ID", "Deleted"}
	rows := [][]string{{result.ID, fmt.Sprintf("%t", result.Deleted)}}
	return headers, rows
}

func subscriptionPriceDeleteResultRows(result *SubscriptionPriceDeleteResult) ([]string, [][]string) {
	headers := []string{"ID", "Deleted"}
	rows := [][]string{{result.ID, fmt.Sprintf("%t", result.Deleted)}}
	return headers, rows
}

func subscriptionIntroductoryOfferCreateSummaryRows(summary *SubscriptionIntroductoryOfferCreateSummary) ([]string, [][]string) {
	headers := []string{"Subscription ID"}
	row := []string{summary.SubscriptionID}
	if summary.AvailabilityID != "" {
		headers = append(headers, "Availability ID")
		row = append(row, summary.AvailabilityID)
	}
	if summary.Territory != "" {
		headers = append(headers, "Territory")
		row = append(row, summary.Territory)
	}
	headers = append(headers, "Dry Run", "Total", "Created", "Skipped", "Failed")
	row = append(
		row,
		fmt.Sprintf("%t", summary.DryRun),
		fmt.Sprintf("%d", summary.Total),
		fmt.Sprintf("%d", summary.Created),
		fmt.Sprintf("%d", summary.Skipped),
		fmt.Sprintf("%d", summary.Failed),
	)
	return headers, [][]string{row}
}

func subscriptionIntroductoryOfferCreateSummarySkipRows(summary *SubscriptionIntroductoryOfferCreateSummary) ([]string, [][]string) {
	rows := make([][]string, 0, len(summary.Skips))
	for _, skip := range summary.Skips {
		rows = append(rows, []string{skip.Territory, skip.Reason})
	}
	return []string{"Skipped Territory", "Reason"}, rows
}

func subscriptionIntroductoryOfferCreateSummaryFailureRows(summary *SubscriptionIntroductoryOfferCreateSummary) ([]string, [][]string) {
	rows := make([][]string, 0, len(summary.Failures))
	for _, failure := range summary.Failures {
		rows = append(rows, []string{failure.Territory, failure.Error})
	}
	return []string{"Failed Territory", "Error"}, rows
}

func subscriptionPriceRelationshipIDs(raw json.RawMessage) (string, string, error) {
	if len(raw) == 0 {
		return "", "", nil
	}
	var relationships SubscriptionPriceRelationships
	if err := json.Unmarshal(raw, &relationships); err != nil {
		return "", "", fmt.Errorf("decode subscription price relationships: %w", err)
	}
	territoryID := ""
	pricePointID := ""
	if relationships.Territory != nil {
		territoryID = relationships.Territory.Data.ID
	}
	if relationships.SubscriptionPricePoint != nil {
		pricePointID = relationships.SubscriptionPricePoint.Data.ID
	}
	return territoryID, pricePointID, nil
}
