package asc

import (
	"strings"
	"testing"
)

func TestSubscriptionPricingDeriveResultRegisteredTableAndMarkdownOutput(t *testing.T) {
	result := &SubscriptionPricingDeriveResult{
		SourceSubscriptionID: "monthly-sub",
		TargetSubscriptionID: "yearly-sub",
		Multiplier:           "10",
		Rounding:             "nearest",
		DryRun:               true,
		Summary: SubscriptionPricingDeriveSummary{
			Total:   1,
			Planned: 1,
		},
		Rows: []SubscriptionPricingDeriveRow{
			{
				Territory:          "SWE",
				Currency:           "SEK",
				SourcePrice:        "9",
				DesiredPrice:       "90",
				CurrentTargetPrice: "129",
				TargetPrice:        "89",
				RequestedMultiple:  "10",
				AchievedMultiple:   "9.888889",
				Action:             "update",
				Status:             "planned",
			},
		},
	}

	ensureOutputRegistryPopulated()
	if !isRegistryTypeRegistered(typeForPtr[SubscriptionPricingDeriveResult]()) {
		t.Fatal("SubscriptionPricingDeriveResult is not registered with the output renderer")
	}

	table := captureStdout(t, func() error { return PrintTable(result) })
	for _, want := range []string{"Source Subscription", "monthly-sub", "Territory", "SWE", "9.888889", "planned"} {
		if !strings.Contains(table, want) {
			t.Fatalf("expected table to contain %q, got %q", want, table)
		}
	}

	markdown := captureStdout(t, func() error { return PrintMarkdown(result) })
	for _, want := range []string{"| Source Subscription", "| monthly-sub", "| Territory", "| SWE", "9.888889"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected markdown to contain %q, got %q", want, markdown)
		}
	}
}
