package cmdtest

import (
	"context"
	"errors"
	"flag"
	"io"
	"testing"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func TestPaginationSelectorsReportCanonicalMissingParameter(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantParameter string
	}{
		{
			name:          "bundle ID capabilities",
			args:          []string{"bundle-ids", "capabilities", "list"},
			wantParameter: "--bundle",
		},
		{
			name:          "bundle ID profiles",
			args:          []string{"bundle-ids", "profiles", "list"},
			wantParameter: "--id",
		},
		{
			name:          "category subcategories",
			args:          []string{"categories", "subcategories"},
			wantParameter: "--category-id",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := RootCommand("1.2.3")
			root.FlagSet.SetOutput(io.Discard)

			if err := root.Parse(test.args); err != nil {
				t.Fatalf("parse error: %v", err)
			}
			err := root.Run(context.Background())
			if !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("expected ErrHelp, got %v", err)
			}

			diagnostic, ok := shared.DiagnosticFromError(err)
			if !ok {
				t.Fatalf("expected structured diagnostic, got %v", err)
			}
			if diagnostic.Code != shared.DiagnosticRequiredInputMissing || diagnostic.Parameter != test.wantParameter {
				t.Fatalf("diagnostic = %+v, want required_input_missing for %q", diagnostic, test.wantParameter)
			}
		})
	}
}
