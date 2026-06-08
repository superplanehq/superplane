package jira

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	CloudID string `json:"cloudId" mapstructure:"cloudId"`
}

type WebhookMetadata struct {
	WebhookID int64 `json:"webhookId" mapstructure:"webhookId"`
}

type JiraWebhookHandler struct{}

func (h *JiraWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	return configA.CloudID == configB.CloudID, nil
}

func (h *JiraWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *JiraWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	webhook, err := client.CreateWebhook(CreateWebhookRequest{
		URL: ctx.Webhook.GetURL(),
		Webhooks: []WebhookRegistration{
			{
				Events: []string{
					jiraWebhookEventCreated,
					jiraWebhookEventUpdated,
					jiraWebhookEventDeleted,
				},
				JQLFilter: "project IS NOT EMPTY",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira webhook: %w", err)
	}

	if len(webhook.WebhookRegistrationResult) == 0 {
		return nil, fmt.Errorf("Jira webhook response did not include a registration result")
	}

	result := webhook.WebhookRegistrationResult[0]
	if result.CreatedWebhookID == 0 {
		return nil, fmt.Errorf("Jira webhook was not created: %v", result.Errors)
	}

	return WebhookMetadata{WebhookID: result.CreatedWebhookID}, nil
}

func (h *JiraWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode Jira webhook metadata: %w", err)
	}

	if metadata.WebhookID == 0 {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %w", err)
	}

	if err := client.DeleteWebhook(metadata.WebhookID); err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusNotFound {
			return nil
		}

		return fmt.Errorf("failed to delete Jira webhook: %w", err)
	}

	return nil
}
