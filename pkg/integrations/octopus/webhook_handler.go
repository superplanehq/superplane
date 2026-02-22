package octopus

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

const subscriptionNamePrefix = "SuperPlane"

type OctopusWebhookHandler struct{}

type WebhookMetadata struct {
	SubscriptionID string `json:"subscriptionId" mapstructure:"subscriptionId"`
	SpaceID        string `json:"spaceId" mapstructure:"spaceId"`
}

type WebhookConfiguration struct {
	EventCategories []string `json:"eventCategories,omitempty" mapstructure:"eventCategories"`
	Projects        []string `json:"projects,omitempty" mapstructure:"projects"`
	Environments    []string `json:"environments,omitempty" mapstructure:"environments"`
}

func (h *OctopusWebhookHandler) CompareConfig(a, b any) (bool, error) {
	// All Octopus webhook configurations are compatible and will be merged
	// into a single subscription with combined event categories.
	return true, nil
}

func (h *OctopusWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig, err := decodeWebhookConfiguration(current)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode current webhook configuration: %w", err)
	}

	requestedConfig, err := decodeWebhookConfiguration(requested)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode requested webhook configuration: %w", err)
	}

	merged := WebhookConfiguration{
		EventCategories: normalizeEventCategories(
			append(currentConfig.EventCategories, requestedConfig.EventCategories...),
		),
		Projects:     mergeStringSlices(currentConfig.Projects, requestedConfig.Projects),
		Environments: mergeStringSlices(currentConfig.Environments, requestedConfig.Environments),
	}

	changed := !slices.Equal(normalizeEventCategories(currentConfig.EventCategories), merged.EventCategories) ||
		!slices.Equal(currentConfig.Projects, merged.Projects) ||
		!slices.Equal(currentConfig.Environments, merged.Environments)

	return merged, changed, nil
}

func (h *OctopusWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	spaceID, err := spaceIDForIntegration(client, ctx.Integration)
	if err != nil {
		return nil, err
	}

	webhookURL := ctx.Webhook.GetURL()
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	webhookConfig, err := decodeWebhookConfiguration(ctx.Webhook.GetConfiguration())
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	// Generate a random secret for webhook verification
	secret, err := generateWebhookSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate webhook secret: %w", err)
	}

	// Clean up any stale subscription with the same name from a previous
	// failed provisioning attempt (e.g., subscription created in Octopus
	// but saving metadata locally failed, causing a retry).
	name := fmt.Sprintf("%s-%s", subscriptionNamePrefix, ctx.Webhook.GetID())
	cleanupStaleSubscription(client, spaceID, name)

	// Create a new subscription in Octopus Deploy.
	subscription, err := client.CreateSubscription(CreateSubscriptionRequest{
		Name:    name,
		SpaceID: spaceID,
		EventNotificationSubscription: &EventNotificationSubscription{
			WebhookURI:         webhookURL,
			WebhookHeaderKey:   webhookHeaderKey,
			WebhookHeaderValue: secret,
			WebhookTimeout:     "00:00:30",
			Filter: &EventSubscriptionFilter{
				EventCategories: webhookConfig.EventCategories,
				Projects:        webhookConfig.Projects,
				Environments:    webhookConfig.Environments,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Octopus Deploy subscription: %w", err)
	}

	if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
		return nil, fmt.Errorf("failed to store webhook secret: %w", err)
	}

	return WebhookMetadata{
		SubscriptionID: subscription.ID,
		SpaceID:        spaceID,
	}, nil
}

func (h *OctopusWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata, err := decodeWebhookMetadata(ctx.Webhook.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	if metadata.SubscriptionID == "" || metadata.SpaceID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	err = client.DeleteSubscription(metadata.SpaceID, metadata.SubscriptionID)
	if err == nil {
		return nil
	}

	apiErr, ok := err.(*APIError)
	if ok && apiErr.StatusCode == 404 {
		return nil
	}

	return err
}

func decodeWebhookConfiguration(configuration any) (WebhookConfiguration, error) {
	webhookConfig := WebhookConfiguration{}
	if configuration == nil {
		return webhookConfig, nil
	}

	if err := mapstructure.Decode(configuration, &webhookConfig); err != nil {
		return WebhookConfiguration{}, err
	}

	webhookConfig.EventCategories = normalizeEventCategories(webhookConfig.EventCategories)
	return webhookConfig, nil
}

func decodeWebhookMetadata(value any) (WebhookMetadata, error) {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(value, &metadata); err != nil {
		return WebhookMetadata{}, err
	}

	return metadata, nil
}

func generateWebhookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// cleanupStaleSubscription deletes any existing Octopus subscription with the
// given name. This handles the case where a previous provisioning attempt
// created the subscription in Octopus but failed before saving the metadata
// locally, causing the provisioner to retry and hit a name conflict.
func cleanupStaleSubscription(client *Client, spaceID, name string) {
	subscriptions, err := client.ListSubscriptions(spaceID)
	if err != nil {
		return
	}

	for _, sub := range subscriptions {
		if sub.Name == name {
			_ = client.DeleteSubscription(spaceID, sub.ID)
			return
		}
	}
}

// mergeStringSlices merges two string slices with deduplication.
// An empty/nil slice means "all" (no filtering), so if either input
// is empty, the result is empty to preserve that "match all" semantics.
func mergeStringSlices(a, b []string) []string {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(a)+len(b))
	merged := make([]string, 0, len(a)+len(b))

	for _, v := range a {
		if !seen[v] {
			seen[v] = true
			merged = append(merged, v)
		}
	}

	for _, v := range b {
		if !seen[v] {
			seen[v] = true
			merged = append(merged, v)
		}
	}

	return merged
}
