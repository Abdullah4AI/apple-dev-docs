package errfmt

import (
	"context"
	"errors"
	"fmt"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

type ClassifiedError struct {
	Message string
	Hint    string
}

func Classify(err error) ClassifiedError {
	if err == nil {
		return ClassifiedError{}
	}

	if errors.Is(err, shared.ErrMissingAuth) {
		return ClassifiedError{
			Message: err.Error(),
			Hint:    "Run `appstore auth login` or `appstore auth init` (or set APPSTORE_KEY_ID/APPSTORE_ISSUER_ID/APPSTORE_PRIVATE_KEY_PATH). Try `appstore auth doctor` if you're unsure what's misconfigured.",
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ClassifiedError{
			Message: err.Error(),
			Hint:    "Increase the request timeout (e.g. set `APPSTORE_TIMEOUT=90s`).",
		}
	}

	if errors.Is(err, asc.ErrForbidden) {
		return ClassifiedError{
			Message: err.Error(),
			Hint:    "Check that your API key has the right role/permissions for this operation in App Store Connect.",
		}
	}

	if errors.Is(err, asc.ErrUnauthorized) {
		return ClassifiedError{
			Message: err.Error(),
			Hint:    "Your credentials may be invalid or expired. Try `appstore auth status` and re-login if needed.",
		}
	}

	return ClassifiedError{
		Message: err.Error(),
		Hint:    "",
	}
}

func FormatStderr(err error) string {
	ce := Classify(err)
	if ce.Message == "" {
		return ""
	}
	if ce.Hint == "" {
		return fmt.Sprintf("Error: %s\n", ce.Message)
	}
	return fmt.Sprintf("Error: %s\nHint: %s\n", ce.Message, ce.Hint)
}
