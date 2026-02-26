package firehydrant

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	Subscriptions []string `json:"subscriptions" mapstructure:"subscriptions"`
}

type WebhookMetadata struct {
	WebhookID string `json:"webhookId" mapstructure:"webhookId"`
}

type FireHydrantWebhookHandler struct{}

func (h *FireHydrantWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	eventsMatch := subscriptionsSubsetOrEqual(configA.Subscriptions, configB.Subscriptions) ||
		subscriptionsSubsetOrEqual(configB.Subscriptions, configA.Subscriptions)

	return eventsMatch, nil
}

func subscriptionsSubsetOrEqual(a, b []string) bool {
	for _, x := range a {
		if !slices.Contains(b, x) {
			return false
		}
	}

	return true
}

func (h *FireHydrantWebhookHandler) Merge(current, requested any) (any, bool, error) {
	cur := WebhookConfiguration{}
	req := WebhookConfiguration{}

	if err := mapstructure.Decode(current, &cur); err != nil {
		return nil, false, err
	}

	if err := mapstructure.Decode(requested, &req); err != nil {
		return nil, false, err
	}

	// If the current subscriptions are a strict subset of the requested ones, expand.
	if subscriptionsSubsetOrEqual(cur.Subscriptions, req.Subscriptions) &&
		!subscriptionsSubsetOrEqual(req.Subscriptions, cur.Subscriptions) {
		cur.Subscriptions = req.Subscriptions
		return cur, true, nil
	}

	return current, false, nil
}

func (h *FireHydrantWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	webhookURL := ctx.Webhook.GetURL()
	webhook, err := findOrCreateWebhook(client, webhookURL, string(secret), config.Subscriptions)
	if err != nil {
		return nil, err
	}

	return WebhookMetadata{
		WebhookID: webhook.ID,
	}, nil
}

func findOrCreateWebhook(client *Client, webhookURL, secret string, subscriptions []string) (*Webhook, error) {
	webhooks, err := client.ListWebhooks()
	if err == nil {
		if webhook := findWebhookByURL(webhooks, webhookURL); webhook != nil {
			mergedSubscriptions := mergeSubscriptions(webhook.Subscriptions, subscriptions)
			if subscriptionsEqual(webhook.Subscriptions, mergedSubscriptions) {
				return webhook, nil
			}

			updated, updateErr := client.UpdateWebhook(webhook.ID, webhookURL, mergedSubscriptions)
			if updateErr != nil {
				return nil, fmt.Errorf("error updating webhook: %v", updateErr)
			}

			return updated, nil
		}
	}

	webhook, err := client.CreateWebhook(webhookURL, secret, subscriptions)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	return webhook, nil
}

func findWebhookByURL(webhooks []Webhook, url string) *Webhook {
	for _, webhook := range webhooks {
		if webhook.URL == url {
			return &webhook
		}
	}

	return nil
}

func mergeSubscriptions(current, requested []string) []string {
	merged := append([]string{}, current...)
	for _, sub := range requested {
		if !slices.Contains(merged, sub) {
			merged = append(merged, sub)
		}
	}

	slices.Sort(merged)
	return merged
}

func subscriptionsEqual(a, b []string) bool {
	sortedA := append([]string{}, a...)
	sortedB := append([]string{}, b...)
	slices.Sort(sortedA)
	slices.Sort(sortedB)
	return slices.Equal(sortedA, sortedB)
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
