package gamecenter

import (
	"flag"
	"fmt"
	"os"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

const gameCenterReplacementConfirmWarning = "Warning: Game Center relationship replacement without --confirm is deprecated and will be rejected in 5.0.0; pass --confirm to acknowledge replacing existing relationships."

func validateGameCenterReplacementConfirm(fs *flag.FlagSet, confirmed bool) error {
	confirmProvided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "confirm" {
			confirmProvided = true
		}
	})
	if confirmProvided && !confirmed {
		return shared.UsageError("--confirm must be true when specified")
	}
	if !confirmed {
		fmt.Fprintln(os.Stderr, gameCenterReplacementConfirmWarning)
	}
	return nil
}
