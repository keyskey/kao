package collect

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/scope"
	"github.com/keyskey/kao/internal/storage"
)

type Options struct {
	ConfigPath   string
	Repositories []string
	Output       string
	Date         string
}

func resolveOutput(cfg *config.Config, output string) (storage.Backend, io.Writer, error) {
	switch output {
	case "", "storage":
		backend, err := storage.New(cfg)
		if err != nil {
			return nil, nil, err
		}
		return backend, nil, nil
	case "-":
		return nil, os.Stdout, nil
	default:
		f, err := os.Create(output)
		if err != nil {
			return nil, nil, fmt.Errorf("create output file: %w", err)
		}
		return nil, f, nil
	}
}

func writeJSONL(w io.Writer, items []any) error {
	enc := json.NewEncoder(w)
	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return nil
}

func loadConfig(path string) (*config.Config, error) {
	if path == "" {
		path = config.DefaultConfigPath
	}
	return config.Load(path)
}

func repoList(cfg *config.Config, domain string, filter []string) []string {
	rs := cfg.RepositoriesFor(domain)
	return scope.Resolve(rs.Include, rs.Exclude, filter)
}
