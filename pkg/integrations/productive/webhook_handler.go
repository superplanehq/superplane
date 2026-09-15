package productive

import (
	"errors"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration scopes a Productive.io webhook to one project.
// Productive.io webhooks are managed per project, so every onTask node
// watching the same project shares one webhook.
type WebhookConfiguration struct {
	ProjectID string `json:"projectId" mapstructure:"projectId"`
}

// WebhookMetadata is stored on the webhook record after it is created, so it
// can be deleted from Productive.io again on cleanup.
type WebhookMetadata struct {
	ID string `json:"id" mapstructure:"id"`
}

// ProductiveWebhookHandler creates and tears down the Productive.io webhook
// the onTask trigger subscribes to.
type ProductiveWebhookHandler struct{}

func (h *ProductiveWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *ProductiveWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.ProjectID == configB.ProjectID, nil
}

func (h *ProductiveWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode webhook config: %v", err)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	webhook, err := client.CreateWebhook(config.ProjectID, ctx.Webhook.GetURL(), string(secret))
	if err != nil {
		if errors.Is(err, ErrWebhooksLimitExceeded) {
			return nil, fmt.Errorf("Productive.io does not offer webhooks on this plan, so the On Task trigger cannot be set up: %w", err)
		}

		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	return &WebhookMetadata{ID: webhook.ID}, nil
}

func (h *ProductiveWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %v", err)
	}

	// If the webhook was never created (Setup failed), there's nothing to clean up.
	if metadata.ID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	if err := client.DeleteWebhook(metadata.ID); err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}
