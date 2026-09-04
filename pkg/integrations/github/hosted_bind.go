package github

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

var (
	newInstallationClient = newClientForAppInstallation
	newAppJWTClient       = newClientForApp
	listInstallationRepos = listInstallationRepositories
)

func (g *GitHub) bindHostedInstallation(ctx core.HTTPRequestContext, metadata common.Metadata, installationID string) error {
	metadata.InstallationID = installationID
	client, err := newInstallationClient(ctx.Integration, metadata.GitHubApp.ID, installationID)
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
		appClient, err := newAppJWTClient(ctx.Integration, metadata.GitHubApp.ID)
		if err != nil {
			ctx.Logger.Errorf("failed to create app client: %v", err)
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

	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.RemoveBrowserAction()
	ctx.Integration.Ready()

	ctx.Logger.Infof("Successfully installed GitHub App %s - installation=%s", metadata.GitHubApp.Slug, metadata.InstallationID)
	ctx.Logger.Infof("Repositories: %v", metadata.Repositories)
	return nil
}
