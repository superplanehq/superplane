package firehydrant

import (
	"crypto/sha256"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	// FireHydrant webhooks subscribe to all incident events;
	// filtering is done in the trigger's HandleWebhook.
}

type WebhookMetadata struct {
	WebhookID string `json:"webhookId"`
}

type FireHydrantWebhookHandler struct{}

func (h *FireHydrantWebhookHandler) CompareConfig(a, b any) (bool, error) {
	// FireHydrant webhooks are coarse (all incident events per webhook),
	// so any two webhook configs are compatible — they can share a webhook.
	return true, nil
}

func (h *FireHydrantWebhookHandler) Merge(current, requested any) (any, bool, error) {
	// Nothing to merge since all incident events are delivered to the same webhook.
	return current, false, nil
}

func (h *FireHydrantWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	webhookURL := ctx.Webhook.GetURL()
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	webhook, err := client.CreateWebhook(webhookURL, string(secret))
	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	// If FireHydrant returned a secret (when we didn't provide one),
	// update the stored secret for signature verification.
	if webhook.Secret != "" && len(secret) == 0 {
		err = ctx.Webhook.SetSecret([]byte(webhook.Secret))
		if err != nil {
			return nil, fmt.Errorf("error updating webhook secret: %v", err)
		}
	}

	return WebhookMetadata{
		WebhookID: webhook.ID,
	}, nil
}

func (h *FireHydrantWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	if metadata.WebhookID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	err = client.DeleteWebhook(metadata.WebhookID)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}

// firehydrantWebhookName creates a deterministic webhook name based on the webhook ID.
func firehydrantWebhookName(webhookID string) string {
	hash := sha256.New()
	hash.Write([]byte(webhookID))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	return fmt.Sprintf("SuperPlane-%s", suffix[:12])
}
