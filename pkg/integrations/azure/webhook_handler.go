package azure

import (
	"context"
	"fmt"
	"net/http"
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
type AzureWebhookHandler struct {
	integration *AzureIntegration
}

func (h *AzureWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	webhookURL := ctx.Webhook.GetURL()
	ctx.Logger.Infof("Setting up Azure Event Grid subscription for webhook: %s", webhookURL)

	provider, err := h.integration.ensureProvider(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("Azure provider not available: %w", err)
	}

	// Ensure Microsoft.EventGrid resource provider is registered before
	// creating the subscription, otherwise the LRO will fail.
	ctx.Logger.Info("Ensuring Microsoft.EventGrid resource provider is registered")
	if err := provider.getClient().ensureResourceProviderRegistered(context.Background(), "Microsoft.EventGrid"); err != nil {
		return nil, fmt.Errorf("failed to register Microsoft.EventGrid resource provider: %w", err)
	}

	config, err := decodeAzureWebhookConfiguration(ctx.Webhook.GetConfiguration())
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	subName := fmt.Sprintf("superplane-%s", ctx.Webhook.GetID())
	scope := fmt.Sprintf("/subscriptions/%s", provider.GetSubscriptionID())

	// Build subject filter.
	// Note: Event Grid subjects for resource events look like:
	//   /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/virtualMachines/{name}
	// The subject ends with the resource name, so we cannot use subjectEndsWith
	// to filter by resource type. Instead, we use subjectBeginsWith when a resource
	// group is specified and rely on handler-side filtering for the resource type.
	var subjectBeginsWith string
	if config.ResourceGroup != "" {
		subjectBeginsWith = fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s",
			provider.GetSubscriptionID(), config.ResourceGroup,
		)
	}

	// Get webhook secret for delivery authentication
	secret, _ := ctx.Webhook.GetSecret()
	var deliveryAttributes []map[string]any
	if len(secret) > 0 {
		deliveryAttributes = []map[string]any{
			{
				"name": "X-Webhook-Secret",
				"type": "Static",
				"properties": map[string]any{
					"value":    string(secret),
					"isSecret": true,
				},
			},
		}
	}

	// Build advanced filters to reduce noise at the Azure level.
	// Filter by subject containing the resource type so Azure only delivers
	// events for the resource type we care about (e.g., virtualMachines).
	var advancedFilters []map[string]any
	if config.ResourceType != "" {
		advancedFilters = append(advancedFilters, map[string]any{
			"operatorType": "StringContains",
			"key":          "subject",
			"values":       []string{config.ResourceType},
		})
	}

	body := map[string]any{
		"properties": map[string]any{
			"destination": map[string]any{
				"endpointType": "WebHook",
				"properties": map[string]any{
					"endpointUrl":               webhookURL,
					"deliveryAttributeMappings": deliveryAttributes,
				},
			},
			"filter": map[string]any{
				"includedEventTypes": config.EventTypes,
				"subjectBeginsWith":  subjectBeginsWith,
				"advancedFilters":    advancedFilters,
			},
		},
	}

	url := fmt.Sprintf("%s%s/providers/Microsoft.EventGrid/eventSubscriptions/%s?api-version=%s",
		armBaseURL, scope, subName, armAPIVersionEventGrid)

	_, err = provider.getClient().putAndPoll(context.Background(), url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create Event Grid subscription: %w", err)
	}

	ctx.Logger.Infof("Event Grid subscription created: %s", subName)

	return map[string]any{
		"mode":             "automatic",
		"subscriptionName": subName,
		"scope":            scope,
		"url":              webhookURL,
	}, nil
}

func (h *AzureWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	ctx.Logger.Info("Cleaning up Azure Event Grid subscription")

	provider, err := h.integration.ensureProvider(ctx.Integration)
	if err != nil {
		ctx.Logger.Warnf("Azure provider not available; skipping Event Grid cleanup: %v", err)
		return nil
	}

	subName := fmt.Sprintf("superplane-%s", ctx.Webhook.GetID())
	scope := fmt.Sprintf("/subscriptions/%s", provider.GetSubscriptionID())

	url := fmt.Sprintf("%s%s/providers/Microsoft.EventGrid/eventSubscriptions/%s?api-version=%s",
		armBaseURL, scope, subName, armAPIVersionEventGrid)

	resp, err := provider.getClient().doRequest(context.Background(), http.MethodDelete, url, nil)
	if err != nil {
		ctx.Logger.Warnf("Failed to delete Event Grid subscription: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		ctx.Logger.Warnf("Failed to delete Event Grid subscription, status: %d", resp.StatusCode)
	} else {
		ctx.Logger.Info("Event Grid subscription deleted successfully")
	}

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
