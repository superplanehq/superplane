package linear

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	TeamID       string `json:"teamId" mapstructure:"teamId"`
	ResourceType string `json:"resourceType" mapstructure:"resourceType"`
}

type WebhookMetadata struct {
	ID string `json:"id" mapstructure:"id"`
}

type LinearWebhookHandler struct{}

func (h *LinearWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *LinearWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	if configA.TeamID != configB.TeamID {
		return false, nil
	}

	return configA.ResourceType == configB.ResourceType, nil
}

func (h *LinearWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode webhook config: %v", err)
	}

	//
	// Linear generates a signing secret when one is not supplied, but it accepts
	// ours, so we keep SuperPlane's webhook secret as the single source of truth.
	//
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	webhook, err := client.CreateWebhook(
		ctx.Webhook.GetURL(),
		string(secret),
		"SuperPlane",
		config.TeamID,
		[]string{config.ResourceType},
	)

	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	return &WebhookMetadata{ID: webhook.ID}, nil
}

func (h *LinearWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
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
