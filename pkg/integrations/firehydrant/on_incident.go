package firehydrant

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIncident struct{}

type OnIncidentConfiguration struct {
	Severities []string `json:"severities"`
}

func (t *OnIncident) Name() string {
	return "firehydrant.onIncident"
}

func (t *OnIncident) Label() string {
	return "On New Incident"
}

func (t *OnIncident) Description() string {
	return "Runs when a new incident is created in FireHydrant"
}

func (t *OnIncident) Documentation() string {
	return `The On New Incident trigger starts a workflow execution when a new incident is created in FireHydrant.

## Use Cases

- **Incident response**: Automatically notify Slack, update a status page, or create a Jira ticket when an incident is opened
- **Alert escalation**: Trigger escalation workflows when critical incidents are created
- **Cross-tool sync**: Sync new FireHydrant incidents to other incident management tools

## Configuration

- **Severities** (optional): Filter by severity levels. Only incidents matching the selected severities will trigger the workflow. If empty, all new incidents will trigger.

## Event Data

Each incident event includes:
- **name**: Incident name/title
- **number**: Incident number
- **severity**: Severity level
- **priority**: Priority level
- **current_milestone**: Current milestone (e.g., started, acknowledged)
- **created_at**: Creation timestamp
- **summary**: Incident summary

## Webhook Setup

This trigger automatically sets up a FireHydrant webhook endpoint when configured. The endpoint is managed by SuperPlane and will be cleaned up when the trigger is removed.

## Note

FireHydrant webhooks deliver all incident events. This trigger filters for only ` + "`CREATED`" + ` operations, ensuring only new incidents fire the workflow.`
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
			Name:        "severities",
			Label:       "Severities",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Only trigger for incidents with these severities. Leave empty for all severities.",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "SEV1", Value: "SEV1"},
						{Label: "SEV2", Value: "SEV2"},
						{Label: "SEV3", Value: "SEV3"},
						{Label: "SEV4", Value: "SEV4"},
						{Label: "SEV5", Value: "SEV5"},
						{Label: "UNSET", Value: "UNSET"},
						{Label: "MAINTENANCE", Value: "MAINTENANCE"},
						{Label: "GAMEDAY", Value: "GAMEDAY"},
					},
				},
			},
		},
	}
}

func (t *OnIncident) Setup(ctx core.TriggerContext) error {
	return ctx.Integration.RequestWebhook(WebhookConfiguration{})
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

	// Only process incident creation events
	if payload.Event.ResourceType != "incident" || payload.Event.Operation != "CREATED" {
		return http.StatusOK, nil
	}

	// Apply severity filter if configured
	if len(config.Severities) > 0 {
		incidentSeverity := extractSeveritySlug(payload)
		if incidentSeverity != "" && !containsString(config.Severities, incidentSeverity) {
			return http.StatusOK, nil
		}
	}

	err = ctx.Events.Emit(
		"firehydrant.incident.created",
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
	result := map[string]any{
		"event":         "incident.created",
		"operation":     payload.Event.Operation,
		"resource_type": payload.Event.ResourceType,
	}

	if payload.Data.Incident != nil {
		result["incident"] = payload.Data.Incident
	}

	return result
}

// extractSeveritySlug pulls the severity slug from the webhook incident data.
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
	if !hmacEqual([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// computeHMACSHA256 computes HMAC-SHA256 and returns hex-encoded result
func computeHMACSHA256(key, data []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// hmacEqual compares two HMAC values in constant time
func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}

	return result == 0
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}

	return false
}
