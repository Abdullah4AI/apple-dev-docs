package passtypeids

import (
	"context"
	"errors"
	"flag"
	"testing"
)

func TestPassTypeIDsCommandShape(t *testing.T) {
	cmd := PassTypeIDsCommand()
	if cmd == nil {
		t.Fatal("expected pass-type-ids command")
		return
	}
	if cmd.Name != "pass-type-ids" {
		t.Fatalf("unexpected command name: %q", cmd.Name)
	}
	if len(cmd.Subcommands) != 6 {
		t.Fatalf("expected 6 subcommands, got %d", len(cmd.Subcommands))
	}
	if got := PassTypeIDsCommand(); got == nil {
		t.Fatal("expected Command wrapper to return command")
	}
}

func TestPassTypeIDsValidationErrors(t *testing.T) {
	t.Run("get missing pass-type-id", func(t *testing.T) {
		cmd := PassTypeIDsGetCommand()
		if err := cmd.FlagSet.Parse([]string{}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	t.Run("create missing identifier", func(t *testing.T) {
		cmd := PassTypeIDsCreateCommand()
		if err := cmd.FlagSet.Parse([]string{"--name", "Example"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	t.Run("update missing name", func(t *testing.T) {
		cmd := PassTypeIDsUpdateCommand()
		if err := cmd.FlagSet.Parse([]string{"--pass-type-id", "PASS_ID"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	t.Run("delete missing confirm", func(t *testing.T) {
		cmd := PassTypeIDsDeleteCommand()
		if err := cmd.FlagSet.Parse([]string{"--pass-type-id", "PASS_ID"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})

	t.Run("certificates list missing pass-type-id", func(t *testing.T) {
		cmd := PassTypeIDCertificatesListCommand()
		if err := cmd.FlagSet.Parse([]string{}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := cmd.Exec(context.Background(), nil); !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
	})
}

func TestPassTypeIDHelpers(t *testing.T) {
	if got, err := normalizePassTypeIDInclude("certificates"); err != nil || len(got) != 1 {
		t.Fatalf("expected valid include, got %v err=%v", got, err)
	}
	if _, err := normalizePassTypeIDInclude("bad"); err == nil {
		t.Fatal("expected include validation error")
	}

	if _, err := normalizePassTypeIDFields("invalid", "--fields"); err == nil {
		t.Fatal("expected fields validation error")
	}
}

func TestPassTypeIDFromCertificatesNextURL(t *testing.T) {
	tests := []struct {
		name         string
		next         string
		relationship bool
		want         string
	}{
		{
			name: "certificates",
			next: "https://api.appstoreconnect.apple.com/v1/passTypeIds/pass-1/certificates?cursor=AQ",
			want: "pass-1",
		},
		{
			name:         "certificate relationships",
			next:         "https://api.appstoreconnect.apple.com/v1/passTypeIds/pass-1/relationships/certificates?cursor=AQ",
			relationship: true,
			want:         "pass-1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := passTypeIDFromCertificatesNextURL(test.next, test.relationship)
			if err != nil {
				t.Fatalf("passTypeIDFromCertificatesNextURL() error: %v", err)
			}
			if got != test.want {
				t.Fatalf("passTypeIDFromCertificatesNextURL() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPassTypeIDFromCertificatesNextURLRejectsWrongEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		next         string
		relationship bool
	}{
		{name: "users endpoint", next: "https://api.appstoreconnect.apple.com/v1/users?cursor=AQ"},
		{name: "missing pass type ID", next: "https://api.appstoreconnect.apple.com/v1/passTypeIds//certificates?cursor=AQ"},
		{name: "relationship passed to list", next: "https://api.appstoreconnect.apple.com/v1/passTypeIds/pass-1/relationships/certificates?cursor=AQ"},
		{name: "list passed to relationship", next: "https://api.appstoreconnect.apple.com/v1/passTypeIds/pass-1/certificates?cursor=AQ", relationship: true},
		{name: "encoded slash in ID", next: "https://api.appstoreconnect.apple.com/v1/passTypeIds/pass%2Fone/certificates?cursor=AQ"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := passTypeIDFromCertificatesNextURL(test.next, test.relationship); err == nil {
				t.Fatalf("expected error for %q", test.next)
			}
		})
	}
}
