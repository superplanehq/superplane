package common

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const accessTokenSecret = "accessToken"

// Credentials holds the Azure AD access token for ARM API calls.
type Credentials struct {
	AccessToken string
}

type IntegrationMetadata struct {
	Session *SessionMetadata `json:"session" mapstructure:"session"`
	Tags    []Tag            `json:"tags" mapstructure:"tags"`
}

type SessionMetadata struct {
	TenantID       string `json:"tenantId" mapstructure:"tenantId"`
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
	ClientID       string `json:"clientId" mapstructure:"clientId"`
	ExpiresAt      string `json:"expiresAt" mapstructure:"expiresAt"`
	Location       string `json:"location" mapstructure:"location"`
}

type Tag struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

func CredentialsFromInstallation(ctx core.IntegrationContext) (*Credentials, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure session secrets: %w", err)
	}

	var accessToken string
	for _, secret := range secrets {
		if secret.Name == accessTokenSecret {
			accessToken = string(secret.Value)
			break
		}
	}

	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("Azure access token is missing; integration may need resync")
	}

	return &Credentials{AccessToken: accessToken}, nil
}

func LocationFromInstallation(ctx core.IntegrationContext) string {
	locationBytes, err := ctx.GetConfig("location")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(locationBytes))
}

func SubscriptionIDFromInstallation(ctx core.IntegrationContext) string {
	subBytes, err := ctx.GetConfig("subscriptionId")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(subBytes))
}

func NormalizeTags(tags []Tag) []Tag {
	if len(tags) == 0 {
		return nil
	}

	normalized := make([]Tag, 0, len(tags))
	seen := map[string]int{}
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}

		value := strings.TrimSpace(tag.Value)
		if index, ok := seen[key]; ok {
			normalized[index].Value = value
			continue
		}

		seen[key] = len(normalized)
		normalized = append(normalized, Tag{
			Key:   key,
			Value: value,
		})
	}

	return normalized
}
