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
	requiredReviews := *cfg.Controls.RepoControl.BranchProtection.RequiredReviews

	core := evaluateTraceabilityCore(
		"CM-007",
		"infra_deployment",
		resourceID,
		evidence.EvidenceRef{Type: "infra_deployment", ResourceID: resourceID},
		cc,
		requiredReviews,
		evaluatedAt,
	)
	if !core.ok {
		return []evidence.Evaluation{core.ev}
	}

	ev := core.ev
	if !hasExecutionURL(id.Execution.URL) {
		ev.Status = evidence.StatusFailed
		ev.Expected = "execution.url present"
		ev.Actual = "execution.url missing"
		ev.Reason = "infra_deployment → execution.url missing"
		return []evidence.Evaluation{ev}
	}

	ev.Status = evidence.StatusPassed
	ev.Expected = "traceability chain complete"
	ev.Actual = "repository+sha join, author, approvals, execution.url present"
	return []evidence.Evaluation{ev}
}

func infraDeploymentResourceID(id evidence.InfraDeployment) string {
	return fmt.Sprintf("%s#%s@%s", id.Repository, id.Workspace, id.AppliedAt.UTC().Format(time.RFC3339))
}
