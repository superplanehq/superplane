package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	EnvGitHubAppID            = "SUPERPLANE_GITHUB_APP_ID"
	EnvGitHubAppSlug          = "SUPERPLANE_GITHUB_APP_SLUG"
	EnvGitHubAppPrivateKey    = "SUPERPLANE_GITHUB_APP_PRIVATE_KEY"
	EnvGitHubAppWebhookSecret = "SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET"
)

// HostedApp is SuperPlane Cloud's public GitHub App. The process holds the
// credentials. New integrations store only the GitHub installation id.
type HostedApp struct {
	ID            int64
	Slug          string
	PrivateKey    string
	WebhookSecret string
}

// HostedAppFromEnv returns the public GitHub App when every required
// environment variable is set. Self-hosted leaves them empty.
func HostedAppFromEnv() (HostedApp, bool) {
	idRaw := strings.TrimSpace(os.Getenv(EnvGitHubAppID))
	slug := strings.TrimSpace(os.Getenv(EnvGitHubAppSlug))
	privateKey := normalizePEM(os.Getenv(EnvGitHubAppPrivateKey))
	webhookSecret := strings.TrimSpace(os.Getenv(EnvGitHubAppWebhookSecret))
	if idRaw == "" || slug == "" || privateKey == "" || webhookSecret == "" {
		return HostedApp{}, false
	}

	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil || id <= 0 {
		return HostedApp{}, false
	}

	return HostedApp{
		ID:            id,
		Slug:          slug,
		PrivateKey:    privateKey,
		WebhookSecret: webhookSecret,
	}, true
}

func HostedAppConfigured() bool {
	_, ok := HostedAppFromEnv()
	return ok
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

func normalizePEM(value string) string {
	value = strings.TrimSpace(value)
	return strings.ReplaceAll(value, `\n`, "\n")
}
