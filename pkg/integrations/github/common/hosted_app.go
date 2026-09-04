package common

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	EnvGitHubAppID            = config.EnvGitHubAppID
	EnvGitHubAppSlug          = config.EnvGitHubAppSlug
	EnvGitHubAppPrivateKey    = config.EnvGitHubAppPrivateKey
	EnvGitHubAppWebhookSecret = config.EnvGitHubAppWebhookSecret
	EnvGitHubAppClientID      = config.EnvGitHubAppClientID
	EnvGitHubAppClientSecret  = config.EnvGitHubAppClientSecret
)

// HostedApp is SuperPlane Cloud's public GitHub App. The process holds the
// credentials. New integrations store only the GitHub installation id.
type HostedApp struct {
	ID            int64
	Slug          string
	PrivateKey    string
	WebhookSecret string
	ClientID      string
	ClientSecret  string
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
		ClientID:      cfg.ClientID,
		ClientSecret:  cfg.ClientSecret,
	}, true
}

func (a HostedApp) UserOAuthEnabled() bool {
	return a.ClientID != "" && a.ClientSecret != ""
}

func HostedAppConfigured() bool {
	return config.LoadGitHubHostedAppConfig().Enabled()
}

func HostedAppInstallURL(slug, state string) string {
	return HostedAppInstallURLForAccount(slug, state, 0)
}

func HostedAppInstallURLForAccount(slug, state string, targetID int64) string {
	values := url.Values{}
	if state != "" {
		values.Set("state", state)
	}
	if targetID > 0 {
		values.Set("target_id", strconv.FormatInt(targetID, 10))
	}

	path := fmt.Sprintf("https://github.com/apps/%s/installations/new", slug)
	encoded := values.Encode()
	if encoded == "" {
		return path
	}

	return path + "?" + encoded
}

func HostedAppOAuthCallbackURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/") + "/api/v1/github/app/oauth/callback"
}

func HostedAppAuthorizeURL(clientID, redirectURI, state string) string {
	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("redirect_uri", redirectURI)
	values.Set("state", state)
	return "https://github.com/login/oauth/authorize?" + values.Encode()
}

func HostedAppBindURL(baseURL, state, installationID string) string {
	values := url.Values{}
	values.Set("state", state)
	values.Set("installation_id", installationID)
	return strings.TrimRight(baseURL, "/") + "/api/v1/github/app/bind?" + values.Encode()
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
