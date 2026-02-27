package honeycomb

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	DatasetSlug string   `json:"datasetSlug" mapstructure:"datasetSlug"`
	TriggerIDs  []string `json:"triggerIds" mapstructure:"triggerIds"`
}

type WebhookMetadata struct {
	RecipientID string `json:"recipientId" mapstructure:"recipientId"`
}

type HoneycombWebhookHandler struct{}

// Share webhook if dataset matches - Merge will union the trigger IDs
func (h *HoneycombWebhookHandler) CompareConfig(a, b any) (bool, error) {
	ca := WebhookConfiguration{}
	cb := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &ca); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &cb); err != nil {
		return false, err
	}

	ca.DatasetSlug = strings.TrimSpace(ca.DatasetSlug)
	cb.DatasetSlug = strings.TrimSpace(cb.DatasetSlug)

	if ca.DatasetSlug == "" || cb.DatasetSlug == "" {
		return false, nil
	}

	return ca.DatasetSlug == cb.DatasetSlug, nil
}

func (h *HoneycombWebhookHandler) Merge(current, requested any) (any, bool, error) {
	cc := WebhookConfiguration{}
	rc := WebhookConfiguration{}

	if err := mapstructure.Decode(current, &cc); err != nil {
		return current, false, err
	}
	if err := mapstructure.Decode(requested, &rc); err != nil {
		return current, false, err
	}

	changed := false

	cc.DatasetSlug = strings.TrimSpace(cc.DatasetSlug)
	rc.DatasetSlug = strings.TrimSpace(rc.DatasetSlug)

	if cc.DatasetSlug == "" && rc.DatasetSlug != "" {
		cc.DatasetSlug = rc.DatasetSlug
		changed = true
	}

	for _, tid := range rc.TriggerIDs {
		tid = strings.TrimSpace(tid)
		if tid == "" {
			continue
		}
		if !slices.Contains(cc.TriggerIDs, tid) {
			cc.TriggerIDs = append(cc.TriggerIDs, tid)
			changed = true
		}
	}

	return cc, changed, nil
}

func (h *HoneycombWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	cfg := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &cfg); err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %w", err)
	}
	cfg.DatasetSlug = strings.TrimSpace(cfg.DatasetSlug)
	if cfg.DatasetSlug == "" {
		return nil, fmt.Errorf("datasetSlug is required for webhook")
	}

	secretBytes, err := ctx.Webhook.GetSecret()
	if err != nil || len(secretBytes) == 0 || strings.TrimSpace(string(secretBytes)) == "" {
		token, genErr := generateTokenHex(24)
		if genErr != nil {
			return nil, fmt.Errorf("failed to generate webhook secret: %w", genErr)
		}
		if err := ctx.Webhook.SetSecret([]byte(token)); err != nil {
			return nil, fmt.Errorf("failed to set webhook secret: %w", err)
		}
		secretBytes = []byte(token)
	}
	secret := string(secretBytes)

	webhookURL := strings.TrimSpace(ctx.Webhook.GetURL())
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is empty")
	}

	var recipientID string
	existingMeta := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &existingMeta); err == nil && existingMeta.RecipientID != "" {
		recipientID = existingMeta.RecipientID
	} else {
		recipient, err := client.CreateWebhookRecipient(webhookURL, secret)
		if err != nil {
			return nil, err
		}
		recipientID = recipient.ID
	}

	for _, tid := range cfg.TriggerIDs {
		tid = strings.TrimSpace(tid)
		if tid == "" {
			continue
		}
		if err := client.EnsureRecipientOnTrigger(cfg.DatasetSlug, tid, recipientID); err != nil {
			return nil, fmt.Errorf("failed to attach recipient to trigger %s: %w", tid, err)
		}
	}

	return WebhookMetadata{RecipientID: recipientID}, nil
}
func (h *HoneycombWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	meta := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &meta); err != nil {
		return nil
	}
	if meta.RecipientID == "" {
		return nil
	}

	cfg := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &cfg); err != nil {
		return nil
	}
	return client.DeleteRecipient(meta.RecipientID, cfg.DatasetSlug)
}
