package config

import (
	"os"
	"strings"
)

const (
	EnvSentryAppSlug         = "SUPERPLANE_SENTRY_APP_SLUG"
	EnvSentryAppClientID     = "SUPERPLANE_SENTRY_APP_CLIENT_ID"
	EnvSentryAppClientSecret = "SUPERPLANE_SENTRY_APP_CLIENT_SECRET"
	EnvSentryAppAPIToken     = "SUPERPLANE_SENTRY_APP_API_TOKEN"

	// EnvSentryWebhookDurableAcceptance turns on durable Sentry webhook
	// acceptance in the API. Deploy the Sentry webhook worker first.
	EnvSentryWebhookDurableAcceptance = "SENTRY_WEBHOOK_DURABLE_ACCEPTANCE"
)

// SentryWebhookDurableAcceptance reports whether the API stores verified
// Sentry webhooks in the durable inbox and answers before it processes them.
// When it is off, the API keeps the synchronous delivery path.
func SentryWebhookDurableAcceptance() bool {
	return strings.TrimSpace(os.Getenv(EnvSentryWebhookDurableAcceptance)) == "yes"
}

// SentryHostedAppConfig is SuperPlane Cloud's public Sentry app. The process
// holds the credentials. New connections store only the installation UUID
// and tokens.
type SentryHostedAppConfig struct {
	Slug         string
	ClientID     string
	ClientSecret string
	// APIToken is an optional owner token. When set, SuperPlane reads the
	// public app and refuses a ready connection unless the webhook sends
	// issue events. It is not required to install the app.
	APIToken string
}

// LoadSentryHostedAppConfig reads the public Sentry app from the process
// environment. Self-hosted leaves these empty. Enabled() is false unless
// every required value is set.
func LoadSentryHostedAppConfig() SentryHostedAppConfig {
	slug := strings.TrimSpace(os.Getenv(EnvSentryAppSlug))
	clientID := strings.TrimSpace(os.Getenv(EnvSentryAppClientID))
	clientSecret := strings.TrimSpace(os.Getenv(EnvSentryAppClientSecret))
	if slug == "" || clientID == "" || clientSecret == "" {
		return SentryHostedAppConfig{}
	}

	return SentryHostedAppConfig{
		Slug:         slug,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		APIToken:     strings.TrimSpace(os.Getenv(EnvSentryAppAPIToken)),
	}
}

// Enabled reports whether Cloud holds a complete public Sentry app.
func (c SentryHostedAppConfig) Enabled() bool {
	return c.Slug != "" && c.ClientID != "" && c.ClientSecret != ""
}
