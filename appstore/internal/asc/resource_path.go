package asc

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// reservedPathSegmentCharacters are the delimiters that let a caller-supplied
// identifier escape its own path segment and re-target the request: "/" adds
// segments, "?" starts a query string, "#" truncates the URL, "%" introduces a
// pre-encoded sequence, and "\" is normalized inconsistently across platforms.
const reservedPathSegmentCharacters = `/?#%\`

// validateResourcePathSegment trims an identifier and confirms it occupies
// exactly one path segment.
//
// Reserved delimiters are rejected rather than percent-encoded. Encoding would
// silently turn an operator mistake into a request for a resource that does not
// exist, and validateAPIPath rejects "%" in outbound paths anyway.
func validateResourcePathSegment(id string) (string, error) {
	segment := strings.TrimSpace(id)
	if segment == "" {
		return "", errors.New("resource identifier is required")
	}
	for _, r := range segment {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("resource identifier %q must be a single path segment without control characters", id)
		}
	}
	if strings.ContainsAny(segment, reservedPathSegmentCharacters) {
		return "", fmt.Errorf("resource identifier %q must be a single path segment without any of %q", id, reservedPathSegmentCharacters)
	}
	if segment == "." || segment == ".." {
		return "", fmt.Errorf("resource identifier %q must be a single path segment, not a relative path", id)
	}
	return segment, nil
}

// resourcePath builds a relative API path from a printf-style template whose
// "%s" verbs are filled with caller-supplied resource identifiers. Every
// identifier is validated as a single path segment, so the parsed endpoint of
// the resulting request always matches the template.
func resourcePath(template string, ids ...string) (string, error) {
	if want := strings.Count(template, "%s"); want != len(ids) {
		return "", fmt.Errorf("resource path template %q expects %d identifiers, got %d", template, want, len(ids))
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		segment, err := validateResourcePathSegment(id)
		if err != nil {
			return "", err
		}
		args = append(args, segment)
	}
	return fmt.Sprintf(template, args...), nil
}

// validateMutatingRequestTarget rejects a query string on a mutating request.
// The App Store Connect OpenAPI definition declares no query parameters for
// POST, PATCH, PUT, or DELETE, so a query string can only come from an
// identifier that escaped its path segment and changed the request target.
func validateMutatingRequestTarget(method, path string) error {
	switch method {
	case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
	default:
		return nil
	}
	if strings.Contains(path, "?") {
		return fmt.Errorf("%s API path must not contain a query string: %q", method, path)
	}
	return nil
}
