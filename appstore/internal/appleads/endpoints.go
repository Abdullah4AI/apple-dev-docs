package appleads

import "strings"

// BodyKind describes the JSON body shape accepted by an Apple Ads endpoint.
type BodyKind string

const (
	BodyNone   BodyKind = ""
	BodyObject BodyKind = "object"
	BodyArray  BodyKind = "array"
)

// ParamType describes the primitive type of a path or query parameter.
type ParamType string

const (
	ParamString ParamType = "string"
	ParamInt    ParamType = "int"
	ParamBool   ParamType = "bool"
)

// ParamSpec describes a documented Apple Ads path or query parameter.
type ParamSpec struct {
	Name     string
	Flag     string
	Type     ParamType
	Required bool
	Max      int
	Allowed  []string
}

// EndpointSpec is the single source of truth for the Apple Ads command and
// client surface.
type EndpointSpec struct {
	Name             string
	Method           string
	Path             string
	CommandPath      []string
	BodyKind         BodyKind
	BodyType         string
	ResponseType     string
	RequiresOrg      bool
	RequiresConfirm  bool
	PathParams       []ParamSpec
	QueryParams      []ParamSpec
	SupportsPaginate bool
	DefaultListAlias bool
}

const maxAppleAdsPageLimit = 1000

var (
	adamIDParam      = ParamSpec{Name: "adamId", Flag: "adam-id", Type: ParamInt, Required: true}
	adGroupParam     = ParamSpec{Name: "adgroupId", Flag: "ad-group", Type: ParamInt, Required: true}
	adParam          = ParamSpec{Name: "adId", Flag: "ad", Type: ParamInt, Required: true}
	budgetOrderParam = ParamSpec{Name: "boId", Flag: "budget-order", Type: ParamInt, Required: true}
	campaignParam    = ParamSpec{Name: "campaignId", Flag: "campaign", Type: ParamInt, Required: true}
	creativeParam    = ParamSpec{Name: "creativeId", Flag: "creative", Type: ParamInt, Required: true}
	keywordParam     = ParamSpec{Name: "keywordId", Flag: "keyword", Type: ParamInt, Required: true}
	productPageParam = ParamSpec{Name: "productPageId", Flag: "product-page", Type: ParamString, Required: true}
	reasonParam      = ParamSpec{Name: "productPageReasonId", Flag: "reason", Type: ParamInt, Required: true}
	reportParam      = ParamSpec{Name: "reportId", Flag: "report", Type: ParamInt, Required: true}

	limitParam  = ParamSpec{Name: "limit", Flag: "limit", Type: ParamInt, Max: maxAppleAdsPageLimit}
	offsetParam = ParamSpec{Name: "offset", Flag: "offset", Type: ParamInt}
)

func q(name, flag string, typ ParamType, required bool) ParamSpec {
	return ParamSpec{Name: name, Flag: flag, Type: typ, Required: required}
}

func qAllowed(name, flag string, allowed ...string) ParamSpec {
	return ParamSpec{Name: name, Flag: flag, Type: ParamString, Allowed: append([]string(nil), allowed...)}
}

func qLimitOffset() []ParamSpec {
	return []ParamSpec{limitParam, offsetParam}
}

func endpoint(name, method, path string, commandPath []string, bodyKind BodyKind, bodyType, responseType string, pathParams []ParamSpec, queryParams []ParamSpec) EndpointSpec {
	return EndpointSpec{
		Name:         name,
		Method:       method,
		Path:         path,
		CommandPath:  append([]string(nil), commandPath...),
		BodyKind:     bodyKind,
		BodyType:     bodyType,
		ResponseType: responseType,
		RequiresOrg:  true,
		PathParams:   append([]ParamSpec(nil), pathParams...),
		QueryParams:  append([]ParamSpec(nil), queryParams...),
	}
}

// EndpointSpecs returns the current Apple Ads Campaign Management API v5 surface.
func EndpointSpecs() []EndpointSpec {
	specs := []EndpointSpec{
		endpoint("get-user-acl", "GET", "v5/acls", []string{"acls", "list"}, BodyNone, "", "UserAclListResponse", nil, nil),
		endpoint("get-me-details", "GET", "v5/me", []string{"me", "view"}, BodyNone, "", "MeDetailResponse", nil, nil),

		endpoint("search-for-ios-apps", "GET", "v5/search/apps", []string{"apps", "search"}, BodyNone, "", "AppInfoListResponse", nil, []ParamSpec{limitParam, offsetParam, q("query", "query", ParamString, true), q("returnOwnedApps", "return-owned-apps", ParamBool, false)}),
		endpoint("get-app-details", "GET", "v5/apps/{adamId}", []string{"apps", "view"}, BodyNone, "", "MediaDetailResponse", []ParamSpec{adamIDParam}, nil),
		endpoint("get-localized-app-details", "GET", "v5/apps/{adamId}/locale-details", []string{"apps", "localized-details"}, BodyNone, "", "MediaLocaleDetailResponse", []ParamSpec{adamIDParam}, nil),
		endpoint("find-app-eligibility-records", "POST", "v5/apps/{adamId}/eligibilities/find", []string{"apps", "eligibility", "find"}, BodyObject, "Selector", "EligibilityRecordListResponse", []ParamSpec{adamIDParam}, nil),
		endpoint("find-app-assets", "POST", "v5/apps/{adamId}/assets/find", []string{"apps", "assets", "find"}, BodyObject, "Selector", "AppAssetListResponse", []ParamSpec{adamIDParam}, nil),

		endpoint("get-product-pages", "GET", "v5/apps/{adamId}/product-pages", []string{"product-pages", "list"}, BodyNone, "", "ProductPageDetailListResponse", []ParamSpec{adamIDParam}, []ParamSpec{q("name", "name", ParamString, false), qAllowed("states", "states", "HIDDEN", "VISIBLE")}),
		endpoint("get-product-pages-by-identifier", "GET", "v5/apps/{adamId}/product-pages/{productPageId}", []string{"product-pages", "view"}, BodyNone, "", "ProductPageDetailResponse", []ParamSpec{adamIDParam, productPageParam}, nil),
		endpoint("get-product-page-locales", "GET", "v5/apps/{adamId}/product-pages/{productPageId}/locale-details", []string{"product-pages", "locales", "list"}, BodyNone, "", "ProductPageLocaleDetailListResponse", []ParamSpec{adamIDParam, productPageParam}, []ParamSpec{qAllowed("deviceClasses", "device-classes", "IPAD", "IPHONE"), q("expand", "expand", ParamBool, false), q("languageCodes", "language-codes", ParamString, false), q("languages", "languages", ParamString, false)}),
		endpoint("get-supported-countries-or-regions", "GET", "v5/countries-or-regions", []string{"product-pages", "countries", "list"}, BodyNone, "", "CountriesOrRegionsListResponse", nil, []ParamSpec{q("countriesOrRegions", "countries-or-regions", ParamString, false)}),
		endpoint("get-app-preview-device-sizes", "GET", "v5/creativeappmappings/devices", []string{"product-pages", "devices", "list"}, BodyNone, "", "AppPreviewDevicesMappingResponse", nil, nil),

		endpoint("get-all-budget-orders", "GET", "v5/budgetorders", []string{"budget-orders", "list"}, BodyNone, "", "BudgetOrderInfoListResponse", nil, qLimitOffset()),
		endpoint("create-a-budget-order", "POST", "v5/budgetorders", []string{"budget-orders", "create"}, BodyObject, "BudgetOrderCreate", "BudgetOrderInfoResponse", nil, nil),
		endpoint("get-a-budget-order", "GET", "v5/budgetorders/{boId}", []string{"budget-orders", "view"}, BodyNone, "", "BudgetOrderInfoResponse", []ParamSpec{budgetOrderParam}, nil),
		endpoint("update-a-budget-order", "PUT", "v5/budgetorders/{boId}", []string{"budget-orders", "update"}, BodyObject, "BudgetOrderUpdate", "BudgetOrderInfoResponse", []ParamSpec{budgetOrderParam}, nil),

		endpoint("get-all-campaigns", "GET", "v5/campaigns", []string{"campaigns", "list"}, BodyNone, "", "CampaignListResponse", nil, qLimitOffset()),
		endpoint("create-a-campaign", "POST", "v5/campaigns", []string{"campaigns", "create"}, BodyObject, "Campaign", "CampaignResponse", nil, nil),
		endpoint("find-campaigns", "POST", "v5/campaigns/find", []string{"campaigns", "find"}, BodyObject, "Selector", "CampaignListResponse", nil, nil),
		endpoint("delete-a-campaign", "DELETE", "v5/campaigns/{campaignId}", []string{"campaigns", "delete"}, BodyNone, "", "VoidResponse", []ParamSpec{campaignParam}, nil),
		endpoint("get-a-campaign", "GET", "v5/campaigns/{campaignId}", []string{"campaigns", "view"}, BodyNone, "", "CampaignResponse", []ParamSpec{campaignParam}, nil),
		endpoint("update-a-campaign", "PUT", "v5/campaigns/{campaignId}", []string{"campaigns", "update"}, BodyObject, "UpdateCampaignRequest", "CampaignResponse", []ParamSpec{campaignParam}, nil),

		endpoint("get-all-ad-groups", "GET", "v5/campaigns/{campaignId}/adgroups", []string{"ad-groups", "list"}, BodyNone, "", "AdGroupListResponse", []ParamSpec{campaignParam}, qLimitOffset()),
		endpoint("create-an-ad-group", "POST", "v5/campaigns/{campaignId}/adgroups", []string{"ad-groups", "create"}, BodyObject, "AdGroup", "AdGroupResponse", []ParamSpec{campaignParam}, nil),
		endpoint("find-ad-groups", "POST", "v5/campaigns/{campaignId}/adgroups/find", []string{"ad-groups", "find"}, BodyObject, "Selector", "AdGroupListResponse", []ParamSpec{campaignParam}, nil),
		endpoint("find-ad-groups-across-organization", "POST", "v5/adgroups/find", []string{"ad-groups", "find-org"}, BodyObject, "Selector", "AdGroupListResponse", nil, nil),
		endpoint("delete-an-ad-group", "DELETE", "v5/campaigns/{campaignId}/adgroups/{adgroupId}", []string{"ad-groups", "delete"}, BodyNone, "", "VoidResponse", []ParamSpec{campaignParam, adGroupParam}, nil),
		endpoint("get-an-ad-group", "GET", "v5/campaigns/{campaignId}/adgroups/{adgroupId}", []string{"ad-groups", "view"}, BodyNone, "", "AdGroupResponse", []ParamSpec{campaignParam, adGroupParam}, nil),
		endpoint("update-an-ad-group", "PUT", "v5/campaigns/{campaignId}/adgroups/{adgroupId}", []string{"ad-groups", "update"}, BodyObject, "AdGroupUpdate", "AdGroupResponse", []ParamSpec{campaignParam, adGroupParam}, nil),

		endpoint("get-all-ads", "GET", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/ads", []string{"ads", "list"}, BodyNone, "", "AdListResponse", []ParamSpec{campaignParam, adGroupParam}, nil),
		endpoint("create-an-ad", "POST", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/ads", []string{"ads", "create"}, BodyObject, "AdCreate", "AdResponse", []ParamSpec{campaignParam, adGroupParam}, nil),
		endpoint("delete-an-ad", "DELETE", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/ads/{adId}", []string{"ads", "delete"}, BodyNone, "", "VoidResponse", []ParamSpec{campaignParam, adGroupParam, adParam}, nil),
		endpoint("get-an-ad", "GET", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/ads/{adId}", []string{"ads", "view"}, BodyNone, "", "AdResponse", []ParamSpec{campaignParam, adGroupParam, adParam}, nil),
		endpoint("update-an-ad", "PUT", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/ads/{adId}", []string{"ads", "update"}, BodyObject, "AdUpdate", "AdResponse", []ParamSpec{campaignParam, adGroupParam, adParam}, nil),
		endpoint("find-ads", "POST", "v5/campaigns/{campaignId}/ads/find", []string{"ads", "find"}, BodyObject, "Selector", "AdListResponse", []ParamSpec{campaignParam}, nil),
		endpoint("find-ads-across-organization", "POST", "v5/ads/find", []string{"ads", "find-org"}, BodyObject, "Selector", "AdListResponse", nil, nil),

		endpoint("get-all-targeting-keywords-in-an-ad-group", "GET", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/targetingkeywords", []string{"targeting-keywords", "list"}, BodyNone, "", "KeywordListResponse", []ParamSpec{campaignParam, adGroupParam}, qLimitOffset()),
		endpoint("find-targeting-keywords-in-a-campaign", "POST", "v5/campaigns/{campaignId}/adgroups/targetingkeywords/find", []string{"targeting-keywords", "find"}, BodyObject, "Selector", "KeywordListResponse", []ParamSpec{campaignParam}, nil),
		endpoint("get-a-targeting-keyword-in-an-ad-group", "GET", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/targetingkeywords/{keywordId}", []string{"targeting-keywords", "view"}, BodyNone, "", "KeywordResponse", []ParamSpec{campaignParam, adGroupParam, keywordParam}, nil),
		endpoint("create-targeting-keywords", "POST", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/targetingkeywords/bulk", []string{"targeting-keywords", "create-bulk"}, BodyArray, "[Keyword]", "KeywordListResponse", []ParamSpec{campaignParam, adGroupParam}, nil),
		endpoint("update-targeting-keywords", "PUT", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/targetingkeywords/bulk", []string{"targeting-keywords", "update-bulk"}, BodyArray, "[KeywordUpdateRequest]", "KeywordListResponse", []ParamSpec{campaignParam, adGroupParam}, nil),
		endpoint("delete-a-targeting-keyword", "DELETE", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/targetingkeywords/{keywordId}", []string{"targeting-keywords", "delete"}, BodyNone, "", "VoidResponse", []ParamSpec{campaignParam, adGroupParam, keywordParam}, nil),
		endpoint("delete-targeting-keywords", "POST", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/targetingkeywords/delete/bulk", []string{"targeting-keywords", "delete-bulk"}, BodyArray, "[int64]", "IntegerResponse", []ParamSpec{campaignParam, adGroupParam}, nil),

		endpoint("get-all-campaign-negative-keywords", "GET", "v5/campaigns/{campaignId}/negativekeywords", []string{"campaign-negative-keywords", "list"}, BodyNone, "", "NegativeKeywordListResponse", []ParamSpec{campaignParam}, qLimitOffset()),
		endpoint("find-campaign-negative-keywords", "POST", "v5/campaigns/{campaignId}/negativekeywords/find", []string{"campaign-negative-keywords", "find"}, BodyObject, "Selector", "NegativeKeywordListResponse", []ParamSpec{campaignParam}, nil),
		endpoint("get-a-campaign-negative-keyword", "GET", "v5/campaigns/{campaignId}/negativekeywords/{keywordId}", []string{"campaign-negative-keywords", "view"}, BodyNone, "", "NegativeKeywordResponse", []ParamSpec{campaignParam, keywordParam}, nil),
		endpoint("create-campaign-negative-keywords", "POST", "v5/campaigns/{campaignId}/negativekeywords/bulk", []string{"campaign-negative-keywords", "create-bulk"}, BodyArray, "[NegativeKeyword]", "NegativeKeywordListResponse", []ParamSpec{campaignParam}, nil),
		endpoint("update-campaign-negative-keywords", "PUT", "v5/campaigns/{campaignId}/negativekeywords/bulk", []string{"campaign-negative-keywords", "update-bulk"}, BodyArray, "[NegativeKeyword]", "NegativeKeywordListResponse", []ParamSpec{campaignParam}, nil),
		endpoint("delete-campaign-negative-keywords", "POST", "v5/campaigns/{campaignId}/negativekeywords/delete/bulk", []string{"campaign-negative-keywords", "delete-bulk"}, BodyArray, "[int64]", "IntegerResponse", []ParamSpec{campaignParam}, nil),

		endpoint("get-all-ad-group-negative-keywords", "GET", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/negativekeywords", []string{"ad-group-negative-keywords", "list"}, BodyNone, "", "NegativeKeywordListResponse", []ParamSpec{campaignParam, adGroupParam}, qLimitOffset()),
		endpoint("find-ad-group-negative-keywords", "POST", "v5/campaigns/{campaignId}/adgroups/negativekeywords/find", []string{"ad-group-negative-keywords", "find"}, BodyObject, "Selector", "NegativeKeywordListResponse", []ParamSpec{campaignParam}, nil),
		endpoint("get-an-ad-group-negative-keyword", "GET", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/negativekeywords/{keywordId}", []string{"ad-group-negative-keywords", "view"}, BodyNone, "", "NegativeKeywordResponse", []ParamSpec{campaignParam, adGroupParam, keywordParam}, nil),
		endpoint("create-ad-group-negative-keywords", "POST", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/negativekeywords/bulk", []string{"ad-group-negative-keywords", "create-bulk"}, BodyArray, "[NegativeKeyword]", "NegativeKeywordListResponse", []ParamSpec{campaignParam, adGroupParam}, nil),
		endpoint("update-ad-group-negative-keywords", "PUT", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/negativekeywords/bulk", []string{"ad-group-negative-keywords", "update-bulk"}, BodyArray, "[NegativeKeyword]", "NegativeKeywordListResponse", []ParamSpec{campaignParam, adGroupParam}, nil),
		endpoint("delete-ad-group-negative-keywords", "POST", "v5/campaigns/{campaignId}/adgroups/{adgroupId}/negativekeywords/delete/bulk", []string{"ad-group-negative-keywords", "delete-bulk"}, BodyArray, "[int64]", "IntegerResponse", []ParamSpec{campaignParam, adGroupParam}, nil),

		endpoint("search-for-geolocations", "GET", "v5/search/geo", []string{"geo", "search"}, BodyNone, "", "SearchEntityListResponse", nil, []ParamSpec{q("countrycode", "country-code", ParamString, false), q("entity", "entity", ParamString, false), limitParam, offsetParam, q("query", "query", ParamString, false)}),
		endpoint("get-a-list-of-geo-locations", "POST", "v5/search/geo", []string{"geo", "resolve"}, BodyArray, "[GeoRequest]", "SearchEntityListResponse", nil, qLimitOffset()),

		endpoint("get-all-creatives", "GET", "v5/creatives", []string{"creatives", "list"}, BodyNone, "", "CreativeListResponse", nil, qLimitOffset()),
		endpoint("create-a-creative", "POST", "v5/creatives", []string{"creatives", "create"}, BodyObject, "(CustomProductPageCreative | DefaultProductPageCreative)", "CreativeResponse", nil, nil),
		endpoint("find-creatives", "POST", "v5/creatives/find", []string{"creatives", "find"}, BodyObject, "Selector", "CreativeListResponse", nil, nil),
		endpoint("get-a-creative", "GET", "v5/creatives/{creativeId}", []string{"creatives", "view"}, BodyNone, "", "CreativeResponse", []ParamSpec{creativeParam}, []ParamSpec{q("includeDeletedCreativeSetAssets", "include-deleted-creative-set-assets", ParamBool, false)}),

		endpoint("find-ad-creative-rejection-reasons", "POST", "v5/product-page-reasons/find", []string{"rejection-reasons", "find"}, BodyObject, "Selector", "ProductPageReasonListResponse", nil, nil),
		endpoint("gets-a-product-page-reason", "GET", "v5/product-page-reasons/{productPageReasonId}", []string{"rejection-reasons", "view"}, BodyNone, "", "ProductPageReasonResponse", []ParamSpec{reasonParam}, nil),

		endpoint("get-campaign-level-reports", "POST", "v5/reports/campaigns", []string{"reports", "campaigns"}, BodyObject, "ReportingRequest", "ReportingResponseBody", nil, nil),
		endpoint("get-ad-group-level-reports", "POST", "v5/reports/campaigns/{campaignId}/adgroups", []string{"reports", "ad-groups"}, BodyObject, "ReportingRequest", "ReportingResponseBody", []ParamSpec{campaignParam}, nil),
		endpoint("get-keyword-level-reports", "POST", "v5/reports/campaigns/{campaignId}/keywords", []string{"reports", "keywords"}, BodyObject, "ReportingRequest", "ReportingResponseBody", []ParamSpec{campaignParam}, nil),
		endpoint("get-search-term-level-reports", "POST", "v5/reports/campaigns/{campaignId}/searchterms", []string{"reports", "search-terms"}, BodyObject, "ReportingRequest", "ReportingResponseBody", []ParamSpec{campaignParam}, nil),
		endpoint("get-ad-level-reports", "POST", "v5/reports/campaigns/{campaignId}/ads", []string{"reports", "ads"}, BodyObject, "ReportingRequest", "ReportingResponseBody", []ParamSpec{campaignParam}, nil),
		endpoint("get-keyword-level-within-ad-group-reports", "POST", "v5/reports/campaigns/{campaignId}/adgroups/{adgroupId}/keywords", []string{"reports", "ad-group-keywords"}, BodyObject, "ReportingRequest", "ReportingResponseBody", []ParamSpec{campaignParam, adGroupParam}, nil),
		endpoint("get-search-term-level-within-ad-group-reports", "POST", "v5/reports/campaigns/{campaignId}/adgroups/{adgroupId}/searchterms", []string{"reports", "ad-group-search-terms"}, BodyObject, "ReportingRequest", "ReportingResponseBody", []ParamSpec{campaignParam, adGroupParam}, nil),

		endpoint("get-all-impression-share-reports", "GET", "v5/custom-reports", []string{"impression-share-reports", "list"}, BodyNone, "", "CustomReportResponseBody", nil, []ParamSpec{q("field", "field", ParamString, false), {Name: "limit", Flag: "limit", Type: ParamInt, Max: 50}, offsetParam, q("sortOrder", "sort-order", ParamString, false)}),
		endpoint("impression-share-report", "POST", "v5/custom-reports", []string{"impression-share-reports", "create"}, BodyObject, "CustomReportRequest", "CustomReportResponseBody", nil, nil),
		endpoint("get-a-single-impression-share-report", "GET", "v5/custom-reports/{reportId}", []string{"impression-share-reports", "view"}, BodyNone, "", "CustomReportResponseBody", []ParamSpec{reportParam}, nil),
	}

	for i := range specs {
		specs[i].RequiresConfirm = specs[i].Method == "DELETE" || strings.Contains(specs[i].Path, "/delete/bulk")
		specs[i].SupportsPaginate = hasLimitOffset(specs[i].QueryParams)
		specs[i].DefaultListAlias = len(specs[i].CommandPath) == 2 && (specs[i].CommandPath[1] == "list" || specs[i].Name == "get-me-details")
		if specs[i].Name == "get-user-acl" || specs[i].Name == "get-me-details" {
			specs[i].RequiresOrg = false
		}
	}
	return specs
}

func hasLimitOffset(params []ParamSpec) bool {
	hasLimit := false
	hasOffset := false
	for _, param := range params {
		switch param.Name {
		case "limit":
			hasLimit = true
		case "offset":
			hasOffset = true
		}
	}
	return hasLimit && hasOffset
}

// EndpointByCommandPath returns a spec by command path.
func EndpointByCommandPath(path ...string) (EndpointSpec, bool) {
	joined := strings.Join(path, " ")
	for _, spec := range EndpointSpecs() {
		if strings.Join(spec.CommandPath, " ") == joined {
			return spec, true
		}
	}
	return EndpointSpec{}, false
}
