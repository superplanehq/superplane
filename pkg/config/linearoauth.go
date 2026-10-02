package config

import (
	"os"
	"strings"
)

const (
	EnvLinearOAuthClientID     = "SUPERPLANE_LINEAR_OAUTH_CLIENT_ID"
	EnvLinearOAuthClientSecret = "SUPERPLANE_LINEAR_OAUTH_CLIENT_SECRET"
)

// LinearHostedOAuthConfig is SuperPlane's public Linear OAuth application.
// The process holds the credentials. New connections store only tokens.
type LinearHostedOAuthConfig struct {
	ClientID     string
	ClientSecret string
}

// LoadLinearHostedOAuthConfig reads the public Linear OAuth app from the
// process environment. Enabled() is false unless both values are set.
func LoadLinearHostedOAuthConfig() LinearHostedOAuthConfig {
	clientID := strings.TrimSpace(os.Getenv(EnvLinearOAuthClientID))
	clientSecret := strings.TrimSpace(os.Getenv(EnvLinearOAuthClientSecret))
	if clientID == "" || clientSecret == "" {
		return LinearHostedOAuthConfig{}
	}

	return LinearHostedOAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

// Enabled reports whether the process holds a complete Linear OAuth app.
func (c LinearHostedOAuthConfig) Enabled() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}
