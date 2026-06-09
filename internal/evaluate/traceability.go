package evaluate

import (
	"fmt"
	"time"

	"github.com/keyskey/kao/internal/evidence"
)

type traceabilityCoreResult struct {
	ev   evidence.Evaluation
	refs []evidence.EvidenceRef
	cc   evidence.CodeChange
	ok   bool
}

func evaluateTraceabilityCore(
	controlID, resourceType, resourceID string,
	deploymentRef evidence.EvidenceRef,
	cc *evidence.CodeChange,
	requiredReviews int,
	evaluatedAt time.Time,
) traceabilityCoreResult {
	ev := evidence.NewEvaluation(controlID, resourceType, resourceID, evidence.SeverityCritical)
	ev.EvidenceRefs = []evidence.EvidenceRef{deploymentRef}
	ev.EvaluatedAt = evaluatedAt

	if cc == nil {
		ev.Status = evidence.StatusFailed
		ev.Expected = "code_change joined by repository + commit_sha"
		ev.Actual = "no matching code_change"
		ev.Reason = resourceType + " → code_change join failed"
		return traceabilityCoreResult{ev: ev, ok: false}
	}

	refs := []evidence.EvidenceRef{
		deploymentRef,
		{
			Type:       "code_change",
			ResourceID: fmt.Sprintf("%s#%d", cc.Repository, cc.PRNumber),
		},
	}
	ev.EvidenceRefs = refs

	switch {
	case cc.Author == "":
		ev.Status = evidence.StatusFailed
		ev.Expected = "code_change.author present"
		ev.Actual = "author missing"
		ev.Reason = resourceType + " → code_change → author missing"
		return traceabilityCoreResult{ev: ev, ok: false}
	case cc.ApprovalCount < requiredReviews:
		ev.Status = evidence.StatusFailed
		ev.Expected = fmt.Sprintf("approval_count >= %d", requiredReviews)
		ev.Actual = fmt.Sprintf("approval_count == %d", cc.ApprovalCount)
		ev.Reason = resourceType + " → code_change → approvals insufficient"
		return traceabilityCoreResult{ev: ev, ok: false}
	default:
		return traceabilityCoreResult{ev: ev, refs: refs, cc: *cc, ok: true}
	}
}

func hasCIRunURL(cc evidence.CodeChange) bool {
	for _, run := range cc.CI.Runs {
		if run.URL != "" {
			return true
		}
	}
	return false
}

func hasExecutionURL(url string) bool {
	return url != ""
}
