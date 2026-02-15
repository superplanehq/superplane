package harness

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

const OnPipelineCompletedPayloadType = "harness.pipeline.completed"

type OnPipelineCompleted struct{}

type OnPipelineCompletedConfiguration struct {
	PipelineIdentifier string   `json:"pipelineIdentifier" mapstructure:"pipelineIdentifier"`
	Statuses           []string `json:"statuses" mapstructure:"statuses"`
}

type OnPipelineCompletedMetadata struct {
	PipelineIdentifier string `json:"pipelineIdentifier,omitempty" mapstructure:"pipelineIdentifier"`
	WebhookURL         string `json:"webhookUrl,omitempty" mapstructure:"webhookUrl"`
}

var onPipelineCompletedStatusOptions = []configuration.FieldOption{
	{Label: "Succeeded", Value: "succeeded"},
	{Label: "Failed", Value: "failed"},
	{Label: "Aborted", Value: "aborted"},
	{Label: "Expired", Value: "expired"},
}

var onPipelineCompletedAllowedStatuses = []string{"succeeded", "failed", "aborted", "expired"}

func (t *OnPipelineCompleted) Name() string {
	return "harness.onPipelineCompleted"
}

func (t *OnPipelineCompleted) Label() string {
	return "On Pipeline Completed"
}

func (t *OnPipelineCompleted) Description() string {
	return "Listen to Harness pipeline completion events"
}

func (t *OnPipelineCompleted) Documentation() string {
	return `The On Pipeline Completed trigger starts a workflow when a Harness pipeline execution finishes.

## Use Cases

- **Failure notifications**: Send Slack alerts when critical pipelines fail
- **Release automation**: Trigger post-deploy checks when a deployment pipeline succeeds
- **Incident workflows**: Create tickets for aborted/expired pipeline runs

## Configuration

- **Pipeline Identifier**: Optional pipeline identifier filter. Leave empty to accept all pipeline completions.
- **Statuses**: Completion statuses that should trigger the workflow.

## Webhook Setup

This trigger generates a SuperPlane webhook URL. Configure a Harness notification rule to send pipeline completion events to that URL.

Recommended: set **Webhook Secret** in the Harness integration and send it as ` + "`Authorization: Bearer <secret>`" + ` from Harness.`
}

func (t *OnPipelineCompleted) Icon() string {
	return "workflow"
}

func (t *OnPipelineCompleted) Color() string {
	return "gray"
}

func (t *OnPipelineCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "pipelineIdentifier",
			Label:       "Pipeline",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional pipeline filter",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePipeline,
				},
			},
		},
		{
			Name:        "statuses",
			Label:       "Statuses",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    true,
			Default:     []string{"succeeded", "failed"},
			Description: "Pipeline completion statuses to listen for",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{Options: onPipelineCompletedStatusOptions},
			},
		},
	}
}

func (t *OnPipelineCompleted) Setup(ctx core.TriggerContext) error {
	config, err := decodeOnPipelineCompletedConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if ctx.Webhook == nil {
		return fmt.Errorf("missing webhook context")
	}

	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	return ctx.Metadata.Set(OnPipelineCompletedMetadata{
		PipelineIdentifier: config.PipelineIdentifier,
		WebhookURL:         webhookURL,
	})
}

func (t *OnPipelineCompleted) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnPipelineCompleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnPipelineCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config, err := decodeOnPipelineCompletedConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusBadRequest, err
	}

	if err := authorizeWebhook(ctx); err != nil {
		return http.StatusForbidden, err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	event := extractPipelineWebhookEvent(payload)
	if !isPipelineCompletedEventType(event.EventType) {
		return http.StatusOK, nil
	}

	if !isTerminalStatus(event.Status) {
		return http.StatusOK, nil
	}

	if config.PipelineIdentifier != "" {
		if event.PipelineIdentifier == "" || config.PipelineIdentifier != event.PipelineIdentifier {
			return http.StatusOK, nil
		}
	}

	if len(config.Statuses) > 0 && !statusSelected(config.Statuses, event.Status) {
		return http.StatusOK, nil
	}

	emittedPayload := map[string]any{
		"executionId":        event.ExecutionID,
		"pipelineIdentifier": event.PipelineIdentifier,
		"status":             canonicalStatus(event.Status),
		"eventType":          event.EventType,
		"raw":                payload,
	}

	if err := ctx.Events.Emit(OnPipelineCompletedPayloadType, emittedPayload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnPipelineCompleted) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func decodeOnPipelineCompletedConfiguration(value any) (OnPipelineCompletedConfiguration, error) {
	config := OnPipelineCompletedConfiguration{}
	if err := mapstructure.Decode(value, &config); err != nil {
		return OnPipelineCompletedConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.PipelineIdentifier = strings.TrimSpace(config.PipelineIdentifier)
	config.Statuses = normalizeSelectedStatuses(config.Statuses)
	if len(config.Statuses) == 0 {
		config.Statuses = []string{"succeeded", "failed"}
	}

	return config, nil
}

func normalizeSelectedStatuses(statuses []string) []string {
	selected := make([]string, 0, len(statuses))
	for _, status := range statuses {
		normalized := normalizeStatus(status)
		if normalized == "" {
			continue
		}

		if !contains(onPipelineCompletedAllowedStatuses, normalized) {
			continue
		}

		if contains(selected, normalized) {
			continue
		}

		selected = append(selected, normalized)
	}

	return selected
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func statusSelected(selected []string, currentStatus string) bool {
	normalized := canonicalStatus(currentStatus)
	if normalized == "" {
		return false
	}

	return contains(selected, normalized)
}

func authorizeWebhook(ctx core.WebhookRequestContext) error {
	if ctx.Integration == nil {
		return nil
	}

	secret, err := optionalConfig(ctx.Integration, "webhookSecret")
	if err != nil {
		return fmt.Errorf("failed to read webhookSecret: %w", err)
	}

	if secret == "" {
		return nil
	}

	candidateTokens := []string{}
	authorizationHeader := strings.TrimSpace(ctx.Headers.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorizationHeader), "bearer ") {
		candidateTokens = append(candidateTokens, strings.TrimSpace(authorizationHeader[7:]))
	}

	candidateTokens = append(candidateTokens,
		strings.TrimSpace(ctx.Headers.Get("X-Harness-Webhook-Token")),
		strings.TrimSpace(ctx.Headers.Get("X-Api-Key")),
	)

	for _, token := range candidateTokens {
		if token == "" {
			continue
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1 {
			return nil
		}
	}

	return fmt.Errorf("invalid webhook authorization")
}
