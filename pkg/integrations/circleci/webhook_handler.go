package circleci

import (
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	ProjectSlug string   `json:"projectSlug"`
	Events      []string `json:"events"`
}

type WebhookMetadata struct {
	WebhookID string `json:"webhookId"`
	Name      string `json:"name"`
}

var (
	defaultEvents = []string{"workflow-completed"}
	allowedEvents = map[string]struct{}{
		"workflow-completed": {},
		"job-completed":      {},
	}
)

type CircleCIWebhookHandler struct{}

func (h *CircleCIWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding configuration: %v", err)
	}

	normalizedEvents, err := normalizeEvents(configuration.Events)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook events: %w", err)
	}

	// Generate deterministic webhook name based on webhook ID
	hash := sha256.New()
	hash.Write([]byte(ctx.Webhook.GetID()))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	name := fmt.Sprintf("superplane-webhook-%s", suffix[:16])

	webhookSecret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	webhook, err := upsertWebhook(client, name, ctx.Webhook.GetURL(), string(webhookSecret), configuration.ProjectSlug, normalizedEvents)
	if err != nil {
		return nil, err
	}

	return WebhookMetadata{
		WebhookID: webhook.ID,
		Name:      webhook.Name,
	}, nil
}

func upsertWebhook(client *Client, name, webhookURL, secret, projectSlug string, events []string) (*WebhookResponse, error) {
	//
	// Check if webhook with this name already exists.
	//
	webhooks, err := client.ListWebhooks(projectSlug)
	if err == nil {
		for _, webhook := range webhooks {
			if webhook.Name == name {
				// Webhook with this name already exists, return it.
				// The webhook system will call CompareConfig to verify the configuration matches.
				return &webhook, nil
			}
		}
	}

	//
	// Webhook does not exist, create it.
	//
	webhook, err := client.CreateWebhook(name, webhookURL, secret, projectSlug, events)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	return webhook, nil
}

func (h *CircleCIWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	normalizedA, err := normalizeEvents(configA.Events)
	if err != nil {
		return false, err
	}

	normalizedB, err := normalizeEvents(configB.Events)
	if err != nil {
		return false, err
	}

	if configA.ProjectSlug != configB.ProjectSlug {
		return false, nil
	}

	for _, eventB := range normalizedB {
		if !slices.Contains(normalizedA, eventB) {
			return false, nil
		}
	}

	return true, nil
}

func (h *CircleCIWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(current, &currentConfig); err != nil {
		return nil, false, err
	}

	requestedConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(requested, &requestedConfig); err != nil {
		return nil, false, err
	}

	normalizedRequestedEvents, err := normalizeEvents(requestedConfig.Events)
	if err != nil {
		return nil, false, err
	}

	mergedConfig := WebhookConfiguration{
		ProjectSlug: currentConfig.ProjectSlug,
		Events:      normalizedRequestedEvents,
	}

	return mergedConfig, true, nil
}

func (h *CircleCIWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	return client.DeleteWebhook(metadata.WebhookID)
}

func normalizeEvents(events []string) ([]string, error) {
	if len(events) == 0 {
		return defaultEvents, nil
	}

	unique := make([]string, 0, len(events))
	seen := map[string]struct{}{}

	for _, event := range events {
		if _, ok := allowedEvents[event]; !ok {
			return nil, fmt.Errorf("unsupported CircleCI event type: %s", event)
		}

		if _, exists := seen[event]; exists {
			continue
		}

		seen[event] = struct{}{}
		unique = append(unique, event)
	}

	return unique, nil
}
