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

type OnIncidentFieldValues struct{}

type OnIncidentFieldValuesMetadata struct {
	WebhookRegistered bool `json:"webhookRegistered"`
}

func (t *OnIncidentFieldValues) Name() string {
	return "pagerduty.onIncidentFieldValues"
}

func (t *OnIncidentFieldValues) Label() string {
	return "On Incident Field Values"
}

func (t *OnIncidentFieldValues) Description() string {
	return "Listen to incident custom field values updated events"
}

func (t *OnIncidentFieldValues) Icon() string {
	return "settings"
}

func (t *OnIncidentFieldValues) Color() string {
	return "orange"
}

func (t *OnIncidentFieldValues) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnIncidentFieldValues) Setup(ctx core.TriggerContext) error {
	var metadata OnIncidentFieldValuesMetadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	if metadata.WebhookRegistered {
		return nil
	}

	err = ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		Events: []string{"incident.custom_field_values.updated"},
	})
	if err != nil {
		return err
	}

	metadata.WebhookRegistered = true
	return ctx.MetadataContext.Set(metadata)
}

func (t *OnIncidentFieldValues) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncidentFieldValues) HandleAction(ctx core.TriggerActionContext) error {
	return nil
}

func (t *OnIncidentFieldValues) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
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
	err = ctx.EventContext.Emit("pagerduty.incidentFieldValues", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
