package evaluate

import (
	"testing"
	"time"

	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
)

func TestEvaluateAppDeploymentCM006Pass(t *testing.T) {
	cfg := testConfig()
	deployedAt := time.Date(2026, 6, 5, 16, 5, 0, 0, time.UTC)

	ad := evidence.AppDeployment{
		Application: "kao-prod",
		Repository:  "kao",
		Revision:    "abc123def4567890abc123def4567890abc12345",
		DeployedAt:  deployedAt,
	}
	cc := evidence.CodeChange{
		Repository:    "kao",
		PRNumber:      1,
		CommitSHA:     "abc123def4567890abc123def4567890abc12345",
		Author:        "alice",
		ApprovalCount: 1,
		CI: evidence.CIResult{
			Runs: []evidence.CIRun{{URL: "https://github.com/keyskey/kao/actions/runs/123"}},
		},
	}
	join := map[JoinKey]evidence.CodeChange{
		{Repository: "kao", CommitSHA: ad.Revision}: cc,
	}

	results := EvaluateAll(cfg, nil, nil, []evidence.AppDeployment{ad}, nil, join, deployedAt)
	cm006 := findResult(results, "CM-006")
	if cm006 == nil || cm006.Status != evidence.StatusPassed {
		t.Fatalf("CM-006 should pass, got %+v", cm006)
	}
}

func TestEvaluateAppDeploymentCM006JoinFail(t *testing.T) {
	cfg := testConfig()
	deployedAt := time.Date(2026, 6, 5, 16, 5, 0, 0, time.UTC)

	ad := evidence.AppDeployment{
		Application: "kao-prod",
		Repository:  "kao",
		Revision:    "missing",
		DeployedAt:  deployedAt,
	}

	results := EvaluateAll(cfg, nil, nil, []evidence.AppDeployment{ad}, nil, map[JoinKey]evidence.CodeChange{}, deployedAt)
	cm006 := findResult(results, "CM-006")
	if cm006 == nil || cm006.Status != evidence.StatusFailed {
		t.Fatalf("CM-006 should fail, got %+v", cm006)
	}
}

func TestEvaluateAppDeploymentCM006NoCIURL(t *testing.T) {
	cfg := testConfig()
	deployedAt := time.Date(2026, 6, 5, 16, 5, 0, 0, time.UTC)

	ad := evidence.AppDeployment{
		Application: "kao-prod",
		Repository:  "kao",
		Revision:    "abc123def4567890abc123def4567890abc12345",
		DeployedAt:  deployedAt,
	}
	cc := evidence.CodeChange{
		Repository:    "kao",
		PRNumber:      1,
		CommitSHA:     ad.Revision,
		Author:        "alice",
		ApprovalCount: 1,
		CI:            evidence.CIResult{Runs: []evidence.CIRun{{URL: ""}}},
	}
	join := map[JoinKey]evidence.CodeChange{
		{Repository: "kao", CommitSHA: ad.Revision}: cc,
	}

	results := EvaluateAll(cfg, nil, nil, []evidence.AppDeployment{ad}, nil, join, deployedAt)
	cm006 := findResult(results, "CM-006")
	if cm006 == nil || cm006.Status != evidence.StatusFailed {
		t.Fatalf("CM-006 should fail without CI URL, got %+v", cm006)
	}
}

func testConfig() *config.Config {
	cfg := &config.Config{}
	applyTestDefaults(cfg)
	return cfg
}
