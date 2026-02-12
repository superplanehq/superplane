package jira

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type JiraWebhookHandler struct{}

func (h *JiraWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *JiraWebhookHandler) CompareConfig(a, b any) (bool, error) {
	var configA, configB WebhookConfiguration

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, fmt.Errorf("failed to decode config a: %w", err)
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, fmt.Errorf("failed to decode config b: %w", err)
	}

	return configA.EventType == configB.EventType && configA.Project == configB.Project, nil
}

func (h *JiraWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	var config WebhookConfiguration
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	jqlFilter := fmt.Sprintf("project = %q", config.Project)
	events := []string{config.EventType}
	webhookURL := ctx.Webhook.GetURL()

	response, err := client.RegisterWebhook(webhookURL, jqlFilter, events)
	if err != nil {
		return nil, fmt.Errorf("error registering webhook: %v", err)
	}

	if len(response.WebhookRegistrationResult) == 0 {
		return nil, fmt.Errorf("no webhook registration result returned")
	}

	result := response.WebhookRegistrationResult[0]
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("webhook registration failed: %v", result.Errors)
	}

	return WebhookMetadata{ID: result.CreatedWebhookID}, nil
}

func (h *JiraWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	var metadata WebhookMetadata
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	if metadata.ID == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	return client.DeleteWebhook([]int64{metadata.ID})
}
