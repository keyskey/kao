package evaluate

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
	"github.com/keyskey/kao/internal/storage"
)

type Options struct {
	ConfigPath   string
	Date         string
	Repositories []string
	Input        string
	Output       string
	Format       string

	RepoControlFile string
	CodeChangeFiles []string
}

func Run(ctx context.Context, opts Options) error {
	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		return err
	}

	evaluatedAt := time.Now().UTC()
	var repoControls []evidence.RepoControl
	var codeChanges []evidence.CodeChange

	switch opts.Input {
	case "", "storage":
		backend, err := storage.New(cfg)
		if err != nil {
			return err
		}
		filter := storage.Filter{
			Date:         opts.Date,
			Repositories: opts.Repositories,
		}
		repoControls, err = backend.QueryRepoControl(ctx, filter)
		if err != nil {
			return err
		}
		codeChanges, err = backend.QueryCodeChange(ctx, filter)
		if err != nil {
			return err
		}
	case "files":
		repoControls, err = readRepoControlFile(opts.RepoControlFile)
		if err != nil {
			return err
		}
		for _, path := range opts.CodeChangeFiles {
			changes, err := readCodeChangeFile(path)
			if err != nil {
				return err
			}
			codeChanges = append(codeChanges, changes...)
		}
	default:
		return fmt.Errorf("unsupported input: %q", opts.Input)
	}

	results := EvaluateAll(cfg, repoControls, codeChanges, evaluatedAt)

	return writeResults(ctx, cfg, opts, results)
}

func EvaluateAll(cfg *config.Config, repoControls []evidence.RepoControl, codeChanges []evidence.CodeChange, evaluatedAt time.Time) []evidence.Evaluation {
	var results []evidence.Evaluation
	for _, rc := range repoControls {
		results = append(results, evaluateRepoControl(cfg, rc, evaluatedAt)...)
	}
	for _, cc := range codeChanges {
		rc := findRepoControlForCodeChange(repoControls, cc)
		results = append(results, evaluateCodeChange(cfg, cc, rc, evaluatedAt)...)
	}
	return results
}

func writeResults(ctx context.Context, cfg *config.Config, opts Options, results []evidence.Evaluation) error {
	switch opts.Output {
	case "", "storage":
		backend, err := storage.New(cfg)
		if err != nil {
			return err
		}
		return backend.PutEvaluations(ctx, results)
	case "-":
		return writeEvaluations(os.Stdout, results, opts.Format)
	default:
		f, err := os.Create(opts.Output)
		if err != nil {
			return err
		}
		defer f.Close()
		return writeEvaluations(f, results, opts.Format)
	}
}

func writeEvaluations(w io.Writer, results []evidence.Evaluation, format string) error {
	if format == "" || format == "jsonl" {
		enc := json.NewEncoder(w)
		for _, r := range results {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	}
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	return fmt.Errorf("unsupported format: %q", format)
}

func loadConfig(path string) (*config.Config, error) {
	if path == "" {
		path = config.DefaultConfigPath
	}
	return config.Load(path)
}

func readRepoControlFile(path string) ([]evidence.RepoControl, error) {
	if path == "" {
		return nil, nil
	}
	return readJSONLFile[evidence.RepoControl](path)
}

func readCodeChangeFile(path string) ([]evidence.CodeChange, error) {
	return readJSONLFile[evidence.CodeChange](path)
}

func readJSONLFile[T any](path string) ([]T, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var result []T
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var item T
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, scanner.Err()
}
