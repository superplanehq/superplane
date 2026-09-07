package config

import (
	"os"
	"strings"
)

const (
	EnvSentryAppClientID     = "SUPERPLANE_SENTRY_APP_CLIENT_ID"
	EnvSentryAppClientSecret = "SUPERPLANE_SENTRY_APP_CLIENT_SECRET"
	EnvSentryAppSlug         = "SUPERPLANE_SENTRY_APP_SLUG"
	EnvSentryAppBaseURL      = "SUPERPLANE_SENTRY_BASE_URL"

	DefaultSentryAppBaseURL = "https://sentry.io"
)

// SentryHostedAppConfig is SuperPlane Cloud's public Sentry app. The process
// holds the credentials. New connections store only the installation id.
type SentryHostedAppConfig struct {
	ClientID     string
	ClientSecret string
	Slug         string
	BaseURL      string
}

// LoadSentryHostedAppConfig reads the public Sentry app from the process
// environment. Self-hosted leaves these empty. Enabled() is false unless
// every required value is set.
func LoadSentryHostedAppConfig() SentryHostedAppConfig {
	clientID := strings.TrimSpace(os.Getenv(EnvSentryAppClientID))
	clientSecret := strings.TrimSpace(os.Getenv(EnvSentryAppClientSecret))
	slug := strings.TrimSpace(os.Getenv(EnvSentryAppSlug))
	if clientID == "" || clientSecret == "" || slug == "" {
		return SentryHostedAppConfig{}
	}

	baseURL := strings.TrimSpace(os.Getenv(EnvSentryAppBaseURL))
	if baseURL == "" {
		baseURL = DefaultSentryAppBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	return SentryHostedAppConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Slug:         slug,
		BaseURL:      baseURL,
	}
}

// Enabled reports whether Cloud holds a complete public Sentry app.
func (c SentryHostedAppConfig) Enabled() bool {
	return c.ClientID != "" && c.ClientSecret != "" && c.Slug != "" && c.BaseURL != ""
}
