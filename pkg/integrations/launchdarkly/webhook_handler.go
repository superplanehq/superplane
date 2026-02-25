package launchdarkly

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration is the config stored with the webhook.
type WebhookConfiguration struct {
	Events []string `json:"events"`
}

// WebhookMetadata is stored after Setup. It holds the LaunchDarkly webhook ID
// so we can delete it when the trigger is removed.
type WebhookMetadata struct {
	LDWebhookID string `json:"ldWebhookId"`
}

type LaunchDarklyWebhookHandler struct{}

func (h *LaunchDarklyWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	eventsMatch := subsetOrEqual(configA.Events, configB.Events) || subsetOrEqual(configB.Events, configA.Events)
	return eventsMatch, nil
}

// subsetOrEqual returns true when a is a subset of or equal to b.
func subsetOrEqual(a, b []string) bool {
	for _, x := range a {
		if !slices.Contains(b, x) {
			return false
		}
	}
	return true
}

func (h *LaunchDarklyWebhookHandler) Merge(current, requested any) (any, bool, error) {
	cur := WebhookConfiguration{}
	req := WebhookConfiguration{}

	if err := mapstructure.Decode(current, &cur); err != nil {
		return nil, false, err
	}
	if err := mapstructure.Decode(requested, &req); err != nil {
		return nil, false, err
	}

	changed := false

	if subsetOrEqual(cur.Events, req.Events) && !slices.Equal(cur.Events, req.Events) {
		cur.Events = req.Events
		changed = true
	}

	if changed {
		return cur, true, nil
	}
	return current, false, nil
}

// Setup creates a signed webhook in LaunchDarkly via the API using the integration's API key.
// LaunchDarkly auto-generates the signing secret, which we store encrypted for later verification.
func (h *LaunchDarklyWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create LaunchDarkly client: %w", err)
	}

	webhook, err := client.CreateWebhook(CreateWebhookRequest{
		URL:  ctx.Webhook.GetURL(),
		Sign: true,
		On:   true,
		Name: "SuperPlane",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook in LaunchDarkly: %w", err)
	}

	if err := ctx.Webhook.SetSecret([]byte(webhook.Secret)); err != nil {
		return nil, fmt.Errorf("failed to store webhook signing secret: %w", err)
	}

	return WebhookMetadata{LDWebhookID: webhook.ID}, nil
}

// Cleanup deletes the webhook from LaunchDarkly when the trigger is removed.
func (h *LaunchDarklyWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	if metadata.LDWebhookID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create LaunchDarkly client: %w", err)
	}

	if err := client.DeleteWebhook(metadata.LDWebhookID); err != nil {
		// If the webhook is already gone in LaunchDarkly, treat as success.
		if strings.Contains(err.Error(), "404") {
			return nil
		}
		return fmt.Errorf("failed to delete webhook from LaunchDarkly: %w", err)
	}

	return nil
}
