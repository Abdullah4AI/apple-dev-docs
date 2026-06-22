package storekit

import (
	"flag"
	"strings"

	"github.com/Abdullah4AI/apple-developer-toolkit/appstore/internal/cli/shared"
)

func rejectUnexpectedArgs(args []string) error {
	if len(args) == 0 {
		return nil
	}
	return shared.UsageErrorf("unexpected argument(s): %s", strings.Join(args, " "))
}

func flagWasSet(fs *flag.FlagSet, names ...string) bool {
	set := map[string]struct{}{}
	fs.Visit(func(item *flag.Flag) { set[item.Name] = struct{}{} })
	for _, name := range names {
		if _, ok := set[name]; ok {
			return true
		}
	}
	return false
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
