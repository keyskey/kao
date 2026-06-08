package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/keyskey/kao/internal/evidence"
)

func TestFilesystemPutAndQuery(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFilesystem(filepath.Join(dir, "evidence"))
	if err != nil {
		t.Fatal(err)
	}

	collectedAt := time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC)
	rc := evidence.RepoControl{
		SchemaVersion: "v1",
		Type:          "repo_control",
		Provider:      "github",
		Repository:    "trading-api",
		CollectedAt:   collectedAt,
		BranchProtection: evidence.BranchProtection{Enabled: true},
	}

	ctx := context.Background()
	if err := fs.PutRepoControl(ctx, []evidence.RepoControl{rc}); err != nil {
		t.Fatal(err)
	}

	got, err := fs.QueryRepoControl(ctx, Filter{Date: "2026-06-06"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Repository != "trading-api" {
		t.Fatalf("got %+v", got)
	}
}
