package evaluate

import (
	"fmt"
	"time"

	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
)

func evaluateInfraDeployment(cfg *config.Config, id evidence.InfraDeployment, cc *evidence.CodeChange, evaluatedAt time.Time) []evidence.Evaluation {
	if !*cfg.Controls.InfraDeployment.Traceability.Required {
		return nil
	}

	resourceID := infraDeploymentResourceID(id)
	refs := []evidence.EvidenceRef{{
		Type:       "infra_deployment",
		ResourceID: resourceID,
	}}

	ev := evidence.NewEvaluation("CM-007", "infra_deployment", resourceID, evidence.SeverityCritical)
	ev.EvidenceRefs = refs
	ev.EvaluatedAt = evaluatedAt

	if cc == nil {
		ev.Status = evidence.StatusFailed
		ev.Expected = "code_change joined by repository + commit_sha"
		ev.Actual = "no matching code_change"
		ev.Reason = "infra_deployment → code_change join failed"
		return []evidence.Evaluation{ev}
	}

	refs = append(refs, evidence.EvidenceRef{
		Type:       "code_change",
		ResourceID: fmt.Sprintf("%s#%d", cc.Repository, cc.PRNumber),
	})
	ev.EvidenceRefs = refs

	requiredReviews := *cfg.Controls.RepoControl.BranchProtection.RequiredReviews

	switch {
	case cc.Author == "":
		ev.Status = evidence.StatusFailed
		ev.Expected = "code_change.author present"
		ev.Actual = "author missing"
		ev.Reason = "infra_deployment → code_change → author missing"
	case cc.ApprovalCount < requiredReviews:
		ev.Status = evidence.StatusFailed
		ev.Expected = fmt.Sprintf("approval_count >= %d", requiredReviews)
		ev.Actual = fmt.Sprintf("approval_count == %d", cc.ApprovalCount)
		ev.Reason = "infra_deployment → code_change → approvals insufficient"
	case id.Execution.URL == "":
		ev.Status = evidence.StatusFailed
		ev.Expected = "execution.url present"
		ev.Actual = "execution.url missing"
		ev.Reason = "infra_deployment → execution.url missing"
	default:
		ev.Status = evidence.StatusPassed
		ev.Expected = "traceability chain complete"
		ev.Actual = "repository+sha join, author, approvals, execution.url present"
	}

	return []evidence.Evaluation{ev}
}

func infraDeploymentResourceID(id evidence.InfraDeployment) string {
	return fmt.Sprintf("%s#%s@%s", id.Repository, id.Workspace, id.AppliedAt.UTC().Format(time.RFC3339))
}
