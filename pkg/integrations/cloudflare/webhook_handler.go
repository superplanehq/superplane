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

type cloudflareWebhookKind string

const (
	cloudflareWebhookKindLoadBalancing cloudflareWebhookKind = "load_balancing_health_alert"
	cloudflareWebhookKindTunnelHealth  cloudflareWebhookKind = "tunnel_health_event"
)

type cloudflareWebhookPolicyConfig struct {
	Kind   cloudflareWebhookKind
	Load   OnLoadBalancingHealthAlertSpec
	Tunnel OnTunnelHealthSpec
}

func decodeCloudflareWebhookPolicyConfig(value any) (cloudflareWebhookPolicyConfig, error) {
	if isTunnelHealthWebhookConfiguration(value) {
		tunnel, err := decodeTunnelHealthWebhookConfig(value)
		if err != nil {
			return cloudflareWebhookPolicyConfig{}, err
		}
		return cloudflareWebhookPolicyConfig{
			Kind:   cloudflareWebhookKindTunnelHealth,
			Tunnel: tunnel,
		}, nil
	}

	load, err := decodeHealthAlertWebhookConfig(value)
	if err != nil {
		return cloudflareWebhookPolicyConfig{}, err
	}

	return cloudflareWebhookPolicyConfig{
		Kind: cloudflareWebhookKindLoadBalancing,
		Load: load,
	}, nil
}

func isTunnelHealthWebhookConfiguration(value any) bool {
	if m, ok := value.(map[string]any); ok {
		if _, ok := m["newStatus"]; ok {
			return true
		}
		if _, ok := m["tunnel"]; ok {
			return true
		}
		return false
	}

	var probe struct {
		NewStatus []string `mapstructure:"newStatus"`
		Tunnel    string   `mapstructure:"tunnel"`
	}
	if err := mapstructure.Decode(value, &probe); err != nil {
		return false
	}
	if len(probe.NewStatus) > 0 {
		return true
	}
	return strings.TrimSpace(probe.Tunnel) != ""
}

func (h *CloudflareWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA, err := decodeCloudflareWebhookPolicyConfig(a)
	if err != nil {
		return false, err
	}

	configB, err := decodeCloudflareWebhookPolicyConfig(b)
	if err != nil {
		return false, err
	}

	if configA.Kind != configB.Kind {
		return false, nil
	}

	if configA.Kind == cloudflareWebhookKindTunnelHealth {
		return configA.Tunnel.Tunnel == configB.Tunnel.Tunnel &&
			sameStringSet(configA.Tunnel.NewStatus, configB.Tunnel.NewStatus), nil
	}

	return configA.Load.Pool == configB.Load.Pool &&
		sameStringSet(configA.Load.NewHealth, configB.Load.NewHealth) &&
		sameStringSet(configA.Load.EventSource, configB.Load.EventSource), nil
}

func (h *CloudflareWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig, err := decodeCloudflareWebhookPolicyConfig(current)
	if err != nil {
		return current, false, err
	}

	requestedConfig, err := decodeCloudflareWebhookPolicyConfig(requested)
	if err != nil {
		return current, false, err
	}

	if currentConfig.Kind != requestedConfig.Kind {
		return current, false, nil
	}

	if currentConfig.Kind == cloudflareWebhookKindTunnelHealth {
		if currentConfig.Tunnel.Tunnel != requestedConfig.Tunnel.Tunnel {
			return currentConfig.Tunnel, false, nil
		}

		merged := currentConfig.Tunnel
		merged.NewStatus = mergeStringSets(currentConfig.Tunnel.NewStatus, requestedConfig.Tunnel.NewStatus)

		changed := !sameStringSet(currentConfig.Tunnel.NewStatus, merged.NewStatus)
		return merged, changed, nil
	}

	if currentConfig.Load.Pool != requestedConfig.Load.Pool {
		return currentConfig.Load, false, nil
	}

	merged := currentConfig.Load
	merged.NewHealth = mergeStringSets(currentConfig.Load.NewHealth, requestedConfig.Load.NewHealth)
	merged.EventSource = mergeStringSets(currentConfig.Load.EventSource, requestedConfig.Load.EventSource)

	changed := !sameStringSet(currentConfig.Load.NewHealth, merged.NewHealth) ||
		!sameStringSet(currentConfig.Load.EventSource, merged.EventSource)

	return merged, changed, nil
}

func (h *CloudflareWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	cfg, err := decodeCloudflareWebhookPolicyConfig(ctx.Webhook.GetConfiguration())
	if err != nil {
		return nil, err
	}

	if cfg.Kind == cloudflareWebhookKindTunnelHealth {
		return setupTunnelHealthAlertWebhook(ctx, cfg.Tunnel)
	}

	return setupLoadBalancingHealthAlertWebhook(ctx, cfg.Load)
}

func setupLoadBalancingHealthAlertWebhook(ctx core.WebhookHandlerContext, config OnLoadBalancingHealthAlertSpec) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	accountID, err := accountIDForIntegration(ctx.Integration)
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
			Name:   cloudflareLoadBalancingWebhookName(config),
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
		Name:        cloudflareLoadBalancingWebhookName(config),
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

func setupTunnelHealthAlertWebhook(ctx core.WebhookHandlerContext, config OnTunnelHealthSpec) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	accountID, err := accountIDForIntegration(ctx.Integration)
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
			Name:   cloudflareTunnelHealthWebhookName(config),
			URL:    webhookURL,
			Secret: string(secretBytes),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare alerting webhook destination: %w", err)
	}

	policy, err := client.CreateNotificationPolicy(accountID, CreateNotificationPolicyRequest{
		AlertType:   "tunnel_health_event",
		Enabled:     true,
		Name:        cloudflareTunnelHealthWebhookName(config),
		Description: "Created by SuperPlane for Cloudflare Tunnel health alerts.",
		Mechanisms: NotificationPolicyMechanisms{
			Webhooks: []NotificationMechanism{{ID: destination.ID}},
		},
		Filters: NotificationPolicyFilters{
			TunnelID:  oneOrNil(config.Tunnel),
			NewStatus: config.NewStatus,
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
		if err := client.DeleteNotificationPolicy(accountID, metadata.NotificationPolicyID); err != nil && !isCloudflareNotFound(err) {
			return fmt.Errorf("failed to delete Cloudflare notification policy: %w", err)
		}
	}

	if metadata.DestinationID != "" {
		if err := client.DeleteAlertingWebhookDestination(accountID, metadata.DestinationID); err != nil && !isCloudflareNotFound(err) {
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

func decodeTunnelHealthWebhookConfig(value any) (OnTunnelHealthSpec, error) {
	config := OnTunnelHealthSpec{}
	if err := mapstructure.Decode(value, &config); err != nil {
		return config, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	return normalizeTunnelHealthSpec(config)
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

func cloudflareLoadBalancingWebhookName(config OnLoadBalancingHealthAlertSpec) string {
	if config.Pool == "" {
		return "SuperPlane Load Balancing Health Alert"
	}

	return fmt.Sprintf("SuperPlane Load Balancing Health Alert - %s", config.Pool)
}

func cloudflareTunnelHealthWebhookName(config OnTunnelHealthSpec) string {
	if config.Tunnel == "" {
		return "SuperPlane Tunnel Health Alert"
	}

	return fmt.Sprintf("SuperPlane Tunnel Health Alert - %s", config.Tunnel)
}

func oneOrNil(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return []string{value}
}
