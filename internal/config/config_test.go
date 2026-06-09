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

func TestLoadAppDeploymentDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kao.yaml")
	content := `
version: v1
providers:
  github:
    org: testorg
    token_env: GITHUB_TOKEN
  argocd:
    server_env: ARGOCD_SERVER
    token_env: ARGOCD_TOKEN
storage:
  backend: filesystem
  filesystem:
    path: ./evidence
evidence:
  app_deployment:
    provider: argocd
    scope:
      applications:
        include: [kao-prod]
controls: {}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Evidence.AppDeployment.Environment != "production" {
		t.Errorf("environment = %q, want production", cfg.Evidence.AppDeployment.Environment)
	}
	if *cfg.Controls.AppDeployment.Traceability.Required != true {
		t.Error("expected app_deployment.traceability.required default true")
	}
	if *cfg.Providers.ArgoCD.GRPCWeb != true {
		t.Error("expected grpc_web default true")
	}
	if !cfg.AppDeploymentEnabled() {
		t.Error("expected AppDeploymentEnabled true")
	}
}

func TestNormalizeArgoCDServerAddr(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://argocd.example.com", "argocd.example.com:443"},
		{"argocd.example.com:443", "argocd.example.com:443"},
		{"argocd.example.com", "argocd.example.com:443"},
	}
	for _, tt := range tests {
		got, err := normalizeArgoCDServerAddr(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("normalizeArgoCDServerAddr(%q) = (%q, %v), want %q", tt.in, got, err, tt.want)
		}
	}
}
