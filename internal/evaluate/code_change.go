package evaluate

import (
	"fmt"
	"strings"
	"time"

	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
	gh "github.com/keyskey/kao/internal/provider/github"
)

func evaluateCodeChange(cfg *config.Config, cc evidence.CodeChange, rc *evidence.RepoControl, evaluatedAt time.Time) []evidence.Evaluation {
	resourceID := fmt.Sprintf("%s#%d", cc.Repository, cc.PRNumber)
	refs := []evidence.EvidenceRef{{
		Type:       "code_change",
		ResourceID: resourceID,
	}}
	if rc != nil {
		refs = append(refs, evidence.EvidenceRef{
			Type:        "repo_control",
			ResourceID:  rc.Repository,
			CollectedAt: rc.CollectedAt,
		})
	}

	bp := cfg.Controls.RepoControl.BranchProtection
	ccCfg := cfg.Controls.CodeChange

	var results []evidence.Evaluation

	// CM-002 (code_change part: approval count)
	results = append(results, boolEvalCodeChange("CM-002", resourceID, evidence.SeverityHigh,
		fmt.Sprintf("approval_count >= %d", *bp.RequiredReviews),
		fmt.Sprintf("approval_count == %d", cc.ApprovalCount),
		cc.ApprovalCount >= *bp.RequiredReviews,
		refs, evaluatedAt,
	))

	// CM-005
	results = append(results, boolEvalCodeChange("CM-005", resourceID, evidence.SeverityHigh,
		fmt.Sprintf("ci.passed == %v", *ccCfg.CI.RequirePass),
		fmt.Sprintf("ci.passed == %v", cc.CI.Passed),
		cc.CI.Passed == *ccCfg.CI.RequirePass,
		refs, evaluatedAt,
	))

	// CM-008
	if *ccCfg.Approver.MustBeAuthorized {
		ev := evaluateCM008(cc, rc, resourceID, refs, evaluatedAt)
		results = append(results, ev)
	}

	return results
}

func evaluateCM008(cc evidence.CodeChange, rc *evidence.RepoControl, resourceID string, refs []evidence.EvidenceRef, at time.Time) evidence.Evaluation {
	ev := evidence.NewEvaluation("CM-008", "code_change", resourceID, evidence.SeverityCritical)
	ev.EvidenceRefs = refs
	ev.EvaluatedAt = at

	if rc == nil {
		ev.Status = evidence.StatusFailed
		ev.Expected = "approver ∈ repo_control resolved_users"
		ev.Actual = "no repo_control snapshot available"
		ev.Reason = "Approver authorization requires repo_control snapshot from same daily job"
		return ev
	}

	authorized := gh.ResolvedUsersUnion(rc.Codeowners)
	var unauthorized []string
	for _, a := range cc.Approvals {
		if !authorized[a.Reviewer] {
			unauthorized = append(unauthorized, a.Reviewer)
		}
	}

	if len(unauthorized) == 0 {
		ev.Status = evidence.StatusPassed
		ev.Expected = "all approvers ∈ resolved_users"
		ev.Actual = "all approvers authorized"
	} else {
		ev.Status = evidence.StatusFailed
		ev.Expected = "approver ∈ repo_control resolved_users"
		ev.Actual = fmt.Sprintf("%s ∉ resolved_users", strings.Join(unauthorized, ", "))
		ev.Reason = "Approver is not included in CODEOWNERS resolved users"
	}
	return ev
}

func boolEvalCodeChange(controlID, resourceID, severity, expected, actual string, pass bool, refs []evidence.EvidenceRef, at time.Time) evidence.Evaluation {
	ev := evidence.NewEvaluation(controlID, "code_change", resourceID, severity)
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

func findRepoControlForCodeChange(repoControls []evidence.RepoControl, cc evidence.CodeChange) *evidence.RepoControl {
	var best *evidence.RepoControl
	for i := range repoControls {
		rc := &repoControls[i]
		if rc.Repository != cc.Repository {
			continue
		}
		if rc.CollectedAt.After(cc.MergedAt) {
			if best == nil || rc.CollectedAt.Before(best.CollectedAt) {
				best = rc
			}
		}
	}
	if best == nil {
		for i := range repoControls {
			rc := &repoControls[i]
			if rc.Repository == cc.Repository {
				return rc
			}
		}
	}
	return best
}
