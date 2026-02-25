package jfrogartifactory

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration is the configuration passed from the trigger to the webhook handler.
type WebhookConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

// WebhookMetadata stores the external webhook key after creation, used for cleanup.
type WebhookMetadata struct {
	Key string `json:"key" mapstructure:"key"`
}

// JFrogWebhookHandler implements core.WebhookHandler for JFrog Artifactory.
type JFrogWebhookHandler struct{}

func (h *JFrogWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
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

	key, err := client.CreateWebhook(ctx.Webhook.GetURL(), string(secret), config.Repository)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook in JFrog: %v", err)
	}

	return &WebhookMetadata{Key: key}, nil
}

func (h *JFrogWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %v", err)
	}

	if metadata.Key == "" {
		return nil
	}

	if err := client.DeleteWebhook(metadata.Key); err != nil {
		return fmt.Errorf("error deleting webhook from JFrog: %v", err)
	}

	return nil
}

func (h *JFrogWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.Repository == configB.Repository, nil
}

func (h *JFrogWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}
