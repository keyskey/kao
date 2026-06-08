package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/keyskey/kao/internal/evidence"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(url string) (*Postgres, error) {
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) PutRepoControl(ctx context.Context, records []evidence.RepoControl) error {
	for _, r := range records {
		payload, err := json.Marshal(r)
		if err != nil {
			return err
		}
		_, err = p.pool.Exec(ctx, `
			INSERT INTO repo_control_evidence (repository, collected_at, payload)
			VALUES ($1, $2, $3)
			ON CONFLICT (repository, collected_at) DO UPDATE SET payload = EXCLUDED.payload
		`, r.Repository, r.CollectedAt, payload)
		if err != nil {
			return fmt.Errorf("upsert repo_control %s: %w", r.Repository, err)
		}
	}
	return nil
}

func (p *Postgres) PutCodeChange(ctx context.Context, records []evidence.CodeChange) error {
	for _, r := range records {
		payload, err := json.Marshal(r)
		if err != nil {
			return err
		}
		_, err = p.pool.Exec(ctx, `
			INSERT INTO code_change_evidence (repository, pr_number, payload)
			VALUES ($1, $2, $3)
			ON CONFLICT (repository, pr_number) DO UPDATE SET payload = EXCLUDED.payload
		`, r.Repository, r.PRNumber, payload)
		if err != nil {
			return fmt.Errorf("upsert code_change %s#%d: %w", r.Repository, r.PRNumber, err)
		}
	}
	return nil
}

func (p *Postgres) QueryRepoControl(ctx context.Context, filter Filter) ([]evidence.RepoControl, error) {
	query := `SELECT payload FROM repo_control_evidence WHERE 1=1`
	args := []any{}
	argN := 1

	if filter.Date != "" {
		query += fmt.Sprintf(` AND collected_at::date = $%d::date`, argN)
		args = append(args, filter.Date)
		argN++
	}
	if len(filter.Repositories) > 0 {
		query += fmt.Sprintf(` AND repository = ANY($%d)`, argN)
		args = append(args, filter.Repositories)
	}

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRepoControl(rows)
}

func (p *Postgres) QueryCodeChange(ctx context.Context, filter Filter) ([]evidence.CodeChange, error) {
	query := `SELECT payload FROM code_change_evidence WHERE 1=1`
	args := []any{}
	argN := 1

	if filter.Date != "" {
		query += fmt.Sprintf(` AND (payload->>'merged_at')::timestamptz::date = $%d::date`, argN)
		args = append(args, filter.Date)
		argN++
	}
	if len(filter.Repositories) > 0 {
		query += fmt.Sprintf(` AND repository = ANY($%d)`, argN)
		args = append(args, filter.Repositories)
	}

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCodeChange(rows)
}

func (p *Postgres) QueryCodeChangeForJoin(ctx context.Context, repos, shas []string, lookbackDays int) ([]evidence.CodeChange, error) {
	query := `
		SELECT payload FROM code_change_evidence
		WHERE (payload->>'merged_at')::timestamptz >= NOW() - ($1 || ' days')::interval
	`
	args := []any{lookbackDays}
	argN := 2

	if len(repos) > 0 {
		query += fmt.Sprintf(` AND repository = ANY($%d)`, argN)
		args = append(args, repos)
		argN++
	}
	if len(shas) > 0 {
		query += fmt.Sprintf(` AND payload->>'commit_sha' = ANY($%d)`, argN)
		args = append(args, shas)
	}

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCodeChange(rows)
}

func (p *Postgres) PutEvaluations(ctx context.Context, records []evidence.Evaluation) error {
	for _, r := range records {
		payload, err := json.Marshal(r)
		if err != nil {
			return err
		}
		_, err = p.pool.Exec(ctx, `
			INSERT INTO control_evaluations (control_id, resource_type, resource_id, evaluated_at, payload)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (control_id, resource_type, resource_id, evaluated_at) DO UPDATE SET payload = EXCLUDED.payload
		`, r.ControlID, r.ResourceType, r.ResourceID, r.EvaluatedAt, payload)
		if err != nil {
			return fmt.Errorf("upsert evaluation %s: %w", r.ControlID, err)
		}
	}
	return nil
}

type rowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanRepoControl(rows rowScanner) ([]evidence.RepoControl, error) {
	var result []evidence.RepoControl
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var rc evidence.RepoControl
		if err := json.Unmarshal(payload, &rc); err != nil {
			return nil, err
		}
		result = append(result, rc)
	}
	return result, rows.Err()
}

func scanCodeChange(rows rowScanner) ([]evidence.CodeChange, error) {
	var result []evidence.CodeChange
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var cc evidence.CodeChange
		if err := json.Unmarshal(payload, &cc); err != nil {
			return nil, err
		}
		result = append(result, cc)
	}
	return result, rows.Err()
}
