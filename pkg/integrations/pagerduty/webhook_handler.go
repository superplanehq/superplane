package pagerduty

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {

	//
	// Specific event types, e.g., ["incident.resolved", "incident.triggered"]
	//
	Events []string `json:"events"`

	//
	// Filter for webhook.
	//
	Filter WebhookFilter `json:"filter"`
}

type WebhookFilter struct {
	//
	// Type of filter for event subscription:
	// - account_reference: webhook is created on account level
	// - team_reference: events will be sent only for events related to the specified team
	// - service_reference: events will be sent only for events related ot the specified service.
	//
	Type string `json:"type"`

	//
	// If team_reference is used, this must be the ID of a team.
	// If service_reference is used, this must be the ID of a service.
	//
	ID string `json:"id"`
}

type WebhookMetadata struct {
	SubscriptionID string `json:"subscriptionId"`
}

type PagerDutyWebhookHandler struct{}

func (h *PagerDutyWebhookHandler) CompareConfig(a, b any) (bool, error) {
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

	//
	// The event subscription filter on the webhook must match exactly
	//
	if configA.Filter.Type != configB.Filter.Type || configA.Filter.ID != configB.Filter.ID {
		return false, nil
	}

	// Check if A contains all events from B (A is superset of B)
	// This allows webhook sharing when existing webhook has more events than needed
	for _, eventB := range configB.Events {
		if !slices.Contains(configA.Events, eventB) {
			return false, nil
		}
	}

	return true, nil
}

func (h *PagerDutyWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	//
	// Create webhook subscription.
	// NOTE: PagerDuty returns the secret used for signing webhooks
	// on the subscription response, so we need to update the webhook secret on our end.
	//
	subscription, err := client.CreateWebhookSubscription(ctx.Webhook.GetURL(), configuration.Events, configuration.Filter)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook subscription: %v", err)
	}

	err = ctx.Webhook.SetSecret([]byte(subscription.DeliveryMethod.Secret))
	if err != nil {
		return nil, fmt.Errorf("error updating webhook secret: %v", err)
	}

	return WebhookMetadata{
		SubscriptionID: subscription.ID,
	}, nil
}

func (h *PagerDutyWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	err = client.DeleteWebhookSubscription(metadata.SubscriptionID)
	if err != nil {
		return fmt.Errorf("error deleting webhook subscription: %v", err)
	}

	return nil
}
