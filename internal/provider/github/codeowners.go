package github

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	gogh "github.com/google/go-github/v62/github"
	"github.com/keyskey/kao/internal/evidence"
)

var codeownersPaths = []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"}

func (c *Client) fetchCodeowners(ctx context.Context, owner, repo string) (evidence.Codeowners, error) {
	for _, path := range codeownersPaths {
		content, _, resp, err := c.api.Repositories.GetContents(ctx, owner, repo, path, nil)
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				continue
			}
			return evidence.Codeowners{}, err
		}
		if content == nil {
			continue
		}
		text, err := content.GetContent()
		if err != nil {
			return evidence.Codeowners{}, err
		}
		rules, err := c.parseCodeowners(ctx, text)
		if err != nil {
			return evidence.Codeowners{}, err
		}
		return evidence.Codeowners{
			Exists: true,
			Path:   path,
			Rules:  rules,
		}, nil
	}
	return evidence.Codeowners{Exists: false}, nil
}

func (c *Client) parseCodeowners(ctx context.Context, content string) ([]evidence.CodeownerRule, error) {
	var rules []evidence.CodeownerRule
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		pattern := parts[0]
		var owners []evidence.Owner
		for _, raw := range parts[1:] {
			owner, err := c.resolveOwner(ctx, raw)
			if err != nil {
				return nil, err
			}
			owners = append(owners, owner)
		}
		rules = append(rules, evidence.CodeownerRule{
			Pattern: pattern,
			Owners:  owners,
		})
	}
	return rules, scanner.Err()
}

func (c *Client) resolveOwner(ctx context.Context, raw string) (evidence.Owner, error) {
	name := strings.TrimPrefix(raw, "@")
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		org, teamSlug := parts[0], parts[1]
		users, err := c.resolveTeamMembers(ctx, org, teamSlug)
		if err != nil {
			return evidence.Owner{}, err
		}
		return evidence.Owner{
			Type:          "team",
			Name:          "@" + name,
			ResolvedUsers: users,
		}, nil
	}
	return evidence.Owner{
		Type:          "user",
		Name:          "@" + name,
		ResolvedUsers: []string{name},
	}, nil
}

func (c *Client) resolveTeamMembers(ctx context.Context, org, teamSlug string) ([]string, error) {
	var users []string
	opts := &gogh.TeamListTeamMembersOptions{ListOptions: gogh.ListOptions{PerPage: 100}}
	for {
		members, resp, err := c.api.Teams.ListTeamMembersBySlug(ctx, org, teamSlug, opts)
		if err != nil {
			return nil, fmt.Errorf("list team members %s/%s: %w", org, teamSlug, err)
		}
		for _, m := range members {
			if m.Login != nil {
				users = append(users, *m.Login)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return users, nil
}

func ResolvedUsersUnion(co evidence.Codeowners) map[string]bool {
	set := make(map[string]bool)
	for _, rule := range co.Rules {
		for _, owner := range rule.Owners {
			for _, u := range owner.ResolvedUsers {
				set[u] = true
			}
		}
	}
	return set
}
