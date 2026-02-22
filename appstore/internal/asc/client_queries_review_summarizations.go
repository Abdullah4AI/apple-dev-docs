package asc

import "net/url"

type customerReviewSummarizationsQuery struct {
	listQuery
	platforms       []string
	territories     []string
	fields          []string
	territoryFields []string
	include         []string
}

func buildCustomerReviewSummarizationsQuery(query *customerReviewSummarizationsQuery) string {
	values := url.Values{}
	addCSV(values, "filter[platform]", query.platforms)
	addCSV(values, "filter[territory]", query.territories)
	addCSV(values, "fields[customerReviewSummarizations]", query.fields)
	addCSV(values, "fields[territories]", query.territoryFields)
	addCSV(values, "include", query.include)
	addLimit(values, query.limit)
	return values.Encode()
}
