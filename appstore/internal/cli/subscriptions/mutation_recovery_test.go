package subscriptions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestRunReconciledMutationRetriesOnlyAfterNegativeReadback(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")

	mutations := 0
	readbacks := 0
	status, err := runReconciledMutation(
		context.Background(),
		func(context.Context) (bool, error) {
			readbacks++
			return false, nil
		},
		func(context.Context) error {
			mutations++
			if mutations == 1 {
				return &asc.RetryableError{Err: errors.New("temporary failure")}
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("runReconciledMutation() error: %v", err)
	}
	if status != reconciledMutationCreated || mutations != 2 || readbacks != 2 {
		t.Fatalf("unexpected recovery: status=%q mutations=%d readbacks=%d", status, mutations, readbacks)
	}
}

func TestRunReconciledMutationRetriesChildDeadlineAfterNegativeReadback(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")

	mutations := 0
	readbacks := 0
	status, err := runReconciledMutation(
		context.Background(),
		func(context.Context) (bool, error) {
			readbacks++
			return false, nil
		},
		func(context.Context) error {
			mutations++
			if mutations == 1 {
				return context.DeadlineExceeded
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("runReconciledMutation() error: %v", err)
	}
	if status != reconciledMutationCreated || mutations != 2 || readbacks != 2 {
		t.Fatalf("unexpected recovery: status=%q mutations=%d readbacks=%d", status, mutations, readbacks)
	}
}

func TestRunReconciledMutationReadsAgainBeforeReplay(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")

	mutations := 0
	readbacks := 0
	status, err := runReconciledMutation(
		context.Background(),
		func(context.Context) (bool, error) {
			readbacks++
			return readbacks == 2, nil
		},
		func(context.Context) error {
			mutations++
			return &asc.RetryableError{Err: errors.New("ambiguous failure")}
		},
	)
	if err != nil {
		t.Fatalf("runReconciledMutation() error: %v", err)
	}
	if status != reconciledMutationReconciled || mutations != 1 || readbacks != 2 {
		t.Fatalf("unexpected recovery: status=%q mutations=%d readbacks=%d", status, mutations, readbacks)
	}
}

func TestRunReconciledMutationStopsWhenReadbackFails(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "3")
	mutations := 0
	_, err := runReconciledMutation(
		context.Background(),
		func(context.Context) (bool, error) {
			return false, errors.New("readback unavailable")
		},
		func(context.Context) error {
			mutations++
			return &asc.RetryableError{Err: errors.New("ambiguous failure")}
		},
	)
	if err == nil || mutations != 1 {
		t.Fatalf("expected one mutation and a readback error, mutations=%d err=%v", mutations, err)
	}
}

func TestRunReconciledMutationRespectsCancellationDuringBackoff(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1h")

	ctx, cancel := context.WithCancel(context.Background())
	mutations := 0
	readbacks := 0
	_, err := runReconciledMutation(
		ctx,
		func(context.Context) (bool, error) {
			readbacks++
			cancel()
			return false, nil
		},
		func(context.Context) error {
			mutations++
			return &asc.RetryableError{Err: errors.New("temporary failure"), RetryAfter: time.Hour}
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if mutations != 1 || readbacks != 1 {
		t.Fatalf("expected cancellation during first backoff, mutations=%d readbacks=%d", mutations, readbacks)
	}
}
