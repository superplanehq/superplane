package firehydrant

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	createdOperation = "CREATED"
	updatedOperation = "UPDATED"
)

type OnIncident struct{}

type OnIncidentConfiguration struct {
	Severities       []string `json:"severities"`
	CurrentMilestone []string `json:"current_milestone" mapstructure:"current_milestone"`
}

func (t *OnIncident) Name() string {
	return "firehydrant.onIncident"
}

func (t *OnIncident) Label() string {
	return "On Incident"
}

func (t *OnIncident) Description() string {
	return "Runs when an incident is created or reaches a specific milestone in FireHydrant"
}

func (t *OnIncident) Documentation() string {
	return `The On Incident trigger starts a workflow execution when a FireHydrant incident is created or reaches a specific milestone.

## Use Cases

- **Incident response**: Automatically notify Slack, update a status page, or create a Jira ticket when an incident is opened
- **Alert escalation**: Trigger escalation workflows when critical incidents are created
- **Milestone tracking**: React to incidents reaching specific milestones such as mitigated or resolved
- **Cross-tool sync**: Sync new FireHydrant incidents to other incident management tools

## Configuration

- **Current Milestone**: Select which incident milestones to trigger on (started, acknowledged, mitigated, resolved, etc.). The workflow will trigger when an incident is created or updated to match any of the selected milestones.
- **Severities** (optional): Filter by severity levels. Only incidents matching the selected severities will trigger the workflow. If empty, all severities are accepted.

## Event Data

Each incident event includes:
- **name**: Incident name/title
- **number**: Incident number
- **severity**: Severity level
- **priority**: Priority level
- **current_milestone**: Current milestone (e.g., started, acknowledged)
- **summary**: Incident summary

## Webhook Setup

This trigger automatically sets up a FireHydrant webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnIncident) Icon() string {
	return "flame"
}

func (t *OnIncident) Color() string {
	return "gray"
}

func (t *OnIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "current_milestone",
			Label:       "Current Milestone",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Only trigger for incident events matching the selected milestones.",
			Default:     []string{"started"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Started", Value: "started"},
						{Label: "Detected", Value: "detected"},
						{Label: "Acknowledged", Value: "acknowledged"},
						{Label: "Identified", Value: "identified"},
						{Label: "Mitigated", Value: "mitigated"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Retrospective Started", Value: "retrospective_started"},
						{Label: "Retrospective Completed", Value: "retrospective_completed"},
						{Label: "Closed", Value: "closed"},
					},
				},
			},
		},
		{
			Name:        "severities",
			Label:       "Severities",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only trigger for incidents with these severities. Leave empty for all severities.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "severity",
					UseNameAsValue: true,
					Multi:          true,
				},
			},
		},
	}
}

func (t *OnIncident) Setup(ctx core.TriggerContext) error {
	config := OnIncidentConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Subscriptions: []string{"incidents"},
	})
}

func (t *OnIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIncidentConfiguration{}
	if ctx.Configuration != nil {
		err := mapstructure.Decode(ctx.Configuration, &config)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
		}
	}

	// Verify HMAC-SHA256 signature
	signature := ctx.Headers.Get("fh-signature")
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	if err := verifyWebhookSignature(signature, ctx.Body, secret); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	// Parse webhook payload
	var payload WebhookPayload
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Only process incident events
	if payload.Event.ResourceType != "incident" {
		return http.StatusOK, nil
	}

	op := payload.Event.Operation

	if op != createdOperation && op != updatedOperation {
		return http.StatusOK, nil
	}

	// Only trigger on CREATED event when "started" milestone is configured.
	if op == createdOperation {
		if !slices.Contains(config.CurrentMilestone, "started") {
			return http.StatusOK, nil
		}
	}

	// Only trigger on UPDATED event when the current milestone is configured.
	if op == updatedOperation {
		// Prevent triggering on duplicate events on incident creation
		if milestones, ok := payload.Data.Incident["milestones"].([]any); ok && len(milestones) == 1 {
			return http.StatusOK, nil
		}

		currentMilestone, ok := payload.Data.Incident["current_milestone"].(string)
		if !ok || currentMilestone == "" || !slices.Contains(config.CurrentMilestone, currentMilestone) {
			return http.StatusOK, nil
		}
	}

	// Apply severity filter if configured
	if len(config.Severities) > 0 {
		incidentSeverity := extractSeveritySlug(payload)
		if incidentSeverity == "" || !slices.Contains(config.Severities, incidentSeverity) {
			return http.StatusOK, nil
		}
	}

	emitType := "firehydrant.incident.created"
	if payload.Event.Operation == updatedOperation {
		emitType = "firehydrant.incident.updated"
	}

	err = ctx.Events.Emit(
		emitType,
		buildIncidentPayload(payload),
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (t *OnIncident) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIncident) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIncident) Cleanup(ctx core.TriggerContext) error {
	return nil
}

// WebhookPayload represents the FireHydrant webhook payload
type WebhookPayload struct {
	Data  WebhookData  `json:"data"`
	Event WebhookEvent `json:"event"`
}

// WebhookData contains the incident data from the webhook
type WebhookData struct {
	Incident map[string]any `json:"incident"`
}

// WebhookEvent represents the event metadata in a FireHydrant webhook
type WebhookEvent struct {
	Operation    string `json:"operation"`
	ResourceType string `json:"resource_type"`
}

func buildIncidentPayload(payload WebhookPayload) map[string]any {
	eventName := "incident.created"
	if payload.Event.Operation == "UPDATED" {
		eventName = "incident.updated"
	}

	result := map[string]any{
		"event":         eventName,
		"operation":     payload.Event.Operation,
		"resource_type": payload.Event.ResourceType,
	}

	if payload.Data.Incident != nil {
		result["incident"] = normalizeIncident(payload.Data.Incident)
	}

	return result
}

func normalizeIncident(raw map[string]any) map[string]any {
	normalized := map[string]any{}
	maps.Copy(normalized, raw)
	if severity, ok := raw["severity"].(map[string]any); ok {
		if slug, ok := severity["slug"].(string); ok {
			normalized["severity"] = slug
		}
	}

	if priority, ok := raw["priority"].(map[string]any); ok {
		if slug, ok := priority["slug"].(string); ok {
			normalized["priority"] = slug
		}
	}

	return normalized
}

func extractSeveritySlug(payload WebhookPayload) string {
	if payload.Data.Incident == nil {
		return ""
	}

	severity, ok := payload.Data.Incident["severity"]
	if !ok || severity == nil {
		return ""
	}

	sevMap, ok := severity.(map[string]any)
	if !ok {
		return ""
	}

	slug, ok := sevMap["slug"].(string)
	if !ok {
		return ""
	}

	return slug
}

// verifyWebhookSignature verifies the FireHydrant webhook HMAC-SHA256 signature.
// FireHydrant sends an fh-signature header with a hex-encoded HMAC-SHA256 of the body.
func verifyWebhookSignature(signature string, body, secret []byte) error {
	if len(secret) == 0 {
		// If no secret is configured, skip verification
		return nil
	}

	if signature == "" {
		return fmt.Errorf("missing signature")
	}

	expectedSig := computeHMACSHA256(secret, body)
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

func computeHMACSHA256(key, data []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}
