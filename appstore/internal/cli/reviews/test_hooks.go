package reviews

import "github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"

// SetReviewItemsClientFactory replaces the review-items client factory for tests.
func SetReviewItemsClientFactory(fn func() (*asc.Client, error)) func() {
	previous := reviewItemsClientFactory
	if fn != nil {
		reviewItemsClientFactory = fn
	}
	return func() {
		reviewItemsClientFactory = previous
	}
}

// SetReviewSubmissionsClientFactory replaces the review-submissions client factory for tests.
func SetReviewSubmissionsClientFactory(fn func() (*asc.Client, error)) func() {
	previous := reviewSubmissionsClientFactory
	if fn != nil {
		reviewSubmissionsClientFactory = fn
	}
	return func() {
		reviewSubmissionsClientFactory = previous
	}
}
