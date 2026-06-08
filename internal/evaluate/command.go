package evaluate

import (
	"context"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	var opts Options

	cmd := &cobra.Command{
		Use:   "evaluate",
		Short: "Evaluate collected evidence against control policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run(context.Background(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.ConfigPath, "config", "", "path to kao.yaml")
	cmd.Flags().StringVar(&opts.Date, "date", "", "target date (YYYY-MM-DD, UTC)")
	cmd.Flags().StringArrayVar(&opts.Repositories, "repository", nil, "limit to repository (repeatable)")
	cmd.Flags().StringVar(&opts.Input, "input", "storage", "input source: storage or files")
	cmd.Flags().StringVar(&opts.Output, "output", "storage", "output destination: storage, -, or file path")
	cmd.Flags().StringVar(&opts.Format, "format", "jsonl", "output format when --output is - or file: jsonl, json")

	cmd.Flags().StringVar(&opts.RepoControlFile, "repo-control", "", "repo_control JSONL file (--input files)")
	cmd.Flags().StringArrayVar(&opts.CodeChangeFiles, "code-change", nil, "code_change JSONL file (--input files, repeatable)")

	return cmd
}
