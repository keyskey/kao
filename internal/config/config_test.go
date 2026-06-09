package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kao.yaml")
	content := `
version: v1
providers:
  github:
    org: testorg
    token_env: GITHUB_TOKEN
storage:
  backend: filesystem
  filesystem:
    path: ./evidence
scope:
  defaults:
    repositories:
      include: [repo-a]
evidence:
  repo_control:
    provider: github
  code_change:
    provider: github
    ci: [github_actions]
controls: {}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if *cfg.Controls.RepoControl.BranchProtection.Enabled != true {
		t.Error("expected branch_protection.enabled default true")
	}
	if *cfg.Controls.RepoControl.BranchProtection.RequiredReviews != 1 {
		t.Error("expected required_reviews default 1")
	}
	if *cfg.Evaluation.CodeChange.LookbackDays != 90 {
		t.Error("expected lookback_days default 90")
	}
	if cfg.Evidence.InfraDeployment.Environment != "production" {
		t.Error("expected infra_deployment.environment default production")
	}
	if cfg.Evidence.InfraDeployment.WorkspaceJobNamePattern == "" {
		t.Error("expected workspace_job_name_pattern default")
	}
	if *cfg.Controls.InfraDeployment.Traceability.Required != true {
		t.Error("expected infra_deployment.traceability.required default true")
	}
}
