package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	EnvGitHubAppID            = "SUPERPLANE_GITHUB_APP_ID"
	EnvGitHubAppSlug          = "SUPERPLANE_GITHUB_APP_SLUG"
	EnvGitHubAppPrivateKey    = "SUPERPLANE_GITHUB_APP_PRIVATE_KEY"
	EnvGitHubAppWebhookSecret = "SUPERPLANE_GITHUB_APP_WEBHOOK_SECRET"
	EnvGitHubAppClientID      = "SUPERPLANE_GITHUB_APP_CLIENT_ID"
	EnvGitHubAppClientSecret  = "SUPERPLANE_GITHUB_APP_CLIENT_SECRET"
)

// GitHubHostedAppConfig is SuperPlane Cloud's public GitHub App. The process
// holds the credentials. New connections store only the GitHub installation id.
type GitHubHostedAppConfig struct {
	ID            int64
	Slug          string
	PrivateKey    string
	WebhookSecret string
	ClientID      string
	ClientSecret  string
}

// LoadGitHubHostedAppConfig reads the public GitHub App from the process
// environment. Self-hosted leaves these empty. Enabled() is false unless
// every required value is set.
func LoadGitHubHostedAppConfig() GitHubHostedAppConfig {
	idRaw := strings.TrimSpace(os.Getenv(EnvGitHubAppID))
	slug := strings.TrimSpace(os.Getenv(EnvGitHubAppSlug))
	privateKey := normalizePEM(os.Getenv(EnvGitHubAppPrivateKey))
	webhookSecret := strings.TrimSpace(os.Getenv(EnvGitHubAppWebhookSecret))
	if idRaw == "" || slug == "" || privateKey == "" || webhookSecret == "" {
		return GitHubHostedAppConfig{}
	}

	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil || id <= 0 {
		return GitHubHostedAppConfig{}
	}

	return GitHubHostedAppConfig{
		ID:            id,
		Slug:          slug,
		PrivateKey:    privateKey,
		WebhookSecret: webhookSecret,
		ClientID:      strings.TrimSpace(os.Getenv(EnvGitHubAppClientID)),
		ClientSecret:  strings.TrimSpace(os.Getenv(EnvGitHubAppClientSecret)),
	}
}

// Enabled reports whether Cloud holds a complete public GitHub App.
func (c GitHubHostedAppConfig) Enabled() bool {
	return c.ID > 0 && c.Slug != "" && c.PrivateKey != "" && c.WebhookSecret != ""
}

// UserOAuthEnabled reports whether Cloud can list the current user's installs
// of the public GitHub App. This needs the App client id and client secret
// in addition to Enabled().
func (c GitHubHostedAppConfig) UserOAuthEnabled() bool {
	return c.Enabled() && c.ClientID != "" && c.ClientSecret != ""
}

func normalizePEM(value string) string {
	value = strings.TrimSpace(value)
	if n := len(value); n >= 2 {
		if (value[0] == '"' && value[n-1] == '"') || (value[0] == '\'' && value[n-1] == '\'') {
			value = value[1 : n-1]
		}
	}
	return strings.TrimSpace(strings.ReplaceAll(value, `\n`, "\n"))
}
