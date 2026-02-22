package errfmt

import (
	"context"
	"errors"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestClassify_MissingAuth(t *testing.T) {
	err := errors.New("wrapped")
	err = wrap(err, shared.ErrMissingAuth)
	ce := Classify(err)
	if ce.Hint == "" {
		t.Fatalf("expected hint, got empty")
	}
}

func TestClassify_Forbidden(t *testing.T) {
	apiErr := &asc.APIError{Code: "FORBIDDEN", Title: "Forbidden", Detail: "Nope"}
	ce := Classify(apiErr)
	if ce.Hint == "" {
		t.Fatalf("expected hint, got empty")
	}
}

func TestClassify_Timeout(t *testing.T) {
	ce := Classify(context.DeadlineExceeded)
	if ce.Hint == "" {
		t.Fatalf("expected hint, got empty")
	}
}

// wrap creates an error that Is() matches target without altering the base string.
type isWrapper struct {
	target error
}

func (e isWrapper) Error() string { return "x" }
func (e isWrapper) Is(t error) bool {
	return t == e.target
}

func wrap(base error, target error) error {
	_ = base
	return isWrapper{target: target}
}
