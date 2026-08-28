package users

import "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"

func normalizeUsersFields(value, flagName string) ([]string, error) {
	return shared.NormalizeSelection(value, usersFieldsList(), flagName)
}

func normalizeUsersAppFields(value, flagName string) ([]string, error) {
	return shared.NormalizeSelection(value, usersAppFieldsList(), flagName)
}

func normalizeUsersSort(value, flagName string) ([]string, error) {
	return shared.NormalizeSelection(value, usersSortList(), flagName)
}

func usersSortList() []string {
	return []string{"username", "-username", "lastName", "-lastName"}
}

func usersFieldsList() []string {
	return []string{
		"username",
		"firstName",
		"lastName",
		"roles",
		"allAppsVisible",
		"provisioningAllowed",
		"visibleApps",
	}
}

func usersAppFieldsList() []string {
	return []string{
		"accessibilityUrl",
		"name",
		"bundleId",
		"sku",
		"primaryLocale",
		"isOrEverWasMadeForKids",
		"subscriptionStatusUrl",
		"subscriptionStatusUrlVersion",
		"subscriptionStatusUrlForSandbox",
		"subscriptionStatusUrlVersionForSandbox",
		"contentRightsDeclaration",
		"streamlinedPurchasingEnabled",
		"accessibilityDeclarations",
		"appEncryptionDeclarations",
		"appStoreIcon",
		"ciProduct",
		"betaTesters",
		"betaGroups",
		"appStoreVersions",
		"appTags",
		"preReleaseVersions",
		"betaAppLocalizations",
		"builds",
		"betaLicenseAgreement",
		"betaAppReviewDetail",
		"appInfos",
		"appClips",
		"appPricePoints",
		"endUserLicenseAgreement",
		"appPriceSchedule",
		"appAvailabilityV2",
		"inAppPurchases",
		"subscriptionGroups",
		"gameCenterEnabledVersions",
		"perfPowerMetrics",
		"appCustomProductPages",
		"inAppPurchasesV2",
		"promotedPurchases",
		"appEvents",
		"reviewSubmissions",
		"subscriptionGracePeriod",
		"customerReviews",
		"customerReviewSummarizations",
		"gameCenterDetail",
		"appStoreVersionExperimentsV2",
		"alternativeDistributionKey",
		"analyticsReportRequests",
		"marketplaceSearchDetail",
		"buildUploads",
		"backgroundAssets",
		"betaFeedbackScreenshotSubmissions",
		"betaFeedbackCrashSubmissions",
		"searchKeywords",
		"webhooks",
		"androidToIosAppMappingDetails",
	}
}
