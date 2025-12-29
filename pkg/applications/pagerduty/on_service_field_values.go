package pagerduty

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnServiceFieldValues struct{}

type OnServiceFieldValuesMetadata struct {
	WebhookRegistered bool `json:"webhookRegistered"`
}

func (t *OnServiceFieldValues) Name() string {
	return "pagerduty.onServiceFieldValues"
}

func (t *OnServiceFieldValues) Label() string {
	return "On Service Field Values"
}

func (t *OnServiceFieldValues) Description() string {
	return "Listen to service custom field values updated events"
}

func (t *OnServiceFieldValues) Icon() string {
	return "settings"
}

func (t *OnServiceFieldValues) Color() string {
	return "green"
}

func (t *OnServiceFieldValues) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnServiceFieldValues) Setup(ctx core.TriggerContext) error {
	var metadata OnServiceFieldValuesMetadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	if metadata.WebhookRegistered {
		return nil
	}

	err = ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		Events: []string{"service.custom_field_values.updated"},
	})
	if err != nil {
		return err
	}

	metadata.WebhookRegistered = true
	return ctx.MetadataContext.Set(metadata)
}

func (t *OnServiceFieldValues) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnServiceFieldValues) HandleAction(ctx core.TriggerActionContext) error {
	return nil
}

func (t *OnServiceFieldValues) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// Verify signature
	signature := ctx.Headers.Get("X-PagerDuty-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	parts := strings.SplitN(signature, "=", 2)
	if len(parts) != 2 || parts[0] != "v1" {
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.WebhookContext.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	if err := crypto.VerifySignature(secret, ctx.Body, parts[1]); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	// Parse payload
	data := map[string]any{}
	err = json.Unmarshal(ctx.Body, &data)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Emit event
	err = ctx.EventContext.Emit("pagerduty.serviceFieldValues", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
