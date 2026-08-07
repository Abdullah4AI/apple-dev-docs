package users

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeUserRolesAcceptsExactUserRoleEnumCaseInsensitively(t *testing.T) {
	input := "admin, finance, account_holder, sales, marketing, app_manager, developer, access_to_reports, customer_support, create_apps, cloud_managed_developer_id, cloud_managed_app_distribution, generate_individual_keys"
	want := userRoleList()

	got, err := normalizeUserRoles(input, "--roles")
	if err != nil {
		t.Fatalf("normalizeUserRoles() error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeUserRoles() = %#v, want %#v", got, want)
	}
}

func TestNormalizeUserRolesRejectsUnknownRole(t *testing.T) {
	_, err := normalizeUserRoles("developer,not_a_role", "--role")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--role must be one of: ADMIN, FINANCE, ACCOUNT_HOLDER") {
		t.Fatalf("unexpected error: %v", err)
	}
}
