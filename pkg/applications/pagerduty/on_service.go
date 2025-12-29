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

type OnService struct{}

type OnServiceConfiguration struct {
	Events []string `json:"events"`
}

type OnServiceMetadata struct {
	WebhookRegistered bool `json:"webhookRegistered"`
}

func (t *OnService) Name() string {
	return "pagerduty.onService"
}

func (t *OnService) Label() string {
	return "On Service"
}

func (t *OnService) Description() string {
	return "Listen to service events"
}

func (t *OnService) Icon() string {
	return "server"
}

func (t *OnService) Color() string {
	return "green"
}

func (t *OnService) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"created", "updated"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Updated", Value: "updated"},
						{Label: "Deleted", Value: "deleted"},
					},
				},
			},
		},
	}
}

func (t *OnService) Setup(ctx core.TriggerContext) error {
	var metadata OnServiceMetadata
	err := mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	if metadata.WebhookRegistered {
		return nil
	}

	config := OnServiceConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Build full event names
	fullEventNames := make([]string, len(config.Events))
	for i, event := range config.Events {
		fullEventNames[i] = fmt.Sprintf("service.%s", event)
	}

	err = ctx.AppInstallationContext.RequestWebhook(WebhookConfiguration{
		Events: fullEventNames,
	})
	if err != nil {
		return err
	}

	metadata.WebhookRegistered = true
	return ctx.MetadataContext.Set(metadata)
}

func (t *OnService) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnService) HandleAction(ctx core.TriggerActionContext) error {
	return nil
}

func (t *OnService) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnServiceConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

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

	// Filter by event type
	if !whitelistedEvent(data, config.Events, "service") {
		return http.StatusOK, nil
	}

	// Emit event
	err = ctx.EventContext.Emit("pagerduty.service", data)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}
