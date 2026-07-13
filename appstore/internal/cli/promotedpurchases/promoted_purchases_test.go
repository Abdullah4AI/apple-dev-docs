package promotedpurchases

import (
	"context"
	"errors"
	"flag"
	"testing"
)

func TestPromotedPurchasesListValidation(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	cmd := PromotedPurchasesListCommand()
	if err := cmd.FlagSet.Parse([]string{}); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}
}
