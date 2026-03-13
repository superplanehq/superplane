package sentry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnIssueEvent struct{}

type OnIssueEventMetadata struct {
	Project *Project `json:"project"`
}

// Service hook events supported by Sentry API
var AllEventTypes = []configuration.FieldOption{
	{Label: "Event Created", Value: "event.created"},
	{Label: "Event Alert", Value: "event.alert"},
}

type OnIssueEventConfiguration struct {
	Project    string   `json:"project" mapstructure:"project"`
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

func (t *OnIssueEvent) Name() string {
	return "sentry.onIssueEvent"
}

func (t *OnIssueEvent) Label() string {
	return "On Issue Event"
}

func (t *OnIssueEvent) Description() string {
	return "Listen to Sentry issue events"
}

func (t *OnIssueEvent) Documentation() string {
	return `The On Issue Event trigger starts a workflow execution when a Sentry event occurs.

## Overview

This trigger uses Sentry's Service Hooks API to automatically create webhooks when configured. SuperPlane will create and manage the webhook in Sentry for you - no manual setup required.

## Use Cases

- **Incident response**: Automatically create Jira tickets or PagerDuty incidents when critical errors occur
- **Slack notifications**: Alert your team channel when new events are captured
- **Deployment validation**: Monitor for new errors after deployments and auto-rollback if issues spike
- **Error aggregation**: Forward events to external logging or analytics systems

## Configuration

| Field | Description |
|-------|-------------|
| **Project** | The Sentry project to monitor for events |
| **Event Types** | Filter which event types trigger the workflow |

## Supported Event Types

| Event Type | Description |
|------------|-------------|
| ` + "`event.created`" + ` | A new event has been processed by Sentry |
| ` + "`event.alert`" + ` | An alert rule has been triggered for an event |

## Event Payload

The trigger receives the Sentry service hook payload. Key fields available in expressions:

` + "```" + `
$['trigger'].event                     # Event type: "event.created" or "event.alert"
$['trigger'].data.event.event_id       # Unique event ID
$['trigger'].data.event.message        # Event message/error
$['trigger'].data.event.level          # Severity: "error", "warning", "info"
$['trigger'].data.event.platform       # Platform (e.g., "python", "javascript")
$['trigger'].data.event.timestamp      # When the event occurred
$['trigger'].data.event.tags           # Event tags
$['trigger'].data.event.url            # Link to event in Sentry
` + "```" + `

## How It Works

1. When you create this trigger, SuperPlane automatically creates a service hook in your Sentry project
2. Sentry sends webhook notifications to SuperPlane when events occur
3. The webhook signature is automatically verified using the secret provided by Sentry
4. When the trigger is removed, SuperPlane automatically deletes the service hook from Sentry

## Requirements

Your Sentry Auth Token must have the ` + "`project:write`" + ` scope to create service hooks.

## Notes

- Webhooks must respond within 1 second or Sentry considers them timed out
- The trigger verifies webhook signatures using HMAC-SHA256 for security
- Service hooks require the 'servicehooks' feature to be enabled for your project`
}

func (t *OnIssueEvent) Icon() string {
	return "sentry"
}

func (t *OnIssueEvent) Color() string {
	return "purple"
}

func (t *OnIssueEvent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "project",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:     "eventTypes",
			Label:    "Event Types",
			Type:     configuration.FieldTypeMultiSelect,
			Required: false,
			Default:  []string{"event.created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: AllEventTypes,
				},
			},
		},
	}
}

func (t *OnIssueEvent) Setup(ctx core.TriggerContext) error {
	var metadata OnIssueEventMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// If metadata is set, it means the trigger was already setup
	if metadata.Project != nil {
		return nil
	}

	config := OnIssueEventConfiguration{}
	err = mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	// If this is the same project, nothing to do
	if metadata.Project != nil && config.Project == metadata.Project.Slug {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(config.Project)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", config.Project, err)
	}

	err = ctx.Metadata.Set(OnIssueEventMetadata{
		Project: &Project{
			ID:   project.ID,
			Slug: project.Slug,
			Name: project.Name,
		},
	})

	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		Project: project.Slug,
	})
}

func (t *OnIssueEvent) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIssueEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssueEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnIssueEventConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Verify signature using Sentry-Hook-Signature header
	signature := ctx.Headers.Get("Sentry-Hook-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	// Parse the webhook payload
	payload := map[string]any{}
	err = json.Unmarshal(ctx.Body, &payload)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	// Get the event type from the Sentry-Hook-Resource header or payload
	// Service hooks use: event.created, event.alert
	eventType := ctx.Headers.Get("Sentry-Hook-Resource")
	if eventType == "" {
		// Try to get from payload
		if ev, ok := payload["event"].(string); ok {
			eventType = ev
		}
	}

	// Normalize event type format
	if !strings.HasPrefix(eventType, "event.") && eventType != "" {
		eventType = "event." + eventType
	}

	if eventType == "" {
		return http.StatusBadRequest, fmt.Errorf("missing event type")
	}

	// Filter by configured event types
	if len(config.EventTypes) > 0 {
		if !slices.Contains(config.EventTypes, eventType) {
			ctx.Logger.Infof("event type %s does not match the allowed types: %v", eventType, config.EventTypes)
			return http.StatusOK, nil
		}
	}

	// Add event type to payload for easier access
	payload["event"] = eventType

	// Emit the event
	emitEventType := fmt.Sprintf("sentry.%s", strings.ReplaceAll(eventType, ".", "_"))
	err = ctx.Events.Emit(emitEventType, payload)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
	}

	return http.StatusOK, nil
}

func (t *OnIssueEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}
