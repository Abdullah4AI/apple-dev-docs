package validation

// TestFlightInput collects the TestFlight validation inputs.
type TestFlightInput struct {
	AppID            string
	AppPrimaryLocale string

	BuildID    string
	Build      *Build
	BuildAppID string

	BetaReviewDetails      *BetaReviewDetails
	BetaBuildLocalizations []BetaBuildLocalization
}

// TestFlightReport is the top-level validate testflight output.
type TestFlightReport struct {
	AppID        string        `json:"appId"`
	BuildID      string        `json:"buildId"`
	BuildVersion string        `json:"buildVersion,omitempty"`
	Summary      Summary       `json:"summary"`
	Checks       []CheckResult `json:"checks"`
	Strict       bool          `json:"strict,omitempty"`
}

// BetaReviewDetails represents TestFlight beta app review details.
type BetaReviewDetails struct {
	ID                  string
	ContactFirstName    string
	ContactLastName     string
	ContactEmail        string
	ContactPhone        string
	DemoAccountName     string
	DemoAccountPassword string
	DemoAccountRequired bool
	Notes               string
}

// BetaBuildLocalization represents a build localization (TestFlight "What to Test").
type BetaBuildLocalization struct {
	ID       string
	Locale   string
	WhatsNew string
}
