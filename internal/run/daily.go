package run

import (
	"context"
	"fmt"
	"time"

	"github.com/keyskey/kao/internal/collect"
	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evaluate"
)

func loadDailyConfig(path string) (*config.Config, error) {
	if path == "" {
		path = config.DefaultConfigPath
	}
	return config.Load(path)
}

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

	cfg, err := loadDailyConfig(opts.ConfigPath)
	if err != nil {
		return err
	}
	if cfg.AppDeploymentEnabled() {
		if err := collect.RunAppDeployment(ctx, collectOpts); err != nil {
			return fmt.Errorf("app-deployment collect: %w", err)
		}
	}

	if cfg.InfraDeploymentEnabled() {
		if err := collect.RunInfraDeployment(ctx, collectOpts); err != nil {
			return fmt.Errorf("infra-deployment collect: %w", err)
		}
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
