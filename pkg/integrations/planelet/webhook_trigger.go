package planelet

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookTrigger struct{}

type WebhookTriggerConfiguration struct {
	TriggerID string `json:"triggerId" mapstructure:"triggerId"`
}

type WebhookTriggerMetadata struct {
	TriggerID        string         `json:"triggerId" mapstructure:"triggerId"`
	WebhookURL       string         `json:"webhookUrl" mapstructure:"webhookUrl"`
	Parameters       map[string]any `json:"parameters,omitempty" mapstructure:"parameters"`
	PlaneletMetadata map[string]any `json:"planeletMetadata,omitempty" mapstructure:"planeletMetadata"`
}

func (t *WebhookTrigger) Name() string {
	return "planelet.webhookTrigger"
}

func (t *WebhookTrigger) Label() string {
	return "On Planelet Webhook"
}

func (t *WebhookTrigger) Description() string {
	return "Start a workflow from a webhook trigger exposed by the connected Planelet server"
}

func (t *WebhookTrigger) Documentation() string {
	return `Triggers a workflow from a third-party webhook managed by the connected Planelet server.

## How It Works

1. Select a trigger from the Planelet manifest.
2. Configure the trigger's parameters.
3. When the workflow is published, SuperPlane generates a webhook URL and asks the Planelet server to register it with the third-party provider.
4. Incoming third-party webhook requests are forwarded to the Planelet server so it can verify, filter, and normalize the event before SuperPlane emits it into the workflow.`
}

func (t *WebhookTrigger) Icon() string {
	return "webhook"
}

func (t *WebhookTrigger) Color() string {
	return "gray"
}

func (t *WebhookTrigger) ExampleData() map[string]any {
	return map[string]any{
		"type": "planelet.webhook",
		"data": map[string]any{
			"eventType": "example.created",
			"payload": map[string]any{
				"id": "example_123",
			},
		},
	}
}

func (t *WebhookTrigger) Configuration() []configuration.Field {
	fields := []configuration.Field{
		{
			Name:     "triggerId",
			Label:    "Trigger",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "trigger",
				},
			},
			Description: "The Planelet webhook trigger to configure",
		},
	}

	manifest := getCachedManifest()
	if manifest != nil {
		for _, trigger := range manifest.Triggers {
			fields = append(fields, manifestParametersToConfig(trigger.Parameters, "triggerId", trigger.ID)...)
		}
	}

	return fields
}

func (t *WebhookTrigger) Setup(ctx core.TriggerContext) error {
	var config WebhookTriggerConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.TriggerID == "" {
		return fmt.Errorf("triggerId is required")
	}

	client, err := NewClientWithHTTP(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to create Planelet client: %w", err)
	}

	manifest, err := client.FetchManifest()
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	if !manifestHasTrigger(manifest, config.TriggerID) {
		return fmt.Errorf("trigger %q not found in Planelet manifest", config.TriggerID)
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return fmt.Errorf("failed to read webhook secret: %w", err)
	}

	params := extractPlaneletParams(ctx.Configuration, config.TriggerID)
	result, err := client.SetupTrigger(config.TriggerID, params, webhookURL, string(secret))
	if err != nil {
		return fmt.Errorf("failed to setup Planelet trigger: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("Planelet trigger setup failed: %s", result.Error)
	}

	return ctx.Metadata.Set(WebhookTriggerMetadata{
		TriggerID:        config.TriggerID,
		WebhookURL:       webhookURL,
		Parameters:       params,
		PlaneletMetadata: result.Metadata,
	})
}

func (t *WebhookTrigger) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	var config WebhookTriggerConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.TriggerID == "" {
		return http.StatusInternalServerError, nil, fmt.Errorf("triggerId is required")
	}

	var metadata WebhookTriggerMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	client, err := NewClientWithHTTP(ctx.Integration, ctx.HTTP)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to create Planelet client: %w", err)
	}

	method := ctx.Method
	if method == "" {
		method = http.MethodPost
	}

	result, err := client.HandleTriggerWebhook(config.TriggerID, HandleTriggerWebhookRequest{
		Parameters: extractPlaneletParams(ctx.Configuration, config.TriggerID),
		Metadata:   metadata.PlaneletMetadata,
		Request: ForwardedWebhookRequest{
			Method:        method,
			Headers:       copyHeaderValues(ctx.Headers),
			Query:         ctx.Query,
			RawBodyBase64: base64.StdEncoding.EncodeToString(ctx.Body),
		},
	})
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to handle Planelet webhook: %w", err)
	}

	if !result.Success {
		status := result.Status
		if status == 0 {
			status = http.StatusInternalServerError
		}

		return status, nil, fmt.Errorf("Planelet webhook failed: %s", result.Error)
	}

	status, response := webhookHTTPResponse(result.Response)
	if !result.Emit {
		return status, response, nil
	}

	eventType := result.EventType
	if eventType == "" {
		eventType = config.TriggerID
	}

	if err := ctx.Events.Emit(eventType, result.Payload); err != nil {
		return http.StatusInternalServerError, response, fmt.Errorf("failed to emit Planelet webhook event: %w", err)
	}

	return status, response, nil
}

func (t *WebhookTrigger) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *WebhookTrigger) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	return nil, nil
}

func (t *WebhookTrigger) Cleanup(ctx core.TriggerContext) error {
	var config WebhookTriggerConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	var metadata WebhookTriggerMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	triggerID := metadata.TriggerID
	if triggerID == "" {
		triggerID = config.TriggerID
	}

	if triggerID == "" {
		return nil
	}

	params := metadata.Parameters
	if params == nil {
		params = extractPlaneletParams(ctx.Configuration, triggerID)
	}

	client, err := NewClientWithHTTP(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to create Planelet client: %w", err)
	}

	result, err := client.CleanupTrigger(triggerID, params, metadata.PlaneletMetadata)
	if err != nil {
		return fmt.Errorf("failed to cleanup Planelet trigger: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("Planelet trigger cleanup failed: %s", result.Error)
	}

	return nil
}

func manifestHasTrigger(manifest *Manifest, triggerID string) bool {
	for _, trigger := range manifest.Triggers {
		if trigger.ID == triggerID {
			return true
		}
	}

	return false
}

func copyHeaderValues(headers http.Header) map[string][]string {
	values := make(map[string][]string, len(headers))
	for name, headerValues := range headers {
		values[name] = append([]string{}, headerValues...)
	}

	return values
}

func webhookHTTPResponse(response *WebhookHTTPResponse) (int, *core.WebhookResponseBody) {
	if response == nil {
		return http.StatusOK, nil
	}

	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}

	body := &core.WebhookResponseBody{
		Body:    []byte(response.Body),
		Headers: response.Headers,
	}

	if response.Headers != nil {
		body.ContentType = response.Headers["Content-Type"]
	}

	return status, body
}
