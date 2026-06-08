package collect

import (
	"context"
	"fmt"
	"os"

	"github.com/keyskey/kao/internal/evidence"
	gh "github.com/keyskey/kao/internal/provider/github"
)

func RunCodeChange(ctx context.Context, opts Options) error {
	if opts.Date == "" {
		return fmt.Errorf("--date is required for code-change collection")
	}

	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	client, err := gh.New(cfg)
	if err != nil {
		return err
	}

	repos := repoList(cfg, "code_change", opts.Repositories)
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

	var records []evidence.CodeChange
	var failures []error

	for _, repo := range repos {
		ghRepo, _, err := client.API().Repositories.Get(ctx, client.Org(), repo)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: get repo: %w", repo, err))
			continue
		}
		defaultBranch := ghRepo.GetDefaultBranch()
		if defaultBranch == "" {
			defaultBranch = "main"
		}

		changes, err := client.CollectCodeChange(ctx, repo, defaultBranch, opts.Date)
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", repo, err))
			continue
		}
		records = append(records, changes...)
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
		if err := backend.PutCodeChange(ctx, records); err != nil {
			return err
		}
	}

	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "collect code-change: %v\n", f)
		}
		return fmt.Errorf("%d repository(s) failed", len(failures))
	}
	return nil
}
