package asc

import (
	"net/url"
	"testing"
)

func TestBuildSubscriptionIntroductoryOffersQueryFieldsAndInclude(t *testing.T) {
	query := &subscriptionIntroductoryOffersQuery{}
	WithSubscriptionIntroductoryOffersFields([]string{
		"startDate",
		"endDate",
		"duration",
		"offerMode",
		"numberOfPeriods",
		"targetSubscriptionPlanType",
		"territory",
		"subscriptionPricePoint",
	})(query)
	WithSubscriptionIntroductoryOffersInclude([]string{"territory", "subscriptionPricePoint"})(query)

	values, err := url.ParseQuery(buildSubscriptionIntroductoryOffersQuery(query))
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if got := values.Get("fields[subscriptionIntroductoryOffers]"); got != "startDate,endDate,duration,offerMode,numberOfPeriods,targetSubscriptionPlanType,territory,subscriptionPricePoint" {
		t.Fatalf("unexpected introductory offer fields: %q", got)
	}
	if got := values.Get("include"); got != "territory,subscriptionPricePoint" {
		t.Fatalf("unexpected introductory offer include: %q", got)
	}
}

func TestBuildSubscriptionLocalizationQueriesFields(t *testing.T) {
	subscriptionQuery := &subscriptionLocalizationsQuery{}
	WithSubscriptionLocalizationsFields([]string{"name", "locale", "description"})(subscriptionQuery)
	subscriptionValues, err := url.ParseQuery(buildSubscriptionLocalizationsQuery(subscriptionQuery))
	if err != nil {
		t.Fatalf("parse subscription localization query: %v", err)
	}
	if got := subscriptionValues.Get("fields[subscriptionLocalizations]"); got != "name,locale,description" {
		t.Fatalf("unexpected subscription localization fields: %q", got)
	}

	groupQuery := &subscriptionGroupLocalizationsQuery{}
	WithSubscriptionGroupLocalizationsFields([]string{"name", "locale", "customAppName"})(groupQuery)
	groupValues, err := url.ParseQuery(buildSubscriptionGroupLocalizationsQuery(groupQuery))
	if err != nil {
		t.Fatalf("parse group localization query: %v", err)
	}
	if got := groupValues.Get("fields[subscriptionGroupLocalizations]"); got != "name,locale,customAppName" {
		t.Fatalf("unexpected group localization fields: %q", got)
	}
}

func TestBuildSubscriptionPricesQueryPlanType(t *testing.T) {
	query := &subscriptionPricesQuery{}
	WithSubscriptionPricesPlanType(SubscriptionPlanTypeMonthly)(query)

	values, err := url.ParseQuery(buildSubscriptionPricesQuery(query))
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if got := values.Get("filter[planType]"); got != "MONTHLY" {
		t.Fatalf("expected filter[planType]=MONTHLY, got %q", got)
	}
}

func TestBuildSubscriptionPricesQueryFields(t *testing.T) {
	query := &subscriptionPricesQuery{}
	WithSubscriptionPricesFields([]string{"startDate", "preserved", "planType", "territory", "subscriptionPricePoint"})(query)

	values, err := url.ParseQuery(buildSubscriptionPricesQuery(query))
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if got := values.Get("fields[subscriptionPrices]"); got != "startDate,preserved,planType,territory,subscriptionPricePoint" {
		t.Fatalf("unexpected subscription price fields: %q", got)
	}
}

func TestBuildSubscriptionPricesQueryRejectsEmptyPlanType(t *testing.T) {
	query := &subscriptionPricesQuery{}
	WithSubscriptionPricesPlanType("")(query)

	values, err := url.ParseQuery(buildSubscriptionPricesQuery(query))
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if got := values.Get("filter[planType]"); got != "" {
		t.Fatalf("expected no planType filter, got %q", got)
	}
}

func TestBuildSubscriptionPricesQueryPlanTypeWithOtherFilters(t *testing.T) {
	query := &subscriptionPricesQuery{}
	WithSubscriptionPricesPlanType(SubscriptionPlanTypeUpfront)(query)
	WithSubscriptionPricesTerritory("nor")(query)
	WithSubscriptionPricesInclude([]string{"subscriptionPricePoint", "territory"})(query)
	WithSubscriptionPricesLimit(25)(query)

	values, err := url.ParseQuery(buildSubscriptionPricesQuery(query))
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if got := values.Get("filter[planType]"); got != "UPFRONT" {
		t.Fatalf("expected filter[planType]=UPFRONT, got %q", got)
	}
	if got := values.Get("filter[territory]"); got != "NOR" {
		t.Fatalf("expected filter[territory]=NOR, got %q", got)
	}
	if got := values.Get("include"); got != "subscriptionPricePoint,territory" {
		t.Fatalf("expected combined include values, got %q", got)
	}
	if got := values.Get("limit"); got != "25" {
		t.Fatalf("expected limit=25, got %q", got)
	}
}
