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
	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("grafana webhook setup: falling back to manual setup (client unavailable): %v", err)
		}
		return nil, nil
	}

	config := OnAlertFiringConfig{}
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	name := buildContactPointName(ctx.Webhook.GetID())
	sharedSecret := strings.TrimSpace(config.SharedSecret)

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
	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return nil
	}

	metadata := GrafanaWebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return nil
	}

	return client.DeleteContactPoint(strings.TrimSpace(metadata.ContactPointUID))
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

	merged := OnAlertFiringConfig{
		SharedSecret: strings.TrimSpace(requestedConfig.SharedSecret),
	}

	changed := strings.TrimSpace(currentConfig.SharedSecret) != merged.SharedSecret
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
