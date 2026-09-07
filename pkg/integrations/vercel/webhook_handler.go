package vercel

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookHandler struct{}

type WebhookMetadata struct {
	WebhookID string `json:"webhookId" mapstructure:"webhookId"`
}

// CompareConfig always matches: all triggers share one account-level webhook,
// and per-trigger filtering happens when events are handled.
func (h *WebhookHandler) CompareConfig(a, b any) (bool, error) {
	return true, nil
}

// Merge keeps the current configuration. The subscription is fixed to all
// supported deployment events, so no update is ever needed.
func (h *WebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *WebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	webhookURL := ctx.Webhook.GetURL()
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	created, err := client.CreateWebhook(createWebhookRequest{
		URL:    webhookURL,
		Events: allowedDeploymentEventTypes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vercel webhook: %w", err)
	}

	secret := created.signingSecret()
	if secret == "" {
		return nil, fmt.Errorf("Vercel webhook signing secret is missing")
	}

	if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
		return nil, fmt.Errorf("failed to store webhook secret: %w", err)
	}

	return WebhookMetadata{WebhookID: created.ID}, nil
}

func (h *WebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	if metadata.WebhookID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	err = client.DeleteWebhook(metadata.WebhookID)
	if err == nil {
		return nil
	}

	apiErr, ok := err.(*APIError)
	if ok && apiErr.StatusCode == http.StatusNotFound {
		return nil
	}

	return err
}
