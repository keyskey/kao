package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	gogh "github.com/google/go-github/v62/github"
	"github.com/keyskey/kao/internal/evidence"
)

func (c *Client) CollectCodeChange(ctx context.Context, repo, defaultBranch, date string) ([]evidence.CodeChange, error) {
	owner := c.org
	query := fmt.Sprintf(
		"repo:%s/%s is:pr is:merged base:%s merged:%s..%s",
		owner, repo, defaultBranch, date, date,
	)

	var result []evidence.CodeChange
	opts := &gogh.SearchOptions{ListOptions: gogh.ListOptions{PerPage: 100}}
	for {
		searchResult, resp, err := c.api.Search.Issues(ctx, query, opts)
		if err != nil {
			return nil, fmt.Errorf("search merged PRs %s: %w", repo, err)
		}
		for _, issue := range searchResult.Issues {
			if issue.PullRequestLinks == nil || issue.Number == nil {
				continue
			}
			cc, err := c.buildCodeChange(ctx, owner, repo, defaultBranch, *issue.Number)
			if err != nil {
				return nil, err
			}
			if cc.MergedAt.UTC().Format("2006-01-02") == date {
				result = append(result, cc)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return result, nil
}

func (c *Client) buildCodeChange(ctx context.Context, owner, repo, defaultBranch string, prNumber int) (evidence.CodeChange, error) {
	pr, _, err := c.api.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return evidence.CodeChange{}, fmt.Errorf("get PR %s#%d: %w", repo, prNumber, err)
	}

	cc := evidence.NewCodeChange()
	cc.Provider = "github"
	cc.Repository = repo
	cc.DefaultBranch = defaultBranch
	cc.PRNumber = prNumber
	cc.URL = pr.GetHTMLURL()

	if pr.User != nil && pr.User.Login != nil {
		cc.Author = *pr.User.Login
	}
	if pr.MergedBy != nil && pr.MergedBy.Login != nil {
		cc.MergedBy = *pr.MergedBy.Login
	}
	if pr.MergeCommitSHA != nil {
		cc.CommitSHA = *pr.MergeCommitSHA
	}
	if pr.MergedAt != nil {
		cc.MergedAt = pr.MergedAt.Time
	}

	approvals, count, err := c.fetchApprovals(ctx, owner, repo, prNumber)
	if err != nil {
		return evidence.CodeChange{}, err
	}
	cc.Approvals = approvals
	cc.ApprovalCount = count

	ci, err := c.fetchCI(ctx, owner, repo, cc.CommitSHA)
	if err != nil {
		return evidence.CodeChange{}, err
	}
	cc.CI = ci

	return cc, nil
}

func (c *Client) fetchApprovals(ctx context.Context, owner, repo string, prNumber int) ([]evidence.Approval, int, error) {
	reviews, _, err := c.api.PullRequests.ListReviews(ctx, owner, repo, prNumber, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("list reviews %s#%d: %w", repo, prNumber, err)
	}

	seen := make(map[string]bool)
	var approvals []evidence.Approval
	for _, review := range reviews {
		if review.State == nil || *review.State != "APPROVED" {
			continue
		}
		if review.User == nil || review.User.Login == nil {
			continue
		}
		reviewer := *review.User.Login
		if seen[reviewer] {
			continue
		}
		seen[reviewer] = true
		var submittedAt time.Time
		if review.SubmittedAt != nil {
			submittedAt = review.SubmittedAt.Time
		}
		approvals = append(approvals, evidence.Approval{
			Reviewer:    reviewer,
			State:       "APPROVED",
			SubmittedAt: submittedAt,
		})
	}
	return approvals, len(approvals), nil
}

func (c *Client) fetchCI(ctx context.Context, owner, repo, commitSHA string) (evidence.CIResult, error) {
	if commitSHA == "" {
		return evidence.CIResult{Passed: false, Runs: []evidence.CIRun{}}, nil
	}

	opts := &gogh.ListWorkflowRunsOptions{
		HeadSHA: commitSHA,
		ListOptions: gogh.ListOptions{PerPage: 100},
	}
	runs, _, err := c.api.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
	if err != nil {
		return evidence.CIResult{}, fmt.Errorf("list workflow runs %s@%s: %w", repo, commitSHA, err)
	}

	var ciRuns []evidence.CIRun
	passed := len(runs.WorkflowRuns) > 0
	for _, run := range runs.WorkflowRuns {
		status := strings.ToLower(run.GetConclusion())
		if status == "" {
			status = strings.ToLower(run.GetStatus())
		}
		if status != "success" {
			passed = false
		}
		ciRuns = append(ciRuns, evidence.CIRun{
			Provider:  "github_actions",
			Workflow:  run.GetName(),
			RunID:     strconv.FormatInt(run.GetID(), 10),
			Status:    status,
			CommitSHA: commitSHA,
			URL:       run.GetHTMLURL(),
		})
	}
	if len(runs.WorkflowRuns) == 0 {
		passed = false
	}

	return evidence.CIResult{Passed: passed, Runs: ciRuns}, nil
}
