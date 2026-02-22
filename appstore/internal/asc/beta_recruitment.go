package asc

// BetaRecruitmentCriteriaAttributes describes beta recruitment criteria metadata.
type BetaRecruitmentCriteriaAttributes struct {
	LastModifiedDate             string                        `json:"lastModifiedDate,omitempty"`
	DeviceFamilyOsVersionFilters []DeviceFamilyOsVersionFilter `json:"deviceFamilyOsVersionFilters,omitempty"`
}

// BetaRecruitmentCriteriaCreateAttributes describes create attributes.
type BetaRecruitmentCriteriaCreateAttributes struct {
	DeviceFamilyOsVersionFilters []DeviceFamilyOsVersionFilter `json:"deviceFamilyOsVersionFilters"`
}

// BetaRecruitmentCriteriaUpdateAttributes describes update attributes.
type BetaRecruitmentCriteriaUpdateAttributes struct {
	DeviceFamilyOsVersionFilters []DeviceFamilyOsVersionFilter `json:"deviceFamilyOsVersionFilters,omitempty"`
}

// BetaRecruitmentCriteriaResponse is the response from beta recruitment criteria endpoints.
type BetaRecruitmentCriteriaResponse = SingleResponse[BetaRecruitmentCriteriaAttributes]

// BetaRecruitmentCriteriaDeleteResult represents CLI output for deletions.
type BetaRecruitmentCriteriaDeleteResult struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// BetaRecruitmentCriterionCompatibleBuildCheckAttributes describes compatible build check attributes.
type BetaRecruitmentCriterionCompatibleBuildCheckAttributes struct {
	HasCompatibleBuild bool `json:"hasCompatibleBuild,omitempty"`
}

// BetaRecruitmentCriterionCompatibleBuildCheckResponse is the response for compatible build checks.
type BetaRecruitmentCriterionCompatibleBuildCheckResponse = SingleResponse[BetaRecruitmentCriterionCompatibleBuildCheckAttributes]

// BetaRecruitmentCriteriaRelationships describes relationships for recruitment criteria.
type BetaRecruitmentCriteriaRelationships struct {
	BetaGroup *Relationship `json:"betaGroup,omitempty"`
}

// BetaRecruitmentCriteriaCreateData is the data portion of a criteria create request.
type BetaRecruitmentCriteriaCreateData struct {
	Type          ResourceType                            `json:"type"`
	Attributes    BetaRecruitmentCriteriaCreateAttributes `json:"attributes"`
	Relationships *BetaRecruitmentCriteriaRelationships   `json:"relationships"`
}

// BetaRecruitmentCriteriaCreateRequest is a request to create beta recruitment criteria.
type BetaRecruitmentCriteriaCreateRequest struct {
	Data BetaRecruitmentCriteriaCreateData `json:"data"`
}

// BetaRecruitmentCriteriaUpdateData is the data portion of a criteria update request.
type BetaRecruitmentCriteriaUpdateData struct {
	Type       ResourceType                             `json:"type"`
	ID         string                                   `json:"id"`
	Attributes *BetaRecruitmentCriteriaUpdateAttributes `json:"attributes,omitempty"`
}

// BetaRecruitmentCriteriaUpdateRequest is a request to update beta recruitment criteria.
type BetaRecruitmentCriteriaUpdateRequest struct {
	Data BetaRecruitmentCriteriaUpdateData `json:"data"`
}

// BetaRecruitmentCriterionOptionAttributes describes recruitment criteria options.
type BetaRecruitmentCriterionOptionAttributes struct {
	Identifier             string                                       `json:"identifier,omitempty"`
	Name                   string                                       `json:"name,omitempty"`
	Category               string                                       `json:"category,omitempty"`
	DeviceFamilyOsVersions []BetaRecruitmentCriterionOptionDeviceFamily `json:"deviceFamilyOsVersions,omitempty"`
}

// BetaRecruitmentCriterionOptionDeviceFamily describes device families and OS versions for options.
type BetaRecruitmentCriterionOptionDeviceFamily struct {
	DeviceFamily DeviceFamily `json:"deviceFamily,omitempty"`
	OSVersions   []string     `json:"osVersions,omitempty"`
}

// BetaRecruitmentCriterionOptionsResponse is the response from recruitment criteria options list.
type BetaRecruitmentCriterionOptionsResponse = Response[BetaRecruitmentCriterionOptionAttributes]

// BetaGroupMetricAttributes represents metric attributes returned by metrics endpoints.
type BetaGroupMetricAttributes map[string]any

// BetaGroupPublicLinkUsagesResponse is the response from public link usage metrics.
type BetaGroupPublicLinkUsagesResponse = Response[BetaGroupMetricAttributes]

// BetaGroupTesterUsagesResponse is the response from beta tester usage metrics.
type BetaGroupTesterUsagesResponse = Response[BetaGroupMetricAttributes]
