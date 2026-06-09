package evaluate

import (
	"fmt"
	"time"

	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
)

func evaluateAppDeployment(cfg *config.Config, ad evidence.AppDeployment, cc *evidence.CodeChange, evaluatedAt time.Time) []evidence.Evaluation {
	if !*cfg.Controls.AppDeployment.Traceability.Required {
		return nil
	}

	resourceID := appDeploymentResourceID(ad)
	requiredReviews := *cfg.Controls.RepoControl.BranchProtection.RequiredReviews

	core := evaluateTraceabilityCore(
		"CM-006",
		"app_deployment",
		resourceID,
		evidence.EvidenceRef{Type: "app_deployment", ResourceID: resourceID},
		cc,
		requiredReviews,
		evaluatedAt,
	)
	if !core.ok {
		return []evidence.Evaluation{core.ev}
	}

	ev := core.ev
	if !hasCIRunURL(core.cc) {
		ev.Status = evidence.StatusFailed
		ev.Expected = "code_change.ci.runs[].url present"
		ev.Actual = "no CI run URL"
		ev.Reason = "app_deployment → code_change → ci.runs[].url missing"
		return []evidence.Evaluation{ev}
	}

	ev.Status = evidence.StatusPassed
	ev.Expected = "traceability chain complete"
	ev.Actual = "repository+sha join, author, approvals, ci.runs[].url present"
	return []evidence.Evaluation{ev}
}

func appDeploymentResourceID(ad evidence.AppDeployment) string {
	return fmt.Sprintf("%s@%s", ad.Application, ad.DeployedAt.UTC().Format(time.RFC3339))
}
