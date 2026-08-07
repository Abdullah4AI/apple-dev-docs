package versions

import "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"

func normalizeAppStoreVersionInclude(value string) ([]string, error) {
	return shared.NormalizeSelection(value, appStoreVersionIncludeList(), "--include")
}

func appStoreVersionIncludeList() []string {
	return []string{
		"ageRatingDeclaration",
		"app",
		"appStoreVersionLocalizations",
		"build",
		"appStoreVersionPhasedRelease",
		"gameCenterAppVersion",
		"routingAppCoverage",
		"appStoreReviewDetail",
		"appStoreVersionSubmission",
		"appClipDefaultExperience",
		"appStoreVersionExperiments",
		"appStoreVersionExperimentsV2",
		"alternativeDistributionPackage",
	}
}
