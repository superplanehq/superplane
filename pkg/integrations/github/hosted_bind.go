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
	newInstallationClient = newClientForAppInstallation
	newAppJWTClient       = newClientForApp
	listInstallationRepos = listInstallationRepositories
	listAppInstallations  = listAppInstallationsFromGitHub
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

// adoptRequestedInstallation binds a pending connection whose install request
// was approved on GitHub. The approve callback carries no CSRF state and the
// installation webhook cannot find a connection without an installation id,
// so Sync asks GitHub whether the requested account has the App installed.
func (g *GitHub) adoptRequestedInstallation(ctx core.SyncContext, app common.HostedApp, metadata common.Metadata) (bool, error) {
	account := strings.TrimSpace(metadata.InstallRequestedAccount)
	if account == "" {
		return false, nil
	}

	client, err := newAppJWTClient(ctx.Integration, app.ID)
	if err != nil {
		return false, fmt.Errorf("failed to create app client: %w", err)
	}

	installationID, err := listAppInstallations(context.Background(), client, account)
	if err != nil {
		return false, fmt.Errorf("failed to list app installations: %w", err)
	}
	if installationID == "" {
		return false, nil
	}

	if err := g.bindHostedInstallationWith(ctx.Integration, ctx.Logger, metadata, installationID); err != nil {
		return false, err
	}
	return true, nil
}

// listAppInstallationsFromGitHub returns the id of the App installation owned
// by the account, or an empty string when the account has no installation.
func listAppInstallationsFromGitHub(ctx context.Context, client *github.Client, account string) (string, error) {
	opts := &github.ListOptions{PerPage: 100}
	for {
		installations, response, err := client.Apps.ListInstallations(ctx, opts)
		if err != nil {
			return "", err
		}

		for _, installation := range installations {
			if strings.EqualFold(installation.GetAccount().GetLogin(), account) {
				return strconv.FormatInt(installation.GetID(), 10), nil
			}
		}

		if response == nil || response.NextPage == 0 {
			return "", nil
		}
		opts.Page = response.NextPage
	}
}
