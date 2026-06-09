package evaluate

import (
	"github.com/keyskey/kao/internal/evidence"
)

type JoinKey struct {
	Repository string
	CommitSHA  string
}

// JoinCodeChange maps repository+commit_sha to the code_change with the newest merged_at.
func JoinCodeChange(changes []evidence.CodeChange) map[JoinKey]evidence.CodeChange {
	best := make(map[JoinKey]evidence.CodeChange)
	for _, cc := range changes {
		if cc.Repository == "" || cc.CommitSHA == "" {
			continue
		}
		key := JoinKey{Repository: cc.Repository, CommitSHA: cc.CommitSHA}
		existing, ok := best[key]
		if !ok || cc.MergedAt.After(existing.MergedAt) {
			best[key] = cc
		}
	}
	return best
}
