package github

import (
	gogh "github.com/google/go-github/v62/github"
	"github.com/keyskey/kao/internal/config"
)

type Client struct {
	api *gogh.Client
	org string
}

func New(cfg *config.Config) (*Client, error) {
	token, err := cfg.GitHubToken()
	if err != nil {
		return nil, err
	}
	return &Client{
		api: gogh.NewClient(nil).WithAuthToken(token),
		org: cfg.Providers.GitHub.Org,
	}, nil
}

func (c *Client) Org() string {
	return c.org
}

func (c *Client) API() *gogh.Client {
	return c.api
}
