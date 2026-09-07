package shared

import (
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// VisibleUsageFunc renders command help while omitting deprecated commands from
// nested subcommand listings. Root-level deprecated commands are already hidden
// elsewhere; this keeps nested canonical help focused on current surfaces.
func VisibleUsageFunc(c *ffcli.Command) string {
	clone := *c
	if len(c.Subcommands) > 0 {
		visible := make([]*ffcli.Command, 0, len(c.Subcommands))
		for _, sub := range c.Subcommands {
			if sub == nil {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(sub.ShortHelp), "DEPRECATED:") {
				continue
			}
			visible = append(visible, sub)
		}
		clone.Subcommands = visible
	}
	return DefaultUsageFunc(&clone)
}
