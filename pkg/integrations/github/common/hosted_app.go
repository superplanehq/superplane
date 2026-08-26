package common

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	EnvGitHubAppID            = config.EnvGitHubAppID
	EnvGitHubAppSlug          = config.EnvGitHubAppSlug
	EnvGitHubAppPrivateKey    = config.EnvGitHubAppPrivateKey
	EnvGitHubAppWebhookSecret = config.EnvGitHubAppWebhookSecret
)

// HostedApp is SuperPlane Cloud's public GitHub App. The process holds the
// credentials. New integrations store only the GitHub installation id.
type HostedApp struct {
	ID            int64
	Slug          string
	PrivateKey    string
	WebhookSecret string
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
	return fmt.Sprintf("https://github.com/apps/%s/installations/new?state=%s", slug, state)
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
