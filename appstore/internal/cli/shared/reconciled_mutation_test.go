package shared

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
)

func TestRunReconciledMutationReadsAgainBeforeReplay(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")
	mutations := 0
	readbacks := 0

	value, status, err := RunReconciledMutation(
		context.Background(),
		func(context.Context) (string, error) {
			mutations++
			return "", &asc.RetryableError{Err: errors.New("ambiguous timeout")}
		},
		func(context.Context) (string, bool, error) {
			readbacks++
			return "localization-id", readbacks == 2, nil
		},
	)
	if err != nil {
		t.Fatalf("RunReconciledMutation() error: %v", err)
	}
	if value != "localization-id" || status != ReconciledMutationRecovered || mutations != 1 || readbacks != 2 {
		t.Fatalf("unexpected recovery: value=%q status=%q mutations=%d readbacks=%d", value, status, mutations, readbacks)
	}
}

func TestRunReconciledMutationDoesNotRetryRateLimitRejection(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")

	mutations := 0
	readbacks := 0
	_, _, err := RunReconciledMutation(
		context.Background(),
		func(context.Context) (string, error) {
			mutations++
			return "", &asc.RetryableError{
				Err: &asc.APIError{Code: "RATE_LIMIT_EXCEEDED", StatusCode: http.StatusTooManyRequests},
			}
		},
		func(context.Context) (string, bool, error) {
			readbacks++
			return "", false, nil
		},
	)
	if err == nil {
		t.Fatal("expected rate-limit error, got nil")
	}
	if mutations != 1 {
		t.Fatalf("expected one mutation invocation after the client retry budget was exhausted, got %d", mutations)
	}
	if readbacks != 1 {
		t.Fatalf("expected one reconciliation readback for a rejected request, got %d", readbacks)
	}
	if !asc.IsRetryable(err) {
		t.Fatalf("expected retryable rate-limit error, got %v", err)
	}
	var statusErr interface{ HTTPStatusCode() int }
	if !errors.As(err, &statusErr) || statusErr.HTTPStatusCode() != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 to remain inspectable, got %v", err)
	}
}

func TestIsTransientMutationErrorRejectsUnhonoredRetryDelay(t *testing.T) {
	_, err := asc.WithRetry(context.Background(), func() (struct{}, error) {
		return struct{}{}, &asc.RetryableError{
			Err:        &asc.APIError{Code: "RATE_LIMIT_EXCEEDED", StatusCode: http.StatusTooManyRequests},
			RetryAfter: time.Hour,
		}
	}, asc.RetryOptions{MaxRetries: 1, BaseDelay: time.Millisecond, MaxDelay: time.Second})
	if err == nil {
		t.Fatal("expected unhonored Retry-After error, got nil")
	}
	if !asc.IsRetryDelayExceeded(err) {
		t.Fatalf("expected retry-delay marker, got %v", err)
	}
	if !asc.IsRetryable(err) {
		t.Fatalf("expected original retryable cause to remain inspectable, got %v", err)
	}
	if IsTransientMutationError(context.Background(), err) {
		t.Fatalf("expected unhonored Retry-After to be terminal to mutation recovery, got %v", err)
	}
}

func TestRunReconciledMutationDoesNotReplayRateLimitAfterRetryCancellation(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")

	rateErr := &asc.RetryableError{
		RetryAfter: time.Hour,
		Err:        &asc.APIError{Code: "RATE_LIMIT_EXCEEDED", StatusCode: http.StatusTooManyRequests},
	}
	mutationErr := fmt.Errorf("retry cancelled: %w", errors.Join(context.DeadlineExceeded, rateErr))
	mutations := 0
	readbacks := 0
	_, _, err := RunReconciledMutation(
		context.Background(),
		func(context.Context) (string, error) {
			mutations++
			return "", mutationErr
		},
		func(context.Context) (string, bool, error) {
			readbacks++
			return "", false, nil
		},
	)
	if err == nil {
		t.Fatal("expected retry cancellation error, got nil")
	}
	if mutations != 1 || readbacks != 1 {
		t.Fatalf("expected one mutation and one reconciliation readback, got mutations=%d readbacks=%d", mutations, readbacks)
	}
	if !errors.Is(err, context.DeadlineExceeded) || !asc.IsRetryable(err) {
		t.Fatalf("expected timeout and retryable causes to remain inspectable, got %v", err)
	}
	if got := asc.GetRetryAfter(err); got != time.Hour {
		t.Fatalf("expected Retry-After to remain inspectable, got %s", got)
	}
}

func TestRunReconciledMutationUsesFreshTimeoutForReplay(t *testing.T) {
	t.Setenv("ASC_TIMEOUT", "20ms")
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")
	mutations := 0
	readbacks := 0

	value, status, err := RunReconciledMutation(
		context.Background(),
		func(ctx context.Context) (string, error) {
			mutations++
			if mutations == 1 {
				<-ctx.Done()
				return "", ctx.Err()
			}
			if err := ctx.Err(); err != nil {
				t.Fatalf("replay received expired context: %v", err)
			}
			return "localization-id", nil
		},
		func(context.Context) (string, bool, error) {
			readbacks++
			return "", false, nil
		},
	)
	if err != nil {
		t.Fatalf("RunReconciledMutation() error: %v", err)
	}
	if value != "localization-id" || status != ReconciledMutationApplied || mutations != 2 || readbacks != 2 {
		t.Fatalf("unexpected replay: value=%q status=%q mutations=%d readbacks=%d", value, status, mutations, readbacks)
	}
}

func TestRunReconciledMutationStopsDuringBackoffCancellation(t *testing.T) {
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1h")
	t.Setenv("ASC_MAX_DELAY", "1h")
	ctx, cancel := context.WithCancel(context.Background())
	readbackDone := make(chan struct{})

	go func() {
		<-readbackDone
		cancel()
	}()
	_, _, err := RunReconciledMutation(
		ctx,
		func(context.Context) (string, error) {
			return "", &asc.RetryableError{Err: errors.New("temporary")}
		},
		func(context.Context) (string, bool, error) {
			close(readbackDone)
			return "", false, nil
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestRunReconciledMutationRetriesReadbackChildDeadline(t *testing.T) {
	t.Setenv("ASC_TIMEOUT", "10ms")
	t.Setenv("ASC_MAX_RETRIES", "1")
	t.Setenv("ASC_BASE_DELAY", "1ms")
	t.Setenv("ASC_MAX_DELAY", "1ms")
	readbacks := 0

	value, status, err := RunReconciledMutation(
		context.Background(),
		func(context.Context) (string, error) {
			return "", &asc.RetryableError{Err: errors.New("ambiguous")}
		},
		func(ctx context.Context) (string, bool, error) {
			value, readErr := RetryReadWithFreshTimeout(ctx, func(requestCtx context.Context) (string, error) {
				readbacks++
				if readbacks == 1 {
					<-requestCtx.Done()
					return "", requestCtx.Err()
				}
				return "localization-id", nil
			})
			return value, value != "", readErr
		},
	)
	if err != nil {
		t.Fatalf("RunReconciledMutation() error: %v", err)
	}
	if value != "localization-id" || status != ReconciledMutationRecovered || readbacks != 2 {
		t.Fatalf("unexpected recovery: value=%q status=%q readbacks=%d", value, status, readbacks)
	}
}

func TestRunReconciledMutationDoesNotStartAfterParentDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	mutations := 0

	_, _, err := RunReconciledMutation(
		ctx,
		func(context.Context) (string, error) {
			mutations++
			return "", nil
		},
		func(context.Context) (string, bool, error) { return "", false, nil },
	)
	if !errors.Is(err, context.DeadlineExceeded) || mutations != 0 {
		t.Fatalf("expected parent deadline before mutation, err=%v mutations=%d", err, mutations)
	}
}
