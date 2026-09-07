package productive

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	ProjectID string   `json:"projectId" mapstructure:"projectId"`
	Events    []string `json:"events" mapstructure:"events"`
}

type WebhookMetadata struct {
	ID string `json:"id" mapstructure:"id"`
}

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

	if configA.ProjectID != configB.ProjectID {
		return false, nil
	}

	// A webhook already covering all of configB's events can be shared, even
	// if it also carries events configB does not need.
	for _, event := range configB.Events {
		if !slices.Contains(configA.Events, event) {
			return false, nil
		}
	}

	return true, nil
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

	webhook, err := client.CreateWebhook(
		ctx.Webhook.GetURL(),
		string(secret),
		config.Events,
		config.ProjectID,
	)

	if err != nil {
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
