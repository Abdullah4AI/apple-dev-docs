package subscriptions

import (
	"flag"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

// Product-scoped subscription localization commands sit between two selector
// vocabularies: sibling subscription commands select a product with
// `--subscription-id`/`--product-id`, while the version-scoped replacements
// select a subscription version with `--version-id`. Spellings that mean the
// same resource as the canonical flag become hidden aliases. Spellings that
// name a different resource are recognized only so the command can reject them
// with the distinction spelled out, because leaving them unknown lets the
// parser suggest a same-shaped flag that would honor the wrong mental model.

const (
	subscriptionLocalizationCreateVersionSelectorGuidance      = "subscriptions localizations create: --version-id is not accepted here; this product-scoped command selects the subscription with --subscription-id. Use `asc subscriptions versions localizations create --version-id \"SUBSCRIPTION_VERSION_ID\" --locale \"LOCALE\" --name \"NAME\"` for version-scoped localizations."
	subscriptionLocalizationUpdateSubscriptionSelectorGuidance = "subscriptions localizations update: --subscription-id is not accepted here; --id is the localization ID, not the subscription ID. Run `asc subscriptions localizations list --subscription-id \"SUB_ID\"` to find LOCALIZATION_ID, then rerun with --id \"LOCALIZATION_ID\"."
)

// subscriptionLocalizationRejectedSelector records whether a caller supplied a
// selector spelling that names a different resource than the command addresses.
type subscriptionLocalizationRejectedSelector struct {
	value string
	set   bool
}

// String returns the supplied value for flag.Value formatting.
func (f *subscriptionLocalizationRejectedSelector) String() string {
	if f == nil {
		return ""
	}
	return f.value
}

// Set records that the rejected spelling was explicitly supplied.
func (f *subscriptionLocalizationRejectedSelector) Set(value string) error {
	f.value = value
	f.set = true
	return nil
}

// Used reports whether the rejected spelling was explicitly supplied.
func (f *subscriptionLocalizationRejectedSelector) Used() bool {
	return f != nil && f.set
}

// bindSubscriptionLocalizationRejectedSelector recognizes a selector spelling
// without advertising it, so the command can explain the resource distinction
// instead of failing as an unknown flag.
func bindSubscriptionLocalizationRejectedSelector(fs *flag.FlagSet, name, usage string) *subscriptionLocalizationRejectedSelector {
	selector := &subscriptionLocalizationRejectedSelector{}
	fs.Var(selector, name, usage)
	shared.HideFlagFromHelp(fs.Lookup(name))
	return selector
}

// rejectSubscriptionLocalizationSelector fails before command side effects when
// a caller selected the wrong resource, and never populates the canonical flag.
func rejectSubscriptionLocalizationSelector(selector *subscriptionLocalizationRejectedSelector, guidance, parameter string) error {
	if !selector.Used() {
		return nil
	}
	return shared.WithDiagnostic(
		shared.UsageError(guidance),
		shared.DiagnosticInvalidInput,
		parameter,
	)
}
