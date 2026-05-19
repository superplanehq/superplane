package jira

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	AuthTypeOAuth = "oauth"

	OAuthAccessToken  = "accessToken"
	OAuthRefreshToken = "refreshToken"
)

// NodeMetadata stores metadata on trigger/component nodes.
type NodeMetadata struct {
	Project *Project `json:"project,omitempty"`
}

func getConfigString(ctx core.IntegrationContext, name string) string {
	value, err := ctx.GetConfig(name)
	if err != nil {
		return ""
	}

	return string(value)
}

func loadConfiguration(ctx core.IntegrationContext) Configuration {
	config := Configuration{
		ClientID:     getConfigString(ctx, "clientId"),
		ClientSecret: getConfigString(ctx, "clientSecret"),
	}

	config.ClientID = strings.TrimSpace(config.ClientID)
	config.ClientSecret = strings.TrimSpace(config.ClientSecret)
	return config
}

func findSecret(integration core.IntegrationContext, name string) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == name {
			return string(secret.Value), nil
		}
	}

	return "", nil
}

func requireOAuthSecret(integration core.IntegrationContext, name string) (string, error) {
	value, err := findSecret(integration, name)
	if err != nil {
		return "", err
	}

	if value == "" {
		return "", fmt.Errorf("OAuth %s not found", name)
	}

	return value, nil
}
