package honeycomb

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnAlertFired struct{}

type OnAlertFiredConfiguration struct {
	DatasetSlug string `json:"datasetSlug" mapstructure:"datasetSlug"`
	Trigger     string `json:"trigger" mapstructure:"trigger"`
}

type OnAlertFiredNodeMetadata struct {
	TriggerID string `json:"triggerId" mapstructure:"triggerId"`
}

func (t *OnAlertFired) Name() string {
	return "honeycomb.onAlertFired"
}

func (t *OnAlertFired) Label() string {
	return "On Alert Fired"
}

func (t *OnAlertFired) Description() string {
	return "Triggers when a Honeycomb Trigger fires"
}

func (t *OnAlertFired) Icon() string {
	return "honeycomb"
}

func (t *OnAlertFired) Color() string {
	return "yellow"
}

func (t *OnAlertFired) Documentation() string {
	return `
Starts a workflow execution when a Honeycomb Trigger fires.

**Configuration:**
- **Dataset Slug**: The slug of the dataset that contains your Honeycomb trigger. Found in the dataset URL: honeycomb.io/<team>/datasets/<dataset-slug>.
- **Trigger**: The exact name of the Honeycomb trigger to listen to (case-insensitive). Found in your dataset under Triggers.

**How it works:**
SuperPlane automatically creates a webhook recipient in Honeycomb and attaches it to the selected trigger. No manual webhook setup is required.

When the trigger fires, SuperPlane receives the webhook and starts a workflow execution with the full alert payload.
`
}

func (t *OnAlertFired) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "datasetSlug",
			Label:       "Dataset Slug",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The dataset slug containing your Honeycomb trigger.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "dataset",
					UseNameAsValue: false,
				},
			},
		},
		{
			Name:        "trigger",
			Label:       "Trigger",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The name of the Honeycomb trigger to listen to (case-insensitive).",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "trigger",
					UseNameAsValue: true,
					Parameters: []configuration.ParameterRef{
						{
							Name: "datasetSlug",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "datasetSlug",
							},
						},
					},
				},
			},
		},
	}
}

func (t *OnAlertFired) Setup(ctx core.TriggerContext) error {
	cfg := OnAlertFiredConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	cfg.DatasetSlug = strings.TrimSpace(cfg.DatasetSlug)
	cfg.Trigger = strings.TrimSpace(cfg.Trigger)
	triggerName := cfg.Trigger

	if cfg.DatasetSlug == "" {
		return fmt.Errorf("datasetSlug is required")
	}
	if triggerName == "" {
		return fmt.Errorf("trigger is required")
	}

	if ctx.Integration == nil {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	teamAny, err := ctx.Integration.GetConfig("teamSlug")
	if err == nil && strings.TrimSpace(string(teamAny)) != "" {
		if err := client.EnsureConfigurationKey(strings.TrimSpace(string(teamAny))); err != nil {
			return fmt.Errorf("failed to ensure configuration key: %w", err)
		}
	}

	triggers, err := listDatasetAndEnvironmentTriggers(client, cfg.DatasetSlug)
	if err != nil {
		return fmt.Errorf("failed to list triggers: %w", err)
	}

	var triggerID string
	triggerDatasetSlug := cfg.DatasetSlug
	for _, tr := range triggers {
		if strings.EqualFold(strings.TrimSpace(tr.Name), triggerName) {
			triggerID = tr.ID
			if datasetFromTrigger, ok := tr.Raw["dataset_slug"].(string); ok && strings.TrimSpace(datasetFromTrigger) != "" {
				triggerDatasetSlug = strings.TrimSpace(datasetFromTrigger)
			}
			break
		}
	}

	if triggerID == "" {
		return fmt.Errorf("trigger with name %q not found in dataset %q", triggerName, cfg.DatasetSlug)
	}

	if err := ctx.Metadata.Set(OnAlertFiredNodeMetadata{TriggerID: triggerID}); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	if err := ctx.Integration.RequestWebhook(map[string]any{
		"datasetSlug": triggerDatasetSlug,
		"triggerIds":  []string{triggerID},
	}); err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	return nil
}

func (t *OnAlertFired) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnAlertFired) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnAlertFired) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func (t *OnAlertFired) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	cfg := OnAlertFiredConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &cfg); err != nil {
		return http.StatusInternalServerError, err
	}

	secretBytes, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, err
	}
	secret := string(secretBytes)

	provided := strings.TrimSpace(ctx.Headers.Get("X-Honeycomb-Webhook-Token"))
	if provided == "" {
		auth := strings.TrimSpace(ctx.Headers.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			provided = strings.TrimSpace(auth[len("bearer "):])
		}
	}

	if provided == "" {
		return http.StatusUnauthorized, fmt.Errorf("missing webhook token")
	}

	if subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) != 1 {
		return http.StatusForbidden, fmt.Errorf("invalid webhook token")
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		payload = map[string]any{"raw": string(ctx.Body)}
	}

	meta := OnAlertFiredNodeMetadata{}
	raw := ctx.Metadata.Get()
	if err := mapstructure.Decode(raw, &meta); err == nil && meta.TriggerID != "" {
		if !payloadHasTriggerID(payload, meta.TriggerID) {
			return http.StatusOK, nil
		}
	}

	if err := ctx.Events.Emit("honeycomb.alert.fired", payload); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func payloadHasTriggerID(payload map[string]any, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return true
	}

	if id, ok := payload["id"].(string); ok {
		return strings.EqualFold(strings.TrimSpace(id), want)
	}

	if id, ok := payload["trigger_id"].(string); ok {
		return strings.EqualFold(strings.TrimSpace(id), want)
	}

	if tr, ok := payload["trigger"].(map[string]any); ok {
		if id, ok := tr["id"].(string); ok {
			return strings.EqualFold(strings.TrimSpace(id), want)
		}
	}

	return false
}
