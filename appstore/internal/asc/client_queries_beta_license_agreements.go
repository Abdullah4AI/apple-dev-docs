package asc

import "net/url"

type betaLicenseAgreementsQuery struct {
	listQuery
	appIDs    []string
	fields    []string
	appFields []string
	include   []string
}

type betaLicenseAgreementQuery struct {
	fields    []string
	appFields []string
	include   []string
}

func buildBetaLicenseAgreementsQuery(query *betaLicenseAgreementsQuery) string {
	values := url.Values{}
	addCSV(values, "filter[app]", query.appIDs)
	addCSV(values, "fields[betaLicenseAgreements]", query.fields)
	addCSV(values, "fields[apps]", query.appFields)
	addCSV(values, "include", query.include)
	addLimit(values, query.limit)
	return values.Encode()
}

func buildBetaLicenseAgreementQuery(query *betaLicenseAgreementQuery) string {
	values := url.Values{}
	addCSV(values, "fields[betaLicenseAgreements]", query.fields)
	addCSV(values, "fields[apps]", query.appFields)
	addCSV(values, "include", query.include)
	return values.Encode()
}
