package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromotedPurchasesValidationErrors(t *testing.T) {
	t.Setenv("ASC_APP_ID", "")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "promoted-purchases list missing app",
			args:    []string{"promoted-purchases", "list"},
			wantErr: "--app is required",
		},
		{
			name:    "promoted-purchases list invalid limit",
			args:    []string{"promoted-purchases", "list", "--app", "APP_ID", "--limit", "201"},
			wantErr: "--limit must be between 1 and 200",
		},
		{
			name:    "promoted-purchases get missing id",
			args:    []string{"promoted-purchases", "get"},
			wantErr: "--promoted-purchase-id is required",
		},
		{
			name:    "promoted-purchases create missing app",
			args:    []string{"promoted-purchases", "create", "--product-id", "PRODUCT_ID", "--product-type", "SUBSCRIPTION", "--visible-for-all-users", "true"},
			wantErr: "--app is required",
		},
		{
			name:    "promoted-purchases create missing product id",
			args:    []string{"promoted-purchases", "create", "--app", "APP_ID", "--product-type", "SUBSCRIPTION", "--visible-for-all-users", "true"},
			wantErr: "--product-id is required",
		},
		{
			name:    "promoted-purchases create missing product type",
			args:    []string{"promoted-purchases", "create", "--app", "APP_ID", "--product-id", "PRODUCT_ID", "--visible-for-all-users", "true"},
			wantErr: "--product-type is required",
		},
		{
			name:    "promoted-purchases create invalid product type",
			args:    []string{"promoted-purchases", "create", "--app", "APP_ID", "--product-id", "PRODUCT_ID", "--product-type", "INVALID", "--visible-for-all-users", "true"},
			wantErr: "--product-type must be one of",
		},
		{
			name:    "promoted-purchases create missing visibility",
			args:    []string{"promoted-purchases", "create", "--app", "APP_ID", "--product-id", "PRODUCT_ID", "--product-type", "SUBSCRIPTION"},
			wantErr: "--visible-for-all-users is required",
		},
		{
			name:    "promoted-purchases update missing id",
			args:    []string{"promoted-purchases", "update", "--visible-for-all-users", "true"},
			wantErr: "--promoted-purchase-id is required",
		},
		{
			name:    "promoted-purchases update missing updates",
			args:    []string{"promoted-purchases", "update", "--promoted-purchase-id", "PROMO_ID"},
			wantErr: "at least one update flag is required",
		},
		{
			name:    "promoted-purchases delete missing confirm",
			args:    []string{"promoted-purchases", "delete", "--promoted-purchase-id", "PROMO_ID"},
			wantErr: "--confirm is required",
		},
		{
			name:    "promoted-purchases delete missing id",
			args:    []string{"promoted-purchases", "delete", "--confirm"},
			wantErr: "--promoted-purchase-id is required",
		},
		{
			name:    "promoted-purchases link missing app",
			args:    []string{"promoted-purchases", "link", "--promoted-purchase-id", "PROMO_ID"},
			wantErr: "--app is required",
		},
		{
			name:    "promoted-purchases link missing promoted purchase id",
			args:    []string{"promoted-purchases", "link", "--app", "APP_ID"},
			wantErr: "--promoted-purchase-id is required",
		},
		{
			name:    "promoted-purchases link clear missing confirm",
			args:    []string{"promoted-purchases", "link", "--app", "APP_ID", "--clear"},
			wantErr: "--confirm is required with --clear",
		},
		{
			name:    "promoted-purchases link clear with ids",
			args:    []string{"promoted-purchases", "link", "--app", "APP_ID", "--clear", "--confirm", "--promoted-purchase-id", "PROMO_ID"},
			wantErr: "--clear cannot be used with --promoted-purchase-id",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			stdout, stderr := captureOutput(t, func() {
				if err := root.Parse(test.args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := root.Run(context.Background())
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected ErrHelp, got %v", err)
				}
			})

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected error %q, got %q", test.wantErr, stderr)
			}
		})
	}
}
