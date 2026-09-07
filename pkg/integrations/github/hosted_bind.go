package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

var (
	newInstallationClient       = newClientForAppInstallation
	newAppJWTClient             = newClientForApp
	listInstallationRepos       = listInstallationRepositories
	listAppInstallations        = listAppInstallationsFromGitHub
	listAppInstallationRequests = listAppInstallationRequestsFromGitHub
)

func (g *GitHub) bindHostedInstallation(ctx core.HTTPRequestContext, metadata common.Metadata, installationID string) error {
	return g.bindHostedInstallationWith(ctx.Integration, ctx.Logger, metadata, installationID)
}

func (g *GitHub) bindHostedInstallationWith(
	integration core.IntegrationContext,
	logger *logrus.Entry,
	metadata common.Metadata,
	installationID string,
) error {
	metadata.InstallationID = installationID
	client, err := newInstallationClient(integration, metadata.GitHubApp.ID, installationID)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if metadata.Owner == "" && !metadata.HostedApp {
		ghApp, _, err := client.Apps.Get(context.Background(), metadata.GitHubApp.Slug)
		if err != nil {
			return fmt.Errorf("failed to get app: %w", err)
		}

		metadata.Owner = ghApp.Owner.GetLogin()
	}

	repos, err := listInstallationRepos(context.Background(), client)
	if err != nil {
		return fmt.Errorf("failed to list repos: %w", err)
	}

	if metadata.Owner == "" {
		appClient, err := newAppJWTClient(integration, metadata.GitHubApp.ID)
		if err != nil {
			logger.Errorf("failed to create app client: %v", err)
		}
		metadata.Owner = resolveInstallationOwner(context.Background(), appClient, installationID, repos)
	}

	if metadata.Owner == "" {
		return fmt.Errorf("installation owner is empty for installation %s", installationID)
	}

	metadata.Repositories = repos
	metadata.State = ""
	metadata.PendingInstallations = nil
	metadata.InstallRequested = false
	metadata.InstallRequestedAccount = ""

	integration.SetMetadata(metadata)
	integration.RemoveBrowserAction()
	integration.Ready()

	logger.Infof("Successfully installed GitHub App %s - installation=%s", metadata.GitHubApp.Slug, metadata.InstallationID)
	logger.Infof("Repositories: %v", metadata.Repositories)
	return nil
}

// adoptRequestedInstallation resolves a pending install request once an owner
// approved it on GitHub. The approve callback carries no CSRF state and the
// installation webhook cannot find a connection without an installation id,
// so Sync asks GitHub whether the requested account has the App installed.
// An approved installation joins the account picker; the member still picks
// the account, a silent bind must not happen.
//
// The request callback from GitHub also does not name the requested account,
// so when it is unknown Sync finds the member's open install request on
// GitHub and records the account on the metadata for the next sync and the
// waiting screen.
func (g *GitHub) adoptRequestedInstallation(ctx core.SyncContext, app common.HostedApp, metadata *common.Metadata) error {
	client, err := newAppJWTClient(ctx.Integration, app.ID)
	if err != nil {
		return fmt.Errorf("failed to create app client: %w", err)
	}

	account := strings.TrimSpace(metadata.InstallRequestedAccount)
	if account == "" {
		account, err = listAppInstallationRequests(context.Background(), client, metadata.StartedByGitHubLogin)
		if err != nil {
			return fmt.Errorf("failed to list app installation requests: %w", err)
		}
		if account == "" {
			return nil
		}
		metadata.InstallRequestedAccount = account
	}

	installation, found, err := listAppInstallations(context.Background(), client, account)
	if err != nil {
		return fmt.Errorf("failed to list app installations: %w", err)
	}
	if !found {
		return nil
	}

	if !metadata.AllowsPendingInstallation(installation.ID) {
		metadata.PendingInstallations = append(metadata.PendingInstallations, installation)
	}
	metadata.InstallRequested = false
	metadata.InstallRequestedAccount = ""
	return nil
}

// listAppInstallationRequestsFromGitHub returns the account login of the open
// App install request made by the requester, or an empty string when the
// requester has no open request.
func listAppInstallationRequestsFromGitHub(ctx context.Context, client *github.Client, requesterLogin string) (string, error) {
	if strings.TrimSpace(requesterLogin) == "" {
		return "", nil
	}

	opts := &github.ListOptions{PerPage: 100}
	for {
		requests, response, err := client.Apps.ListInstallationRequests(ctx, opts)
		if err != nil {
			return "", err
		}

		for _, request := range requests {
			if strings.EqualFold(request.GetRequester().GetLogin(), requesterLogin) {
				return request.GetAccount().GetLogin(), nil
			}
		}

		if response == nil || response.NextPage == 0 {
			return "", nil
		}
		opts.Page = response.NextPage
	}
}

// listAppInstallationsFromGitHub returns the App installation owned by the
// account, or found=false when the account has no installation.
func listAppInstallationsFromGitHub(ctx context.Context, client *github.Client, account string) (common.PendingInstallation, bool, error) {
	opts := &github.ListOptions{PerPage: 100}
	for {
		installations, response, err := client.Apps.ListInstallations(ctx, opts)
		if err != nil {
			return common.PendingInstallation{}, false, err
		}

		for _, installation := range installations {
			if strings.EqualFold(installation.GetAccount().GetLogin(), account) {
				return common.PendingInstallation{
					ID:           strconv.FormatInt(installation.GetID(), 10),
					AccountLogin: installation.GetAccount().GetLogin(),
					AccountType:  installation.GetAccount().GetType(),
				}, true, nil
			}
		}

		if response == nil || response.NextPage == 0 {
			return common.PendingInstallation{}, false, nil
		}
		opts.Page = response.NextPage
	}
}
