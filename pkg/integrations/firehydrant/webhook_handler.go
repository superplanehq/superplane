package firehydrant

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

type FireHydrantWebhookHandler struct{}

func (h *FireHydrantWebhookHandler) CompareConfig(a, b any) (bool, error) {
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

func (h *FireHydrantWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *FireHydrantWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	// FireHydrant webhooks can be configured to listen to multiple events
	// We create a single webhook endpoint that receives all configured events
	endpoint, err := client.CreateWebhookEndpoint(ctx.Webhook.GetURL())
	if err != nil {
		return nil, fmt.Errorf("error creating webhook endpoint: %v", err)
	}

	err = ctx.Webhook.SetSecret([]byte(endpoint.Secret))
	if err != nil {
		return nil, fmt.Errorf("error updating webhook secret: %v", err)
	}

	return WebhookMetadata{
		EndpointID: endpoint.ID,
	}, nil
}

func (h *FireHydrantWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
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
