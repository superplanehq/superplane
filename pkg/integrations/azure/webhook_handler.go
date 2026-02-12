package azure

import (
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// AzureWebhookConfiguration defines how a webhook should be configured for Azure triggers.
type AzureWebhookConfiguration struct {
	EventTypes    []string `json:"eventTypes" mapstructure:"eventTypes"`
	ResourceType  string   `json:"resourceType" mapstructure:"resourceType"`
	ResourceGroup string   `json:"resourceGroup,omitempty" mapstructure:"resourceGroup"`
}

// AzureWebhookHandler manages webhook lifecycle for Azure integration triggers.
// Event Grid subscription setup is currently manual, so setup/cleanup are no-ops.
type AzureWebhookHandler struct{}

func (h *AzureWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	ctx.Logger.Infof("Azure webhook ready at %s (manual Event Grid setup required)", ctx.Webhook.GetURL())
	return map[string]any{"mode": "manual", "url": ctx.Webhook.GetURL()}, nil
}

func (h *AzureWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	ctx.Logger.Info("Azure webhook cleanup completed (no external resources to remove)")
	return nil
}

func (h *AzureWebhookHandler) CompareConfig(a, b any) (bool, error) {
	left, err := decodeAzureWebhookConfiguration(a)
	if err != nil {
		return false, err
	}

	right, err := decodeAzureWebhookConfiguration(b)
	if err != nil {
		return false, err
	}

	slices.Sort(left.EventTypes)
	slices.Sort(right.EventTypes)

	if left.ResourceType != right.ResourceType {
		return false, nil
	}

	if left.ResourceGroup != right.ResourceGroup {
		return false, nil
	}

	if len(left.EventTypes) != len(right.EventTypes) {
		return false, nil
	}

	for i := range left.EventTypes {
		if left.EventTypes[i] != right.EventTypes[i] {
			return false, nil
		}
	}

	return true, nil
}

func (h *AzureWebhookHandler) Merge(current, requested any) (merged any, changed bool, err error) {
	currentConfig, err := decodeAzureWebhookConfiguration(current)
	if err != nil {
		return nil, false, err
	}

	requestedConfig, err := decodeAzureWebhookConfiguration(requested)
	if err != nil {
		return nil, false, err
	}

	// Keep webhook semantics deterministic: if configs differ, prefer requested.
	equal, err := h.CompareConfig(currentConfig, requestedConfig)
	if err != nil {
		return nil, false, err
	}

	if equal {
		return currentConfig, false, nil
	}

	return requestedConfig, true, nil
}

func decodeAzureWebhookConfiguration(raw any) (AzureWebhookConfiguration, error) {
	config := AzureWebhookConfiguration{}
	if err := mapstructure.Decode(raw, &config); err != nil {
		return AzureWebhookConfiguration{}, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	if config.ResourceType == "" {
		return AzureWebhookConfiguration{}, fmt.Errorf("resourceType is required")
	}

	if len(config.EventTypes) == 0 {
		return AzureWebhookConfiguration{}, fmt.Errorf("eventTypes is required")
	}

	return config, nil
}

