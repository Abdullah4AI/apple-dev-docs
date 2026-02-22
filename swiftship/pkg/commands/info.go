package commands

import (
	"github.com/Abdullah4AI/apple-developer-toolkit/swiftship/internal/service"
	"github.com/Abdullah4AI/apple-developer-toolkit/swiftship/internal/terminal"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show project status",
	Long:  "Display information about the current project.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfigWithProject()
		if err != nil {
			terminal.Info("No projects yet. Run `swiftship` to create one.")
			return nil
		}

		svc, err := service.NewService(cfg)
		if err != nil {
			return err
		}

		return svc.Info()
	},
}
