package subscriptions

import (
	"flag"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func requiredPositiveIntegerUsageError(fs *flag.FlagSet, name string) error {
	parameter := "--" + name
	if flagWasProvided(fs, name) {
		return shared.InvalidValueUsageError(parameter)
	}
	return shared.MissingRequiredUsageError(parameter)
}
