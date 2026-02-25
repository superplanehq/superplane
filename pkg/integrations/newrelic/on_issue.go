package newrelic

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssue struct{}

type OnIssueConfiguration struct {
	Account    string   `json:"account" yaml:"account" mapstructure:"account"`
	Priorities []string `json:"priorities" yaml:"priorities" mapstructure:"priorities"`
	States     []string `json:"states" yaml:"states" mapstructure:"states"`
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

- **Account**: The New Relic account to monitor for issues.
- **Priorities**: Filter by priority (CRITICAL, HIGH, MEDIUM, LOW). Leave empty for all.
- **States**: Filter by state (ACTIVATED, CLOSED, CREATED). Leave empty for all.

## How It Works

When you save this trigger, Superplane automatically creates a **Webhook Destination** and a **Notification Channel** in your New Relic account via the NerdGraph API. New Relic issues matching your filter criteria will be forwarded to Superplane automatically — no manual configuration required.

When the trigger is deleted, the destination and channel are automatically cleaned up from your New Relic account.
`
}

func (t *OnIssue) Icon() string {
	return "newrelic"
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
			Required:    true,
			Description: "The New Relic account to monitor for issues",
			Placeholder: "Select an account",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "account",
				},
			},
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

// No trigger-specific metadata needed anymore as it's managed by the WebhookHandler

func (t *OnIssue) Setup(ctx core.TriggerContext) error {
	// NerdGraph API calls (creating/deleting destinations and channels) require a User API Key
	userAPIKey, err := ctx.Integration.GetConfig("userApiKey")
	if err != nil || len(userAPIKey) == 0 {
		msg := "User API Key is required for this trigger. Please configure it in the Integration settings."
		return fmt.Errorf("%s", msg)
	}

	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Account == "" {
		return fmt.Errorf("account is required")
	}

	// Request the webhook with the account ID.
	// This will trigger NewrelicWebhookHandler.Setup, which creates the destination and channel.
	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Account: config.Account,
	})
}

func (t *OnIssue) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIssue) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

type NewrelicIssue struct {
	IssueID  string `json:"issue_id"`
	Title    string `json:"title"`
	Priority string `json:"priority"`
	State    string `json:"state"`
	Owner    string `json:"owner"`
	URL      string `json:"issue_url"`
}

// HandleWebhook processes incoming New Relic webhook requests
func (t *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// 0. Verify Authentication
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get webhook secret: %w", err)
	}

	providedSecret := ctx.Headers.Get("X-Superplane-Secret")
	if providedSecret == "" {
		return http.StatusUnauthorized, fmt.Errorf("missing X-Superplane-Secret header")
	}

	if subtle.ConstantTimeCompare([]byte(providedSecret), secret) != 1 {
		return http.StatusUnauthorized, fmt.Errorf("invalid webhook secret")
	}

	// 1. Decode Configuration
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		log.Errorf("Error decoding configuration: %v", err)
		return http.StatusBadRequest, fmt.Errorf("error decoding configuration: %w", err)
	}

	// 2. Parse Payload into Map (Handshake Check)
	var rawPayload map[string]any
	if len(ctx.Body) == 0 {
		log.Infof("New Relic Validation Ping Received (Empty Body)")
		return http.StatusOK, nil
	}
	if err := json.Unmarshal(ctx.Body, &rawPayload); err != nil {
		log.Errorf("Error parsing webhook body: %v", err)
		return http.StatusBadRequest, fmt.Errorf("error parsing webhook body: %w", err)
	}

	// 3. Handshake Logic
	_, hasIssueID := rawPayload["issue_id"]
	if !hasIssueID {
		log.Infof("New Relic Validation Ping Received")
		return http.StatusOK, nil
	}

	// 4. Decode Payload (Mapstructure)
	var issue NewrelicIssue
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &issue,
		TagName:  "json",
	})
	if err != nil {
		log.Errorf("Error creating decoder: %v", err)
		return http.StatusInternalServerError, fmt.Errorf("error creating decoder: %w", err)
	}

	if err := decoder.Decode(rawPayload); err != nil {
		log.Errorf("Error decoding payload: %v", err)
		return http.StatusBadRequest, fmt.Errorf("error decoding payload: %w", err)
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
	case "CREATED":
		eventName = "newrelic.issue_created"
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
	// Webhook cleanup is handled by the core calling NewrelicWebhookHandler.Cleanup
	return nil
}
