package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	gogh "github.com/google/go-github/v62/github"
	"github.com/keyskey/kao/internal/config"
	"github.com/keyskey/kao/internal/evidence"
	"github.com/keyskey/kao/internal/scope"
)

func (c *Client) CollectInfraDeployment(ctx context.Context, repo, date string, cfg *config.Config) ([]evidence.InfraDeployment, error) {
	idCfg := cfg.InfraDeploymentConfig()
	wsScope := cfg.WorkspacesForInfraDeployment()

	pattern, err := regexp.Compile(idCfg.WorkspaceJobNamePattern)
	if err != nil {
		return nil, fmt.Errorf("compile workspace_job_name_pattern: %w", err)
	}

	targetDay, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("parse date %q: %w", date, err)
	}
	prevDay := targetDay.AddDate(0, 0, -1)

	owner := c.org
	workflowPaths := make(map[int64]string)

	var result []evidence.InfraDeployment
	opts := &gogh.ListWorkflowRunsOptions{
		Status: "completed",
		ListOptions: gogh.ListOptions{PerPage: 100},
	}
	created := fmt.Sprintf("%s..%s", prevDay.Format("2006-01-02"), date)
	opts.Created = created

	for {
		runs, resp, err := c.api.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("list workflow runs %s: %w", repo, err)
		}

		for _, run := range runs.WorkflowRuns {
			if !runSucceeded(run) {
				continue
			}

			workflowPath, err := c.workflowPath(ctx, owner, repo, run, workflowPaths)
			if err != nil {
				return nil, err
			}
			if !workflowMatches(workflowPath, idCfg.ApplyWorkflows) {
				continue
			}

			records, err := c.infraDeploymentsFromRun(ctx, owner, repo, run, workflowPath, date, idCfg, pattern, wsScope)
			if err != nil {
				return nil, err
			}
			result = append(result, records...)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return result, nil
}

func (c *Client) workflowPath(ctx context.Context, owner, repo string, run *gogh.WorkflowRun, cache map[int64]string) (string, error) {
	workflowID := run.GetWorkflowID()
	if path, ok := cache[workflowID]; ok {
		return path, nil
	}
	wf, _, err := c.api.Actions.GetWorkflowByID(ctx, owner, repo, workflowID)
	if err != nil {
		return "", fmt.Errorf("get workflow %d: %w", workflowID, err)
	}
	path := wf.GetPath()
	cache[workflowID] = path
	return path, nil
}

func (c *Client) infraDeploymentsFromRun(
	ctx context.Context,
	owner, repo string,
	run *gogh.WorkflowRun,
	workflowPath, date string,
	idCfg config.EvidenceInfraDeployment,
	pattern *regexp.Regexp,
	wsScope config.RepositoryScope,
) ([]evidence.InfraDeployment, error) {
	jobs, _, err := c.api.Actions.ListWorkflowJobs(ctx, owner, repo, run.GetID(), &gogh.ListWorkflowJobsOptions{
		Filter:      "latest",
		ListOptions: gogh.ListOptions{PerPage: 100},
	})
	if err != nil {
		return nil, fmt.Errorf("list workflow jobs run %d: %w", run.GetID(), err)
	}

	var result []evidence.InfraDeployment
	for _, job := range jobs.Jobs {
		if !jobSucceeded(job) {
			continue
		}
		if !jobNameMatches(job.GetName(), idCfg.ApplyJobNames) {
			continue
		}

		workspace, err := extractWorkspace(job.GetName(), pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "infra-deployment %s run %d job %q: %v\n", repo, run.GetID(), job.GetName(), err)
			continue
		}
		if !scope.Matches(workspace, wsScope.Include, wsScope.Exclude) {
			continue
		}

		appliedAt := job.GetCompletedAt().Time
		if appliedAt.UTC().Format("2006-01-02") != date {
			continue
		}

		triggeredBy := ""
		if run.Actor != nil && run.Actor.Login != nil {
			triggeredBy = *run.Actor.Login
		}

		id := evidence.NewInfraDeployment()
		id.Provider = idCfg.Provider
		if id.Provider == "" {
			id.Provider = "terraform"
		}
		id.Repository = repo
		id.Workspace = workspace
		id.Environment = idCfg.Environment
		id.CommitSHA = job.GetHeadSHA()
		id.AppliedAt = appliedAt.UTC()
		id.Execution = evidence.Execution{
			CIProvider:  "github_actions",
			Workflow:    filepath.Base(workflowPath),
			RunID:       strconv.FormatInt(run.GetID(), 10),
			Status:      "succeeded",
			TriggeredBy: triggeredBy,
			URL:         run.GetHTMLURL(),
		}
		result = append(result, id)
	}
	return result, nil
}

func runSucceeded(run *gogh.WorkflowRun) bool {
	if strings.ToLower(run.GetStatus()) != "completed" {
		return false
	}
	conclusion := strings.ToLower(run.GetConclusion())
	return conclusion == "success"
}

func jobSucceeded(job *gogh.WorkflowJob) bool {
	if strings.ToLower(job.GetStatus()) != "completed" {
		return false
	}
	conclusion := strings.ToLower(job.GetConclusion())
	return conclusion == "success"
}

func workflowMatches(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	base := filepath.Base(path)
	for _, p := range patterns {
		if p == path || p == base {
			return true
		}
		if strings.HasSuffix(path, p) {
			return true
		}
		if ok, _ := filepath.Match(p, base); ok {
			return true
		}
		if ok, _ := filepath.Match(p, path); ok {
			return true
		}
	}
	return false
}

func jobNameMatches(name string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func extractWorkspace(jobName string, pattern *regexp.Regexp) (string, error) {
	matches := pattern.FindStringSubmatch(jobName)
	if len(matches) < 2 || strings.TrimSpace(matches[1]) == "" {
		return "", fmt.Errorf("workspace not found in job name")
	}
	return strings.TrimSpace(matches[1]), nil
}
