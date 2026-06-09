package evaluate

import (
	"testing"
	"time"

	"github.com/keyskey/kao/internal/evidence"
)

func TestJoinCodeChangePicksNewestMergedAt(t *testing.T) {
	older := evidence.CodeChange{
		Repository: "platform-infra",
		CommitSHA:  "abc",
		PRNumber:   1,
		MergedAt:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
	newer := evidence.CodeChange{
		Repository: "platform-infra",
		CommitSHA:  "abc",
		PRNumber:   2,
		MergedAt:   time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
	}

	joined := JoinCodeChange([]evidence.CodeChange{older, newer})
	got := joined[JoinKey{Repository: "platform-infra", CommitSHA: "abc"}]
	if got.PRNumber != 2 {
		t.Fatalf("got PR %d, want 2", got.PRNumber)
	}
}
