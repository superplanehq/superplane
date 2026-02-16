package sentry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnIssue struct{}

type OnIssueConfiguration struct {
	Events []string `json:"events"`
}

func (t *OnIssue) Name() string {
	return "sentry.onIssue"
}

func (t *OnIssue) Label() string {
	return "On Issue"
}

func (t *OnIssue) Description() string {
	return "Listen to issue events from Sentry"
}

func (t *OnIssue) Documentation() string {
	return `The On Issue trigger starts a workflow execution when Sentry issue events occur.

## Use Cases

- **Issue automation**: Automate responses to issue events
- **Notification workflows**: Send notifications when issues are created or resolved
- **Integration workflows**: Sync issues with external systems
- **Assignment handling**: Handle issue assignments automatically

## Configuration

- **Events**: Select which issue events to listen for (created, resolved, assigned, ignored, unresolved)

## Event Data

Each issue event includes:
- **event**: Event type (issue.created, issue.resolved, issue.assigned, issue.ignored, issue.unresolved)
- **issue**: Complete issue information including title, status, assignee, project
- **actionUser**: User who triggered the event (if applicable)

## Webhook Setup

This trigger uses a Sentry Internal Integration to receive webhook events. The integration is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnIssue) Icon() string {
	return "alert-circle"
}

func (t *OnIssue) Color() string {
	return "purple"
}

func (t *OnIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "events",
			Label:    "Events",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"issue.created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "issue.created"},
						{Label: "Resolved", Value: "issue.resolved"},
						{Label: "Assigned", Value: "issue.assigned"},
						{Label: "Ignored", Value: "issue.ignored"},
						{Label: "Unresolved", Value: "issue.unresolved"},
					},
				},
			},
		},
	}
}

func (t *OnIssue) Setup(ctx core.TriggerContext) error {
	metadata := NodeMetadata{}
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	//
	// If metadata is already set, skip setup
	//
	return ctx.Integration.RequestWebhook(WebhookConfiguration{})
}

func (t *OnIssue) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIssue) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIssueConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Verify signature using the client secret from the Sentry App
	signature := ctx.Headers.Get("Sentry-Hook-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	// Get the client secret from integration secrets
	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secrets: %v", err)
	}

	var clientSecret []byte
	for _, secret := range secrets {
		if secret.Name == "sentryClientSecret" {
			clientSecret = secret.Value
			break
		}
	}

	if clientSecret == nil {
		return http.StatusForbidden, fmt.Errorf("missing client secret")
	}

	// Verify signature
	if err := crypto.VerifySignature(clientSecret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
	}

	// Parse webhook payload
	var webhook SentryWebhook
	err = json.Unmarshal(ctx.Body, &webhook)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	//
	// Filter events by type - webhook may receive events for all configured issue types
	//
	if !slices.Contains(config.Events, webhook.Action) {
		return http.StatusOK, nil
	}

	err = ctx.Events.Emit(
		fmt.Sprintf("sentry.%s", webhook.Action),
		map[string]any{
			"event":      webhook.Action,
			"issue":      webhook.Data.Issue,
			"actionUser": webhook.Data.ActionUser,
		},
	)

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

// SentryWebhook represents a Sentry webhook payload
type SentryWebhook struct {
	Action string            `json:"action"` // e.g., "created", "resolved", "assigned", "ignored", "unresolved"
	Data   SentryWebhookData `json:"data"`
}

// SentryWebhookData contains the data from the webhook
type SentryWebhookData struct {
	ActionUser *map[string]any `json:"actionUser,omitempty"`
	Issue      *IssueData      `json:"issue,omitempty"`
}

// IssueData represents a Sentry issue from the webhook
type IssueData struct {
	ID       string         `json:"id"`
	ShortID  string         `json:"shortId"`
	Title    string         `json:"title"`
	Level    string         `json:"level"`
	Status   string         `json:"status"`
	Assigned map[string]any `json:"assigned,omitempty"`
	Project  *ProjectRef    `json:"project,omitempty"`
}

// ProjectRef represents a Sentry project reference
type ProjectRef struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
	return nil
}
