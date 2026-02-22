package promotedpurchases

import (
	"fmt"
	"strings"
)

type promotedPurchaseProductType string

const (
	promotedPurchaseProductTypeSubscription  promotedPurchaseProductType = "SUBSCRIPTION"
	promotedPurchaseProductTypeInAppPurchase promotedPurchaseProductType = "IN_APP_PURCHASE"
)

func normalizePromotedPurchaseProductType(value string) (promotedPurchaseProductType, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	switch normalized {
	case string(promotedPurchaseProductTypeSubscription):
		return promotedPurchaseProductTypeSubscription, nil
	case string(promotedPurchaseProductTypeInAppPurchase):
		return promotedPurchaseProductTypeInAppPurchase, nil
	default:
		return "", fmt.Errorf("--product-type must be one of: SUBSCRIPTION, IN_APP_PURCHASE")
	}
}
