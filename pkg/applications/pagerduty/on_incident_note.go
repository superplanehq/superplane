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

type OnIncidentNote struct{}

type OnIncidentNoteMetadata struct {
	WebhookRegistered bool `json:"webhookRegistered"`
}

func (t *OnIncidentNote) Name() string {
	return "pagerduty.onIncidentNote"
}

func (t *OnIncidentNote) Label() string {
	return "On Incident Note"
}

func (t *OnIncidentNote) Description() string {
	return "Listen to incident annotation events"
}

func (t *OnIncidentNote) Icon() string {
	return "message-square"
}

func (t *OnIncidentNote) Color() string {
	return "blue"
}

func (t *OnIncidentNote) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnIncidentNote) Setup(ctx core.TriggerContext) error {
	var metadata OnIncidentNoteMetadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	if metadata.WebhookRegistered {
		return nil
	}

	// Request webhook for incident.annotated event
	err = ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		Events: []string{"incident.annotated"},
	})
	if err != nil {
		return err
	}

	metadata.WebhookRegistered = true
	return ctx.MetadataContext.Set(metadata)
}

func (t *OnIncidentNote) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncidentNote) HandleAction(ctx core.TriggerActionContext) error {
	return nil
}

func (t *OnIncidentNote) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
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

	// Verify event type is incident.annotated
	event, ok := data["event"].(map[string]any)
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("missing event field")
	}

	eventType, ok := event["event_type"].(string)
	if !ok || eventType != "incident.annotated" {
		return http.StatusOK, nil // Not the event we care about
	}

	// Emit event
	err = ctx.EventContext.Emit("pagerduty.incidentNote", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
