package appleads

import "net/http"

func platformCampaignEndpointSpecs() []EndpointSpec {
	resourceID := func(flag string) ParamSpec {
		return ParamSpec{Name: "id", Flag: flag, Type: ParamString, Required: true}
	}
	adGroupID := resourceID("ad-group")
	adID := resourceID("ad")
	sharedBudgetID := resourceID("budget-order")
	campaignID := resourceID("campaign")
	keywordID := resourceID("keyword")
	negativeKeywordID := resourceID("negative-keyword")

	platform := func(name, method, path string, commandPath []string, context ContextKind, bodyKind BodyKind, bodyOptional bool, bodyType, responseType string, pathParams, queryParams []ParamSpec) EndpointSpec {
		return EndpointSpec{
			Name:            name,
			Method:          method,
			Path:            path,
			Version:         APIVersionPlatformV1,
			Context:         context,
			CommandPath:     append([]string(nil), commandPath...),
			BodyKind:        bodyKind,
			BodyOptional:    bodyOptional,
			BodyType:        bodyType,
			ResponseType:    responseType,
			RequiresConfirm: method == http.MethodDelete,
			PathParams:      append([]ParamSpec(nil), pathParams...),
			QueryParams:     append([]ParamSpec(nil), queryParams...),
		}
	}
	query := func(name, path string, commandPath []string, context ContextKind, responseType string, pathParams []ParamSpec) EndpointSpec {
		spec := platform(name, http.MethodPost, path, commandPath, context, BodyObject, true, "QueryRequest", responseType, pathParams, nil)
		spec.RetrySafe = true
		return spec
	}
	requiredSelectorQuery := func(name, path string, commandPath []string, context ContextKind, responseType string, pathParams []ParamSpec, hint string) EndpointSpec {
		spec := query(name, path, commandPath, context, responseType, pathParams)
		spec.CLIRequiresBody = true
		spec.BodyFileExample = "query.json"
		spec.BodyHint = hint
		return spec
	}
	spendRisk := func(spec EndpointSpec) EndpointSpec {
		spec.RiskConfirm = true
		return spec
	}

	keywordSelectorHint := "Selector filters: include at least one filter for id, adGroupId, or campaignId."
	negativeKeywordSelectorHint := "Selector filters: include a filter for id or adGroupId. For campaign-level negative keywords, combine campaignId with an adGroupId IS_NULL filter."
	campaignCreate := platform("platform-create-campaign", http.MethodPost, "v1/campaigns", []string{"campaigns", "create"}, ContextAdAccount, BodyObject, false, "CampaignCreate", "CampaignResponse", nil, nil)
	campaignCreate.BodyHint = "Required fields: adAccountId, billingEvent, dailyBudget, name, promotedObjectId, promotedObjectType, and targeting. For agent-safe creation, set top-level {\"status\":\"PAUSED\"}; another status or an omitted status requires --confirm."
	campaignCreate.BodyFileExample = "campaign.json"
	campaignCreate.RiskConfirm = true
	campaignCreate.RiskConfirmBodyField = "status"
	campaignCreate.RiskConfirmBodyValue = "PAUSED"
	sharedBudgetCreate := platform("platform-create-shared-budget", http.MethodPost, "v1/shared-budgets", []string{"budget-orders", "create"}, ContextNone, BodyObject, false, "SharedBudgetCreate", "SharedBudgetResponse", nil, nil)
	sharedBudgetCreate.BodyHint = "This creates a billing budget order and requires --confirm. The ad account must have an Apple Ads line of credit (LOC) configured before creation."
	sharedBudgetCreate.BodyFileExample = "shared-budget.json"
	sharedBudgetCreate.RiskConfirm = true
	sharedBudgetUpdate := platform("platform-update-shared-budget", http.MethodPut, "v1/shared-budgets/{id}", []string{"budget-orders", "update"}, ContextNone, BodyObject, false, "SharedBudgetUpdate", "SharedBudgetResponse", []ParamSpec{sharedBudgetID}, nil)
	sharedBudgetUpdate.BodyHint = "This updates a billing budget order and always requires --confirm."
	sharedBudgetUpdate.BodyFileExample = "shared-budget-update.json"
	sharedBudgetUpdate.RiskConfirm = true
	campaignUpdate := platform("platform-update-campaign", http.MethodPut, "v1/campaigns/{id}", []string{"campaigns", "update"}, ContextAdAccount, BodyObject, false, "CampaignUpdate", "CampaignResponse", []ParamSpec{campaignID}, nil)
	campaignUpdate.BodyHint = "A payload containing only name and status=PAUSED may proceed without --confirm. Any dailyBudget, budgetOrder, targeting, bid, start, resume, enabled, or non-PAUSED status change requires --confirm."
	campaignUpdate.BodyFileExample = "campaign-update.json"
	campaignUpdate.RiskConfirm = true
	campaignUpdate.RiskConfirmBodyField = "status"
	campaignUpdate.RiskConfirmBodyValue = "PAUSED"
	campaignUpdate.RiskConfirmAllowedBodyFields = []string{"name", "status"}

	return []EndpointSpec{
		spendRisk(platform("platform-create-ad-group", http.MethodPost, "v1/adgroups", []string{"ad-groups", "create"}, ContextAdAccount, BodyObject, false, "AdGroupCreate", "AdGroupResponse", nil, nil)),
		query("platform-query-ad-groups", "v1/adgroups/query", []string{"ad-groups", "find"}, ContextAdAccount, "AdGroupQueryResponse", nil),
		platform("platform-delete-ad-group", http.MethodDelete, "v1/adgroups/{id}", []string{"ad-groups", "delete"}, ContextAdAccount, BodyNone, false, "", "Response", []ParamSpec{adGroupID}, nil),
		platform("platform-get-ad-group", http.MethodGet, "v1/adgroups/{id}", []string{"ad-groups", "view"}, ContextAdAccount, BodyNone, false, "", "AdGroupResponse", []ParamSpec{adGroupID}, nil),
		spendRisk(platform("platform-update-ad-group", http.MethodPut, "v1/adgroups/{id}", []string{"ad-groups", "update"}, ContextAdAccount, BodyObject, false, "AdGroupUpdate", "AdGroupResponse", []ParamSpec{adGroupID}, nil)),

		spendRisk(platform("platform-create-ad", http.MethodPost, "v1/ads", []string{"ads", "create"}, ContextAdAccount, BodyObject, false, "AdCreate", "AdResponse", nil, nil)),
		query("platform-query-ads", "v1/ads/query", []string{"ads", "find"}, ContextAdAccount, "AdQueryResponse", nil),
		platform("platform-delete-ad", http.MethodDelete, "v1/ads/{id}", []string{"ads", "delete"}, ContextAdAccount, BodyNone, false, "", "Response", []ParamSpec{adID}, nil),
		platform("platform-get-ad", http.MethodGet, "v1/ads/{id}", []string{"ads", "view"}, ContextAdAccount, BodyNone, false, "", "AdResponse", []ParamSpec{adID}, nil),
		spendRisk(platform("platform-update-ad", http.MethodPut, "v1/ads/{id}", []string{"ads", "update"}, ContextAdAccount, BodyObject, false, "AdUpdate", "AdResponse", []ParamSpec{adID}, nil)),

		sharedBudgetCreate,
		query("platform-query-shared-budgets", "v1/shared-budgets/query", []string{"budget-orders", "find"}, ContextAdAccountOptional, "SharedBudgetQueryResponse", nil),
		platform("platform-delete-shared-budget", http.MethodDelete, "v1/shared-budgets/{id}", []string{"budget-orders", "delete"}, ContextNone, BodyNone, false, "", "Response", []ParamSpec{sharedBudgetID}, nil),
		platform("platform-get-shared-budget", http.MethodGet, "v1/shared-budgets/{id}", []string{"budget-orders", "view"}, ContextAdAccountOptional, BodyNone, false, "", "SharedBudgetResponse", []ParamSpec{sharedBudgetID}, nil),
		sharedBudgetUpdate,

		spendRisk(platform("platform-bulk-create-keywords", http.MethodPost, "v1/keywords/bulk-create", []string{"targeting-keywords", "create-bulk"}, ContextAdAccount, BodyObject, false, "KeywordCreateBulkRequest", "KeywordCreateBulkResponse", nil, nil)),
		spendRisk(platform("platform-bulk-update-keywords", http.MethodPost, "v1/keywords/bulk-update", []string{"targeting-keywords", "update-bulk"}, ContextAdAccount, BodyObject, false, "KeywordUpdateBulkRequest", "KeywordUpdateBulkResponse", nil, nil)),
		spendRisk(platform("platform-bulk-create-negative-keywords", http.MethodPost, "v1/negative-keywords/bulk-create", []string{"negative-keywords", "create-bulk"}, ContextAdAccount, BodyObject, false, "NegativeKeywordCreateBulkRequest", "NegativeKeywordCreateBulkResponse", nil, nil)),
		spendRisk(platform("platform-bulk-update-negative-keywords", http.MethodPost, "v1/negative-keywords/bulk-update", []string{"negative-keywords", "update-bulk"}, ContextAdAccount, BodyObject, false, "NegativeKeywordUpdateBulkRequest", "NegativeKeywordUpdateBulkResponse", nil, nil)),

		campaignCreate,
		query("platform-query-campaigns", "v1/campaigns/query", []string{"campaigns", "find"}, ContextAdAccount, "CampaignQueryResponse", nil),
		platform("platform-delete-campaign", http.MethodDelete, "v1/campaigns/{id}", []string{"campaigns", "delete"}, ContextAdAccount, BodyNone, false, "", "Response", []ParamSpec{campaignID}, nil),
		platform("platform-get-campaign", http.MethodGet, "v1/campaigns/{id}", []string{"campaigns", "view"}, ContextAdAccount, BodyNone, false, "", "CampaignResponse", []ParamSpec{campaignID}, nil),
		campaignUpdate,
		platform("platform-get-campaign-legacy-status-reasons", http.MethodGet, "v1/campaigns/{id}/legacy-app-limited-status-reason-details", []string{"campaigns", "legacy-status-reasons"}, ContextAdAccount, BodyNone, false, "", "LegacyAppLimitedStatusReasonDetailsResponse", []ParamSpec{campaignID}, nil),

		platform("platform-search-geo-locations", http.MethodGet, "v1/search/geo", []string{"geo", "search"}, ContextAdAccount, BodyNone, false, "", "GeoSearchResponse", nil, []ParamSpec{
			{Name: "supplySource", Flag: "supply-source", Type: ParamString, Required: true, Allowed: []string{"APPSTORE", "MAPS"}},
			{Name: "query", Flag: "query", Type: ParamString},
			{Name: "entity", Flag: "entity", Type: ParamString, Allowed: []string{"Country", "AdminArea", "Locality", "PostalCode"}},
			{Name: "countrycode", Flag: "country-code", Type: ParamString},
			{Name: "eligible", Flag: "eligible", Type: ParamBool},
			{Name: "offset", Flag: "offset", Type: ParamInt},
			{Name: "pageSize", Flag: "page-size", Type: ParamInt},
		}),
		platform("platform-resolve-geo-locations", http.MethodPost, "v1/search/geo", []string{"geo", "resolve"}, ContextAdAccount, BodyObject, false, "GeoSearchPostRequest", "GeoSearchResponse", nil, nil),

		spendRisk(platform("platform-create-keyword", http.MethodPost, "v1/keywords", []string{"targeting-keywords", "create"}, ContextAdAccount, BodyObject, false, "KeywordCreate", "KeywordResponse", nil, nil)),
		requiredSelectorQuery("platform-query-keywords", "v1/keywords/query", []string{"targeting-keywords", "find"}, ContextAdAccount, "KeywordQueryResponse", nil, keywordSelectorHint),
		platform("platform-delete-keyword", http.MethodDelete, "v1/keywords/{id}", []string{"targeting-keywords", "delete"}, ContextAdAccount, BodyNone, false, "", "Response", []ParamSpec{keywordID}, nil),
		platform("platform-get-keyword", http.MethodGet, "v1/keywords/{id}", []string{"targeting-keywords", "view"}, ContextAdAccount, BodyNone, false, "", "KeywordResponse", []ParamSpec{keywordID}, nil),
		spendRisk(platform("platform-update-keyword", http.MethodPut, "v1/keywords/{id}", []string{"targeting-keywords", "update"}, ContextAdAccount, BodyObject, false, "KeywordUpdate", "KeywordResponse", []ParamSpec{keywordID}, nil)),

		spendRisk(platform("platform-create-negative-keyword", http.MethodPost, "v1/negative-keywords", []string{"negative-keywords", "create"}, ContextAdAccount, BodyObject, false, "NegativeKeywordCreate", "NegativeKeywordResponse", nil, nil)),
		requiredSelectorQuery("platform-query-negative-keywords", "v1/negative-keywords/query", []string{"negative-keywords", "find"}, ContextAdAccount, "NegativeKeywordQueryResponse", nil, negativeKeywordSelectorHint),
		platform("platform-delete-negative-keyword", http.MethodDelete, "v1/negative-keywords/{id}", []string{"negative-keywords", "delete"}, ContextAdAccount, BodyNone, false, "", "Response", []ParamSpec{negativeKeywordID}, nil),
		platform("platform-get-negative-keyword", http.MethodGet, "v1/negative-keywords/{id}", []string{"negative-keywords", "view"}, ContextAdAccount, BodyNone, false, "", "NegativeKeywordResponse", []ParamSpec{negativeKeywordID}, nil),
		spendRisk(platform("platform-update-negative-keyword", http.MethodPut, "v1/negative-keywords/{id}", []string{"negative-keywords", "update"}, ContextAdAccount, BodyObject, false, "NegativeKeywordUpdate", "NegativeKeywordResponse", []ParamSpec{negativeKeywordID}, nil)),

		query("platform-query-app-locales", "v1/apps/{adamId}/locale-details/query", []string{"apps", "locales", "find"}, ContextAdAccount, "AppLocaleDetailsQueryResponse", []ParamSpec{adamIDParam}),
		query("platform-query-product-page-locales", "v1/product-pages/locale-details/query", []string{"product-pages", "locales", "find"}, ContextAdAccount, "ProductPageLocaleDetailsQueryResponse", nil),
		query("platform-query-product-pages", "v1/product-pages/query", []string{"product-pages", "find"}, ContextAdAccount, "ProductPageDetailsQueryResponse", nil),
		platform("platform-get-product-page", http.MethodGet, "v1/product-pages/{productPageId}", []string{"product-pages", "view"}, ContextAdAccount, BodyNone, false, "", "ProductPageDetailsResponse", []ParamSpec{productPageParam}, nil),
	}
}
