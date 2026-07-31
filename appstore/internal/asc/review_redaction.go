package asc

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// RedactedValuePlaceholder marks a secret that was withheld from output. It is
// a presentation marker and must be rejected at credential mutation boundaries
// so exported redacted output cannot overwrite a real credential.
const RedactedValuePlaceholder = "(redacted)"

// RedactSecret returns the placeholder for any non-empty secret and leaves the
// empty value untouched so absent fields stay absent in rendered output. The
// API permits an unconstrained string, so even a whitespace-only value is
// treated as a credential.
func RedactSecret(value string) string {
	if value == "" {
		return value
	}
	return RedactedValuePlaceholder
}

func validateSecretMutationValue(value *string) error {
	if value != nil && *value == RedactedValuePlaceholder {
		return fmt.Errorf("demo account password is the redaction placeholder; provide the actual password or omit the field")
	}
	return nil
}

// ValidateSecretMutationValue rejects presentation placeholders before they
// can cross a credential mutation boundary.
func ValidateSecretMutationValue(value *string) error {
	return validateSecretMutationValue(value)
}

// RedactAppStoreReviewDetailAttributes returns a presentation-safe copy of App
// Store review detail attributes.
func RedactAppStoreReviewDetailAttributes(attrs AppStoreReviewDetailAttributes) AppStoreReviewDetailAttributes {
	attrs.DemoAccountPassword = RedactSecret(attrs.DemoAccountPassword)
	return attrs
}

// RedactBetaAppReviewDetailAttributes returns a presentation-safe copy of
// TestFlight beta app review detail attributes.
func RedactBetaAppReviewDetailAttributes(attrs BetaAppReviewDetailAttributes) BetaAppReviewDetailAttributes {
	attrs.DemoAccountPassword = RedactSecret(attrs.DemoAccountPassword)
	return attrs
}

// RedactAppStoreReviewDetailResponse returns a presentation-safe copy of a
// single App Store review detail response. The argument is left unmodified so
// callers keep the real password for requests, comparison, and validation.
func RedactAppStoreReviewDetailResponse(resp *AppStoreReviewDetailResponse) *AppStoreReviewDetailResponse {
	return redactSingleResponse(resp, RedactAppStoreReviewDetailAttributes)
}

// RedactBetaAppReviewDetailResponse returns a presentation-safe copy of a
// single TestFlight beta app review detail response.
func RedactBetaAppReviewDetailResponse(resp *BetaAppReviewDetailResponse) *BetaAppReviewDetailResponse {
	return redactSingleResponse(resp, RedactBetaAppReviewDetailAttributes)
}

// RedactBetaAppReviewDetailsResponse returns a presentation-safe copy of a
// TestFlight beta app review details list response.
func RedactBetaAppReviewDetailsResponse(resp *BetaAppReviewDetailsResponse) *BetaAppReviewDetailsResponse {
	return redactListResponse(resp, RedactBetaAppReviewDetailAttributes)
}

// RedactAppStoreReviewDetailIncludesInSingleResponse returns a presentation-safe
// copy whose included appStoreReviewDetails resources cannot expose a password.
func RedactAppStoreReviewDetailIncludesInSingleResponse[T any](resp *SingleResponse[T]) (*SingleResponse[T], error) {
	if resp == nil {
		return nil, nil
	}
	safe := *resp
	included, err := redactAppStoreReviewDetailIncluded(resp.Included)
	if err != nil {
		return nil, err
	}
	safe.Included = included
	return &safe, nil
}

// RedactAppStoreReviewDetailIncludesInListResponse returns a presentation-safe
// copy whose included appStoreReviewDetails resources cannot expose a password.
func RedactAppStoreReviewDetailIncludesInListResponse[T any](resp *Response[T]) (*Response[T], error) {
	if resp == nil {
		return nil, nil
	}
	safe := *resp
	included, err := redactAppStoreReviewDetailIncluded(resp.Included)
	if err != nil {
		return nil, err
	}
	safe.Included = included
	return &safe, nil
}

func redactAppStoreReviewDetailIncluded(included json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(included)) == 0 {
		return included, nil
	}

	var resources []json.RawMessage
	if err := json.Unmarshal(included, &resources); err != nil {
		return nil, fmt.Errorf("redact included review details: %w", err)
	}
	for index, rawResource := range resources {
		// Always rebuild the resource object, even when its final attributes
		// member has no password. This collapses duplicate object members so an
		// earlier attributes object cannot smuggle a secret past last-key-wins
		// decoding.
		var resource map[string]json.RawMessage
		if err := json.Unmarshal(rawResource, &resource); err != nil {
			return nil, fmt.Errorf("redact included resource %d: %w", index, err)
		}
		if rawAttributes, present := resource["attributes"]; present {
			var attributes map[string]json.RawMessage
			if err := json.Unmarshal(rawAttributes, &attributes); err != nil {
				return nil, fmt.Errorf("redact included review detail %d attributes: %w", index, err)
			}
			if password, present := attributes["demoAccountPassword"]; present &&
				!bytes.Equal(bytes.TrimSpace(password), []byte("null")) {
				var value string
				if err := json.Unmarshal(password, &value); err != nil || value != "" {
					placeholder, err := json.Marshal(RedactedValuePlaceholder)
					if err != nil {
						return nil, err
					}
					attributes["demoAccountPassword"] = placeholder
				}
			}
			redactedAttributes, err := json.Marshal(attributes)
			if err != nil {
				return nil, fmt.Errorf("redact included review detail %d attributes: %w", index, err)
			}
			resource["attributes"] = redactedAttributes
		}

		redactedResource, err := json.Marshal(resource)
		if err != nil {
			return nil, fmt.Errorf("redact included resource %d: %w", index, err)
		}
		resources[index] = redactedResource
	}
	redacted, err := json.Marshal(resources)
	if err != nil {
		return nil, fmt.Errorf("redact included review details: %w", err)
	}
	return redacted, nil
}

func redactSingleResponse[T any](resp *SingleResponse[T], redact func(T) T) *SingleResponse[T] {
	if resp == nil {
		return nil
	}
	safe := *resp
	safe.Data.Attributes = redact(safe.Data.Attributes)
	return &safe
}

func redactListResponse[T any](resp *Response[T], redact func(T) T) *Response[T] {
	if resp == nil {
		return nil
	}
	safe := *resp
	if len(resp.Data) > 0 {
		safe.Data = make([]Resource[T], len(resp.Data))
		copy(safe.Data, resp.Data)
		for i := range safe.Data {
			safe.Data[i].Attributes = redact(safe.Data[i].Attributes)
		}
	}
	return &safe
}
