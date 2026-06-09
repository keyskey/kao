package run

import (
	"context"
	"fmt"
	"time"

	"github.com/keyskey/kao/internal/collect"
	"github.com/keyskey/kao/internal/evaluate"
)

type DailyOptions struct {
	ConfigPath   string
	Date         string
	Repositories []string
}

func RunDaily(ctx context.Context, opts DailyOptions) error {
	date := opts.Date
	if date == "" {
		date = time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	}

	collectOpts := collect.Options{
		ConfigPath:   opts.ConfigPath,
		Repositories: opts.Repositories,
		Output:       "storage",
		Date:         date,
	}

	if err := collect.RunRepoControl(ctx, collectOpts); err != nil {
		return fmt.Errorf("repo-control collect: %w", err)
	}

	if err := collect.RunCodeChange(ctx, collectOpts); err != nil {
		return fmt.Errorf("code-change collect: %w", err)
	}

	if err := collect.RunInfraDeployment(ctx, collectOpts); err != nil {
		return fmt.Errorf("infra-deployment collect: %w", err)
	}

	evalOpts := evaluate.Options{
		ConfigPath:   opts.ConfigPath,
		Date:         date,
		Repositories: opts.Repositories,
		Input:        "storage",
		Output:       "storage",
	}

	if err := evaluate.Run(ctx, evalOpts); err != nil {
		return fmt.Errorf("evaluate: %w", err)
	}

	return nil
}
