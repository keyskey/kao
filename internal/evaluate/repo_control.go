package evaluate

import (
	"fmt"
	"time"

	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
)

func evaluateRepoControl(cfg *config.Config, rc evidence.RepoControl, evaluatedAt time.Time) []evidence.Evaluation {
	date := rc.CollectedAt.UTC().Format("2006-01-02")
	resourceID := fmt.Sprintf("%s@%s", rc.Repository, date)
	refs := []evidence.EvidenceRef{{
		Type:        "repo_control",
		ResourceID:  rc.Repository,
		CollectedAt: rc.CollectedAt,
	}}

	bp := cfg.Controls.RepoControl.BranchProtection
	co := cfg.Controls.RepoControl.Codeowners

	var results []evidence.Evaluation

	// CM-001
	results = append(results, boolEval("CM-001", resourceID, evidence.SeverityHigh,
		fmt.Sprintf("branch_protection.enabled == %v", *bp.Enabled),
		fmt.Sprintf("branch_protection.enabled == %v", rc.BranchProtection.Enabled),
		rc.BranchProtection.Enabled == *bp.Enabled,
		refs, evaluatedAt,
	))

	// CM-002 (repo_control part)
	results = append(results, intGteEval("CM-002", resourceID, evidence.SeverityHigh,
		fmt.Sprintf("required_reviews >= %d", *bp.RequiredReviews),
		fmt.Sprintf("required_reviews == %d", rc.BranchProtection.RequiredReviews),
		rc.BranchProtection.RequiredReviews >= *bp.RequiredReviews,
		refs, evaluatedAt,
	))

	// CM-003
	results = append(results, boolEval("CM-003", resourceID, evidence.SeverityHigh,
		fmt.Sprintf("require_codeowner_review == %v", *bp.RequireCodeownerReview),
		fmt.Sprintf("require_codeowner_review == %v", rc.BranchProtection.RequireCodeownerReview),
		rc.BranchProtection.RequireCodeownerReview == *bp.RequireCodeownerReview,
		refs, evaluatedAt,
	))

	// CM-004
	results = append(results, boolEval("CM-004", resourceID, evidence.SeverityHigh,
		fmt.Sprintf("allow_force_push == %v", *bp.AllowForcePush),
		fmt.Sprintf("allow_force_push == %v", rc.BranchProtection.AllowForcePush),
		rc.BranchProtection.AllowForcePush == *bp.AllowForcePush,
		refs, evaluatedAt,
	))

	// CM-009
	results = append(results, boolEval("CM-009", resourceID, evidence.SeverityHigh,
		fmt.Sprintf("allow_branch_deletion == %v", *bp.AllowBranchDeletion),
		fmt.Sprintf("allow_branch_deletion == %v", rc.BranchProtection.AllowBranchDeletion),
		rc.BranchProtection.AllowBranchDeletion == *bp.AllowBranchDeletion,
		refs, evaluatedAt,
	))

	// CM-010
	results = append(results, boolEval("CM-010", resourceID, evidence.SeverityHigh,
		fmt.Sprintf("codeowners.exists == %v", *co.Required),
		fmt.Sprintf("codeowners.exists == %v", rc.Codeowners.Exists),
		rc.Codeowners.Exists == *co.Required,
		refs, evaluatedAt,
	))

	return results
}

func boolEval(controlID, resourceID, severity, expected, actual string, pass bool, refs []evidence.EvidenceRef, at time.Time) evidence.Evaluation {
	ev := evidence.NewEvaluation(controlID, "repo_control", resourceID, severity)
	ev.Expected = expected
	ev.Actual = actual
	ev.EvidenceRefs = refs
	ev.EvaluatedAt = at
	if pass {
		ev.Status = evidence.StatusPassed
	} else {
		ev.Status = evidence.StatusFailed
		ev.Reason = fmt.Sprintf("%s: expected %s, got %s", controlID, expected, actual)
	}
	return ev
}

func intGteEval(controlID, resourceID, severity, expected, actual string, pass bool, refs []evidence.EvidenceRef, at time.Time) evidence.Evaluation {
	return boolEval(controlID, resourceID, severity, expected, actual, pass, refs, at)
}
