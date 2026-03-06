package newrelic

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookProvisioningMetadata stores the IDs of the destination and channel
// created in New Relic so they can be cleaned up later.
type WebhookProvisioningMetadata struct {
	DestinationID string `json:"destinationId" mapstructure:"destinationId"`
	ChannelID     string `json:"channelId" mapstructure:"channelId"`
}

// defaultPayloadTemplate is a New Relic Handlebars template that maps
// issue fields to the NewRelicIssuePayload struct expected by HandleWebhook.
const defaultPayloadTemplate = `{
  "issueId": {{json issueId}},
  "issueUrl": {{json issueUrl}},
  "title": {{#if title.[0]}}{{json title.[0]}}{{else}}null{{/if}},
  "priority": {{json priority}},
  "state": {{json state}},
  "policyName": {{#if policyName.[0]}}{{json policyName.[0]}}{{else}}null{{/if}},
  "conditionName": {{#if conditionName.[0]}}{{json conditionName.[0]}}{{else}}null{{/if}},
  "accountId": {{#if accumulations.tag.account.[0]}}{{json accumulations.tag.account.[0]}}{{else}}null{{/if}},
  "createdAt": {{json createdAt}},
  "updatedAt": {{json updatedAt}},
  "sources": {{json sources}}
}`

type NewRelicWebhookHandler struct{}

func (h *NewRelicWebhookHandler) CompareConfig(a any, b any) (bool, error) {
	return true, nil
}

// Setup creates a webhook destination and notification channel in the user's
// New Relic account via NerdGraph, so alerts are forwarded automatically.
func (h *NewRelicWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create New Relic client: %w", err)
	}

	secret := uuid.New().String()
	if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
		return nil, fmt.Errorf("failed to persist webhook secret: %w", err)
	}

	destinationID, err := client.CreateNotificationDestination(context.Background(), ctx.Webhook.GetURL(), secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook destination: %w", err)
	}

	channelID, err := client.CreateNotificationChannel(context.Background(), destinationID, defaultPayloadTemplate)
	if err != nil {
		// Best-effort cleanup of the destination we just created.
		_ = client.DeleteNotificationDestination(context.Background(), destinationID)
		return nil, fmt.Errorf("failed to create notification channel: %w", err)
	}

	return WebhookProvisioningMetadata{
		DestinationID: destinationID,
		ChannelID:     channelID,
	}, nil
}

// Cleanup deletes the notification channel and destination from New Relic.
func (h *NewRelicWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookProvisioningMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create New Relic client: %w", err)
	}

	if metadata.ChannelID != "" {
		if err := client.DeleteNotificationChannel(context.Background(), metadata.ChannelID); err != nil && !isNotFoundOrUnauthorized(err) {
			return fmt.Errorf("failed to delete notification channel: %w", err)
		}
	}

	if metadata.DestinationID != "" {
		if err := client.DeleteNotificationDestination(context.Background(), metadata.DestinationID); err != nil && !isNotFoundOrUnauthorized(err) {
			return fmt.Errorf("failed to delete notification destination: %w", err)
		}
	}

	return nil
}

// isNotFoundOrUnauthorized returns true if the error is an API error with
// status 401 or 404, meaning the resource is already gone or credentials
// are no longer valid. In both cases cleanup can be considered complete.
func isNotFoundOrUnauthorized(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// Merge always keeps the current config because all New Relic triggers share
// a single integration-level webhook with no trigger-specific configuration.
// CompareConfig returns true so all triggers route to the same webhook.
func (h *NewRelicWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}
