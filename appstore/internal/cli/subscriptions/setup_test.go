package subscriptions

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestSubscriptionSetupPriceMatchesTargetRequiresPlanType(t *testing.T) {
	relationships, err := json.Marshal(subscriptionSetupPriceRelationships{
		SubscriptionPricePoint: &asc.Relationship{Data: asc.ResourceData{Type: asc.ResourceTypeSubscriptionPricePoints, ID: "pp-1"}},
		Territory:              &asc.Relationship{Data: asc.ResourceData{Type: asc.ResourceTypeTerritories, ID: "USA"}},
	})
	if err != nil {
		t.Fatalf("marshal relationships: %v", err)
	}

	price := asc.Resource[asc.SubscriptionPriceAttributes]{
		ID:            "price-1",
		Relationships: relationships,
		Attributes: asc.SubscriptionPriceAttributes{
			PlanType: asc.SubscriptionPlanTypeMonthly,
		},
	}

	if subscriptionSetupPriceMatchesTarget(price, "pp-1", "USA", asc.SubscriptionPriceCreateAttributes{PlanType: asc.SubscriptionPlanTypeUpfront}) {
		t.Fatal("monthly price should not satisfy an upfront setup price")
	}

	price.Attributes.PlanType = asc.SubscriptionPlanTypeUpfront
	if !subscriptionSetupPriceMatchesTarget(price, "pp-1", "USA", asc.SubscriptionPriceCreateAttributes{PlanType: asc.SubscriptionPlanTypeUpfront}) {
		t.Fatal("upfront price should satisfy an upfront setup price")
	}
}

func TestValidateExistingSubscriptionSetupLocalizationComparesEmptyDescription(t *testing.T) {
	localization := asc.Resource[asc.SubscriptionLocalizationAttributes]{
		ID: "loc-1",
		Attributes: asc.SubscriptionLocalizationAttributes{
			Locale:      "en-US",
			Name:        "Pro Monthly",
			Description: "Old description.",
		},
	}
	opts := subscriptionsSetupOptions{
		Locale:      "en-US",
		DisplayName: "Pro Monthly",
	}

	err := validateExistingSubscriptionSetupLocalization(localization, opts)
	if err == nil || !strings.Contains(err.Error(), "different description") {
		t.Fatalf("expected different description error, got %v", err)
	}
}
