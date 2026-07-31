package asc

import (
	"encoding/json"
	"strings"
	"testing"
)

const reviewRedactionSentinel = "asc-red-sentinel-asc-demo-pw-2ad57e"

func TestRedactSecretPreservesOnlyTheEmptyValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty stays empty", value: "", want: ""},
		// The API permits an unconstrained string, so a whitespace-only
		// password is still a credential and must not pass through verbatim.
		{name: "whitespace-only is still a secret", value: "   ", want: RedactedValuePlaceholder},
		{name: "secret becomes placeholder", value: reviewRedactionSentinel, want: RedactedValuePlaceholder},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := RedactSecret(test.value); got != test.want {
				t.Fatalf("RedactSecret(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestRedactAppStoreReviewDetailResponseLeavesOriginalIntact(t *testing.T) {
	original := &AppStoreReviewDetailResponse{}
	original.Data.ID = "detail-1"
	original.Data.Attributes = AppStoreReviewDetailAttributes{
		DemoAccountName:     "reviewer@example.com",
		DemoAccountPassword: reviewRedactionSentinel,
		Notes:               "Reviewer notes",
	}

	safe := RedactAppStoreReviewDetailResponse(original)

	if original.Data.Attributes.DemoAccountPassword != reviewRedactionSentinel {
		t.Fatalf("original password was mutated to %q", original.Data.Attributes.DemoAccountPassword)
	}
	if safe.Data.Attributes.DemoAccountPassword != RedactedValuePlaceholder {
		t.Fatalf("redacted password = %q, want %q", safe.Data.Attributes.DemoAccountPassword, RedactedValuePlaceholder)
	}
	if safe.Data.ID != "detail-1" || safe.Data.Attributes.DemoAccountName != "reviewer@example.com" {
		t.Fatalf("redaction dropped non-sensitive fields: %+v", safe.Data)
	}

	encoded, err := json.Marshal(safe)
	if err != nil {
		t.Fatalf("marshal redacted response: %v", err)
	}
	if strings.Contains(string(encoded), reviewRedactionSentinel) {
		t.Fatalf("serialized redacted response leaked the sentinel: %s", encoded)
	}
}

func TestRedactBetaAppReviewDetailsResponseCopiesEveryResource(t *testing.T) {
	original := &BetaAppReviewDetailsResponse{
		Data: []Resource[BetaAppReviewDetailAttributes]{
			{ID: "detail-1", Attributes: BetaAppReviewDetailAttributes{DemoAccountPassword: reviewRedactionSentinel}},
			{ID: "detail-2", Attributes: BetaAppReviewDetailAttributes{DemoAccountName: "reviewer@example.com"}},
		},
	}

	safe := RedactBetaAppReviewDetailsResponse(original)

	if original.Data[0].Attributes.DemoAccountPassword != reviewRedactionSentinel {
		t.Fatal("redaction mutated the original list response")
	}
	if safe.Data[0].Attributes.DemoAccountPassword != RedactedValuePlaceholder {
		t.Fatalf("first resource password = %q, want placeholder", safe.Data[0].Attributes.DemoAccountPassword)
	}
	if safe.Data[1].Attributes.DemoAccountPassword != "" {
		t.Fatalf("empty password became %q, want empty", safe.Data[1].Attributes.DemoAccountPassword)
	}
	if safe.Data[1].Attributes.DemoAccountName != "reviewer@example.com" {
		t.Fatalf("redaction dropped non-sensitive fields: %+v", safe.Data[1])
	}
}

func TestRedactResponseHelpersHandleNil(t *testing.T) {
	if got := RedactAppStoreReviewDetailResponse(nil); got != nil {
		t.Fatalf("RedactAppStoreReviewDetailResponse(nil) = %v, want nil", got)
	}
	if got := RedactBetaAppReviewDetailResponse(nil); got != nil {
		t.Fatalf("RedactBetaAppReviewDetailResponse(nil) = %v, want nil", got)
	}
	if got := RedactBetaAppReviewDetailsResponse(nil); got != nil {
		t.Fatalf("RedactBetaAppReviewDetailsResponse(nil) = %v, want nil", got)
	}
}

func TestRedactAppStoreReviewDetailIncludedResources(t *testing.T) {
	originalIncluded := json.RawMessage(`[
		{"type":"appStoreReviewDetails","id":"detail-1","attributes":{"demoAccountPassword":"` + reviewRedactionSentinel + `","notes":"keep"},"relationships":{"version":{"data":{"type":"appStoreVersions","id":"version-1"}}}},
		{"type":"builds","id":"build-1","attributes":{"version":"42"}}
	]`)
	original := &SingleResponse[AppStoreVersionAttributes]{Included: originalIncluded}

	safe, err := RedactAppStoreReviewDetailIncludesInSingleResponse(original)
	if err != nil {
		t.Fatalf("RedactAppStoreReviewDetailIncludesInSingleResponse() error = %v", err)
	}
	encoded, err := json.Marshal(safe)
	if err != nil {
		t.Fatalf("marshal redacted response: %v", err)
	}
	if strings.Contains(string(encoded), reviewRedactionSentinel) {
		t.Fatalf("included resource leaked password: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"demoAccountPassword":"`+RedactedValuePlaceholder+`"`) {
		t.Fatalf("included resource missing placeholder: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"notes":"keep"`) ||
		!strings.Contains(string(encoded), `"relationships"`) ||
		!strings.Contains(string(encoded), `"type":"builds"`) {
		t.Fatalf("redaction dropped unrelated included data: %s", encoded)
	}
	if !strings.Contains(string(original.Included), reviewRedactionSentinel) {
		t.Fatal("redaction mutated original Included bytes")
	}
}

func TestRedactIncludedPasswordDoesNotTrustResourceType(t *testing.T) {
	original := &SingleResponse[AppStoreVersionAttributes]{
		Included: json.RawMessage(`[
			{"type":"appStoreReviewDetails","type":"misleadingType","attributes":{"demoAccountPassword":"` + reviewRedactionSentinel + `"}}
		]`),
	}

	safe, err := RedactAppStoreReviewDetailIncludesInSingleResponse(original)
	if err != nil {
		t.Fatalf("RedactAppStoreReviewDetailIncludesInSingleResponse() error = %v", err)
	}
	encoded, err := json.Marshal(safe)
	if err != nil {
		t.Fatalf("marshal redacted response: %v", err)
	}
	if strings.Contains(string(encoded), reviewRedactionSentinel) {
		t.Fatalf("misleading resource type bypassed password redaction: %s", encoded)
	}
}

func TestRedactIncludedPasswordCollapsesDuplicateAttributes(t *testing.T) {
	original := &SingleResponse[AppStoreVersionAttributes]{
		Included: json.RawMessage(`[
			{"attributes":{"demoAccountPassword":"` + reviewRedactionSentinel + `"},"attributes":{"notes":"last"}}
		]`),
	}

	safe, err := RedactAppStoreReviewDetailIncludesInSingleResponse(original)
	if err != nil {
		t.Fatalf("RedactAppStoreReviewDetailIncludesInSingleResponse() error = %v", err)
	}
	encoded, err := json.Marshal(safe)
	if err != nil {
		t.Fatalf("marshal redacted response: %v", err)
	}
	if strings.Contains(string(encoded), reviewRedactionSentinel) {
		t.Fatalf("duplicate attributes bypassed password redaction: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"notes":"last"`) {
		t.Fatalf("last attributes object was not preserved: %s", encoded)
	}
}

func TestValidateSecretMutationValueRejectsRedactionPlaceholder(t *testing.T) {
	placeholder := RedactedValuePlaceholder
	if err := validateSecretMutationValue(&placeholder); err == nil {
		t.Fatal("validateSecretMutationValue() error = nil, want placeholder rejection")
	}

	actual := "actual-password"
	if err := validateSecretMutationValue(&actual); err != nil {
		t.Fatalf("validateSecretMutationValue(actual) error = %v", err)
	}
	if err := validateSecretMutationValue(nil); err != nil {
		t.Fatalf("validateSecretMutationValue(nil) error = %v", err)
	}
}
