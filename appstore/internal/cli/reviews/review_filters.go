package reviews

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/asc"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/ascterritory"
	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

const (
	reviewResponseStateAny         = "any"
	reviewResponseStateUnresponded = "unresponded"
	reviewResponseStateResponded   = "responded"
)

// reviewSorts lists the sort values Apple accepts on every customer review
// listing endpoint.
var reviewSorts = []string{"rating", "-rating", "createdDate", "-createdDate"}

// ReviewFilterFlags holds the customer review filter flags. Apple exposes the
// same filter, sort, and include surface on GET /v1/apps/{id}/customerReviews
// and GET /v1/appStoreVersions/{id}/customerReviews, so every command that
// lists customer reviews binds these flags from here and gets identical names,
// help text, and validation.
type ReviewFilterFlags struct {
	Stars             string
	Territory         string
	Sort              string
	ResponseState     string
	OnlyUnresponded   bool
	IncludeResponse   bool
	ResponseFields    string
	starsSet          bool
	territorySet      bool
	sortSet           bool
	responseStateSet  bool
	responseFieldsSet bool
}

type trackedReviewStringFlag struct {
	name           string
	value          *string
	set            *bool
	commaSeparated bool
}

func (f *trackedReviewStringFlag) Set(value string) error {
	if *f.set {
		if f.commaSeparated {
			return fmt.Errorf(
				"--%s specified multiple times; pass one comma-separated list, for example --%s %q",
				f.name, f.name, *f.value+","+value,
			)
		}
		return fmt.Errorf("--%s specified multiple times; pass it once", f.name)
	}
	*f.value = value
	*f.set = true
	return nil
}

func (f *trackedReviewStringFlag) String() string {
	if f == nil || f.value == nil {
		return ""
	}
	return *f.value
}

func bindTrackedReviewStringFlag(fs *flag.FlagSet, name string, value *string, provided *bool, defaultValue string, commaSeparated bool, usage string) {
	*value = defaultValue
	fs.Var(&trackedReviewStringFlag{name: name, value: value, set: provided, commaSeparated: commaSeparated}, name, usage)
}

// BindReviewFilterFlags registers the shared customer review filter flags on fs.
func BindReviewFilterFlags(fs *flag.FlagSet) *ReviewFilterFlags {
	filters := &ReviewFilterFlags{}
	bindTrackedReviewStringFlag(fs, "stars", &filters.Stars, &filters.starsSet, "", true, "Filter by star ratings, comma-separated (1-5)")
	bindTrackedReviewStringFlag(fs, "territory", &filters.Territory, &filters.territorySet, "", true, "Filter by App Store territories, comma-separated (e.g., USA,GBR)")
	bindTrackedReviewStringFlag(fs, "sort", &filters.Sort, &filters.sortSet, "", false, "Sort by "+strings.Join(reviewSorts, ", "))
	bindTrackedReviewStringFlag(fs, "response-state", &filters.ResponseState, &filters.responseStateSet, reviewResponseStateAny, false, "Filter by response state: any, unresponded/unreplied, responded/replied")
	fs.BoolVar(&filters.OnlyUnresponded, "only-unresponded", false, "Only list reviews without a published response")
	fs.BoolVar(&filters.IncludeResponse, "include-response", false, "Include customer review response relationships")
	bindTrackedReviewStringFlag(fs, "response-fields", &filters.ResponseFields, &filters.responseFieldsSet, "", true, "Comma-separated customer review response fields: responseBody,lastModifiedDate,state,review")
	return filters
}

// ValidateReviewNextFlagConflicts rejects query flags that a links.next URL
// already encodes. Inspecting the FlagSet preserves explicit empty and zero
// values, which must not be silently ignored either.
func ValidateReviewNextFlagConflicts(next string, fs *flag.FlagSet, ownerFlag string) error {
	if strings.TrimSpace(next) == "" || fs == nil {
		return nil
	}

	setFlags := make(map[string]struct{})
	fs.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = struct{}{}
	})
	for _, name := range []string{
		ownerFlag,
		"stars",
		"territory",
		"sort",
		"response-state",
		"only-unresponded",
		"include-response",
		"response-fields",
		"limit",
	} {
		if _, ok := setFlags[name]; ok {
			return shared.WithDiagnostic(
				shared.UsageErrorf("--next cannot be combined with --%s", name),
				shared.DiagnosticConflictingInput,
				"",
			)
		}
	}
	return nil
}

// ReviewOptions validates the bound flags and returns the matching query
// options. Invalid input is reported as a usage failure carrying the offending
// parameter so both review listings fail the same way.
func (f *ReviewFilterFlags) ReviewOptions() ([]asc.ReviewOption, error) {
	if f.starsSet && strings.TrimSpace(f.Stars) == "" {
		return nil, shared.WithDiagnostic(shared.UsageError(errInvalidReviewStars().Error()), shared.DiagnosticInvalidInput, "--stars")
	}
	if f.territorySet && strings.TrimSpace(f.Territory) == "" {
		return nil, shared.WithDiagnostic(shared.UsageError("--territory must be a valid App Store territory code"), shared.DiagnosticInvalidInput, "--territory")
	}
	if f.sortSet && strings.TrimSpace(f.Sort) == "" {
		return nil, shared.WithDiagnostic(shared.UsageError("--sort must be one of: "+strings.Join(reviewSorts, ", ")), shared.DiagnosticInvalidInput, "--sort")
	}
	if f.responseStateSet && strings.TrimSpace(f.ResponseState) == "" {
		return nil, shared.WithDiagnostic(shared.UsageError("--response-state must be one of: any, unresponded, unreplied, responded, replied"), shared.DiagnosticInvalidInput, "--response-state")
	}
	if f.responseFieldsSet && strings.TrimSpace(f.ResponseFields) == "" {
		return nil, shared.WithDiagnostic(shared.UsageError("--response-fields must be a comma-separated list of: responseBody,lastModifiedDate,state,review"), shared.DiagnosticInvalidInput, "--response-fields")
	}
	ratings, err := normalizeReviewStars(f.Stars)
	if err != nil {
		return nil, shared.WithDiagnostic(shared.UsageError(err.Error()), shared.DiagnosticInvalidInput, "--stars")
	}
	territory, err := normalizeReviewTerritories(f.Territory)
	if err != nil {
		return nil, shared.WithDiagnostic(
			shared.UsageError(err.Error()),
			shared.DiagnosticInvalidInput,
			"--territory",
		)
	}
	if err := shared.ValidateSort(f.Sort, reviewSorts...); err != nil {
		return nil, shared.WithDiagnostic(shared.NewValidationError(err), shared.DiagnosticInvalidInput, "--sort")
	}
	responseState, err := normalizeReviewResponseState(f.ResponseState)
	if err != nil {
		return nil, shared.WithDiagnostic(shared.UsageError(err.Error()), shared.DiagnosticInvalidInput, "--response-state")
	}
	if f.OnlyUnresponded {
		if responseState == reviewResponseStateResponded {
			return nil, shared.WithDiagnostic(shared.UsageError("--only-unresponded cannot be combined with --response-state responded"), shared.DiagnosticConflictingInput, "")
		}
		responseState = reviewResponseStateUnresponded
	}
	responseFields, err := normalizeReviewResponseFields(f.ResponseFields)
	if err != nil {
		return nil, shared.WithDiagnostic(shared.UsageError(err.Error()), shared.DiagnosticInvalidInput, "--response-fields")
	}

	opts := []asc.ReviewOption{
		asc.WithRatings(ratings),
		asc.WithTerritory(territory),
	}
	if strings.TrimSpace(f.Sort) != "" {
		opts = append(opts, asc.WithReviewSort(f.Sort))
	}
	if exists, ok := publishedResponseExistsFilter(responseState); ok {
		opts = append(opts, asc.WithPublishedResponseExists(exists))
	}
	if f.IncludeResponse {
		opts = append(opts, asc.WithReviewIncludeResponse())
	}
	if len(responseFields) > 0 {
		opts = append(opts, asc.WithReviewIncludeResponse(), asc.WithReviewResponseFields(responseFields))
	}
	return opts, nil
}

func normalizeReviewTerritories(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}

	elements := strings.Split(value, ",")
	territories := make([]string, 0, len(elements))
	for _, element := range elements {
		normalized, err := ascterritory.Normalize(element)
		if err != nil {
			return "", fmt.Errorf("--territory must be a valid App Store territory code or comma-separated list of codes")
		}
		territories = append(territories, normalized)
	}
	return strings.Join(territories, ","), nil
}

func normalizeReviewResponseState(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		normalized = reviewResponseStateAny
	}
	switch normalized {
	case reviewResponseStateAny, reviewResponseStateUnresponded, reviewResponseStateResponded:
		return normalized, nil
	case "unreplied":
		return reviewResponseStateUnresponded, nil
	case "replied":
		return reviewResponseStateResponded, nil
	default:
		return "", fmt.Errorf("--response-state must be one of: any, unresponded, unreplied, responded, replied")
	}
}

// normalizeReviewStars parses the comma-separated --stars value. Apple accepts
// filter[rating] as an array parameter, so callers can ask for several ratings
// in one request. An empty value means "no rating filter"; anything else must
// be a list where every element is a rating between 1 and 5. Empty elements
// (`1,,2`, `1,`) are rejected rather than skipped, so malformed input never
// silently narrows the filter.
func normalizeReviewStars(value string) ([]int, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	elements := strings.Split(value, ",")
	ratings := make([]int, 0, len(elements))
	for _, element := range elements {
		rating, err := strconv.Atoi(strings.TrimSpace(element))
		if err != nil || rating < 1 || rating > 5 {
			return nil, errInvalidReviewStars()
		}
		ratings = append(ratings, rating)
	}
	return ratings, nil
}

func errInvalidReviewStars() error {
	return fmt.Errorf("--stars must be a comma-separated list of star ratings: 1, 2, 3, 4, 5")
}

// normalizeReviewResponseFields parses the comma-separated --response-fields
// value. An empty value means "no sparse fieldset"; anything else must be a
// list where every element names a customer review response field. Empty
// elements (`responseBody,,state`, `,`) are rejected rather than skipped, so
// malformed input never silently changes or drops the fieldset.
func normalizeReviewResponseFields(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	allowed := map[string]bool{
		"responseBody":     true,
		"lastModifiedDate": true,
		"state":            true,
		"review":           true,
	}
	fields := strings.Split(value, ",")
	normalized := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if !allowed[field] {
			return nil, fmt.Errorf("--response-fields must be a comma-separated list of: responseBody,lastModifiedDate,state,review")
		}
		normalized = append(normalized, field)
	}
	return normalized, nil
}

func publishedResponseExistsFilter(responseState string) (bool, bool) {
	switch responseState {
	case reviewResponseStateUnresponded:
		return false, true
	case reviewResponseStateResponded:
		return true, true
	default:
		return false, false
	}
}
