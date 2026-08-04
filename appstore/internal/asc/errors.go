package asc

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrNotFound              = errors.New("resource not found")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden")
	ErrBadRequest            = errors.New("bad request")
	ErrConflict              = errors.New("resource conflict")
	ErrRepeatedPaginationURL = errors.New("detected repeated pagination URL")
)

type responseBodyReadError struct {
	err error
}

func (e *responseBodyReadError) Error() string {
	return fmt.Sprintf("failed to read response body: %v", e.err)
}

func (e *responseBodyReadError) Unwrap() error {
	return e.err
}

func isResponseBodyReadError(err error) bool {
	var readErr *responseBodyReadError
	return errors.As(err, &readErr)
}

type buildUploadFileCommitResponseError struct {
	err error
}

func (e *buildUploadFileCommitResponseError) Error() string {
	return e.err.Error()
}

func (e *buildUploadFileCommitResponseError) Unwrap() error {
	return e.err
}

func newBuildUploadFileCommitResponseError(err error) error {
	return &buildUploadFileCommitResponseError{err: err}
}

// IsBuildUploadFileCommitResponseError reports whether a successful
// build-upload-file commit response could not be read or decoded.
func IsBuildUploadFileCommitResponseError(err error) bool {
	var responseErr *buildUploadFileCommitResponseError
	return errors.As(err, &responseErr)
}

// APIError represents a parsed App Store Connect error response.
type APIError struct {
	Code             string
	Title            string
	Detail           string
	StatusCode       int // HTTP status code that triggered this error (0 if unknown)
	AssociatedErrors map[string][]APIAssociatedError
}

// APIAssociatedError represents an additional actionable error returned
// under errors[].meta.associatedErrors in App Store Connect responses.
type APIAssociatedError struct {
	Code   string
	Detail string
}

func (e *APIError) Error() string {
	title := strings.TrimSpace(SanitizeTerminalText(e.Title))
	detail := strings.TrimSpace(SanitizeTerminalText(e.Detail))
	code := strings.TrimSpace(SanitizeTerminalText(e.Code))
	baseMessage := ""
	switch {
	case title != "" && detail != "":
		baseMessage = fmt.Sprintf("%s: %s", title, detail)
	case title != "":
		baseMessage = title
	case detail != "":
		baseMessage = detail
	case code != "":
		baseMessage = code
	default:
		baseMessage = "API error"
	}

	associated := formatAssociatedErrors(e.AssociatedErrors)
	if associated == "" {
		return baseMessage
	}
	return fmt.Sprintf("%s\n\n%s", baseMessage, associated)
}

func (e *APIError) HTTPStatusCode() int {
	if e == nil {
		return 0
	}
	return e.StatusCode
}

func formatAssociatedErrors(values map[string][]APIAssociatedError) string {
	if len(values) == 0 {
		return ""
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	sections := make([]string, 0, len(keys))
	for _, key := range keys {
		resource := strings.TrimSpace(SanitizeTerminalText(key))
		if resource == "" {
			resource = "(unknown resource)"
		}

		entries := values[key]
		lines := make([]string, 0, len(entries)+1)
		lines = append(lines, fmt.Sprintf("Associated errors for %s:", resource))

		for _, entry := range entries {
			entryDetail := strings.TrimSpace(SanitizeTerminalText(entry.Detail))
			entryCode := strings.TrimSpace(SanitizeTerminalText(entry.Code))
			switch {
			case entryDetail != "":
				lines = append(lines, fmt.Sprintf("  - %s", entryDetail))
			case entryCode != "":
				lines = append(lines, fmt.Sprintf("  - %s", entryCode))
			}
		}

		if len(lines) > 1 {
			sections = append(sections, strings.Join(lines, "\n"))
		}
	}

	if len(sections) == 0 {
		return ""
	}
	return strings.Join(sections, "\n\n")
}

func (e *APIError) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return strings.EqualFold(e.Code, "NOT_FOUND") || e.StatusCode == 404
	case ErrUnauthorized:
		// Apple returns 401 with code NOT_AUTHORIZED, so match on the
		// status code as well as the canonical code string.
		return strings.EqualFold(e.Code, "UNAUTHORIZED") || e.StatusCode == 401
	case ErrForbidden:
		return strings.EqualFold(e.Code, "FORBIDDEN") || e.StatusCode == 403
	case ErrBadRequest:
		return strings.EqualFold(e.Code, "BAD_REQUEST")
	case ErrConflict:
		return strings.EqualFold(e.Code, "CONFLICT")
	default:
		return false
	}
}
