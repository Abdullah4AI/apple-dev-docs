package validation

// App Store metadata character limits.
const (
	LimitDescription     = 4000
	LimitKeywords        = 100
	LimitWhatsNew        = 4000
	LimitPromotionalText = 170
	LimitName            = 30
	LimitSubtitle        = 30
)

// App Store metadata URL length limits.
//
// Apple's App Store Connect OpenAPI specification carries no maxLength for
// these attributes, so the values below track Apple's documented App Store
// Connect field constraints instead of a machine-readable schema. Checks built
// on them are advisory (warning severity) so a longer value Apple happens to
// accept never hard-fails a local run.
const (
	LimitMarketingURL      = 255
	LimitSupportURL        = 255
	LimitPrivacyPolicyURL  = 255
	LimitPrivacyChoicesURL = 255
)

// Minimum lengths that separate real metadata from placeholders such as "X"
// or "TBD". Apple states no exact minimum in a machine-readable form, so these
// only ever produce warnings.
const (
	MinLengthName        = 2
	MinLengthDescription = 10
)
