package cloudflare

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type CloudflareWebhookHandler struct{}

type CloudflareWebhookMetadata struct {
	AccountID            string `json:"accountId" mapstructure:"accountId"`
	DestinationID        string `json:"destinationId" mapstructure:"destinationId"`
	NotificationPolicyID string `json:"notificationPolicyId" mapstructure:"notificationPolicyId"`
}

func (h *CloudflareWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA, err := decodeHealthAlertWebhookConfig(a)
	if err != nil {
		return false, err
	}

	configB, err := decodeHealthAlertWebhookConfig(b)
	if err != nil {
		return false, err
	}

	return configA.Pool == configB.Pool &&
		sameStringSet(configA.NewHealth, configB.NewHealth) &&
		sameStringSet(configA.EventSource, configB.EventSource), nil
}

func (h *CloudflareWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig, err := decodeHealthAlertWebhookConfig(current)
	if err != nil {
		return current, false, err
	}

	requestedConfig, err := decodeHealthAlertWebhookConfig(requested)
	if err != nil {
		return current, false, err
	}

	if currentConfig.Pool != requestedConfig.Pool {
		return currentConfig, false, nil
	}

	merged := currentConfig
	merged.NewHealth = mergeStringSets(currentConfig.NewHealth, requestedConfig.NewHealth)
	merged.EventSource = mergeStringSets(currentConfig.EventSource, requestedConfig.EventSource)

	changed := !sameStringSet(currentConfig.NewHealth, merged.NewHealth) ||
		!sameStringSet(currentConfig.EventSource, merged.EventSource)

	return merged, changed, nil
}

func (h *CloudflareWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	accountID, err := accountIDForIntegration(ctx.Integration)
	if err != nil {
		return nil, err
	}

	config, err := decodeHealthAlertWebhookConfig(ctx.Webhook.GetConfiguration())
	if err != nil {
		return nil, err
	}

	webhookURL := strings.TrimSpace(ctx.Webhook.GetURL())
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is empty")
	}

	secretBytes, err := ctx.Webhook.GetSecret()
	if err != nil || len(secretBytes) == 0 || strings.TrimSpace(string(secretBytes)) == "" {
		secret := uuid.NewString()
		if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
			return nil, fmt.Errorf("failed to set webhook secret: %w", err)
		}
		secretBytes = []byte(secret)
	}

	destination, err := client.CreateAlertingWebhookDestination(
		accountID,
		CreateAlertingWebhookDestinationRequest{
			Name:   cloudflareWebhookName(config),
			URL:    webhookURL,
			Secret: string(secretBytes),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare alerting webhook destination: %w", err)
	}

	policy, err := client.CreateNotificationPolicy(accountID, CreateNotificationPolicyRequest{
		AlertType:   "load_balancing_health_alert",
		Enabled:     true,
		Name:        cloudflareWebhookName(config),
		Description: "Created by SuperPlane for Cloudflare load balancing health alerts.",
		Mechanisms: NotificationPolicyMechanisms{
			Webhooks: []NotificationMechanism{{ID: destination.ID}},
		},
		Filters: NotificationPolicyFilters{
			PoolID:      oneOrNil(config.Pool),
			NewHealth:   config.NewHealth,
			EventSource: config.EventSource,
		},
	})
	if err != nil {
		_ = client.DeleteAlertingWebhookDestination(accountID, destination.ID)
		return nil, fmt.Errorf("failed to create Cloudflare notification policy: %w", err)
	}

	return CloudflareWebhookMetadata{
		AccountID:            accountID,
		DestinationID:        destination.ID,
		NotificationPolicyID: policy.ID,
	}, nil
}

func (h *CloudflareWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	metadata := CloudflareWebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return nil
	}

	accountID := strings.TrimSpace(metadata.AccountID)
	if accountID == "" {
		accountID, err = accountIDForIntegration(ctx.Integration)
		if err != nil {
			return nil
		}
	}

	if metadata.NotificationPolicyID != "" {
		if err := client.DeleteNotificationPolicy(accountID, metadata.NotificationPolicyID); err != nil {
			return fmt.Errorf("failed to delete Cloudflare notification policy: %w", err)
		}
	}

	if metadata.DestinationID != "" {
		if err := client.DeleteAlertingWebhookDestination(accountID, metadata.DestinationID); err != nil {
			return fmt.Errorf("failed to delete Cloudflare alerting webhook destination: %w", err)
		}
	}

	return nil
}

func decodeHealthAlertWebhookConfig(value any) (OnLoadBalancingHealthAlertSpec, error) {
	config := OnLoadBalancingHealthAlertSpec{}
	if err := mapstructure.Decode(value, &config); err != nil {
		return config, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	return normalizeHealthAlertSpec(config)
}

func mergeStringSets(a, b []string) []string {
	result := append([]string{}, a...)
	for _, value := range b {
		if !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	return result
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for _, value := range a {
		if !slices.Contains(b, value) {
			return false
		}
	}

	return true
}

func cloudflareWebhookName(config OnLoadBalancingHealthAlertSpec) string {
	if config.Pool == "" {
		return "SuperPlane Load Balancing Health Alert"
	}

	return fmt.Sprintf("SuperPlane Load Balancing Health Alert - %s", config.Pool)
}

func oneOrNil(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return []string{value}
}
