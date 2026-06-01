package appleads

import (
	"strings"
	"testing"
)

func TestEndpointSpecsCoverCampaignManagementAPI5Surface(t *testing.T) {
	specs := EndpointSpecs()
	if got, want := len(specs), 73; got != want {
		t.Fatalf("EndpointSpecs() count = %d, want %d", got, want)
	}

	names := map[string]struct{}{}
	commandPaths := map[string]struct{}{}
	for _, spec := range specs {
		if strings.TrimSpace(spec.Name) == "" {
			t.Fatalf("empty endpoint name in %+v", spec)
		}
		if strings.TrimSpace(spec.Method) == "" || strings.TrimSpace(spec.Path) == "" {
			t.Fatalf("endpoint %q is missing method or path", spec.Name)
		}
		if len(spec.CommandPath) == 0 {
			t.Fatalf("endpoint %q is missing command path", spec.Name)
		}
		if _, ok := names[spec.Name]; ok {
			t.Fatalf("duplicate endpoint name %q", spec.Name)
		}
		names[spec.Name] = struct{}{}

		commandPath := strings.Join(spec.CommandPath, " ")
		if _, ok := commandPaths[commandPath]; ok {
			t.Fatalf("duplicate command path %q", commandPath)
		}
		commandPaths[commandPath] = struct{}{}

		if (spec.Method == "DELETE" || strings.Contains(spec.Path, "/delete/bulk")) && !spec.RequiresConfirm {
			t.Fatalf("%s %s should require --confirm", spec.Method, spec.Path)
		}
		if spec.SupportsPaginate && !hasLimitOffset(spec.QueryParams) {
			t.Fatalf("%q supports paginate without limit+offset params", spec.Name)
		}
	}

	for _, path := range []string{
		"me view",
		"campaigns list",
		"ad-groups find-org",
		"targeting-keywords delete-bulk",
		"geo resolve",
		"reports ad-group-search-terms",
		"impression-share-reports list",
	} {
		if _, ok := commandPaths[path]; !ok {
			t.Fatalf("expected command path %q", path)
		}
	}
}

func TestEndpointSpecsAuthenticationAndPaginationMetadata(t *testing.T) {
	for _, path := range [][]string{{"me", "view"}, {"acls", "list"}} {
		spec, ok := EndpointByCommandPath(path...)
		if !ok {
			t.Fatalf("missing endpoint %q", strings.Join(path, " "))
		}
		if spec.RequiresOrg {
			t.Fatalf("%q must not require X-AP-Context", strings.Join(path, " "))
		}
	}

	campaigns, ok := EndpointByCommandPath("campaigns", "list")
	if !ok {
		t.Fatal("missing campaigns list endpoint")
	}
	if !campaigns.RequiresOrg {
		t.Fatal("campaigns list should require org context")
	}
	if !campaigns.SupportsPaginate || MaxPageLimit(campaigns) != 1000 {
		t.Fatalf("campaigns list pagination metadata = supports %t max %d, want true max 1000", campaigns.SupportsPaginate, MaxPageLimit(campaigns))
	}

	customReports, ok := EndpointByCommandPath("impression-share-reports", "list")
	if !ok {
		t.Fatal("missing impression-share-reports list endpoint")
	}
	if !customReports.SupportsPaginate || MaxPageLimit(customReports) != 50 {
		t.Fatalf("custom reports pagination metadata = supports %t max %d, want true max 50", customReports.SupportsPaginate, MaxPageLimit(customReports))
	}

	reports, ok := EndpointByCommandPath("reports", "campaigns")
	if !ok {
		t.Fatal("missing reports campaigns endpoint")
	}
	if reports.SupportsPaginate {
		t.Fatal("reporting request body endpoints must not expose automatic query pagination")
	}
}

func TestEndpointSpecsValidatedCSVQueryParameters(t *testing.T) {
	productPages, ok := EndpointByCommandPath("product-pages", "list")
	if !ok {
		t.Fatal("missing product-pages list endpoint")
	}
	states := findQueryParam(productPages, "states")
	if got, want := strings.Join(states.Allowed, ","), "HIDDEN,VISIBLE"; got != want {
		t.Fatalf("states allowed values = %q, want %q", got, want)
	}

	locales, ok := EndpointByCommandPath("product-pages", "locales", "list")
	if !ok {
		t.Fatal("missing product-pages locales list endpoint")
	}
	deviceClasses := findQueryParam(locales, "deviceClasses")
	if got, want := strings.Join(deviceClasses.Allowed, ","), "IPAD,IPHONE"; got != want {
		t.Fatalf("deviceClasses allowed values = %q, want %q", got, want)
	}
}

func findQueryParam(spec EndpointSpec, name string) ParamSpec {
	for _, param := range spec.QueryParams {
		if param.Name == name {
			return param
		}
	}
	return ParamSpec{}
}
