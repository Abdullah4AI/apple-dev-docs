package commands

import (
	"github.com/Abdullah4AI/apple-developer-toolkit/swiftship/internal/service"
	"github.com/Abdullah4AI/apple-developer-toolkit/swiftship/internal/terminal"
	"github.com/spf13/cobra"
)

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Auto-fix compilation errors",
	Long:  "Build the project and automatically fix any compilation errors.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfigWithProject()
		if err != nil {
			terminal.Error("No project found.")
			terminal.Info("Run `swiftship` first to create a project.")
			return err
		}

		svc, err := service.NewService(cfg, service.ServiceOpts{Model: ModelFlag()})
		if err != nil {
			return err
		}

		return svc.Fix(cmd.Context())
	},
}
