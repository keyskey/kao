package collect

import (
	"context"
	"fmt"
	"os"

	"github.com/keyskey/kao/internal/evidence"
	gh "github.com/keyskey/kao/internal/provider/github"
)

func RunInfraDeployment(ctx context.Context, opts Options) error {
	if opts.Date == "" {
		return fmt.Errorf("--date is required for infra-deployment collection")
	}

	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	if !cfg.InfraDeploymentEnabled() {
		return fmt.Errorf("infra_deployment collection requires evidence.infra_deployment.provider and apply_workflows or apply_job_names in config")
	}

	client, err := gh.New(cfg)
	if err != nil {
		return err
	}

	repos := repoList(cfg, "infra_deployment", opts.Repositories)
	if len(repos) == 0 {
		return fmt.Errorf("no repositories to collect")
	}

	backend, writer, err := resolveOutput(cfg, opts.Output)
	if err != nil {
		return err
	}
	if writer != nil {
		defer func() {
			if f, ok := writer.(*os.File); ok && f != os.Stdout {
				f.Close()
			}
		}()
	}

	var records []evidence.InfraDeployment
	var failures []error

	for _, repo := range repos {
		deployments, err := client.CollectInfraDeployment(ctx, repo, opts.Date, cfg)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", repo, err))
			continue
		}
		records = append(records, deployments...)
	}

	if writer != nil {
		items := make([]any, len(records))
		for i, r := range records {
			items[i] = r
		}
		if err := writeJSONL(writer, items); err != nil {
			return err
		}
	} else if backend != nil && len(records) > 0 {
		if err := backend.PutInfraDeployment(ctx, records); err != nil {
			return err
		}
	}

	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "collect infra-deployment: %v\n", f)
		}
		return fmt.Errorf("%d repository(s) failed", len(failures))
	}
	return nil
}
