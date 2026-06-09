package evidence

import "time"

const SchemaVersion = "v1"

type RepoControl struct {
	SchemaVersion    string           `json:"schema_version"`
	Type             string           `json:"type"`
	Provider         string           `json:"provider"`
	Repository       string           `json:"repository"`
	DefaultBranch    string           `json:"default_branch"`
	BranchProtection BranchProtection `json:"branch_protection"`
	Codeowners       Codeowners       `json:"codeowners"`
	CollectedAt      time.Time        `json:"collected_at"`
}

type BranchProtection struct {
	Enabled                bool `json:"enabled"`
	RequiredReviews        int  `json:"required_reviews"`
	RequireCodeownerReview bool `json:"require_codeowner_review"`
	AllowForcePush         bool `json:"allow_force_push"`
	AllowBranchDeletion    bool `json:"allow_branch_deletion"`
}

type Codeowners struct {
	Exists bool            `json:"exists"`
	Path   string          `json:"path,omitempty"`
	Rules  []CodeownerRule `json:"rules,omitempty"`
}

type CodeownerRule struct {
	Pattern string        `json:"pattern"`
	Owners  []Owner       `json:"owners"`
}

type Owner struct {
	Type          string   `json:"type"`
	Name          string   `json:"name"`
	ResolvedUsers []string `json:"resolved_users,omitempty"`
}

type CodeChange struct {
	SchemaVersion string     `json:"schema_version"`
	Type          string     `json:"type"`
	Provider      string     `json:"provider"`
	Repository    string     `json:"repository"`
	DefaultBranch string     `json:"default_branch"`
	PRNumber      int        `json:"pr_number"`
	URL           string     `json:"url"`
	Author        string     `json:"author"`
	MergedBy      string     `json:"merged_by"`
	Approvals     []Approval `json:"approvals"`
	ApprovalCount int        `json:"approval_count"`
	CommitSHA     string     `json:"commit_sha"`
	CI            CIResult   `json:"ci"`
	MergedAt      time.Time  `json:"merged_at"`
}

type Approval struct {
	Reviewer    string    `json:"reviewer"`
	State       string    `json:"state"`
	SubmittedAt time.Time `json:"submitted_at"`
}

type CIResult struct {
	Passed bool    `json:"passed"`
	Runs   []CIRun `json:"runs"`
}

type CIRun struct {
	Provider  string `json:"provider"`
	Workflow  string `json:"workflow"`
	RunID     string `json:"run_id"`
	Status    string `json:"status"`
	CommitSHA string `json:"commit_sha"`
	URL       string `json:"url"`
}

type InfraDeployment struct {
	SchemaVersion string    `json:"schema_version"`
	Type          string    `json:"type"`
	Provider      string    `json:"provider"`
	Repository    string    `json:"repository"`
	Workspace     string    `json:"workspace"`
	Environment   string    `json:"environment"`
	CommitSHA     string    `json:"commit_sha"`
	Execution     Execution `json:"execution"`
	AppliedAt     time.Time `json:"applied_at"`
}

type Execution struct {
	CIProvider  string `json:"ci_provider"`
	Workflow    string `json:"workflow"`
	RunID       string `json:"run_id"`
	Status      string `json:"status"`
	TriggeredBy string `json:"triggered_by"`
	URL         string `json:"url"`
}

type Evaluation struct {
	SchemaVersion string         `json:"schema_version"`
	ControlID     string         `json:"control_id"`
	ResourceType  string         `json:"resource_type"`
	ResourceID    string         `json:"resource_id"`
	Status        string         `json:"status"`
	Severity      string         `json:"severity"`
	Expected      string         `json:"expected"`
	Actual        string         `json:"actual"`
	Reason        string         `json:"reason,omitempty"`
	EvidenceRefs  []EvidenceRef  `json:"evidence_refs"`
	EvaluatedAt   time.Time      `json:"evaluated_at"`
}

type EvidenceRef struct {
	Type         string    `json:"type"`
	ResourceID   string    `json:"resource_id"`
	CollectedAt  time.Time `json:"collected_at,omitempty"`
}

const (
	StatusPassed = "PASSED"
	StatusFailed = "FAILED"

	SeverityHigh     = "HIGH"
	SeverityCritical = "CRITICAL"
)

func NewRepoControl() RepoControl {
	return RepoControl{
		SchemaVersion: SchemaVersion,
		Type:          "repo_control",
	}
}

func NewCodeChange() CodeChange {
	return CodeChange{
		SchemaVersion: SchemaVersion,
		Type:          "code_change",
	}
}

func NewInfraDeployment() InfraDeployment {
	return InfraDeployment{
		SchemaVersion: SchemaVersion,
		Type:          "infra_deployment",
	}
}

func NewEvaluation(controlID, resourceType, resourceID, severity string) Evaluation {
	return Evaluation{
		SchemaVersion: SchemaVersion,
		ControlID:     controlID,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Severity:      severity,
		EvidenceRefs:  []EvidenceRef{},
	}
}
