package fluxcd

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type FluxCDWebhookHandler struct{}

type WebhookConfiguration struct {
	SharedSecret      string `json:"sharedSecret" mapstructure:"sharedSecret"`
	WebhookBindingKey string `json:"webhookBindingKey,omitempty" mapstructure:"webhookBindingKey"`
}

func (h *FluxCDWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config := WebhookConfiguration{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	sharedSecret := strings.TrimSpace(config.SharedSecret)
	if err := ctx.Webhook.SetSecret([]byte(sharedSecret)); err != nil {
		return nil, fmt.Errorf("failed to persist shared secret: %w", err)
	}

	return nil, nil
}

func (h *FluxCDWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

func (h *FluxCDWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := WebhookConfiguration{}
	if err := mapstructure.Decode(b, &configB); err != nil {
		return false, err
	}

	bindingKeyA := strings.TrimSpace(configA.WebhookBindingKey)
	bindingKeyB := strings.TrimSpace(configB.WebhookBindingKey)
	if bindingKeyA != "" || bindingKeyB != "" {
		return bindingKeyA != "" && bindingKeyA == bindingKeyB, nil
	}

	return strings.TrimSpace(configA.SharedSecret) == strings.TrimSpace(configB.SharedSecret), nil
}

func (h *FluxCDWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(current, &currentConfig); err != nil {
		return nil, false, err
	}

	requestedConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(requested, &requestedConfig); err != nil {
		return nil, false, err
	}

	sharedSecretProvided := false
	webhookBindingKeyProvided := false
	if requestedMap, ok := requested.(map[string]any); ok {
		_, sharedSecretProvided = requestedMap["sharedSecret"]
		_, webhookBindingKeyProvided = requestedMap["webhookBindingKey"]
	}

	mergedSharedSecret := strings.TrimSpace(currentConfig.SharedSecret)
	if sharedSecretProvided {
		mergedSharedSecret = strings.TrimSpace(requestedConfig.SharedSecret)
	}

	mergedWebhookBindingKey := strings.TrimSpace(currentConfig.WebhookBindingKey)
	if webhookBindingKeyProvided && strings.TrimSpace(requestedConfig.WebhookBindingKey) != "" {
		mergedWebhookBindingKey = strings.TrimSpace(requestedConfig.WebhookBindingKey)
	}

	merged := WebhookConfiguration{
		SharedSecret:      mergedSharedSecret,
		WebhookBindingKey: mergedWebhookBindingKey,
	}

	changed := strings.TrimSpace(currentConfig.SharedSecret) != merged.SharedSecret ||
		strings.TrimSpace(currentConfig.WebhookBindingKey) != merged.WebhookBindingKey
	return merged, changed, nil
}
