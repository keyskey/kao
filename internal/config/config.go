package config

import (
	"fmt"
	"os"

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
	RepoControl EvidenceRepoControl `yaml:"repo_control"`
	CodeChange  EvidenceCodeChange  `yaml:"code_change"`
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

type Controls struct {
	RepoControl RepoControlControls `yaml:"repo_control"`
	CodeChange  CodeChangeControls  `yaml:"code_change"`
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
	}
	return c.Scope.Defaults.Repositories
}
