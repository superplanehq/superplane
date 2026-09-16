package config

import (
	"os"
	"strings"
)

const (
	EnvSentryAppSlug         = "SUPERPLANE_SENTRY_APP_SLUG"
	EnvSentryAppClientID     = "SUPERPLANE_SENTRY_APP_CLIENT_ID"
	EnvSentryAppClientSecret = "SUPERPLANE_SENTRY_APP_CLIENT_SECRET"
)

// SentryHostedAppConfig is SuperPlane Cloud's public Sentry app. The process
// holds the credentials. New connections store only the installation UUID
// and tokens.
type SentryHostedAppConfig struct {
	Slug         string
	ClientID     string
	ClientSecret string
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
	}
}

// Enabled reports whether Cloud holds a complete public Sentry app.
func (c SentryHostedAppConfig) Enabled() bool {
	return c.Slug != "" && c.ClientID != "" && c.ClientSecret != ""
}
