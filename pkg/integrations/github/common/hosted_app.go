package common

import (
	"fmt"
	"net/url"

	"github.com/bradleyfalzon/ghinstallation/v2"
	githubauth "github.com/google/go-github/v75/github"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	EnvGitHubAppID            = config.EnvGitHubAppID
	EnvGitHubAppSlug          = config.EnvGitHubAppSlug
	EnvGitHubAppPrivateKey    = config.EnvGitHubAppPrivateKey
	EnvGitHubAppWebhookSecret = config.EnvGitHubAppWebhookSecret
)

// HostedApp is SuperPlane Cloud's public GitHub App. The process holds the
// credentials. Organization integrations reference a global installation.
type HostedApp struct {
	ID            int64
	Slug          string
	PrivateKey    string
	WebhookSecret string
}

type HostedAppBinding struct {
	App           HostedApp
	Installation  *models.VCSProviderInstallation
	ID            int64
	RepositoryIDs []int64
}

func ResolveHostedAppBinding(ctx core.IntegrationContext) (*HostedAppBinding, error) {
	binding, err := models.FindVCSProviderIntegrationBinding(database.Conn(), ctx.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to find global GitHub App binding: %w", err)
	}
	if binding.Provider != models.ProviderGitHub {
		return nil, fmt.Errorf("hosted GitHub integration has provider %q", binding.Provider)
	}
	installation, err := models.FindVCSProviderInstallation(database.Conn(), binding.Provider, binding.InstallationID)
	if err != nil {
		return nil, fmt.Errorf("failed to find global GitHub App installation: %w", err)
	}
	if installation.SuspendedAt != nil {
		return nil, fmt.Errorf("global GitHub App installation is suspended")
	}
	repositories, err := models.ListVCSProviderBindingRepositories(database.Conn(), ctx.ID())
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories granted to the GitHub App binding: %w", err)
	}
	if len(repositories) == 0 {
		return nil, fmt.Errorf("hosted GitHub integration has no granted repositories")
	}
	repositoryIDs := make([]int64, 0, len(repositories))
	for _, repository := range repositories {
		repositoryIDs = append(repositoryIDs, repository.RepositoryID)
	}
	app, ok := HostedAppFromEnv()
	if !ok {
		return nil, fmt.Errorf("hosted GitHub App is not configured")
	}
	return &HostedAppBinding{
		App:           app,
		Installation:  installation,
		ID:            binding.InstallationID,
		RepositoryIDs: repositoryIDs,
	}, nil
}

func RestrictHostedAppTransport(transport *ghinstallation.Transport, repositoryIDs []int64) error {
	if transport == nil {
		return fmt.Errorf("GitHub App transport is required")
	}
	if len(repositoryIDs) == 0 {
		return fmt.Errorf("hosted GitHub integration has no granted repositories")
	}
	transport.InstallationTokenOptions = &githubauth.InstallationTokenOptions{
		RepositoryIDs: append([]int64(nil), repositoryIDs...),
	}
	return nil
}

// HostedAppFromEnv returns the public GitHub App when Cloud holds complete
// credentials. Self-hosted leaves them empty.
func HostedAppFromEnv() (HostedApp, bool) {
	cfg := config.LoadGitHubHostedAppConfig()
	if !cfg.Enabled() {
		return HostedApp{}, false
	}

	return HostedApp{
		ID:            cfg.ID,
		Slug:          cfg.Slug,
		PrivateKey:    cfg.PrivateKey,
		WebhookSecret: cfg.WebhookSecret,
	}, true
}

func HostedAppConfigured() bool {
	return config.LoadGitHubHostedAppConfig().Enabled()
}

func HostedAppInstallURL(slug, state string) string {
	return fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s", slug, url.QueryEscape(state))
}

// LegacyAppPrivateKey returns the PEM for a legacy GitHub App connection.
// Hosted connections use the process key. Customer-created apps use the
// integration secret.
func LegacyAppPrivateKey(ctx core.IntegrationContext, metadata Metadata) (string, error) {
	if metadata.HostedApp {
		app, ok := HostedAppFromEnv()
		if !ok {
			return "", fmt.Errorf("hosted GitHub App is not configured")
		}
		return app.PrivateKey, nil
	}

	return FindSecret(ctx, GitHubAppPEM)
}
