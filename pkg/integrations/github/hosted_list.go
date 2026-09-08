package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

// NewHostedAppClient returns a GitHub App JWT client from process credentials.
func NewHostedAppClient(app common.HostedApp) (*github.Client, error) {
	if app.ID <= 0 || app.PrivateKey == "" {
		return nil, fmt.Errorf("hosted GitHub App is not configured")
	}

	transport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, app.ID, []byte(app.PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub App transport: %w", err)
	}

	return github.NewClient(&http.Client{Transport: transport}), nil
}

// ListHostedAppInstallations lists every GitHub account that has installed
// the hosted SuperPlane GitHub App.
func ListHostedAppInstallations(ctx context.Context, app common.HostedApp) ([]common.PendingInstallation, error) {
	client, err := NewHostedAppClient(app)
	if err != nil {
		return nil, err
	}

	var result []common.PendingInstallation
	opts := &github.ListOptions{PerPage: 100}
	for {
		installations, response, err := client.Apps.ListInstallations(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list GitHub App installations: %w", err)
		}

		for _, installation := range installations {
			if installation == nil || installation.GetID() == 0 {
				continue
			}
			result = append(result, common.PendingInstallation{
				ID:           strconv.FormatInt(installation.GetID(), 10),
				AccountLogin: installation.GetAccount().GetLogin(),
				AccountType:  installation.GetAccount().GetType(),
			})
		}

		if response == nil || response.NextPage == 0 {
			return result, nil
		}
		opts.Page = response.NextPage
	}
}
