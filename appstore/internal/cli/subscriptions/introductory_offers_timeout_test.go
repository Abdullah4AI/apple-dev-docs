package subscriptions

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestContextWithSubscriptionIntroductoryOfferCreateTimeoutUsesOperationDefault(t *testing.T) {
	t.Setenv("ASC_TIMEOUT", "")
	t.Setenv("ASC_TIMEOUT_SECONDS", "")

	ctx, cancel := contextWithSubscriptionIntroductoryOfferCreateTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected create operation deadline")
	}
	remaining := time.Until(deadline)
	if remaining < 4*time.Minute+59*time.Second || remaining > 5*time.Minute+time.Second {
		t.Fatalf("expected create operation timeout near 5m, got %v", remaining)
	}

	requestCtx, requestCancel := shared.ContextWithTimeout(ctx)
	defer requestCancel()
	requestDeadline, ok := requestCtx.Deadline()
	if !ok {
		t.Fatal("expected individual request deadline")
	}
	requestRemaining := time.Until(requestDeadline)
	if requestRemaining < 29*time.Second || requestRemaining > 31*time.Second {
		t.Fatalf("expected individual request timeout near 30s, got %v", requestRemaining)
	}
}

func TestContextWithSubscriptionIntroductoryOfferCreateTimeoutRespectsASCTimeout(t *testing.T) {
	t.Setenv("ASC_TIMEOUT", "45s")
	t.Setenv("ASC_TIMEOUT_SECONDS", "")

	ctx, cancel := contextWithSubscriptionIntroductoryOfferCreateTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected create operation deadline")
	}
	remaining := time.Until(deadline)
	if remaining < 44*time.Second || remaining > 46*time.Second {
		t.Fatalf("expected create operation timeout near 45s from ASC_TIMEOUT, got %v", remaining)
	}
}

func TestContextWithSubscriptionIntroductoryOfferCreateTimeoutRespectsASCTimeoutSeconds(t *testing.T) {
	t.Setenv("ASC_TIMEOUT", "")
	if err := os.Unsetenv("ASC_TIMEOUT"); err != nil {
		t.Fatalf("unset ASC_TIMEOUT: %v", err)
	}
	t.Setenv("ASC_TIMEOUT_SECONDS", "45")

	ctx, cancel := contextWithSubscriptionIntroductoryOfferCreateTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected create operation deadline")
	}
	remaining := time.Until(deadline)
	if remaining < 44*time.Second || remaining > 46*time.Second {
		t.Fatalf("expected create operation timeout near 45s from ASC_TIMEOUT_SECONDS, got %v", remaining)
	}
}
