package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "kao.yaml"

type Config struct {
	Version    string           `yaml:"version"`
	Providers  Providers        `yaml:"providers"`
	Scope      Scope            `yaml:"scope"`
	Evidence   Evidence         `yaml:"evidence"`
	Controls   Controls         `yaml:"controls"`
	Storage    Storage          `yaml:"storage"`
	Evaluation EvaluationConfig `yaml:"evaluation"`
}

type Providers struct {
	GitHub GitHubProvider `yaml:"github"`
	ArgoCD ArgoCDProvider `yaml:"argocd"`
}

type ArgoCDProvider struct {
	ServerEnv string `yaml:"server_env"`
	TokenEnv  string `yaml:"token_env"`
	GRPCWeb   *bool  `yaml:"grpc_web,omitempty"`
	Insecure  *bool  `yaml:"insecure,omitempty"`
}

type GitHubProvider struct {
	Org      string `yaml:"org"`
	TokenEnv string `yaml:"token_env"`
}

type Scope struct {
	Defaults ScopeDefaults `yaml:"defaults"`
}

type ScopeDefaults struct {
	Repositories RepositoryScope `yaml:"repositories"`
}

type RepositoryScope struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

type Evidence struct {
	RepoControl     EvidenceRepoControl     `yaml:"repo_control"`
	CodeChange      EvidenceCodeChange      `yaml:"code_change"`
	AppDeployment   EvidenceAppDeployment   `yaml:"app_deployment"`
	InfraDeployment EvidenceInfraDeployment `yaml:"infra_deployment"`
}

type AppDeploymentScope struct {
	Applications RepositoryScope `yaml:"applications"`
}

type EvidenceAppDeployment struct {
	Provider    string              `yaml:"provider"`
	Environment string              `yaml:"environment"`
	Scope       *AppDeploymentScope `yaml:"scope,omitempty"`
}

type EvidenceRepoControl struct {
	Provider string           `yaml:"provider"`
	Scope    *RepositoryScope `yaml:"scope,omitempty"`
}

type EvidenceCodeChange struct {
	Provider string           `yaml:"provider"`
	CI       []string         `yaml:"ci"`
	Scope    *RepositoryScope `yaml:"scope,omitempty"`
}

type InfraDeploymentScope struct {
	Repositories RepositoryScope `yaml:"repositories"`
	Workspaces   RepositoryScope `yaml:"workspaces"`
}

type EvidenceInfraDeployment struct {
	Provider                string                `yaml:"provider"`
	Environment             string                `yaml:"environment"`
	Scope                   *InfraDeploymentScope `yaml:"scope,omitempty"`
	ApplyWorkflows          []string              `yaml:"apply_workflows"`
	ApplyJobNames           []string              `yaml:"apply_job_names"`
	WorkspaceJobNamePattern string                `yaml:"workspace_job_name_pattern"`
}

type Controls struct {
	RepoControl     RepoControlControls     `yaml:"repo_control"`
	CodeChange      CodeChangeControls      `yaml:"code_change"`
	AppDeployment   AppDeploymentControls   `yaml:"app_deployment"`
	InfraDeployment InfraDeploymentControls `yaml:"infra_deployment"`
}

type AppDeploymentControls struct {
	Traceability AppDeploymentTraceabilityControls `yaml:"traceability"`
}

type AppDeploymentTraceabilityControls struct {
	Required *bool `yaml:"required,omitempty"`
}

type InfraDeploymentControls struct {
	Traceability InfraDeploymentTraceabilityControls `yaml:"traceability"`
}

type InfraDeploymentTraceabilityControls struct {
	Required *bool `yaml:"required,omitempty"`
}

type RepoControlControls struct {
	BranchProtection BranchProtectionControls `yaml:"branch_protection"`
	Codeowners       CodeownersControls       `yaml:"codeowners"`
}

type BranchProtectionControls struct {
	Enabled                *bool `yaml:"enabled,omitempty"`
	RequiredReviews        *int  `yaml:"required_reviews,omitempty"`
	RequireCodeownerReview *bool `yaml:"require_codeowner_review,omitempty"`
	AllowForcePush         *bool `yaml:"allow_force_push,omitempty"`
	AllowBranchDeletion    *bool `yaml:"allow_branch_deletion,omitempty"`
}

type CodeownersControls struct {
	Required *bool `yaml:"required,omitempty"`
}

type CodeChangeControls struct {
	CI       CodeChangeCIControls       `yaml:"ci"`
	Approver CodeChangeApproverControls `yaml:"approver"`
}

type CodeChangeCIControls struct {
	RequirePass *bool `yaml:"require_pass,omitempty"`
}

type CodeChangeApproverControls struct {
	MustBeAuthorized *bool `yaml:"must_be_authorized,omitempty"`
}

type Storage struct {
	Backend    string             `yaml:"backend"`
	Postgres   *PostgresStorage   `yaml:"postgres,omitempty"`
	Filesystem *FilesystemStorage `yaml:"filesystem,omitempty"`
}

type PostgresStorage struct {
	URLEnv string `yaml:"url_env"`
}

type FilesystemStorage struct {
	Path string `yaml:"path"`
}

type EvaluationConfig struct {
	CodeChange CodeChangeEvaluation `yaml:"code_change"`
}

type CodeChangeEvaluation struct {
	LookbackDays *int `yaml:"lookback_days,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	bp := &cfg.Controls.RepoControl.BranchProtection
	if bp.Enabled == nil {
		v := true
		bp.Enabled = &v
	}
	if bp.RequiredReviews == nil {
		v := 1
		bp.RequiredReviews = &v
	}
	if bp.RequireCodeownerReview == nil {
		v := true
		bp.RequireCodeownerReview = &v
	}
	if bp.AllowForcePush == nil {
		v := false
		bp.AllowForcePush = &v
	}
	if bp.AllowBranchDeletion == nil {
		v := false
		bp.AllowBranchDeletion = &v
	}

	co := &cfg.Controls.RepoControl.Codeowners
	if co.Required == nil {
		v := true
		co.Required = &v
	}

	cc := &cfg.Controls.CodeChange
	if cc.CI.RequirePass == nil {
		v := true
		cc.CI.RequirePass = &v
	}
	if cc.Approver.MustBeAuthorized == nil {
		v := true
		cc.Approver.MustBeAuthorized = &v
	}

	if cfg.Evaluation.CodeChange.LookbackDays == nil {
		v := 90
		cfg.Evaluation.CodeChange.LookbackDays = &v
	}

	id := &cfg.Evidence.InfraDeployment
	if id.Environment == "" {
		id.Environment = "production"
	}
	if id.WorkspaceJobNamePattern == "" {
		id.WorkspaceJobNamePattern = `terraform apply \(([^,)]+)`
	}

	ad := &cfg.Evidence.AppDeployment
	if ad.Environment == "" {
		ad.Environment = "production"
	}

	adc := &cfg.Controls.AppDeployment.Traceability
	if adc.Required == nil {
		v := true
		adc.Required = &v
	}

	if cfg.Providers.ArgoCD.GRPCWeb == nil {
		v := true
		cfg.Providers.ArgoCD.GRPCWeb = &v
	}
	if cfg.Providers.ArgoCD.Insecure == nil {
		v := false
		cfg.Providers.ArgoCD.Insecure = &v
	}

	idc := &cfg.Controls.InfraDeployment.Traceability
	if idc.Required == nil {
		v := true
		idc.Required = &v
	}
}

func (c *Config) GitHubToken() (string, error) {
	env := c.Providers.GitHub.TokenEnv
	if env == "" {
		env = "GITHUB_TOKEN"
	}
	token := os.Getenv(env)
	if token == "" {
		return "", fmt.Errorf("environment variable %s is not set", env)
	}
	return token, nil
}

func (c *Config) DatabaseURL() (string, error) {
	if c.Storage.Postgres == nil {
		return "", fmt.Errorf("storage.postgres is not configured")
	}
	env := c.Storage.Postgres.URLEnv
	if env == "" {
		env = "DATABASE_URL"
	}
	url := os.Getenv(env)
	if url == "" {
		return "", fmt.Errorf("environment variable %s is not set", env)
	}
	return url, nil
}

func (c *Config) RepositoriesFor(domain string) RepositoryScope {
	switch domain {
	case "repo_control":
		if c.Evidence.RepoControl.Scope != nil {
			return *c.Evidence.RepoControl.Scope
		}
	case "code_change":
		if c.Evidence.CodeChange.Scope != nil {
			return *c.Evidence.CodeChange.Scope
		}
	case "infra_deployment":
		if c.Evidence.InfraDeployment.Scope != nil && len(c.Evidence.InfraDeployment.Scope.Repositories.Include) > 0 {
			return c.Evidence.InfraDeployment.Scope.Repositories
		}
	}
	return c.Scope.Defaults.Repositories
}

func (c *Config) WorkspacesForInfraDeployment() RepositoryScope {
	if c.Evidence.InfraDeployment.Scope != nil {
		return c.Evidence.InfraDeployment.Scope.Workspaces
	}
	return RepositoryScope{}
}

func (c *Config) InfraDeploymentConfig() EvidenceInfraDeployment {
	return c.Evidence.InfraDeployment
}

func (c *Config) AppDeploymentEnabled() bool {
	if c.Evidence.AppDeployment.Provider == "" {
		return false
	}
	p := c.Providers.ArgoCD
	return p.ServerEnv != "" || p.TokenEnv != ""
}

func (c *Config) InfraDeploymentEnabled() bool {
	id := c.Evidence.InfraDeployment
	if id.Provider == "" {
		return false
	}
	return len(id.ApplyWorkflows) > 0 || len(id.ApplyJobNames) > 0
}

func (c *Config) ApplicationsForAppDeployment() RepositoryScope {
	if c.Evidence.AppDeployment.Scope != nil {
		return c.Evidence.AppDeployment.Scope.Applications
	}
	return RepositoryScope{}
}

func (c *Config) AppDeploymentConfig() EvidenceAppDeployment {
	return c.Evidence.AppDeployment
}

func (c *Config) ArgoCDServer() (string, error) {
	env := c.Providers.ArgoCD.ServerEnv
	if env == "" {
		env = "ARGOCD_SERVER"
	}
	raw := os.Getenv(env)
	if raw == "" {
		return "", fmt.Errorf("environment variable %s is not set", env)
	}
	return normalizeArgoCDServerAddr(raw)
}

func (c *Config) ArgoCDToken() (string, error) {
	env := c.Providers.ArgoCD.TokenEnv
	if env == "" {
		env = "ARGOCD_TOKEN"
	}
	token := os.Getenv(env)
	if token == "" {
		return "", fmt.Errorf("environment variable %s is not set", env)
	}
	return token, nil
}

func normalizeArgoCDServerAddr(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty Argo CD server address")
	}

	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("parse Argo CD server URL: %w", err)
		}
		host := u.Host
		if host == "" {
			return "", fmt.Errorf("invalid Argo CD server URL: %q", raw)
		}
		if !strings.Contains(host, ":") {
			if u.Scheme == "https" {
				host += ":443"
			} else {
				host += ":80"
			}
		}
		return host, nil
	}

	if !strings.Contains(raw, ":") {
		return raw + ":443", nil
	}
	return raw, nil
}
