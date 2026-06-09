package storage

import (
	"context"
	"fmt"

	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
)

type Filter struct {
	Date         string
	Repositories []string
}

type Backend interface {
	PutRepoControl(ctx context.Context, records []evidence.RepoControl) error
	PutCodeChange(ctx context.Context, records []evidence.CodeChange) error
	PutAppDeployment(ctx context.Context, records []evidence.AppDeployment) error
	PutInfraDeployment(ctx context.Context, records []evidence.InfraDeployment) error
	QueryRepoControl(ctx context.Context, filter Filter) ([]evidence.RepoControl, error)
	QueryCodeChange(ctx context.Context, filter Filter) ([]evidence.CodeChange, error)
	QueryAppDeployment(ctx context.Context, filter Filter) ([]evidence.AppDeployment, error)
	QueryInfraDeployment(ctx context.Context, filter Filter) ([]evidence.InfraDeployment, error)
	QueryCodeChangeForJoin(ctx context.Context, repos, shas []string, lookbackDays int) ([]evidence.CodeChange, error)
	PutEvaluations(ctx context.Context, records []evidence.Evaluation) error
}

func New(cfg *config.Config) (Backend, error) {
	switch cfg.Storage.Backend {
	case "filesystem":
		if cfg.Storage.Filesystem == nil {
			return nil, fmt.Errorf("storage.filesystem is not configured")
		}
		return NewFilesystem(cfg.Storage.Filesystem.Path)
	case "postgres":
		url, err := cfg.DatabaseURL()
		if err != nil {
			return nil, err
		}
		return NewPostgres(url)
	default:
		return nil, fmt.Errorf("unsupported storage backend: %q", cfg.Storage.Backend)
	}
}
