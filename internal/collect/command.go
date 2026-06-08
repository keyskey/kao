package collect

import (
	"context"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	var configPath string
	var repositories []string
	var output string
	var date string

	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect evidence from providers",
	}

	repoControlCmd := &cobra.Command{
		Use:   "repo-control",
		Short: "Collect repository control snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRepoControl(context.Background(), Options{
				ConfigPath:   configPath,
				Repositories: repositories,
				Output:       output,
			})
		},
	}

	codeChangeCmd := &cobra.Command{
		Use:   "code-change",
		Short: "Collect merged PR evidence for a date",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunCodeChange(context.Background(), Options{
				ConfigPath:   configPath,
				Repositories: repositories,
				Output:       output,
				Date:         date,
			})
		},
	}

	for _, c := range []*cobra.Command{cmd, repoControlCmd, codeChangeCmd} {
		c.PersistentFlags().StringVar(&configPath, "config", "", "path to kao.yaml")
	}
	repoControlCmd.Flags().StringVar(&output, "output", "storage", "output destination: storage, -, or file path")
	codeChangeCmd.Flags().StringVar(&output, "output", "storage", "output destination: storage, -, or file path")
	codeChangeCmd.Flags().StringVar(&date, "date", "", "target date (YYYY-MM-DD, UTC)")

	repoControlCmd.Flags().StringArrayVar(&repositories, "repository", nil, "limit to repository (repeatable)")
	codeChangeCmd.Flags().StringArrayVar(&repositories, "repository", nil, "limit to repository (repeatable)")

	cmd.AddCommand(repoControlCmd)
	cmd.AddCommand(codeChangeCmd)
	return cmd
}
