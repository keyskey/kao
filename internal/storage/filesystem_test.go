package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/keyskey/kao/internal/evidence"
)

func TestFilesystemPutAndQuery(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFilesystem(filepath.Join(dir, "evidence"))
	if err != nil {
		t.Fatal(err)
	}

	collectedAt := time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC)
	rc := evidence.RepoControl{
		SchemaVersion: "v1",
		Type:          "repo_control",
		Provider:      "github",
		Repository:    "trading-api",
		CollectedAt:   collectedAt,
		BranchProtection: evidence.BranchProtection{Enabled: true},
	}

	ctx := context.Background()
	if err := fs.PutRepoControl(ctx, []evidence.RepoControl{rc}); err != nil {
		t.Fatal(err)
	}

	got, err := fs.QueryRepoControl(ctx, Filter{Date: "2026-06-06"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Repository != "trading-api" {
		t.Fatalf("got %+v", got)
	}
}

func TestFilesystemPutAndQueryInfraDeployment(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFilesystem(filepath.Join(dir, "evidence"))
	if err != nil {
		t.Fatal(err)
	}

	appliedAt := time.Date(2026, 6, 5, 16, 5, 0, 0, time.UTC)
	id := evidence.InfraDeployment{
		SchemaVersion: "v1",
		Type:          "infra_deployment",
		Provider:      "terraform",
		Repository:    "platform-infra",
		Workspace:     "prod-gke",
		AppliedAt:     appliedAt,
		Execution:     evidence.Execution{URL: "https://example.com/runs/1"},
	}

	ctx := context.Background()
	if err := fs.PutInfraDeployment(ctx, []evidence.InfraDeployment{id}); err != nil {
		t.Fatal(err)
	}

	got, err := fs.QueryInfraDeployment(ctx, Filter{Date: "2026-06-05"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Workspace != "prod-gke" {
		t.Fatalf("got %+v", got)
	}
}

func TestFilesystemPutAndQueryAppDeployment(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFilesystem(filepath.Join(dir, "evidence"))
	if err != nil {
		t.Fatal(err)
	}

	deployedAt := time.Date(2026, 6, 5, 16, 5, 0, 0, time.UTC)
	ad := evidence.AppDeployment{
		SchemaVersion: "v1",
		Type:          "app_deployment",
		Provider:      "argocd",
		Application:   "kao-prod",
		Repository:    "kao",
		Revision:      "abc123def4567890abc123def4567890abc12345",
		DeployedAt:    deployedAt,
	}

	ctx := context.Background()
	if err := fs.PutAppDeployment(ctx, []evidence.AppDeployment{ad}); err != nil {
		t.Fatal(err)
	}

	got, err := fs.QueryAppDeployment(ctx, Filter{Date: "2026-06-05", Repositories: []string{"kao"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Application != "kao-prod" {
		t.Fatalf("got %+v", got)
	}
}
