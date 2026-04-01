package grafana

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type GrafanaWebhookHandler struct{}

type GrafanaWebhookMetadata struct {
	ContactPointUID  string `json:"contactPointUid" mapstructure:"contactPointUid"`
	ContactPointName string `json:"contactPointName" mapstructure:"contactPointName"`
}

func (h *GrafanaWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	config := OnAlertFiringConfig{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	sharedSecret, err := resolveOrCreateWebhookSecret(ctx.Webhook, config)
	if err != nil {
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

	if err := client.UpsertNotificationPolicyRoute(name, config.AlertNames); err != nil {
		return nil, fmt.Errorf("grafana webhook setup: notification policy route: %w", err)
	}

	return GrafanaWebhookMetadata{
		ContactPointUID:  uid,
		ContactPointName: name,
	}, nil
}

func resolveOrCreateWebhookSecret(webhook core.WebhookContext, config OnAlertFiringConfig) (string, error) {
	legacy := strings.TrimSpace(config.SharedSecret)

	secret, err := webhook.GetSecret()
	if err != nil {
		return "", err
	}
	stored := strings.TrimSpace(string(secret))

	var sharedSecret string
	switch {
	case legacy != "":
		// Legacy trigger config wins over an existing stored secret so contact points and
		// any manually configured Grafana bearer tokens stay aligned with user intent.
		sharedSecret = legacy
	case stored != "":
		sharedSecret = stored
	default:
		sharedSecret, err = crypto.Base64String(32)
		if err != nil {
			return "", err
		}
	}

	if err := webhook.SetSecret([]byte(sharedSecret)); err != nil {
		return "", err
	}

	return sharedSecret, nil
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

	contactPointName := strings.TrimSpace(metadata.ContactPointName)
	if contactPointName == "" {
		// Backward compat: recompute name from webhook ID if not stored.
		contactPointName = buildContactPointName(ctx.Webhook.GetID())
	}

	if err := client.RemoveNotificationPolicyRoute(contactPointName); err != nil {
		return fmt.Errorf("grafana webhook cleanup: notification policy route: %w", err)
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
		if bindingKeyA == "" || bindingKeyB == "" || bindingKeyA != bindingKeyB {
			return false, nil
		}
	}

	trimA := strings.TrimSpace(configA.SharedSecret)
	trimB := strings.TrimSpace(configB.SharedSecret)
	secretsMatch := trimA == trimB
	if !secretsMatch && bindingKeyA != "" && bindingKeyA == bindingKeyB && trimB == "" {
		// New canvases omit sharedSecret; treat as compatible with an existing webhook record
		// that still carries a legacy or stored secret so we do not force reprovisioning.
		secretsMatch = true
	}

	return secretsMatch && predicatesSliceEqual(configA.AlertNames, configB.AlertNames), nil
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
	alertNamesProvided := false
	if requestedMap, ok := requested.(map[string]any); ok {
		_, sharedSecretProvided = requestedMap["sharedSecret"]
		_, webhookBindingKeyProvided = requestedMap["webhookBindingKey"]
		_, alertNamesProvided = requestedMap["alertNames"]
	}

	mergedSharedSecret := strings.TrimSpace(currentConfig.SharedSecret)
	if sharedSecretProvided {
		mergedSharedSecret = strings.TrimSpace(requestedConfig.SharedSecret)
	}

	mergedWebhookBindingKey := strings.TrimSpace(currentConfig.WebhookBindingKey)
	if webhookBindingKeyProvided && strings.TrimSpace(requestedConfig.WebhookBindingKey) != "" {
		mergedWebhookBindingKey = strings.TrimSpace(requestedConfig.WebhookBindingKey)
	}

	mergedAlertNames := currentConfig.AlertNames
	if alertNamesProvided {
		mergedAlertNames = requestedConfig.AlertNames
	}

	merged := OnAlertFiringConfig{
		SharedSecret:      mergedSharedSecret,
		WebhookBindingKey: mergedWebhookBindingKey,
		AlertNames:        mergedAlertNames,
	}

	changed := strings.TrimSpace(currentConfig.SharedSecret) != merged.SharedSecret ||
		strings.TrimSpace(currentConfig.WebhookBindingKey) != merged.WebhookBindingKey ||
		!predicatesSliceEqual(currentConfig.AlertNames, merged.AlertNames)
	return merged, changed, nil
}

func predicatesSliceEqual(a, b []configuration.Predicate) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
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
