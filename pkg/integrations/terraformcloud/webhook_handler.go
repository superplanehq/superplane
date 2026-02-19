package terraformcloud

import (
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	WorkspaceID string   `json:"workspaceId"`
	Triggers    []string `json:"triggers"`
}

type WebhookMetadata struct {
	NotificationConfigurationID string `json:"notificationConfigurationId"`
	Name                        string `json:"name"`
}

var (
	defaultTriggers = []string{"run:completed", "run:errored"}
	allowedTriggers = map[string]struct{}{
		"run:completed":      {},
		"run:errored":        {},
		"run:needs_attention": {},
	}
)

type TerraformCloudWebhookHandler struct{}

func (h *TerraformCloudWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return nil, fmt.Errorf("error decoding configuration: %v", err)
	}

	normalizedTriggers, err := normalizeTriggers(configuration.Triggers)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook triggers: %w", err)
	}

	hash := sha256.New()
	hash.Write([]byte(ctx.Webhook.GetID()))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	name := fmt.Sprintf("superplane-webhook-%s", suffix[:16])

	webhookSecret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	existing, err := client.ListNotificationConfigurations(configuration.WorkspaceID)
	if err == nil {
		for _, notif := range existing {
			if notif.Attributes.Name == name {
				return WebhookMetadata{
					NotificationConfigurationID: notif.ID,
					Name:                        notif.Attributes.Name,
				}, nil
			}
		}
	}

	notif, err := client.CreateNotificationConfiguration(
		configuration.WorkspaceID,
		name,
		ctx.Webhook.GetURL(),
		string(webhookSecret),
		normalizedTriggers,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating notification configuration: %v", err)
	}

	return WebhookMetadata{
		NotificationConfigurationID: notif.ID,
		Name:                        notif.Attributes.Name,
	}, nil
}

func (h *TerraformCloudWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	if configA.WorkspaceID != configB.WorkspaceID {
		return false, nil
	}

	normalizedA, err := normalizeTriggers(configA.Triggers)
	if err != nil {
		return false, err
	}

	normalizedB, err := normalizeTriggers(configB.Triggers)
	if err != nil {
		return false, err
	}

	for _, triggerB := range normalizedB {
		if !slices.Contains(normalizedA, triggerB) {
			return false, nil
		}
	}

	return true, nil
}

func (h *TerraformCloudWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(current, &currentConfig); err != nil {
		return nil, false, err
	}

	requestedConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(requested, &requestedConfig); err != nil {
		return nil, false, err
	}

	normalizedRequestedTriggers, err := normalizeTriggers(requestedConfig.Triggers)
	if err != nil {
		return nil, false, err
	}

	mergedConfig := WebhookConfiguration{
		WorkspaceID: currentConfig.WorkspaceID,
		Triggers:    normalizedRequestedTriggers,
	}

	return mergedConfig, true, nil
}

func (h *TerraformCloudWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	return client.DeleteNotificationConfiguration(metadata.NotificationConfigurationID)
}

func normalizeTriggers(triggers []string) ([]string, error) {
	if len(triggers) == 0 {
		return defaultTriggers, nil
	}

	unique := make([]string, 0, len(triggers))
	seen := map[string]struct{}{}

	for _, trigger := range triggers {
		if _, ok := allowedTriggers[trigger]; !ok {
			return nil, fmt.Errorf("unsupported Terraform Cloud trigger: %s", trigger)
		}

		if _, exists := seen[trigger]; exists {
			continue
		}

		seen[trigger] = struct{}{}
		unique = append(unique, trigger)
	}

	return unique, nil
}
