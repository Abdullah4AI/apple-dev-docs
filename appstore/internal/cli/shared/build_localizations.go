package shared

import (
	"fmt"
	"regexp"
)

var buildLocalizationLocaleRegex = regexp.MustCompile(`^[a-zA-Z]{2,3}(-[a-zA-Z0-9]+)*$`)

// ValidateBuildLocalizationLocales validates multiple locale strings.
func ValidateBuildLocalizationLocales(locales []string) error {
	for _, locale := range locales {
		if err := ValidateBuildLocalizationLocale(locale); err != nil {
			return err
		}
	}
	return nil
}

// ValidateBuildLocalizationLocale validates a locale string.
func ValidateBuildLocalizationLocale(locale string) error {
	if locale == "" || !buildLocalizationLocaleRegex.MatchString(locale) {
		return fmt.Errorf("invalid locale %q: must match pattern like en or en-US", locale)
	}
	return nil
}
