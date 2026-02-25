package firehydrant

import (
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	Events []string `json:"events"`
}

type WebhookMetadata struct {
	WebhookID string `json:"webhookId"`
}

type FireHydrantWebhookHandler struct{}

const firehydrantLegacyWebhookName = "SuperPlane"

func (h *FireHydrantWebhookHandler) CompareConfig(a, b any) (bool, error) {
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

	for _, eventB := range configB.Events {
		if !slices.Contains(configA.Events, eventB) {
			return false, nil
		}
	}

	return true, nil
}

func (h *FireHydrantWebhookHandler) Merge(current, requested any) (any, bool, error) {
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

func (h *FireHydrantWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	webhookURL := ctx.Webhook.GetURL()
	webhookID := ctx.Webhook.GetID()

	config := WebhookConfiguration{}
	err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return nil, fmt.Errorf("error decoding webhook configuration: %v", err)
	}

	// FireHydrant webhooks are set up manually by the user in the FireHydrant UI.
	// We generate a secret and store it for verifying incoming requests.
	// The user must copy the webhook URL and secret into FireHydrant settings.
	secret, err := ctx.Webhook.GetSecret()
	if err != nil || len(secret) == 0 {
		// The webhook provisioner generates a secret; we just use what's there.
		ctx.Logger.Info("FireHydrant webhook provisioned — user must configure it in FireHydrant Settings > Webhooks")
	}

	return WebhookMetadata{
		WebhookID: firehydrantWebhookName(webhookID, webhookURL),
	}, nil
}

func (h *FireHydrantWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	return nil
}

func firehydrantWebhookName(webhookID, webhookURL string) string {
	hash := sha256.New()
	hash.Write([]byte(webhookID))
	hash.Write([]byte(webhookURL))
	suffix := fmt.Sprintf("%x", hash.Sum(nil))
	return fmt.Sprintf("SuperPlane-%s", suffix[:12])
}
