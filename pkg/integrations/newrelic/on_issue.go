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
	Priorities      []string `json:"priorities" yaml:"priorities" mapstructure:"priorities"`
	States          []string `json:"states" yaml:"states" mapstructure:"states"`
	Account         string   `json:"account" yaml:"account" mapstructure:"account"`
	ManualAccountID string   `json:"manualAccountId" yaml:"manualAccountId" mapstructure:"manualAccountId"`
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
  "issue_id": "{{issueId}}",
  "title": "{{annotations.title.[0]}}",
  "priority": "{{priority}}",
  "issue_url": "{{issuePageUrl}}",
  "state": "{{state}}",
  "owner": "{{owner}}"
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
			Name:        "account",
			Label:       "Account",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false, // Optional to prevent blocking
			Description: "The New Relic account (optional for webhook)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "account",
				},
			},
		},
		{
			Name:        "manualAccountId",
			Label:       "Manual Account ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Manually enter Account ID if dropdown fails",
			Placeholder: "1234567",
		},
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

// OnIssueMetadata holds the metadata stored on the canvas node for the UI.
type OnIssueMetadata struct {
	URL    string `json:"url" mapstructure:"url"`
	Manual bool   `json:"manual" mapstructure:"manual"`
}

func (t *OnIssue) Setup(ctx core.TriggerContext) error {
	// 1. Always ensure manual: true in metadata so the UI refreshes correctly
	//    to show the webhook URL once set up.
	var metadata OnIssueMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		// If decode fails, start fresh
		metadata = OnIssueMetadata{}
	}
	
	metadata.Manual = true
	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	// 2. Check if URL is already set (idempotency guard)
	if metadata.URL != "" {
		return nil
	}

	// 3. Decode configuration
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// 4. Create the webhook and get the URL
	webhookURL, err := ctx.Webhook.Setup()
	if err != nil {
		return fmt.Errorf("failed to setup webhook: %w", err)
	}

	// 5. Store the URL in node metadata
	metadata.URL = webhookURL
	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	ctx.Logger.Infof("New Relic OnIssue webhook URL: %s", webhookURL)
	return nil
}

func (t *OnIssue) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIssue) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

type NewRelicIssue struct {
	IssueID  string `json:"issue_id"`
	Title    string `json:"title"`
	Priority string `json:"priority"`
	State    string `json:"state"`
	Owner    string `json:"owner"`
	URL      string `json:"issue_url"`
}

// HandleWebhook processes incoming New Relic webhook requests
//
// Represents the "Azure Pattern" adapted for New Relic:
// 1. Handshake/Validation check (Test notification)
// 2. Mapstructure decoding
// 3. Event processing
func (t *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// 1. Decode Configuration
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		fmt.Printf("Error decoding configuration: %v\n", err)
		// Azure Pattern: Return 200 OK on malformed config to avoid retries/disable
		return http.StatusOK, nil
	}

	// 2. Parse Payload into Map (Handshake Check)
	var rawPayload map[string]any
	if len(ctx.Body) == 0 {
		fmt.Println("New Relic Validation Ping Received (Empty Body)")
		return http.StatusOK, nil
	}
	if err := json.Unmarshal(ctx.Body, &rawPayload); err != nil {
		fmt.Printf("Error parsing webhook body: %v\n", err)
		return http.StatusOK, nil
	}

	// 3. Handshake Logic (Mirroring Azure SubscriptionValidation)
	// If the payload is missing issue_id (indicating a New Relic "Test Connection" ping)
	_, hasIssueID := rawPayload["issue_id"]

	if !hasIssueID {
		fmt.Println("New Relic Validation Ping Received")
		return http.StatusOK, nil
	}

	// 4. Decode Payload (Mapstructure)
	var issue NewRelicIssue
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &issue,
		TagName:  "json",
	})
	if err != nil {
		fmt.Printf("Error creating decoder: %v\n", err)
		return http.StatusOK, nil
	}

	if err := decoder.Decode(rawPayload); err != nil {
		fmt.Printf("Error decoding payload: %v\n", err)
		return http.StatusOK, nil
	}

	// 5. Filter Logic
	if !allowedPriority(issue.Priority, config.Priorities) {
		return http.StatusOK, nil
	}
	if !allowedState(issue.State, config.States) {
		return http.StatusOK, nil
	}

	var eventName string
	switch issue.State {
	case "ACTIVATED":
		eventName = "newrelic.issue_activated"
	case "CLOSED":
		eventName = "newrelic.issue_closed"
	default:
		eventName = "newrelic.issue_updated"
	}

	// 6. Emit Event
	eventData := map[string]any{
		"issueId":  issue.IssueID,
		"title":    issue.Title,
		"priority": issue.Priority,
		"state":    issue.State,
		"issueUrl": issue.URL,
		"owner":    issue.Owner,
	}

	if err := ctx.Events.Emit(eventName, eventData); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	return http.StatusOK, nil
}

func allowedPriority(priority string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	return slices.Contains(allowed, priority)
}

func allowedState(state string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	return slices.Contains(allowed, state)
}

func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}