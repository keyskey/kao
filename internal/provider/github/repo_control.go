package github

import (
	"context"
	"fmt"
	"time"

	"github.com/keyskey/kao/internal/evidence"
)

func (c *Client) CollectRepoControl(ctx context.Context, repo string) (evidence.RepoControl, error) {
	owner := c.org
	ghRepo, _, err := c.api.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return evidence.RepoControl{}, fmt.Errorf("get repository %s: %w", repo, err)
	}

	defaultBranch := "main"
	if ghRepo.DefaultBranch != nil {
		defaultBranch = *ghRepo.DefaultBranch
	}

	bp := evidence.BranchProtection{Enabled: false}
	protection, resp, err := c.api.Repositories.GetBranchProtection(ctx, owner, repo, defaultBranch)
	if err == nil && protection != nil {
		bp.Enabled = true
		if protection.RequiredPullRequestReviews != nil {
			bp.RequiredReviews = protection.RequiredPullRequestReviews.RequiredApprovingReviewCount
			bp.RequireCodeownerReview = protection.RequiredPullRequestReviews.RequireCodeOwnerReviews
		}
		if protection.AllowForcePushes != nil {
			bp.AllowForcePush = protection.AllowForcePushes.Enabled
		}
		if protection.AllowDeletions != nil {
			bp.AllowBranchDeletion = protection.AllowDeletions.Enabled
		}
	} else if resp != nil && resp.StatusCode != 404 {
		return evidence.RepoControl{}, fmt.Errorf("get branch protection %s: %w", repo, err)
	}

	codeowners, err := c.fetchCodeowners(ctx, owner, repo)
	if err != nil {
		return evidence.RepoControl{}, fmt.Errorf("fetch codeowners %s: %w", repo, err)
	}

	rc := evidence.NewRepoControl()
	rc.Provider = "github"
	rc.Repository = repo
	rc.DefaultBranch = defaultBranch
	rc.BranchProtection = bp
	rc.Codeowners = codeowners
	rc.CollectedAt = time.Now().UTC()

	return rc, nil
}
