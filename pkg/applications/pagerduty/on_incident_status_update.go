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

type OnIncidentStatusUpdate struct{}

type OnIncidentStatusUpdateMetadata struct {
	WebhookRegistered bool `json:"webhookRegistered"`
}

func (t *OnIncidentStatusUpdate) Name() string {
	return "pagerduty.onIncidentStatusUpdate"
}

func (t *OnIncidentStatusUpdate) Label() string {
	return "On Incident Status Update"
}

func (t *OnIncidentStatusUpdate) Description() string {
	return "Listen to incident status update published events"
}

func (t *OnIncidentStatusUpdate) Icon() string {
	return "info"
}

func (t *OnIncidentStatusUpdate) Color() string {
	return "cyan"
}

func (t *OnIncidentStatusUpdate) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (t *OnIncidentStatusUpdate) Setup(ctx core.TriggerContext) error {
	var metadata OnIncidentStatusUpdateMetadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	if metadata.WebhookRegistered {
		return nil
	}

	err = ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		Events: []string{"incident.status_update_published"},
	})
	if err != nil {
		return err
	}

	metadata.WebhookRegistered = true
	return ctx.MetadataContext.Set(metadata)
}

func (t *OnIncidentStatusUpdate) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncidentStatusUpdate) HandleAction(ctx core.TriggerActionContext) error {
	return nil
}

func (t *OnIncidentStatusUpdate) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
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
	err = ctx.EventContext.Emit("pagerduty.incidentStatusUpdate", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
