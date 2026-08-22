package apps

import (
	"fmt"
	"slices"
	"strings"
)

// appVersionStateFilterList mirrors the enum documented for
// filter[appStoreVersions.appVersionState] on GET /v1/apps. It is deliberately
// narrower than the states accepted by the App Store version endpoints.
func appVersionStateFilterList() []string {
	return []string{
		"ACCEPTED",
		"DEVELOPER_REJECTED",
		"IN_REVIEW",
		"INVALID_BINARY",
		"METADATA_REJECTED",
		"PENDING_APPLE_RELEASE",
		"PENDING_DEVELOPER_RELEASE",
		"PREPARE_FOR_SUBMISSION",
		"PROCESSING_FOR_DISTRIBUTION",
		"READY_FOR_DISTRIBUTION",
		"READY_FOR_REVIEW",
		"REJECTED",
		"REPLACED_WITH_NEW_VERSION",
		"WAITING_FOR_EXPORT_COMPLIANCE",
		"WAITING_FOR_REVIEW",
	}
}

// reviewSubmissionStateFilterList mirrors the enum documented for
// filter[reviewSubmissions.state] on GET /v1/apps.
func reviewSubmissionStateFilterList() []string {
	return []string{
		"READY_FOR_REVIEW",
		"WAITING_FOR_REVIEW",
		"IN_REVIEW",
		"UNRESOLVED_ISSUES",
		"CANCELING",
		"COMPLETING",
		"COMPLETE",
	}
}

func normalizeAppVersionStateFilters(values []string) ([]string, error) {
	return normalizeStateFilters(values, "--version-state", appVersionStateFilterList())
}

func normalizeReviewSubmissionStateFilters(values []string) ([]string, error) {
	return normalizeStateFilters(values, "--review-submission-state", reviewSubmissionStateFilterList())
}

func normalizeStateFilters(values []string, flagName string, allowed []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	for _, value := range values {
		if !slices.Contains(allowed, value) {
			return nil, fmt.Errorf("%s must be one of: %s", flagName, strings.Join(allowed, ", "))
		}
	}
	return values, nil
}
