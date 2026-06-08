package run

import (
	"context"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	var configPath string
	var date string
	var repositories []string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run scheduled jobs",
	}

	dailyCmd := &cobra.Command{
		Use:   "daily",
		Short: "Run daily collection and evaluation (GitHub evidence only in MVP)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDaily(context.Background(), DailyOptions{
				ConfigPath:   configPath,
				Date:         date,
				Repositories: repositories,
			})
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", "", "path to kao.yaml")
	dailyCmd.Flags().StringVar(&date, "date", "", "target date (YYYY-MM-DD, UTC). default: yesterday")
	dailyCmd.Flags().StringArrayVar(&repositories, "repository", nil, "limit to repository (repeatable)")

	cmd.AddCommand(dailyCmd)
	return cmd
}
