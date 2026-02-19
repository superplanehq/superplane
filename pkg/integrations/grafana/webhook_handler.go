package grafana

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type GrafanaWebhookHandler struct{}

type GrafanaWebhookMetadata struct {
	ContactPointUID string `json:"contactPointUid" mapstructure:"contactPointUid"`
}

func (h *GrafanaWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config := OnAlertFiringConfig{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	sharedSecret := strings.TrimSpace(config.SharedSecret)
	if err := ctx.Webhook.SetSecret([]byte(sharedSecret)); err != nil {
		return nil, fmt.Errorf("failed to persist shared secret in webhook storage: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("grafana webhook setup: falling back to manual setup (client unavailable): %v", err)
		}
		return nil, nil
	}

	name := buildContactPointName(ctx.Webhook.GetID())

	uid, err := client.UpsertWebhookContactPoint(name, ctx.Webhook.GetURL(), sharedSecret)
	if err != nil {
		if !shouldFallbackToManualSetup(err) {
			return nil, fmt.Errorf("grafana webhook setup: contact point provisioning will be retried: %w", err)
		}

		if ctx.Logger != nil {
			ctx.Logger.Warnf("grafana webhook setup: falling back to manual setup (contact point provisioning failed): %v", err)
		}
		return nil, nil
	}

	return GrafanaWebhookMetadata{
		ContactPointUID: uid,
	}, nil
}

func (h *GrafanaWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	if ctx.Webhook.GetMetadata() == nil {
		return nil
	}

	metadata := GrafanaWebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return err
	}

	contactPointUID := strings.TrimSpace(metadata.ContactPointUID)
	if contactPointUID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return err
	}

	return client.DeleteContactPoint(contactPointUID)
}

func (h *GrafanaWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := OnAlertFiringConfig{}
	if err := mapstructure.Decode(a, &configA); err != nil {
		return false, err
	}

	configB := OnAlertFiringConfig{}
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

func (h *GrafanaWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig := OnAlertFiringConfig{}
	if err := mapstructure.Decode(current, &currentConfig); err != nil {
		return nil, false, err
	}

	requestedConfig := OnAlertFiringConfig{}
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

	merged := OnAlertFiringConfig{
		SharedSecret:      mergedSharedSecret,
		WebhookBindingKey: mergedWebhookBindingKey,
	}

	changed := strings.TrimSpace(currentConfig.SharedSecret) != merged.SharedSecret ||
		strings.TrimSpace(currentConfig.WebhookBindingKey) != merged.WebhookBindingKey
	return merged, changed, nil
}

func buildContactPointName(webhookID string) string {
	hash := sha256.New()
	hash.Write([]byte(webhookID))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	return fmt.Sprintf("superplane-%s", suffix[:16])
}

func shouldFallbackToManualSetup(err error) bool {
	var statusErr *apiStatusError
	if !errors.As(err, &statusErr) {
		return false
	}

	switch statusErr.StatusCode {
	case http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusMethodNotAllowed,
		http.StatusUnprocessableEntity:
		return true
	default:
		return false
	}
}
