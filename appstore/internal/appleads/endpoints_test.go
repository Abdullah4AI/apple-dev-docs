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

func TestEndpointSpecsMarkOnlyOperationalV5MutationsForRiskConfirmation(t *testing.T) {
	wants := map[string]struct {
		method string
		path   string
	}{
		"budget-orders create":                   {"POST", "v5/budgetorders"},
		"budget-orders update":                   {"PUT", "v5/budgetorders/{boId}"},
		"campaigns create":                       {"POST", "v5/campaigns"},
		"campaigns update":                       {"PUT", "v5/campaigns/{campaignId}"},
		"ad-groups create":                       {"POST", "v5/campaigns/{campaignId}/adgroups"},
		"ad-groups update":                       {"PUT", "v5/campaigns/{campaignId}/adgroups/{adgroupId}"},
		"ads create":                             {"POST", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/ads"},
		"ads update":                             {"PUT", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/ads/{adId}"},
		"targeting-keywords create-bulk":         {"POST", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/targetingkeywords/bulk"},
		"targeting-keywords update-bulk":         {"PUT", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/targetingkeywords/bulk"},
		"campaign-negative-keywords create-bulk": {"POST", "v5/campaigns/{campaignId}/negativekeywords/bulk"},
		"campaign-negative-keywords update-bulk": {"PUT", "v5/campaigns/{campaignId}/negativekeywords/bulk"},
		"ad-group-negative-keywords create-bulk": {"POST", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/negativekeywords/bulk"},
		"ad-group-negative-keywords update-bulk": {"PUT", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/negativekeywords/bulk"},
	}

	got := map[string]EndpointSpec{}
	for _, spec := range EndpointSpecs() {
		if spec.RiskConfirm {
			got[strings.Join(spec.CommandPath, " ")] = spec
		}
	}
	if len(got) != len(wants) {
		t.Fatalf("risk-confirm v5 endpoints = %d, want %d: %+v", len(got), len(wants), got)
	}
	for command, want := range wants {
		spec, ok := got[command]
		if !ok {
			t.Errorf("missing risk confirmation for %q", command)
			continue
		}
		if spec.Method != want.method || spec.Path != want.path {
			t.Errorf("%q risk contract = %s %s, want %s %s", command, spec.Method, spec.Path, want.method, want.path)
		}
		if spec.RiskConfirmBodyField != "" || spec.RiskConfirmBodyValue != "" {
			t.Errorf("%q risk confirmation must be unconditional: %+v", command, spec)
		}
	}

	for _, command := range [][]string{
		{"campaigns", "find"},
		{"reports", "campaigns"},
		{"geo", "resolve"},
		{"creatives", "create"},
		{"impression-share-reports", "create"},
	} {
		spec, ok := EndpointByCommandPath(command...)
		if !ok || spec.RiskConfirm {
			t.Errorf("read-like or benign v5 command %q must not require risk confirmation: %+v", strings.Join(command, " "), spec)
		}
	}
}

func TestPlatformEndpointSpecsCoverAccountAndAppSurface(t *testing.T) {
	specs := PlatformEndpointSpecs()

	wants := map[string]struct {
		method  string
		path    string
		context ContextKind
	}{
		"me view":                       {"GET", "v1/me", ContextNone},
		"acls list":                     {"GET", "v1/acls", ContextNone},
		"orgs view":                     {"GET", "v1/orgs/{id}", ContextNone},
		"ad-accounts create":            {"POST", "v1/ad-accounts", ContextNone},
		"ad-accounts view":              {"GET", "v1/ad-accounts/{id}", ContextAdAccount},
		"ad-accounts update":            {"PUT", "v1/ad-accounts/{id}", ContextAdAccount},
		"advertiser-resources list":     {"GET", "v1/advertiser-resources", ContextNone},
		"apps search":                   {"GET", "v1/search/apps", ContextAdAccount},
		"apps view":                     {"GET", "v1/apps/{adamId}", ContextAdAccount},
		"apps supported-languages find": {"POST", "v1/metadata/apps/supported-languages/query", ContextAdAccount},
		"apps eligibility find":         {"POST", "v1/eligibilities/apps/query", ContextAdAccount},
		"rejection-reasons apps find":   {"POST", "v1/rejection-reasons/apps/query", ContextAdAccount},
		"rejection-reasons apps view":   {"GET", "v1/rejection-reasons/apps/{rejectionReasonId}", ContextAdAccount},
	}

	for _, spec := range specs {
		key := strings.Join(spec.CommandPath, " ")
		want, ok := wants[key]
		if !ok {
			continue
		}
		if spec.Version != APIVersionPlatformV1 || spec.Method != want.method || spec.Path != want.path || spec.Context != want.context {
			t.Fatalf("%s contract = %s %s version=%q context=%v", key, spec.Method, spec.Path, spec.Version, spec.Context)
		}
		delete(wants, key)
	}
	if len(wants) != 0 {
		t.Fatalf("missing platform commands: %+v", wants)
	}

	create, _ := PlatformEndpointByCommandPath("ad-accounts", "create")
	update, _ := PlatformEndpointByCommandPath("ad-accounts", "update")
	if create.BodyOptional || update.BodyOptional || create.BodyKind != BodyObject || update.BodyKind != BodyObject {
		t.Fatal("ad-account create/update must require JSON object files")
	}
	if update.ConfirmBodyField != "delegations" {
		t.Fatalf("ad-account update confirm field = %q, want delegations", update.ConfirmBodyField)
	}
	if !create.RiskConfirm {
		t.Fatal("ad-account create must require risk confirmation")
	}
	if create.RiskConfirmBodyField != "" || create.RiskConfirmBodyValue != "" {
		t.Fatalf("ad-account create risk exemption = %q=%q, want unconditional confirmation", create.RiskConfirmBodyField, create.RiskConfirmBodyValue)
	}

	for _, path := range [][]string{{"apps", "supported-languages", "find"}, {"apps", "eligibility", "find"}, {"rejection-reasons", "apps", "find"}} {
		spec, ok := PlatformEndpointByCommandPath(path...)
		if !ok || !spec.BodyOptional || !spec.RetrySafe {
			t.Fatalf("%q query body must be optional and retry-safe", strings.Join(path, " "))
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
