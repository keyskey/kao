package evaluate

import (
	"testing"
	"time"

	"github.com/keyskey/kao/internal/evidence"
)

func TestEvaluateTraceabilityCorePass(t *testing.T) {
	cc := &evidence.CodeChange{
		Repository:    "kao",
		PRNumber:      1,
		Author:        "alice",
		ApprovalCount: 1,
	}
	core := evaluateTraceabilityCore(
		"CM-006", "app_deployment", "kao-prod@t",
		evidence.EvidenceRef{Type: "app_deployment", ResourceID: "kao-prod@t"},
		cc, 1, time.Now().UTC(),
	)
	if !core.ok {
		t.Fatalf("expected ok, got %+v", core.ev)
	}
}

func TestHasCIRunURL(t *testing.T) {
	cc := evidence.CodeChange{
		CI: evidence.CIResult{
			Runs: []evidence.CIRun{{URL: "https://example.com/run/1"}},
		},
	}
	if !hasCIRunURL(cc) {
		t.Fatal("expected true")
	}
}

func TestHasExecutionURL(t *testing.T) {
	if !hasExecutionURL("https://example.com") {
		t.Fatal("expected true")
	}
	if hasExecutionURL("") {
		t.Fatal("expected false")
	}
}
