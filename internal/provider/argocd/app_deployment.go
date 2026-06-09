package argocd

import (
	"context"
	"fmt"
	"os"
	"regexp"

	applicationpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
	"github.com/keyskey/kao/internal/giturl"
	"github.com/keyskey/kao/internal/scope"
)

var fullSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

func (c *Client) CollectAppDeployment(ctx context.Context, date string, cfg *config.Config) ([]evidence.AppDeployment, error) {
	apps := applicationList(cfg)
	if len(apps) == 0 {
		return nil, fmt.Errorf("evidence.app_deployment.scope.applications.include must list at least one application")
	}

	adCfg := cfg.AppDeploymentConfig()
	var result []evidence.AppDeployment
	var failures []error

	for _, name := range apps {
		appName := name
		app, err := c.app.Get(ctx, &applicationpkg.ApplicationQuery{Name: &appName})
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
			continue
		}
		result = append(result, appDeploymentsFromApplication(app, date, adCfg)...)
	}

	if len(failures) > 0 {
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "collect app-deployment: %v\n", f)
		}
		return result, fmt.Errorf("%d application(s) failed", len(failures))
	}
	return result, nil
}

func applicationList(cfg *config.Config) []string {
	s := cfg.ApplicationsForAppDeployment()
	return scope.Resolve(s.Include, s.Exclude, nil)
}

func appDeploymentsFromApplication(app *v1alpha1.Application, date string, adCfg config.EvidenceAppDeployment) []evidence.AppDeployment {
	if app == nil {
		return nil
	}

	var result []evidence.AppDeployment
	for _, hist := range app.Status.History {
		if hist.DeployedAt.IsZero() {
			continue
		}
		deployedAt := hist.DeployedAt.Time.UTC()
		if deployedAt.Format("2006-01-02") != date {
			continue
		}

		if !fullSHA.MatchString(hist.Revision) {
			fmt.Fprintf(os.Stderr, "app-deployment %s history id %d: skipping non-SHA revision %q\n",
				app.Name, hist.ID, hist.Revision)
			continue
		}

		repoURL := repoURLFromHistory(hist, app.Spec)
		repo, ok := giturl.RepositoryFromURL(repoURL)
		if !ok || repo == "" {
			fmt.Fprintf(os.Stderr, "app-deployment %s history id %d: cannot parse repository from %q\n",
				app.Name, hist.ID, repoURL)
			continue
		}

		ad := evidence.NewAppDeployment()
		ad.Provider = adCfg.Provider
		if ad.Provider == "" {
			ad.Provider = "argocd"
		}
		ad.Application = app.Name
		ad.Repository = repo
		ad.Environment = adCfg.Environment
		ad.Revision = hist.Revision
		ad.SyncInitiatedBy = hist.InitiatedBy.Username
		ad.Status = "succeeded"
		ad.DeployedAt = deployedAt
		result = append(result, ad)
	}
	return result
}

func repoURLFromHistory(hist v1alpha1.RevisionHistory, spec v1alpha1.ApplicationSpec) string {
	if hist.Source.RepoURL != "" {
		return hist.Source.RepoURL
	}
	if len(hist.Sources) > 0 && hist.Sources[0].RepoURL != "" {
		return hist.Sources[0].RepoURL
	}
	if spec.Source != nil && spec.Source.RepoURL != "" {
		return spec.Source.RepoURL
	}
	if len(spec.Sources) > 0 && spec.Sources[0].RepoURL != "" {
		return spec.Sources[0].RepoURL
	}
	return ""
}

func FilterByRepository(records []evidence.AppDeployment, repos []string) []evidence.AppDeployment {
	if len(repos) == 0 {
		return records
	}
	set := make(map[string]bool, len(repos))
	for _, r := range repos {
		set[r] = true
	}
	var out []evidence.AppDeployment
	for _, ad := range records {
		if set[ad.Repository] {
			out = append(out, ad)
		}
	}
	return out
}
