package argocd

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
)

func TestAppDeploymentsFromApplication(t *testing.T) {
	deployedAt := metav1.Time{Time: time.Date(2026, 6, 5, 16, 5, 0, 0, time.UTC)}
	app := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "kao-prod"},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{
				RepoURL: "https://github.com/keyskey/kao.git",
			},
		},
		Status: v1alpha1.ApplicationStatus{
			History: v1alpha1.RevisionHistories{
				{
					Revision:   "abc123def4567890abc123def4567890abc12345",
					DeployedAt: deployedAt,
					InitiatedBy: v1alpha1.OperationInitiator{Username: "alice"},
				},
				{
					Revision:   "def4567890abcdef4567890abcdef4567890ab",
					DeployedAt: metav1.Time{Time: time.Date(2026, 6, 4, 16, 5, 0, 0, time.UTC)},
				},
			},
		},
	}

	adCfg := config.EvidenceAppDeployment{
		Provider:    "argocd",
		Environment: "production",
	}

	got := appDeploymentsFromApplication(app, "2026-06-05", adCfg)
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	if got[0].Repository != "kao" || got[0].Revision != "abc123def4567890abc123def4567890abc12345" {
		t.Fatalf("got %+v", got[0])
	}
	if got[0].SyncInitiatedBy != "alice" {
		t.Fatalf("sync_initiated_by = %q", got[0].SyncInitiatedBy)
	}
}

func TestAppDeploymentsFromApplicationSkipsNonSHA(t *testing.T) {
	app := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{Name: "kao-prod"},
		Spec: v1alpha1.ApplicationSpec{
			Source: &v1alpha1.ApplicationSource{RepoURL: "https://github.com/keyskey/kao.git"},
		},
		Status: v1alpha1.ApplicationStatus{
			History: v1alpha1.RevisionHistories{
				{
					Revision:   "main",
					DeployedAt: metav1.Time{Time: time.Date(2026, 6, 5, 16, 5, 0, 0, time.UTC)},
				},
			},
		},
	}
	got := appDeploymentsFromApplication(app, "2026-06-05", config.EvidenceAppDeployment{Environment: "production"})
	if len(got) != 0 {
		t.Fatalf("expected 0 records, got %d", len(got))
	}
}

func TestFilterByRepository(t *testing.T) {
	records := []evidence.AppDeployment{
		{Repository: "kao"},
		{Repository: "other"},
	}
	got := FilterByRepository(records, []string{"kao"})
	if len(got) != 1 || got[0].Repository != "kao" {
		t.Fatalf("got %+v", got)
	}
}
