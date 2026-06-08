package collect

import (
	"context"
	"fmt"
	"os"

	"github.com/keyskey/kao/internal/evidence"
	gh "github.com/keyskey/kao/internal/provider/github"
)

func RunRepoControl(ctx context.Context, opts Options) error {
	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	client, err := gh.New(cfg)
	if err != nil {
		return err
	}

	repos := repoList(cfg, "repo_control", opts.Repositories)
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

	var records []evidence.RepoControl
	var failures []error

	for _, repo := range repos {
		rc, err := client.CollectRepoControl(ctx, repo)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", repo, err))
			continue
		}
		records = append(records, rc)
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
		if err := backend.PutRepoControl(ctx, records); err != nil {
			return err
		}
	}

	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "collect repo-control: %v\n", f)
		}
		return fmt.Errorf("%d repository(s) failed", len(failures))
	}
	return nil
}
