package argocd

import (
	"io"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient"
	applicationpkg "github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/keyskey/kao/internal/config"
)

type Client struct {
	app    applicationpkg.ApplicationServiceClient
	closer io.Closer
}

func New(cfg *config.Config) (*Client, error) {
	server, err := cfg.ArgoCDServer()
	if err != nil {
		return nil, err
	}
	token, err := cfg.ArgoCDToken()
	if err != nil {
		return nil, err
	}

	api, err := apiclient.NewClient(&apiclient.ClientOptions{
		ServerAddr: server,
		AuthToken:  token,
		GRPCWeb:    *cfg.Providers.ArgoCD.GRPCWeb,
		Insecure:   *cfg.Providers.ArgoCD.Insecure,
	})
	if err != nil {
		return nil, err
	}

	conn, appClient, err := api.NewApplicationClient()
	if err != nil {
		return nil, err
	}

	return &Client{app: appClient, closer: conn}, nil
}

func (c *Client) Close() error {
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}
