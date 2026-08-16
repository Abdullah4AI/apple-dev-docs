package appleads

import "strings"

// platformMapsEndpointSpecs returns the Platform API v1 brand, location,
// creative, and asset surface. Asset upload is inventory metadata for the
// dedicated multipart command; it is not executed by the generic JSON path.
func platformMapsEndpointSpecs() []EndpointSpec {
	id := func(flag string) []ParamSpec {
		return []ParamSpec{{Name: "id", Flag: flag, Type: ParamString, Required: true}}
	}
	query := func(name, path string, commandPath []string, optional bool, bodyType, responseType string) EndpointSpec {
		return platformEndpoint(name, "POST", path, commandPath, ContextAdAccount, BodyObject, optional, bodyType, responseType, nil, nil)
	}
	assetQuery := query("platform-query-assets", "v1/assets/query", []string{"assets", "find"}, true, "QueryRequest", "AssetQueryResponse")
	assetQuery.BodyFileExample = "query.json"
	assetQuery.BodyHint = "Omit --file to return the default page of non-deleted assets in the selected ad account. For narrower results, use promotedObjectId, promotedObjectType, providerAssetId, or assetType (IMAGE)."
	locationGroupUpdate := platformEndpoint("platform-update-location-group", "PUT", "v1/location-groups/{id}", []string{"location-groups", "update"}, ContextAdAccount, BodyObject, false, "LocationGroupUpdate", "LocationGroupResponse", id("location-group"), nil)
	locationGroupUpdate.BodyHint = "Updating a location group can immediately change targeting for linked campaigns and requires --confirm."
	locationGroupUpdate.RiskConfirm = true

	specs := []EndpointSpec{
		assetQuery,
		platformEndpoint("platform-upload-asset", "POST", "v1/assets/upload", []string{"assets", "upload"}, ContextAdAccount, BodyMultipart, false, "form-data: file (binary; required), promotedObjectId (string; required), promotedObjectType (string; required; allowed=BUSINESS_BRAND)", "AssetResponse", nil, nil),
		platformEndpoint("platform-delete-asset", "DELETE", "v1/assets/{id}", []string{"assets", "delete"}, ContextAdAccount, BodyNone, false, "", "Response", id("asset"), nil),
		platformEndpoint("platform-get-asset", "GET", "v1/assets/{id}", []string{"assets", "view"}, ContextAdAccount, BodyNone, false, "", "AssetResponse", id("asset"), nil),

		query("platform-query-brands", "v1/business-brands/query", []string{"brands", "find"}, false, "QueryRequest", "BrandQueryResponse"),
		platformEndpoint("platform-get-brand", "GET", "v1/business-brands/{id}", []string{"brands", "view"}, ContextAdAccount, BodyNone, false, "", "BrandResponse", id("brand"), nil),
		query("platform-query-business-categories", "v1/business-categories/query", []string{"business-categories", "find"}, false, "QueryRequest", "BusinessCategoryQueryResponse"),
		platformEndpoint("platform-get-business-category", "GET", "v1/business-categories/{id}", []string{"business-categories", "view"}, ContextAdAccount, BodyNone, false, "", "BusinessCategoryResponse", id("category"), nil),
		query("platform-query-brand-rejection-reasons", "v1/rejection-reasons/business-brands/query", []string{"rejection-reasons", "brands", "find"}, true, "PolicyAssignmentQueryRequest", "PolicyAssignmentQueryResponse"),

		platformEndpoint("platform-create-creative", "POST", "v1/creatives", []string{"creatives", "create"}, ContextAdAccount, BodyObject, false, "CreativeCreate", "CreativeResponse", nil, nil),
		query("platform-query-creatives", "v1/creatives/query", []string{"creatives", "find"}, true, "QueryRequest", "CreativeQueryResponse"),
		platformEndpoint("platform-delete-creative", "DELETE", "v1/creatives/{id}", []string{"creatives", "delete"}, ContextAdAccount, BodyNone, false, "", "Response", id("creative"), nil),
		platformEndpoint("platform-get-creative", "GET", "v1/creatives/{id}", []string{"creatives", "view"}, ContextAdAccount, BodyNone, false, "", "CreativeResponse", id("creative"), nil),
		platformEndpoint("platform-update-creative", "PUT", "v1/creatives/{id}", []string{"creatives", "update"}, ContextAdAccount, BodyObject, false, "CreativeUpdate", "CreativeResponse", id("creative"), nil),

		platformEndpoint("platform-create-location-group", "POST", "v1/location-groups", []string{"location-groups", "create"}, ContextAdAccount, BodyObject, false, "LocationGroupCreate", "LocationGroupResponse", nil, nil),
		query("platform-query-location-groups", "v1/location-groups/query", []string{"location-groups", "find"}, false, "QueryRequest", "LocationGroupQueryResponse"),
		platformEndpoint("platform-delete-location-group", "DELETE", "v1/location-groups/{id}", []string{"location-groups", "delete"}, ContextAdAccount, BodyNone, false, "", "LocationGroupResponse", id("location-group"), nil),
		platformEndpoint("platform-get-location-group", "GET", "v1/location-groups/{id}", []string{"location-groups", "view"}, ContextAdAccount, BodyNone, false, "", "LocationGroupResponse", id("location-group"), nil),
		locationGroupUpdate,

		query("platform-query-locations", "v1/locations/query", []string{"locations", "find"}, false, "QueryRequest", "LocationQueryResponse"),
		platformEndpoint("platform-get-location", "GET", "v1/locations/{id}", []string{"locations", "view"}, ContextAdAccount, BodyNone, false, "", "LocationResponse", id("location"), nil),
	}
	for index := range specs {
		if specs[index].Method == "POST" && strings.HasSuffix(specs[index].Path, "/query") {
			specs[index].RetrySafe = true
		}
		if specs[index].Method == "DELETE" {
			specs[index].RequiresConfirm = true
		}
	}
	return specs
}
