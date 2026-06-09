package collect

import (
	"context"
	"fmt"
	"os"

	ac "github.com/keyskey/kao/internal/provider/argocd"
)

func RunAppDeployment(ctx context.Context, opts Options) error {
	if opts.Date == "" {
		return fmt.Errorf("--date is required for app-deployment collection")
	}

	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	if !cfg.AppDeploymentEnabled() {
		return fmt.Errorf("app_deployment collection requires providers.argocd and evidence.app_deployment.provider in config")
	}

	client, err := ac.New(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

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

	records, err := client.CollectAppDeployment(ctx, opts.Date, cfg)
	if err != nil && len(records) == 0 {
		return err
	}

	records = ac.FilterByRepository(records, opts.Repositories)

	if writer != nil {
		items := make([]any, len(records))
		for i, r := range records {
			items[i] = r
		}
		if err := writeJSONL(writer, items); err != nil {
			return err
		}
	} else if backend != nil && len(records) > 0 {
		if err := backend.PutAppDeployment(ctx, records); err != nil {
			return err
		}
	}

	return err
}
