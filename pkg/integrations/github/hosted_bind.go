package github

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

var (
	newInstallationClient = newClientForAppInstallation
	newAppJWTClient       = newClientForApp
	listInstallationRepos = listInstallationRepositories
)

func (g *GitHub) bindLegacyInstallation(ctx core.HTTPRequestContext, metadata common.Metadata, installationID string) error {
	return g.bindLegacyInstallationWith(ctx.Integration, ctx.Logger, metadata, installationID)
}

func (g *GitHub) bindLegacyInstallationWith(
	integration core.IntegrationContext,
	logger *logrus.Entry,
	metadata common.Metadata,
	installationID string,
) error {
	// A rebind moves the connection to another account, so the owner below
	// comes from the new installation.
	if metadata.InstallationID != "" && metadata.InstallationID != installationID {
		metadata.Owner = ""
	}
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
	remainingRequests := slices.DeleteFunc(metadata.CurrentInstallRequests(), func(request common.InstallRequest) bool {
		return request.AccountLogin == "" || strings.EqualFold(request.AccountLogin, metadata.Owner)
	})
	metadata.SetInstallRequests(remainingRequests)

	integration.SetMetadata(metadata)
	integration.RemoveBrowserAction()
	integration.Ready()

	logger.Infof("Successfully installed GitHub App %s - installation=%s", metadata.GitHubApp.Slug, metadata.InstallationID)
	logger.Infof("Repositories: %v", metadata.Repositories)
	return nil
}
