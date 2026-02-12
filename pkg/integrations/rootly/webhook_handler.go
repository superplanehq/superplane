package rootly

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	Events []string `json:"events"`
}

type WebhookMetadata struct {
	EndpointID string `json:"endpointId"`
}

type RootlyWebhookHandler struct{}

func (h *RootlyWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	err := mapstructure.Decode(a, &configA)
	if err != nil {
		return false, err
	}

	err = mapstructure.Decode(b, &configB)
	if err != nil {
		return false, err
	}

	// Check if A contains all events from B (A is superset of B)
	for _, eventB := range configB.Events {
		if !slices.Contains(configA.Events, eventB) {
			return false, nil
		}
	}

	return true, nil
}

func (h *RootlyWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig := WebhookConfiguration{}
	requestedConfig := WebhookConfiguration{}

	if err := mapstructure.Decode(current, &currentConfig); err != nil {
		return nil, false, err
	}

	if err := mapstructure.Decode(requested, &requestedConfig); err != nil {
		return nil, false, err
	}

	merged := WebhookConfiguration{
		Events: append([]string{}, currentConfig.Events...),
	}

	changed := false
	for _, eventType := range requestedConfig.Events {
		if !slices.Contains(merged.Events, eventType) {
			merged.Events = append(merged.Events, eventType)
			changed = true
		}
	}

	if !changed {
		return current, false, nil
	}

	return merged, changed, nil
}

func (h *RootlyWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	webhookName := fmt.Sprintf("SuperPlane %s", ctx.Webhook.GetID())

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	metadata := WebhookMetadata{}
	if rawMetadata := ctx.Webhook.GetMetadata(); rawMetadata != nil {
		if err := mapstructure.Decode(rawMetadata, &metadata); err != nil {
			return nil, fmt.Errorf("error decoding webhook metadata: %v", err)
		}
	}

	var endpoint *WebhookEndpoint
	if metadata.EndpointID != "" {
		endpoint, err = client.UpdateWebhookEndpoint(metadata.EndpointID, webhookName, config.Events)
		if err != nil {
			return nil, fmt.Errorf("error updating webhook endpoint: %v", err)
		}
	} else {
		endpoint, err = client.CreateWebhookEndpoint(webhookName, ctx.Webhook.GetURL(), config.Events)
		if err != nil {
			return nil, fmt.Errorf("error creating webhook endpoint: %v", err)
		}
	}

	if endpoint.Secret != "" {
		err = ctx.Webhook.SetSecret([]byte(endpoint.Secret))
		if err != nil {
			return nil, fmt.Errorf("error updating webhook secret: %v", err)
		}
	}

	return WebhookMetadata{
		EndpointID: endpoint.ID,
	}, nil
}

func (h *RootlyWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	err = client.DeleteWebhookEndpoint(metadata.EndpointID)
	if err != nil {
		return fmt.Errorf("error deleting webhook endpoint: %v", err)
	}

	return nil
}
