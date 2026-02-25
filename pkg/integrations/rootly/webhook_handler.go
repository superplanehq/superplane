package rootly

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

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

const rootlyLegacyWebhookName = "SuperPlane"

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

	for _, event := range requestedConfig.Events {
		if !slices.Contains(merged.Events, event) {
			merged.Events = append(merged.Events, event)
		}
	}

	changed := len(merged.Events) != len(currentConfig.Events)
	return merged, changed, nil
}

func (h *RootlyWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	webhookURL := ctx.Webhook.GetURL()
	endpoint, err := h.findOrCreateWebhookEndpoint(client, ctx.Webhook.GetID(), webhookURL, config.Events)
	if err != nil {
		return nil, err
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

	if metadata.EndpointID == "" {
		return nil
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

// locates a matching Rootly endpoint for this webhook and updates its event types if needed.
func (h *RootlyWebhookHandler) findOrCreateWebhookEndpoint(client *Client, webhookID, webhookURL string, requestedEvents []string) (*WebhookEndpoint, error) {
	webhookName := rootlyWebhookName(webhookID)
	endpoints, err := client.ListWebhookEndpoints()
	if err == nil {
		if endpoint := selectWebhookEndpoint(endpoints, webhookName, webhookURL); endpoint != nil {
			mergedEvents := mergeEventTypes(endpoint.Events, requestedEvents)
			if slices.Equal(normalizeEventTypes(endpoint.Events), normalizeEventTypes(mergedEvents)) {
				return &WebhookEndpoint{
					ID:     endpoint.ID,
					Name:   endpoint.Name,
					URL:    endpoint.URL,
					Secret: "",
					Events: endpoint.Events,
				}, nil
			}

			updated, updateErr := client.UpdateWebhookEndpoint(endpoint.ID, webhookName, endpoint.URL, mergedEvents, true)
			if updateErr != nil {
				return nil, fmt.Errorf("error updating webhook endpoint: %v", updateErr)
			}
			return updated, nil
		}
	}

	endpoint, err := client.CreateWebhookEndpoint(webhookName, webhookURL, requestedEvents)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook endpoint: %v", err)
	}

	return endpoint, nil
}

// prefers the deterministic name and falls back to matching webhook URLs.
func selectWebhookEndpoint(endpoints []WebhookEndpoint, webhookName, webhookURL string) *WebhookEndpoint {
	for _, endpoint := range endpoints {
		if endpoint.Name == webhookName {
			return &endpoint
		}
	}

	var urlMatches []WebhookEndpoint
	for _, endpoint := range endpoints {
		if endpoint.URL == webhookURL {
			urlMatches = append(urlMatches, endpoint)
		}
	}

	if len(urlMatches) == 0 {
		return nil
	}

	for _, endpoint := range urlMatches {
		if endpoint.Name == rootlyLegacyWebhookName {
			return &endpoint
		}
	}

	return &urlMatches[0]
}

// unions event types and normalizes ordering for stable comparisons.
func mergeEventTypes(current, requested []string) []string {
	merged := append([]string{}, current...)
	for _, event := range requested {
		if !slices.Contains(merged, event) {
			merged = append(merged, event)
		}
	}

	return normalizeEventTypes(merged)
}

// deduplicates and sorts event types for stable comparisons/updates.
func normalizeEventTypes(events []string) []string {
	normalized := make([]string, 0, len(events))
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		if !slices.Contains(normalized, event) {
			normalized = append(normalized, event)
		}
	}

	slices.Sort(normalized)
	return normalized
}

// creates a deterministic Rootly webhook name,
func rootlyWebhookName(webhookID string) string {
	hash := sha256.New()
	hash.Write([]byte(webhookID))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	return fmt.Sprintf("SuperPlane-%s", suffix[:12])
}
