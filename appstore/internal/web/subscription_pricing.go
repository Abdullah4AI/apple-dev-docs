package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SubscriptionPlanPricesResult identifies the paired billing-plan prices created.
type SubscriptionPlanPricesResult struct {
	SubscriptionID      string `json:"subscriptionId"`
	UpfrontPricePointID string `json:"upfrontPricePointId"`
	MonthlyPricePointID string `json:"monthlyPricePointId"`
}

// SubscriptionPrice is one applied or scheduled web subscription price record.
type SubscriptionPrice struct {
	ID           string `json:"id"`
	PlanType     string `json:"planType,omitempty"`
	Territory    string `json:"territory,omitempty"`
	PricePointID string `json:"pricePointId,omitempty"`
	StartDate    string `json:"startDate,omitempty"`
	Preserved    bool   `json:"preserved,omitempty"`
}

// SubscriptionPlanPrice identifies one billing plan's price and scheduling attributes.
type SubscriptionPlanPrice struct {
	PlanType             string
	PricePointID         string
	StartDate            string
	PreserveCurrentPrice bool
}

// CreateSubscriptionPlanPrices creates paired upfront and monthly prices through
// the inline subscription PATCH used by App Store Connect.
func (c *Client) CreateSubscriptionPlanPrices(ctx context.Context, subscriptionID, upfrontPricePointID, monthlyPricePointID string) (*SubscriptionPlanPricesResult, error) {
	return c.SetSubscriptionPlanPrices(ctx, subscriptionID, []SubscriptionPlanPrice{
		{PlanType: "UPFRONT", PricePointID: upfrontPricePointID},
		{PlanType: "MONTHLY", PricePointID: monthlyPricePointID},
	})
}

// SetSubscriptionPlanPrices creates or schedules paired plan prices through the inline PATCH.
func (c *Client) SetSubscriptionPlanPrices(ctx context.Context, subscriptionID string, prices []SubscriptionPlanPrice) (*SubscriptionPlanPricesResult, error) {
	subscriptionID = strings.TrimSpace(subscriptionID)
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription id is required")
	}
	if len(prices) != 2 {
		return nil, fmt.Errorf("exactly two subscription plan prices are required")
	}

	upfrontID := "${current-upfront}"
	monthlyID := "${current-monthly}"
	priceRef := func(id string) map[string]string {
		return map[string]string{"type": "subscriptionPrices", "id": id}
	}
	includedPrice := func(id string, price SubscriptionPlanPrice) map[string]any {
		attributes := map[string]any{"planType": price.PlanType}
		if strings.TrimSpace(price.StartDate) != "" {
			attributes["startDate"] = strings.TrimSpace(price.StartDate)
			attributes["preserveCurrentPrice"] = price.PreserveCurrentPrice
		}
		return map[string]any{
			"type":       "subscriptionPrices",
			"id":         id,
			"attributes": attributes,
			"relationships": map[string]any{
				"subscriptionPricePoint": map[string]any{
					"data": map[string]string{
						"type": "subscriptionPricePoints",
						"id":   price.PricePointID,
					},
				},
			},
		}
	}
	byType := map[string]SubscriptionPlanPrice{}
	for _, price := range prices {
		price.PlanType = strings.ToUpper(strings.TrimSpace(price.PlanType))
		price.PricePointID = strings.TrimSpace(price.PricePointID)
		if price.PlanType != "UPFRONT" && price.PlanType != "MONTHLY" {
			return nil, fmt.Errorf(`plan type must be "UPFRONT" or "MONTHLY"`)
		}
		if price.PricePointID == "" {
			return nil, fmt.Errorf("%s price point id is required", strings.ToLower(price.PlanType))
		}
		byType[price.PlanType] = price
	}
	upfront, upfrontOK := byType["UPFRONT"]
	monthly, monthlyOK := byType["MONTHLY"]
	if !upfrontOK || !monthlyOK {
		return nil, fmt.Errorf("both UPFRONT and MONTHLY prices are required")
	}
	requestBody := map[string]any{
		"data": map[string]any{
			"type": "subscriptions",
			"id":   subscriptionID,
			"relationships": map[string]any{
				"prices": map[string]any{
					"data": []map[string]string{
						priceRef(upfrontID),
						priceRef(monthlyID),
					},
				},
			},
		},
		"included": []map[string]any{
			includedPrice(upfrontID, upfront),
			includedPrice(monthlyID, monthly),
		},
	}

	responseBody, err := c.doRequest(ctx, http.MethodPatch, "/subscriptions/"+url.PathEscape(subscriptionID), requestBody)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Data jsonAPIResource `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse subscription pricing response: %w", err)
	}
	if returnedID := strings.TrimSpace(payload.Data.ID); returnedID != "" && returnedID != subscriptionID {
		return nil, fmt.Errorf("apple returned subscription %q after patching %q", returnedID, subscriptionID)
	}
	return &SubscriptionPlanPricesResult{
		SubscriptionID:      subscriptionID,
		UpfrontPricePointID: upfront.PricePointID,
		MonthlyPricePointID: monthly.PricePointID,
	}, nil
}

// ListSubscriptionPrices lists applied and scheduled prices for one subscription territory.
// The path is the iris counterpart of OpenAPI GET /v1/subscriptions/{id}/prices
// (operation subscriptions_prices_getToManyRelated), the relationship collection
// written by the captured inline PATCH /subscriptions/{id} prices workflow.
func (c *Client) ListSubscriptionPrices(ctx context.Context, subscriptionID, territory string) ([]SubscriptionPrice, error) {
	subscriptionID = strings.TrimSpace(subscriptionID)
	territory = strings.ToUpper(strings.TrimSpace(territory))
	if subscriptionID == "" {
		return nil, fmt.Errorf("subscription id is required")
	}
	if len(territory) != 3 {
		return nil, fmt.Errorf("territory must be a three-letter territory id")
	}
	query := url.Values{}
	query.Set("filter[territory]", territory)
	query.Set("include", "subscriptionPricePoint,territory")
	query.Set("limit", "200")
	nextPath := queryPath("/subscriptions/"+url.PathEscape(subscriptionID)+"/prices", query)
	result := make([]SubscriptionPrice, 0)
	visited := map[string]struct{}{}
	for nextPath != "" {
		if _, seen := visited[nextPath]; seen {
			return nil, fmt.Errorf("subscription prices pagination loop detected")
		}
		visited[nextPath] = struct{}{}
		responseBody, err := c.doRequest(ctx, http.MethodGet, nextPath, nil)
		if err != nil {
			return nil, err
		}
		var payload jsonAPIListPayload
		if err := json.Unmarshal(responseBody, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse subscription prices response: %w", err)
		}
		for _, resource := range payload.Data {
			result = append(result, decodeSubscriptionPriceResource(resource, territory))
		}
		nextLink, err := extractNextLink(payload.Links)
		if err != nil {
			return nil, fmt.Errorf("failed to parse subscription prices pagination links: %w", err)
		}
		if strings.TrimSpace(nextLink) == "" {
			break
		}
		nextPath, err = normalizeNextPath(nextLink, c.baseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize subscription prices pagination link: %w", err)
		}
	}
	return result, nil
}

func decodeSubscriptionPriceResource(resource jsonAPIResource, fallbackTerritory string) SubscriptionPrice {
	territory := strings.ToUpper(strings.TrimSpace(fallbackTerritory))
	if ref := firstRelationshipRef(resource, "territory"); ref != nil {
		if id := strings.ToUpper(strings.TrimSpace(ref.ID)); id != "" {
			territory = id
		}
	}
	pricePointID := ""
	if ref := firstRelationshipRef(resource, "subscriptionPricePoint"); ref != nil {
		pricePointID = strings.TrimSpace(ref.ID)
	}
	return SubscriptionPrice{
		ID:           strings.TrimSpace(resource.ID),
		PlanType:     strings.ToUpper(strings.TrimSpace(stringAttr(resource.Attributes, "planType"))),
		Territory:    territory,
		PricePointID: pricePointID,
		StartDate:    NormalizeSubscriptionPriceStartDate(stringAttr(resource.Attributes, "startDate")),
		Preserved:    boolAttr(resource.Attributes, "preserved"),
	}
}

// NormalizeSubscriptionPriceStartDate reduces Apple date or datetime values to YYYY-MM-DD.
func NormalizeSubscriptionPriceStartDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 10 && value[4] == '-' && value[7] == '-' {
		return value[:10]
	}
	return value
}

// FindSubscriptionPrice locates a price record matching plan type, territory, and price point.
// A requested startDate matches that scheduled record. An empty startDate selects the latest
// non-future effective record for the plan and territory, then checks the price point.
func FindSubscriptionPrice(prices []SubscriptionPrice, planType, territory, pricePointID, startDate string, now time.Time) (SubscriptionPrice, bool) {
	planType = strings.ToUpper(strings.TrimSpace(planType))
	territory = strings.ToUpper(strings.TrimSpace(territory))
	pricePointID = strings.TrimSpace(pricePointID)
	startDate = NormalizeSubscriptionPriceStartDate(startDate)
	if startDate != "" {
		for _, price := range prices {
			if !strings.EqualFold(price.PlanType, planType) || !strings.EqualFold(price.Territory, territory) {
				continue
			}
			if strings.TrimSpace(price.PricePointID) != pricePointID {
				continue
			}
			if NormalizeSubscriptionPriceStartDate(price.StartDate) != startDate {
				continue
			}
			return price, true
		}
		return SubscriptionPrice{}, false
	}
	asOf := subscriptionPriceDateOnlyUTC(now)
	var selected *SubscriptionPrice
	selectedStart := time.Time{}
	for _, price := range prices {
		if !strings.EqualFold(price.PlanType, planType) || !strings.EqualFold(price.Territory, territory) {
			continue
		}
		start := time.Time{}
		if date := NormalizeSubscriptionPriceStartDate(price.StartDate); date != "" {
			parsed, err := time.Parse("2006-01-02", date)
			if err != nil || parsed.After(asOf) {
				continue
			}
			start = parsed
		}
		if selected == nil || start.After(selectedStart) ||
			(start.Equal(selectedStart) && selected.Preserved && !price.Preserved) {
			candidate := price
			selected = &candidate
			selectedStart = start
		}
	}
	if selected == nil || strings.TrimSpace(selected.PricePointID) != pricePointID {
		return SubscriptionPrice{}, false
	}
	return *selected, true
}

func subscriptionPriceDateOnlyUTC(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
