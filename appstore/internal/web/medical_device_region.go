package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
)

const medicalDeviceRegionSupportInfoMaxLength = 5000

// MedicalDeviceRegionSupportInfo is one localized support-information row in
// Apple's detailed regulated-medical-device form.
type MedicalDeviceRegionSupportInfo struct {
	Locale      string `json:"locale"`
	Instruction string `json:"instruction"`
	Statement   string `json:"statement"`
	SafetyInfo  string `json:"safetyInfo"`
}

// MedicalDeviceRegionOptions controls one detailed regional medical-device
// answer. The app-level declaration is managed by SetMedicalDeviceDeclaration.
type MedicalDeviceRegionOptions struct {
	Declaration        bool
	RegistrationNumber string
	SupportInfo        []MedicalDeviceRegionSupportInfo
}

// MedicalDeviceRegionResult reports the verified detailed regional answer.
// It intentionally contains no registration, support, or contact values.
type MedicalDeviceRegionResult struct {
	AppID           string `json:"appId"`
	RequirementID   string `json:"requirementId"`
	RequirementName string `json:"requirementName"`
	Status          string `json:"status,omitempty"`
	FormID          string `json:"formId,omitempty"`
	Region          string `json:"region"`
	Declared        bool   `json:"declared"`
	Changed         bool   `json:"changed"`
}

// ValidateMedicalDeviceRegionOptions validates the source-backed fields used
// by one detailed regional write. It does not validate contact information;
// that requires the current form and is checked immediately before a write.
func ValidateMedicalDeviceRegionOptions(region string, options MedicalDeviceRegionOptions) error {
	return validateMedicalDeviceRegionOptions(region, options)
}

func validateMedicalDeviceRegionOptions(region string, options MedicalDeviceRegionOptions) error {
	region = normalizeMedicalDeviceRegion(region)
	if region == "" {
		return fmt.Errorf("medical device country/region is required")
	}
	if _, ok := medicalDeviceSupportedRegionSet[region]; !ok {
		return fmt.Errorf("unsupported medical device country/region %q (supported: EEA, GBR, USA)", region)
	}

	if !options.Declaration {
		if strings.TrimSpace(options.RegistrationNumber) != "" {
			return fmt.Errorf("registrationNumber is only supported when declaration is true")
		}
		if len(options.SupportInfo) > 0 {
			return fmt.Errorf("supportInfo is only supported when declaration is true")
		}
		return nil
	}

	if region != "GBR" && strings.TrimSpace(options.RegistrationNumber) == "" {
		return fmt.Errorf("registrationNumber is required for medical device region %s", region)
	}
	if region == "GBR" && strings.TrimSpace(options.RegistrationNumber) != "" {
		return fmt.Errorf("registrationNumber is not supported for medical device region GBR")
	}
	if len(options.SupportInfo) == 0 {
		return fmt.Errorf("supportInfo must include at least one locale for medical device region %s", region)
	}

	seenLocales := make(map[string]struct{}, len(options.SupportInfo))
	for index, support := range options.SupportInfo {
		locale := strings.TrimSpace(support.Locale)
		if locale == "" {
			return fmt.Errorf("supportInfo[%d].locale is required", index)
		}
		if _, exists := seenLocales[locale]; exists {
			return fmt.Errorf("supportInfo contains duplicate locale %q", locale)
		}
		seenLocales[locale] = struct{}{}

		instruction := support.Instruction
		if strings.TrimSpace(instruction) == "" {
			return fmt.Errorf("supportInfo[%d].instruction is required", index)
		}
		// This mirrors Apple's captured KE validator: it checks the prefix and
		// does not promise that the linked document is reachable.
		if !strings.HasPrefix(instruction, "http://") && !strings.HasPrefix(instruction, "https://") {
			return fmt.Errorf("supportInfo[%d].instruction must begin with http:// or https://", index)
		}

		if strings.TrimSpace(support.Statement) == "" {
			return fmt.Errorf("supportInfo[%d].statement is required", index)
		}
		if medicalDeviceUTF16Length(support.Statement) > medicalDeviceRegionSupportInfoMaxLength {
			return fmt.Errorf("supportInfo[%d].statement must be at most %d characters", index, medicalDeviceRegionSupportInfoMaxLength)
		}
		if strings.TrimSpace(support.SafetyInfo) == "" {
			return fmt.Errorf("supportInfo[%d].safetyInfo is required", index)
		}
		if medicalDeviceUTF16Length(support.SafetyInfo) > medicalDeviceRegionSupportInfoMaxLength {
			return fmt.Errorf("supportInfo[%d].safetyInfo must be at most %d characters", index, medicalDeviceRegionSupportInfoMaxLength)
		}
	}
	return nil
}

func medicalDeviceUTF16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

var medicalDeviceSupportedRegionSet = map[string]struct{}{
	"EEA": {},
	"GBR": {},
	"USA": {},
}

func medicalDeviceRegionConstraintValues(constraints map[string]complianceConstraint) ([]string, error) {
	seen := make(map[string]struct{}, len(medicalDeviceSupportedRegions))
	for key, constraint := range constraints {
		// Nested contact and registration selectors describe their own rows;
		// only the captured top-level selector controls region availability.
		if key != "$[*].countriesOrRegions" || strings.TrimSpace(constraint.AttributeName) != "countriesOrRegions" {
			continue
		}
		for _, option := range constraint.Options {
			values := append([]string{option.Value}, option.ListValues...)
			for _, value := range values {
				region := normalizeMedicalDeviceRegion(value)
				if _, supported := medicalDeviceSupportedRegionSet[region]; !supported {
					continue
				}
				seen[region] = struct{}{}
			}
		}
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("medical device country/region constraints are missing from form metadata")
	}
	regions := make([]string, 0, len(seen))
	for region := range seen {
		regions = append(regions, region)
	}
	slices.Sort(regions)
	return regions, nil
}

func medicalDeviceRegionAllowedByConstraints(constraints map[string]complianceConstraint, region string) error {
	regions, err := medicalDeviceRegionConstraintValues(constraints)
	if err != nil {
		return err
	}
	if !slices.Contains(regions, region) {
		return fmt.Errorf("medical device country/region %q is not available in the form constraints", region)
	}
	return nil
}

func medicalDeviceContactRegion(value any, region string) bool {
	// The captured UI compares contact region arrays with its canonical
	// EEA/GBR/USA enum using literal membership. The command canonicalizes the
	// requested region before this check, so do not broaden contact selection
	// with scalar values or the EU input alias.
	check := func(candidate string) bool { return candidate == region }
	switch values := value.(type) {
	case []string:
		for _, candidate := range values {
			if check(candidate) {
				return true
			}
		}
	case []any:
		for _, candidate := range values {
			if text, ok := candidate.(string); ok && check(text) {
				return true
			}
		}
	}
	return false
}

func medicalDeviceNonEmptyString(value any) bool {
	text, ok := value.(string)
	return ok && strings.TrimSpace(text) != ""
}

func medicalDeviceNonEmptyAddress(value any) bool {
	address, ok := value.(map[string]any)
	return ok && len(address) > 0
}

var medicalDeviceContactConstraintPattern = regexp.MustCompile(`^\$\[\*\]\.medicalDeviceData\.contactInformation\[(\d+)\]\.(.+)$`)

func medicalDeviceConstraintOptionValue(option complianceConstraintOption) (any, bool) {
	// Xh in Apple's form state takes the first option's value, falling back to
	// its listValues when value is absent. The captured constraints only expose
	// scalar values and list values for contact candidates.
	if option.Value != "" {
		return option.Value, true
	}
	if len(option.ListValues) > 0 {
		return append([]string(nil), option.ListValues...), true
	}
	return nil, false
}

func medicalDeviceContactConstraintCandidates(constraints map[string]complianceConstraint) map[int]map[string]any {
	candidates := make(map[int]map[string]any)
	for key, constraint := range constraints {
		match := medicalDeviceContactConstraintPattern.FindStringSubmatch(key)
		if len(match) != 3 || len(constraint.Options) == 0 {
			continue
		}
		value, ok := medicalDeviceConstraintOptionValue(constraint.Options[0])
		if !ok {
			continue
		}
		index, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		fieldPath := strings.Split(match[2], ".")
		if len(fieldPath) == 0 || len(fieldPath) > 2 || fieldPath[0] == "" {
			continue
		}
		candidate := candidates[index]
		if candidate == nil {
			candidate = make(map[string]any)
			candidates[index] = candidate
		}
		if len(fieldPath) == 1 {
			candidate[fieldPath[0]] = value
			continue
		}
		if fieldPath[0] != "address" || fieldPath[1] == "" {
			continue
		}
		address, _ := candidate["address"].(map[string]any)
		if address == nil {
			address = make(map[string]any)
			candidate["address"] = address
		}
		address[fieldPath[1]] = value
	}
	return candidates
}

func medicalDeviceContactCoversAnySupportedRegion(value any) bool {
	for region := range medicalDeviceSupportedRegionSet {
		if medicalDeviceContactRegion(value, region) {
			return true
		}
	}
	return false
}

func medicalDeviceExistingContactEntityIDs(medicalData map[string]any) (map[string]struct{}, error) {
	ids := make(map[string]struct{})
	raw, ok := medicalData["contactInformation"]
	if !ok || raw == nil {
		return ids, nil
	}
	contacts, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("medical device contactInformation must be an array")
	}
	for index, rawContact := range contacts {
		contact, ok := rawContact.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("medical device contactInformation[%d] must be an object", index)
		}
		entityID, ok := contact["legalEntityId"].(string)
		if !ok || strings.TrimSpace(entityID) == "" || !medicalDeviceContactCoversAnySupportedRegion(contact["countriesOrRegions"]) {
			continue
		}
		ids[entityID] = struct{}{}
	}
	return ids, nil
}

func validateMedicalDeviceRegionConstraintContactCandidates(medicalData map[string]any, constraints map[string]complianceConstraint, region string) error {
	candidates := medicalDeviceContactConstraintCandidates(constraints)
	if len(candidates) == 0 {
		return nil
	}
	existingIDs, err := medicalDeviceExistingContactEntityIDs(medicalData)
	if err != nil {
		return err
	}
	indexes := make([]int, 0, len(candidates))
	for index := range candidates {
		indexes = append(indexes, index)
	}
	slices.Sort(indexes)
	for _, index := range indexes {
		candidate := candidates[index]
		entityID, ok := candidate["legalEntityId"].(string)
		// Xh filters materialized candidates to legalEntityId plus at least one
		// country/region. Fields without those values are not required rows.
		if !ok || strings.TrimSpace(entityID) == "" || !medicalDeviceContactRegion(candidate["countriesOrRegions"], region) {
			continue
		}
		if _, present := existingIDs[entityID]; present {
			continue
		}
		return fmt.Errorf("medical device region %s requires a form-defined contact candidate at index %d; configure contact information in App Store Connect before retrying", region, index)
	}
	return nil
}

func validateMedicalDeviceRegionContactCoverage(medicalData map[string]any, region string) error {
	raw, ok := medicalData["contactInformation"]
	if !ok || raw == nil {
		return fmt.Errorf("medical device region %s requires an existing contact covering that region; configure contact information in App Store Connect", region)
	}
	contacts, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("medical device contactInformation must be an array")
	}
	if len(contacts) == 0 {
		return fmt.Errorf("medical device region %s requires an existing contact covering that region; configure contact information in App Store Connect", region)
	}

	covered := false
	for index, rawContact := range contacts {
		contact, ok := rawContact.(map[string]any)
		if !ok {
			return fmt.Errorf("medical device contactInformation[%d] must be an object", index)
		}
		// Apple's selectors first limit contacts to rows whose region list
		// contains the target region and that have a legal entity. Contacts for
		// another region or without a legal entity do not participate in this
		// regional form's Fh validation.
		if !medicalDeviceNonEmptyString(contact["legalEntityId"]) || !medicalDeviceContactRegion(contact["countriesOrRegions"], region) {
			continue
		}
		// These are the source UI's Fh/kh prerequisites from the captured
		// bundle: legal entity + region coverage, phone, email, and a
		// non-empty address object. Do not impose invented address subfields.
		if !medicalDeviceNonEmptyString(contact["phone"]) {
			return fmt.Errorf("medical device contactInformation[%d].phone is required", index)
		}
		if !medicalDeviceNonEmptyString(contact["email"]) {
			return fmt.Errorf("medical device contactInformation[%d].email is required", index)
		}
		if !medicalDeviceNonEmptyAddress(contact["address"]) {
			return fmt.Errorf("medical device contactInformation[%d].address is required", index)
		}
		covered = true
	}
	if !covered {
		return fmt.Errorf("medical device region %s has no complete contact coverage; configure contact information in App Store Connect", region)
	}
	return nil
}

func medicalDeviceRegionSupportRows(existing map[string]any, supportInfo []MedicalDeviceRegionSupportInfo) []any {
	existingByLocale := make(map[string]map[string]any)
	if raw, ok := existing["supportInfo"].([]any); ok {
		for _, value := range raw {
			row, ok := value.(map[string]any)
			if !ok {
				continue
			}
			locale, _ := row["locale"].(string)
			if locale != "" {
				existingByLocale[locale] = row
			}
		}
	}

	rows := make([]any, 0, len(supportInfo))
	for _, support := range supportInfo {
		row := cloneMedicalDeviceMap(existingByLocale[strings.TrimSpace(support.Locale)])
		if row == nil {
			row = make(map[string]any)
		}
		row["locale"] = strings.TrimSpace(support.Locale)
		row["instruction"] = support.Instruction
		row["statement"] = support.Statement
		row["safetyInfo"] = support.SafetyInfo
		rows = append(rows, row)
	}
	return rows
}

func buildMedicalDeviceRegionRequestBody(form *medicalDeviceFormResponse, region string, options MedicalDeviceRegionOptions) (map[string]any, error) {
	formData, err := medicalDeviceFormObject(form)
	if err != nil {
		return nil, err
	}
	medicalData, _, err := medicalDeviceFormMedicalData(form, formData)
	if err != nil {
		return nil, err
	}
	medicalData = cloneMedicalDeviceMap(medicalData)

	existingRows, err := medicalDeviceRegistrationInfo(medicalData)
	if err != nil {
		return nil, err
	}
	var existingTarget map[string]any
	targetIndex := -1
	targetCount := 0
	for index, value := range existingRows {
		if _, ok := value.(map[string]any); !ok || medicalDeviceRegistrationRegion(value) == "" {
			return nil, fmt.Errorf("medical device registrationInfo contains a row without a country/region")
		}
		if medicalDeviceRegistrationRegion(value) == region {
			targetCount++
			targetIndex = index
			if row, ok := value.(map[string]any); ok {
				existingTarget = row
			}
		}
	}
	if targetCount == 0 {
		return nil, fmt.Errorf("medical device registrationInfo row for region %s is required", region)
	}
	if targetCount > 1 {
		return nil, fmt.Errorf("medical device registrationInfo contains duplicate rows for region %s", region)
	}

	target := cloneMedicalDeviceMap(existingTarget)
	if target == nil {
		target = make(map[string]any)
	}
	target["countriesOrRegions"] = []string{region}
	targetDeclaration := medicalDeviceDeclarationNo
	if options.Declaration {
		targetDeclaration = medicalDeviceDeclarationYes
		target["supportInfo"] = medicalDeviceRegionSupportRows(target, options.SupportInfo)
		if region != "GBR" {
			target["registrationNumber"] = strings.TrimSpace(options.RegistrationNumber)
		} else {
			// The source builder only includes registrationNumber for USA/EEA.
			delete(target, "registrationNumber")
		}
	}
	target["declaration"] = targetDeclaration
	updatedRows := append([]any(nil), existingRows...)
	updatedRows[targetIndex] = target
	medicalData["registrationInfo"] = updatedRows

	body := cloneMedicalDeviceMap(formData)
	body["medicalDeviceData"] = medicalData
	return body, nil
}

func medicalDeviceJSONEqual(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func medicalDeviceRowsWithoutRegion(rows []any, region string) []any {
	filtered := make([]any, 0, len(rows))
	for _, row := range rows {
		if medicalDeviceRegistrationRegion(row) != region {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func medicalDeviceFindUniqueRegionRow(rows []any, region string) (map[string]any, error) {
	var found map[string]any
	count := 0
	for _, value := range rows {
		if medicalDeviceRegistrationRegion(value) != region {
			continue
		}
		row, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("medical device registrationInfo row for region %s is not an object", region)
		}
		count++
		found = row
	}
	if count != 1 {
		return nil, fmt.Errorf("medical device registrationInfo readback contains %d rows for region %s", count, region)
	}
	return found, nil
}

func medicalDeviceRowsPreserved(expected, actual []any, targetRegion string) bool {
	expectedOther := medicalDeviceRowsWithoutRegion(expected, targetRegion)
	actualOther := medicalDeviceRowsWithoutRegion(actual, targetRegion)
	if len(expectedOther) != len(actualOther) {
		return false
	}
	used := make([]bool, len(actualOther))
	for _, expectedRow := range expectedOther {
		matched := false
		for index, actualRow := range actualOther {
			if used[index] || medicalDeviceRegistrationRegion(expectedRow) != medicalDeviceRegistrationRegion(actualRow) {
				continue
			}
			if medicalDeviceJSONEqual(expectedRow, actualRow) {
				used[index] = true
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func validateMedicalDeviceRegionFormIdentity(data, previousBody map[string]any, accountID, appID string, requirement *complianceRequirement) error {
	if requirement == nil {
		return fmt.Errorf("medical device region form is missing the compliance requirement")
	}
	expectedIdentity := []struct {
		field    string
		expected string
	}{
		{field: "accountId", expected: strings.TrimSpace(accountID)},
		{field: "contentId", expected: strings.TrimSpace(appID)},
		{field: "requirementId", expected: strings.TrimSpace(requirement.ID)},
		{field: "requirementName", expected: strings.TrimSpace(requirement.Name)},
		{field: "formId", expected: strings.TrimSpace(requirement.FormID)},
	}
	for _, identity := range expectedIdentity {
		field, expected := identity.field, identity.expected
		value, present := data[field]
		if !present || value == nil {
			continue
		}
		actual, ok := value.(string)
		if !ok {
			return fmt.Errorf("medical device region form returned an invalid %s", field)
		}
		actual = strings.TrimSpace(actual)
		if actual == "" {
			continue
		}
		if expected != "" && actual != expected {
			return fmt.Errorf("medical device region form returned a different %s", field)
		}
		if previous, ok := previousBody[field].(string); ok && strings.TrimSpace(previous) != "" && actual != strings.TrimSpace(previous) {
			return fmt.Errorf("medical device region form returned a different %s", field)
		}
	}
	return nil
}

func verifyMedicalDeviceRegionReadback(form *medicalDeviceFormResponse, accountID, appID string, requirement *complianceRequirement, region string, options MedicalDeviceRegionOptions, expectedBody map[string]any) error {
	if form.declaration() != medicalDeviceDeclarationYes {
		return fmt.Errorf("medical device readback returned an overall declaration other than yes")
	}
	gotData, err := medicalDeviceFormObject(form)
	if err != nil {
		return err
	}
	if err := validateMedicalDeviceRegionFormIdentity(gotData, expectedBody, accountID, appID, requirement); err != nil {
		return err
	}
	gotMedical, _, err := medicalDeviceFormMedicalData(form, gotData)
	if err != nil {
		return err
	}
	expectedMedical, ok := expectedBody["medicalDeviceData"].(map[string]any)
	if !ok {
		return fmt.Errorf("medical device region request is missing medicalDeviceData")
	}
	gotRows, err := medicalDeviceRegistrationInfo(gotMedical)
	if err != nil {
		return err
	}
	expectedRows, err := medicalDeviceRegistrationInfo(expectedMedical)
	if err != nil {
		return err
	}
	expectedTarget, err := medicalDeviceFindUniqueRegionRow(expectedRows, region)
	if err != nil {
		return fmt.Errorf("medical device region verification failed: %w", err)
	}
	gotTarget, err := medicalDeviceFindUniqueRegionRow(gotRows, region)
	if err != nil {
		return fmt.Errorf("medical device region verification failed: %w", err)
	}
	if !medicalDeviceJSONEqual(expectedTarget, gotTarget) {
		return fmt.Errorf("medical device region verification failed: Apple returned a different target row")
	}
	if !medicalDeviceRowsPreserved(expectedRows, gotRows, region) {
		return fmt.Errorf("medical device region verification failed: Apple changed an unrelated region row")
	}

	withoutMedicalData := cloneMedicalDeviceMap(expectedBody)
	delete(withoutMedicalData, "medicalDeviceData")
	gotWithoutMedicalData := cloneMedicalDeviceMap(gotData)
	delete(gotWithoutMedicalData, "medicalDeviceData")
	// Identity metadata is optional in legacy readbacks and was compared above
	// wherever available. Preserve exact comparison for every other outer field.
	for _, field := range []string{"accountId", "contentId", "requirementId", "requirementName", "formId"} {
		delete(withoutMedicalData, field)
		delete(gotWithoutMedicalData, field)
	}
	if !medicalDeviceJSONEqual(withoutMedicalData, gotWithoutMedicalData) {
		return fmt.Errorf("medical device region verification failed: Apple changed an outer form field")
	}

	expectedOpaqueMedical := cloneMedicalDeviceMap(expectedMedical)
	gotOpaqueMedical := cloneMedicalDeviceMap(gotMedical)
	delete(expectedOpaqueMedical, "registrationInfo")
	delete(gotOpaqueMedical, "registrationInfo")
	if !medicalDeviceJSONEqual(expectedOpaqueMedical, gotOpaqueMedical) {
		return fmt.Errorf("medical device region verification failed: Apple changed an unrelated medical form field")
	}

	// The explicit target declaration is checked above via the full target row.
	// Keep this branch visible so the result contract remains tied to the input.
	if options.Declaration && gotTarget["declaration"] != medicalDeviceDeclarationYes {
		return fmt.Errorf("medical device region verification failed: affirmative target was not persisted")
	}
	if !options.Declaration && gotTarget["declaration"] != medicalDeviceDeclarationNo {
		return fmt.Errorf("medical device region verification failed: negative target was not persisted")
	}
	return nil
}

// SetMedicalDeviceRegion updates and verifies one detailed regional answer.
// It requires the app-level declaration to already be yes. The method sends
// one full-form PUT only after all source-backed preflight checks pass.
func (c *Client) SetMedicalDeviceRegion(ctx context.Context, accountID, appID, region string, options MedicalDeviceRegionOptions) (*MedicalDeviceRegionResult, error) {
	region = normalizeMedicalDeviceRegion(region)
	if err := validateMedicalDeviceRegionOptions(region, options); err != nil {
		return nil, err
	}

	requirement, form, err := c.medicalDeviceRequirementAndForm(ctx, accountID, appID)
	if err != nil {
		return nil, err
	}
	if form.declaration() != medicalDeviceDeclarationYes {
		return nil, fmt.Errorf("medical device region %s requires the overall medical device declaration to be yes; set the app-level declaration first", region)
	}
	if err := medicalDeviceRegionAllowedByConstraints(form.Constraints, region); err != nil {
		return nil, err
	}

	formData, err := medicalDeviceFormObject(form)
	if err != nil {
		return nil, err
	}
	if err := validateMedicalDeviceRegionFormIdentity(formData, nil, accountID, appID, requirement); err != nil {
		return nil, err
	}
	medicalData, _, err := medicalDeviceFormMedicalData(form, formData)
	if err != nil {
		return nil, err
	}
	if options.Declaration {
		if err := validateMedicalDeviceRegionConstraintContactCandidates(medicalData, form.Constraints, region); err != nil {
			return nil, err
		}
		if err := validateMedicalDeviceRegionContactCoverage(medicalData, region); err != nil {
			return nil, err
		}
	}

	requestBody, err := buildMedicalDeviceRegionRequestBody(form, region, options)
	if err != nil {
		return nil, err
	}
	fillMedicalDeviceFormIdentity(requestBody, accountID, appID, requirement)
	// Missing identity metadata alone must not turn an unchanged answer into
	// a write. Compare both forms with the same compatibility fallbacks.
	fillMedicalDeviceFormIdentity(formData, accountID, appID, requirement)
	result := &MedicalDeviceRegionResult{
		AppID:           strings.TrimSpace(appID),
		RequirementID:   requirement.ID,
		RequirementName: requirement.Name,
		Status:          requirement.Status,
		FormID:          requirement.FormID,
		Region:          region,
		Declared:        options.Declaration,
	}
	if medicalDeviceJSONEqual(formData, requestBody) {
		return result, nil
	}

	path := "/ppm/complianceform/v1/accounts/" + url.PathEscape(strings.TrimSpace(accountID)) +
		"/contents/" + url.PathEscape(strings.TrimSpace(appID)) +
		"/requirements/" + url.PathEscape(requirement.ID) + "/forms"
	if _, err := c.doAppComplianceRequest(ctx, appID, http.MethodPut, path, requestBody); err != nil {
		return nil, err
	}

	updatedRequirement, updatedForm, err := c.medicalDeviceRequirementAndForm(ctx, accountID, appID)
	if err != nil {
		return nil, fmt.Errorf("medical device region verification failed: %w", err)
	}
	if err := verifyMedicalDeviceRegionReadback(updatedForm, accountID, appID, updatedRequirement, region, options, requestBody); err != nil {
		return nil, err
	}
	result.RequirementID = updatedRequirement.ID
	result.RequirementName = updatedRequirement.Name
	result.Status = updatedRequirement.Status
	result.FormID = updatedRequirement.FormID
	result.Changed = true
	return result, nil
}
