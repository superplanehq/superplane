package newrelic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssue struct{}

type OnIssueConfiguration struct {
	Priorities []string `json:"priorities"`
	States     []string `json:"states"`
}

func (t *OnIssue) Name() string {
	return "newrelic.onIssue"
}

func (t *OnIssue) Label() string {
	return "On Issue"
}

func (t *OnIssue) Description() string {
	return "Listen to New Relic issue events"
}

func (t *OnIssue) Documentation() string {
	return `The On Issue trigger starts a workflow execution when New Relic issues are created or updated.

## Use Cases

- **Incident Response**: automated remediation or notification when critical issues occur.
- **Sync**: synchronize New Relic issues with Jira or other tracking systems.

## Configuration

- **Priorities**: Filter by priority (CRITICAL, HIGH, MEDIUM, LOW). Leave empty for all.
- **States**: Filter by state (ACTIVATED, CLOSED, CREATED). Leave empty for all.

## Webhook Setup

This trigger generates a webhook URL. You must configure a **Workflow** in New Relic to send a webhook to this URL.

**IMPORTANT**: You must use the following JSON payload template in your New Relic Webhook configuration:

` + "```json" + `
{
  "issue_id": "{{ issueId }}",
  "title": "{{ annotations.title }}",
  "priority": "{{ priority }}",
  "state": "{{ state }}",
  "issue_url": "{{ issuePageUrl }}",
  "owner": "{{ owner }}",
  "impacted_entities": {{ json entitiesData.names }},
  "total_incidents": {{ totalIncidents }}
}
` + "```" + `
`
}

func (t *OnIssue) Icon() string {
	return "alert-triangle"
}

func (t *OnIssue) Color() string {
	return "teal"
}

func (t *OnIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "priorities",
			Label:       "Priorities",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter issues by priority",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Critical", Value: "CRITICAL"},
						{Label: "High", Value: "HIGH"},
						{Label: "Medium", Value: "MEDIUM"},
						{Label: "Low", Value: "LOW"},
					},
				},
			},
		},
		{
			Name:        "states",
			Label:       "States",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Filter issues by state",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Activated", Value: "ACTIVATED"},
						{Label: "Closed", Value: "CLOSED"},
						{Label: "Created", Value: "CREATED"},
					},
				},
			},
		},
	}
}

func (t *OnIssue) Setup(ctx core.TriggerContext) error {
	config := OnIssueConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// We simply request a webhook. The user will manually configure it in New Relic
	// or we implement the WebhookHandler to do it automatically via NerdGraph.
	// For now, consistent with "minimally viable" and manual setup expectation if automation isn't trivial.
	return ctx.Integration.RequestWebhook(nil)
}

func (t *OnIssue) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIssue) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

type WebhookPayload struct {
	IssueID          string   `json:"issue_id"`
	Title            string   `json:"title"`
	Priority         string   `json:"priority"`
	State            string   `json:"state"`
	Owner            string   `json:"owner"`
	IssueURL         string   `json:"issue_url"`
	ImpactedEntities []string `json:"impacted_entities"`
	TotalIncidents   int      `json:"total_incidents"`
}

func (t *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIssueConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	var payload WebhookPayload
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse webhook body: %w", err)
	}

	// Validate required fields
	if payload.IssueID == "" {
		return http.StatusBadRequest, fmt.Errorf("missing issue_id in payload")
	}

	// Normalize Priority and State to match New Relic values usually sent as uppercase or capitalized
	// We assume the user uses the template which usually outputs standard New Relic values.
	// We'll trust the payload but maybe do case-insensitive comparison if needed.

	// Filter by Priority
	if len(config.Priorities) > 0 && !slices.Contains(config.Priorities, payload.Priority) {
		return http.StatusOK, nil
	}

	// Filter by State
	if len(config.States) > 0 && !slices.Contains(config.States, payload.State) {
		return http.StatusOK, nil
	}

	var eventName string
	switch payload.State {
	case "ACTIVATED":
		eventName = "newrelic.issue_activated"
	case "CLOSED":
		eventName = "newrelic.issue_closed"
	default:
		eventName = "newrelic.issue_updated"
	}

	// Convert payload back to map for event emission or use struct if emitter supports it
	// Using map for flexibility
	var eventData map[string]any
	if err := json.Unmarshal(ctx.Body, &eventData); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to re-marshal payload: %w", err)
	}

	if err := ctx.Events.Emit(eventName, eventData); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil
}

func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}
