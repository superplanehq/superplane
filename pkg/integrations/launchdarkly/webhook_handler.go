package launchdarkly

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration is the config stored with the webhook.
type WebhookConfiguration struct {
	ProjectKey string `json:"projectKey" mapstructure:"projectKey"`
}

// WebhookMetadata is stored after Setup. It holds the LaunchDarkly webhook ID
// so we can delete it when the trigger is removed.
type WebhookMetadata struct {
	LDWebhookID string `json:"ldWebhookId"`
}

type LaunchDarklyWebhookHandler struct{}

func (h *LaunchDarklyWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.ProjectKey == configB.ProjectKey, nil
}

func (h *LaunchDarklyWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

// Setup creates a signed webhook in LaunchDarkly via the API using the integration's API key.
// LaunchDarkly auto-generates the signing secret, which we store encrypted for later verification.
func (h *LaunchDarklyWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create LaunchDarkly client: %w", err)
	}

	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	resource := fmt.Sprintf("proj/%s:env/*:flag/*", config.ProjectKey)
	webhook, err := client.CreateWebhook(CreateWebhookRequest{
		URL:  ctx.Webhook.GetURL(),
		Sign: true,
		On:   true,
		Name: "SuperPlane",
		Statements: []WebhookStatement{
			{
				Effect:    "allow",
				Resources: []string{resource},
				Actions:   []string{"*"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook in LaunchDarkly: %w", err)
	}

	if err := ctx.Webhook.SetSecret([]byte(webhook.Secret)); err != nil {
		return nil, fmt.Errorf("failed to store webhook signing secret: %w", err)
	}

	return WebhookMetadata{LDWebhookID: webhook.ID}, nil
}

// Cleanup deletes the webhook from LaunchDarkly when the trigger is removed.
func (h *LaunchDarklyWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	if metadata.LDWebhookID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create LaunchDarkly client: %w", err)
	}

	if err := client.DeleteWebhook(metadata.LDWebhookID); err != nil {
		// If the webhook is already gone in LaunchDarkly, treat as success.
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("failed to delete webhook from LaunchDarkly: %w", err)
	}

	return nil
}
