package oci

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// WebhookConfiguration is stored on each OCI webhook record so the handler
// knows which ONS topic to subscribe / unsubscribe.
type WebhookConfiguration struct {
	// CompartmentID is used when creating the ONS subscription (required by OCI).
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
	// TopicID is the OCID of the ONS topic to subscribe to.
	TopicID string `json:"topicId" mapstructure:"topicId"`
}

// WebhookMetadata is persisted after a successful subscription and used during cleanup.
type WebhookMetadata struct {
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
}

type WebhookHandler struct{}

// Setup creates an HTTPS subscription on the configured OCI Notifications topic,
// pointing to the SuperPlane-generated webhook URL.
func (h *WebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	var config WebhookConfiguration
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("failed to decode OCI webhook configuration: %w", err)
	}

	if config.TopicID == "" {
		return nil, fmt.Errorf("topicId is required in webhook configuration")
	}
	if config.CompartmentID == "" {
		return nil, fmt.Errorf("compartmentId is required in webhook configuration")
	}

	endpointURL := ctx.Webhook.GetURL()
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	sub, err := client.CreateONSSubscription(config.CompartmentID, config.TopicID, endpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONS subscription: %w", err)
	}

	return WebhookMetadata{SubscriptionID: sub.ID}, nil
}

// Cleanup deletes the ONS subscription created during Setup.
func (h *WebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	var metadata WebhookMetadata
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode OCI webhook metadata: %w", err)
	}

	if metadata.SubscriptionID == "" {
		// Nothing to clean up — subscription was never created.
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	if err := client.DeleteONSSubscription(metadata.SubscriptionID); err != nil {
		return fmt.Errorf("failed to delete ONS subscription %q: %w", metadata.SubscriptionID, err)
	}

	return nil
}

// CompareConfig returns true when both configs point to the same ONS topic.
func (h *WebhookHandler) CompareConfig(a, b any) (bool, error) {
	var ca, cb WebhookConfiguration
	if err := mapstructure.Decode(a, &ca); err != nil {
		return false, err
	}
	if err := mapstructure.Decode(b, &cb); err != nil {
		return false, err
	}
	return ca.TopicID == cb.TopicID && ca.CompartmentID == cb.CompartmentID, nil
}

// Merge returns the requested config unchanged (no merging needed for OCI webhooks).
func (h *WebhookHandler) Merge(current, requested any) (any, bool, error) {
	equal, err := h.CompareConfig(current, requested)
	if err != nil {
		return nil, false, err
	}
	return requested, !equal, nil
}
