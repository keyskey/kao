package evaluate

import (
	"testing"
	"time"

	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
)

func TestEvaluateRepoControlCM001Pass(t *testing.T) {
	cfg := &config.Config{}
	applyTestDefaults(cfg)

	rc := evidence.RepoControl{
		Repository: "trading-api",
		BranchProtection: evidence.BranchProtection{
			Enabled:                true,
			RequiredReviews:        1,
			RequireCodeownerReview: true,
			AllowForcePush:         false,
			AllowBranchDeletion:    false,
		},
		Codeowners:  evidence.Codeowners{Exists: true},
		CollectedAt: time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC),
	}

	results := evaluateRepoControl(cfg, rc, rc.CollectedAt)
	cm001 := findResult(results, "CM-001")
	if cm001 == nil || cm001.Status != evidence.StatusPassed {
		t.Fatalf("CM-001 should pass, got %+v", cm001)
	}
}

func TestEvaluateCodeChangeCM008Fail(t *testing.T) {
	cfg := &config.Config{}
	applyTestDefaults(cfg)

	cc := evidence.CodeChange{
		Repository:    "trading-api",
		PRNumber:      123,
		ApprovalCount: 1,
		CI:            evidence.CIResult{Passed: true},
		Approvals: []evidence.Approval{{
			Reviewer: "unknown-user",
			State:    "APPROVED",
		}},
		MergedAt: time.Date(2026, 6, 5, 16, 0, 0, 0, time.UTC),
	}

	rc := &evidence.RepoControl{
		Repository:  "trading-api",
		CollectedAt: time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC),
		Codeowners: evidence.Codeowners{
			Exists: true,
			Rules: []evidence.CodeownerRule{{
				Pattern: "*",
				Owners: []evidence.Owner{{
					Type:          "user",
					ResolvedUsers: []string{"alice"},
				}},
			}},
		},
	}

	results := evaluateCodeChange(cfg, cc, rc, rc.CollectedAt)
	cm008 := findResult(results, "CM-008")
	if cm008 == nil || cm008.Status != evidence.StatusFailed {
		t.Fatalf("CM-008 should fail, got %+v", cm008)
	}
}

func applyTestDefaults(cfg *config.Config) {
	trueVal := true
	falseVal := false
	one := 1
	cfg.Controls.RepoControl.BranchProtection.Enabled = &trueVal
	cfg.Controls.RepoControl.BranchProtection.RequiredReviews = &one
	cfg.Controls.RepoControl.BranchProtection.RequireCodeownerReview = &trueVal
	cfg.Controls.RepoControl.BranchProtection.AllowForcePush = &falseVal
	cfg.Controls.RepoControl.BranchProtection.AllowBranchDeletion = &falseVal
	cfg.Controls.RepoControl.Codeowners.Required = &trueVal
	cfg.Controls.CodeChange.CI.RequirePass = &trueVal
	cfg.Controls.CodeChange.Approver.MustBeAuthorized = &trueVal
}

func findResult(results []evidence.Evaluation, controlID string) *evidence.Evaluation {
	for i := range results {
		if results[i].ControlID == controlID {
			return &results[i]
		}
	}
	return nil
}
