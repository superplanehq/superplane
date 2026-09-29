package config

import (
	"os"
	"strings"
)

const (
	EnvJiraOAuthClientID     = "SUPERPLANE_JIRA_OAUTH_CLIENT_ID"
	EnvJiraOAuthClientSecret = "SUPERPLANE_JIRA_OAUTH_CLIENT_SECRET"
)

// JiraHostedOAuthConfig is SuperPlane's Atlassian OAuth 2.0 (3LO) app.
// The process holds the credentials. New connections store only tokens.
type JiraHostedOAuthConfig struct {
	ClientID     string
	ClientSecret string
}

// LoadJiraHostedOAuthConfig reads the public Jira OAuth app from the process
// environment. Self-hosted leaves these empty. Enabled() is false unless
// both values are set.
func LoadJiraHostedOAuthConfig() JiraHostedOAuthConfig {
	clientID := strings.TrimSpace(os.Getenv(EnvJiraOAuthClientID))
	clientSecret := strings.TrimSpace(os.Getenv(EnvJiraOAuthClientSecret))
	if clientID == "" || clientSecret == "" {
		return JiraHostedOAuthConfig{}
	}

	return JiraHostedOAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}

// Enabled reports whether Cloud holds a complete Jira OAuth app.
func (c JiraHostedOAuthConfig) Enabled() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}
