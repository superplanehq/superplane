package dockerhub

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	Namespace  string `json:"namespace" mapstructure:"namespace"`
	Repository string `json:"repository" mapstructure:"repository"`
}

type WebhookMetadata struct {
	ID   string `json:"id" mapstructure:"id"`
	Name string `json:"name" mapstructure:"name"`
}

type DockerHubWebhookHandler struct{}

func (h *DockerHubWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.Namespace == configB.Namespace && configA.Repository == configB.Repository, nil
}

func (h *DockerHubWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	webhookName := fmt.Sprintf("superplane-%s", ctx.Webhook.GetID())
	webhook, err := client.CreateWebhook(config.Namespace, config.Repository, webhookName, ctx.Webhook.GetURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker Hub webhook: %w", err)
	}

	return WebhookMetadata{
		ID:   string(webhook.ID),
		Name: webhook.Name,
	}, nil
}

func (h *DockerHubWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	if metadata.ID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	return client.DeleteWebhook(config.Namespace, config.Repository, metadata.ID)
}
