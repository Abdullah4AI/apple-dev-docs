package web

import (
	"testing"

	webcore "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/web"
)

func TestWebIAPTaxCategoryRejectsWrongConditionResourceType(t *testing.T) {
	category := webcore.TaxCategory{ID: "C003", ProductType: "ADDON", Conditions: []webcore.TaxCategoryReference{{ID: "condition-1", Type: "apps"}}}
	if err := validateIAPTaxConditions([]webcore.TaxCategory{category}, &category, []string{"condition-1"}); err == nil {
		t.Fatal("accepted a non-taxConditions reference as a compatible condition")
	}
}
