package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/keyskey/kao/internal/evidence"
)

type Filesystem struct {
	basePath string
}

func NewFilesystem(basePath string) (*Filesystem, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("create storage path: %w", err)
	}
	return &Filesystem{basePath: basePath}, nil
}

func (f *Filesystem) PutRepoControl(ctx context.Context, records []evidence.RepoControl) error {
	byDate := make(map[string][]evidence.RepoControl)
	for _, r := range records {
		d := r.CollectedAt.UTC().Format("2006-01-02")
		byDate[d] = append(byDate[d], r)
	}
	for date, batch := range byDate {
		if err := f.writeJSONLBatch("repo_control", date, toAnySlice(batch)); err != nil {
			return err
		}
	}
	return nil
}

func (f *Filesystem) PutInfraDeployment(ctx context.Context, records []evidence.InfraDeployment) error {
	byDate := make(map[string][]evidence.InfraDeployment)
	for _, r := range records {
		d := r.AppliedAt.UTC().Format("2006-01-02")
		byDate[d] = append(byDate[d], r)
	}
	for date, batch := range byDate {
		if err := f.writeJSONLBatch("infra_deployment", date, toAnySlice(batch)); err != nil {
			return err
		}
	}
	return nil
}

func (f *Filesystem) PutCodeChange(ctx context.Context, records []evidence.CodeChange) error {
	byDate := make(map[string][]evidence.CodeChange)
	for _, r := range records {
		d := r.MergedAt.UTC().Format("2006-01-02")
		byDate[d] = append(byDate[d], r)
	}
	for date, batch := range byDate {
		if err := f.writeJSONLBatch("code_change", date, toAnySlice(batch)); err != nil {
			return err
		}
	}
	return nil
}

func toAnySlice[T any](items []T) []any {
	out := make([]any, len(items))
	for i, v := range items {
		out[i] = v
	}
	return out
}

func (f *Filesystem) writeJSONLBatch(evidenceType, date string, batch []any) error {
	path := filepath.Join(f.basePath, evidenceType, date+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	existing, _ := f.readJSONLFile(path)
	merged := append(existing, batch...)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(file)
	for _, item := range merged {
		data, err := json.Marshal(item)
		if err != nil {
			file.Close()
			return err
		}
		if _, err := w.Write(data); err != nil {
			file.Close()
			return err
		}
		if err := w.WriteByte('\n'); err != nil {
			file.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func (f *Filesystem) QueryRepoControl(ctx context.Context, filter Filter) ([]evidence.RepoControl, error) {
	all, err := f.readRepoControlFile(filter.Date)
	if err != nil {
		return nil, err
	}
	return filterRepoControl(all, filter.Repositories), nil
}

func (f *Filesystem) QueryInfraDeployment(ctx context.Context, filter Filter) ([]evidence.InfraDeployment, error) {
	all, err := f.readInfraDeploymentFile(filter.Date)
	if err != nil {
		return nil, err
	}
	return filterInfraDeployment(all, filter.Repositories), nil
}

func (f *Filesystem) QueryCodeChange(ctx context.Context, filter Filter) ([]evidence.CodeChange, error) {
	all, err := f.readCodeChangeFile(filter.Date)
	if err != nil {
		return nil, err
	}
	return filterCodeChange(all, filter.Repositories), nil
}

func (f *Filesystem) readRepoControlFile(date string) ([]evidence.RepoControl, error) {
	path := filepath.Join(f.basePath, "repo_control", date+".jsonl")
	raw, err := f.readJSONLFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []evidence.RepoControl
	for _, item := range raw {
		if rc, ok := item.(evidence.RepoControl); ok {
			result = append(result, rc)
		}
	}
	return result, nil
}

func (f *Filesystem) readInfraDeploymentFile(date string) ([]evidence.InfraDeployment, error) {
	path := filepath.Join(f.basePath, "infra_deployment", date+".jsonl")
	raw, err := f.readJSONLFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []evidence.InfraDeployment
	for _, item := range raw {
		if id, ok := item.(evidence.InfraDeployment); ok {
			result = append(result, id)
		}
	}
	return result, nil
}

func (f *Filesystem) readCodeChangeFile(date string) ([]evidence.CodeChange, error) {
	path := filepath.Join(f.basePath, "code_change", date+".jsonl")
	raw, err := f.readJSONLFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result []evidence.CodeChange
	for _, item := range raw {
		if cc, ok := item.(evidence.CodeChange); ok {
			result = append(result, cc)
		}
	}
	return result, nil
}

func (f *Filesystem) QueryCodeChangeForJoin(ctx context.Context, repos, shas []string, lookbackDays int) ([]evidence.CodeChange, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -lookbackDays)
	var result []evidence.CodeChange

	dir := filepath.Join(f.basePath, "code_change")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		dateStr := strings.TrimSuffix(e.Name(), ".jsonl")
		d, err := time.Parse("2006-01-02", dateStr)
		if err != nil || d.Before(cutoff) {
			continue
		}
		items, err := f.readJSONLFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			cc, ok := item.(evidence.CodeChange)
			if !ok {
				continue
			}
			if matchesJoin(cc, repos, shas) {
				result = append(result, cc)
			}
		}
	}
	return result, nil
}

func (f *Filesystem) PutEvaluations(ctx context.Context, records []evidence.Evaluation) error {
	if len(records) == 0 {
		return nil
	}
	date := records[0].EvaluatedAt.UTC().Format("2006-01-02")
	path := filepath.Join(f.basePath, "evaluations", date+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	existing, _ := f.readJSONLFile(path)
	merged := append(existing, toAnySlice(records)...)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	w := bufio.NewWriter(file)
	for _, item := range merged {
		data, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := w.Write(append(data, '\n')); err != nil {
			return err
		}
	}
	return w.Flush()
}

func (f *Filesystem) readJSONLFile(path string) ([]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []any
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, err
		}
		var typeCheck struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeCheck); err != nil {
			return nil, err
		}
		switch typeCheck.Type {
		case "repo_control":
			var rc evidence.RepoControl
			if err := json.Unmarshal(raw, &rc); err != nil {
				return nil, err
			}
			result = append(result, rc)
		case "code_change":
			var cc evidence.CodeChange
			if err := json.Unmarshal(raw, &cc); err != nil {
				return nil, err
			}
			result = append(result, cc)
		case "infra_deployment":
			var id evidence.InfraDeployment
			if err := json.Unmarshal(raw, &id); err != nil {
				return nil, err
			}
			result = append(result, id)
		default:
			var ev evidence.Evaluation
			if err := json.Unmarshal(raw, &ev); err == nil && ev.ControlID != "" {
				result = append(result, ev)
			}
		}
	}
	return result, scanner.Err()
}

func filterRepoControl(items []evidence.RepoControl, repos []string) []evidence.RepoControl {
	if len(repos) == 0 {
		return items
	}
	set := toSet(repos)
	var out []evidence.RepoControl
	for _, item := range items {
		if set[item.Repository] {
			out = append(out, item)
		}
	}
	return out
}

func filterInfraDeployment(items []evidence.InfraDeployment, repos []string) []evidence.InfraDeployment {
	if len(repos) == 0 {
		return items
	}
	set := toSet(repos)
	var out []evidence.InfraDeployment
	for _, item := range items {
		if set[item.Repository] {
			out = append(out, item)
		}
	}
	return out
}

func filterCodeChange(items []evidence.CodeChange, repos []string) []evidence.CodeChange {
	if len(repos) == 0 {
		return items
	}
	set := toSet(repos)
	var out []evidence.CodeChange
	for _, item := range items {
		if set[item.Repository] {
			out = append(out, item)
		}
	}
	return out
}

func matchesJoin(cc evidence.CodeChange, repos, shas []string) bool {
	repoOK := len(repos) == 0
	for _, r := range repos {
		if r == cc.Repository {
			repoOK = true
			break
		}
	}
	shaOK := len(shas) == 0
	for _, s := range shas {
		if s == cc.CommitSHA {
			shaOK = true
			break
		}
	}
	return repoOK && shaOK
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, s := range items {
		m[s] = true
	}
	return m
}
