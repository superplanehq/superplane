package sentry

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssueEvent struct{}

var AllEventTypes = []configuration.FieldOption{
	{Label: "Created", Value: "issue.created"},
	{Label: "Resolved", Value: "issue.resolved"},
	{Label: "Assigned", Value: "issue.assigned"},
	{Label: "Ignored", Value: "issue.ignored"},
	{Label: "Unresolved", Value: "issue.unresolved"},
}

type OnIssueEventConfiguration struct {
	Project    string   `json:"project" mapstructure:"project"`
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

type OnIssueEventMetadata struct {
	SubscriptionID string   `json:"subscriptionId,omitempty" mapstructure:"subscriptionId"`
	Project        string   `json:"project"`
	EventTypes     []string `json:"eventTypes"`
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
	return `The On Issue Event trigger starts a workflow execution when a Sentry issue changes state.

## Overview

This trigger listens for issue lifecycle events from Sentry — when issues are created, resolved, assigned, ignored, or become unresolved again. SuperPlane automatically creates and manages the webhook in Sentry for you.

## Use Cases

- **Incident response**: Create Jira tickets or PagerDuty incidents when a new issue is created
- **Slack notifications**: Alert your team when a critical issue is assigned
- **Auto-close tickets**: Close linked tickets when an issue is resolved
- **Regression alerts**: Notify on-call when a previously resolved issue becomes unresolved

## Configuration

| Field | Description |
|-------|-------------|
| **Project** | The Sentry project to monitor |
| **Event Types** | Filter which issue lifecycle events trigger the workflow (all selected by default) |

## Supported Event Types

| Event Type | Description |
|------------|-------------|
| ` + "`issue.created`" + ` | A new issue has been created in Sentry |
| ` + "`issue.resolved`" + ` | An issue has been marked as resolved |
| ` + "`issue.assigned`" + ` | An issue has been assigned to a user or team |
| ` + "`issue.ignored`" + ` | An issue has been ignored/archived |
| ` + "`issue.unresolved`" + ` | A previously resolved or ignored issue has become active again |

## Event Payload

Key fields available in expressions:

` + "```" + `
$['trigger'].event              # Event type, e.g. "issue.created"
$['trigger'].issue.id           # Sentry issue ID
$['trigger'].issue.shortId      # Short ID like "PROJ-123"
$['trigger'].issue.title        # Issue title
$['trigger'].issue.status       # Issue status
$['trigger'].issue.level        # Severity: "error", "warning", "info"
$['trigger'].issue.project.slug # Project slug
$['trigger'].actionUser         # User who triggered the event (if applicable)
` + "```" + `

## How It Works

1. When you install the Sentry integration, SuperPlane automatically creates a Sentry Internal Integration with webhook
2. Sentry sends webhook notifications to SuperPlane when issues change state
3. Each trigger filters events by project and event type
4. When the integration is uninstalled, SuperPlane automatically cleans up the Sentry App

## Requirements

Your Sentry Auth Token must have the ` + "`org:admin`" + ` scope to create Internal Integrations.`
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
			Default:  []string{"issue.created", "issue.resolved", "issue.assigned", "issue.ignored", "issue.unresolved"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: AllEventTypes,
				},
			},
		},
	}
}

func (t *OnIssueEvent) Setup(ctx core.TriggerContext) error {
	config := OnIssueEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	metadata := OnIssueEventMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err == nil {
		if metadata.SubscriptionID != "" {
			metadata.Project = config.Project
			metadata.EventTypes = config.EventTypes
			return ctx.Metadata.Set(metadata)
		}
	}

	//
	// NOTE: we don't include anything in the subscription itself for now.
	// All the filters are applied as part of OnIntegrationMessage().
	//
	subscriptionID, err := ctx.Integration.Subscribe(SubscriptionConfiguration{})
	if err != nil {
		return fmt.Errorf("failed to subscribe to sentry notifications: %w", err)
	}

	metadata.SubscriptionID = subscriptionID.String()
	metadata.Project = config.Project
	metadata.EventTypes = config.EventTypes

	return ctx.Metadata.Set(metadata)
}

func (t *OnIssueEvent) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIssueEvent) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssueEvent) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	// no-op, since events are received through the integration
	// and routed to OnIntegrationMessage()
	return http.StatusOK, nil
}

func (t *OnIssueEvent) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnIssueEventConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	ctx.Logger.Infof("Received issue event: %+v", ctx.Message)

	payload, ok := ctx.Message.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid message format")
	}

	var eventType string
	if action, ok := payload["action"].(string); ok && action != "" {
		eventType = "issue." + action
	} else if ev, ok := payload["event"].(string); ok {
		eventType = ev
	}

	if eventType == "" {
		ctx.Logger.Infof("Missing event type, ignoring message")
		return nil
	}

	if len(config.EventTypes) > 0 && !slices.Contains(config.EventTypes, eventType) {
		ctx.Logger.Infof("Ignoring event type %s (allowed: %v)", eventType, config.EventTypes)
		return nil
	}

	data, _ := payload["data"].(map[string]any)
	issue, _ := data["issue"].(map[string]any)
	project, _ := issue["project"].(map[string]any)
	projectSlug, _ := project["slug"].(string)

	if projectSlug != config.Project {
		ctx.Logger.Infof("Ignoring event for project %s (configured: %s)", projectSlug, config.Project)
		return nil
	}

	payload["event"] = eventType

	emitEventType := fmt.Sprintf("sentry.%s", strings.ReplaceAll(eventType, ".", "_"))

	return ctx.Events.Emit(emitEventType, payload)
}

func (t *OnIssueEvent) Cleanup(ctx core.TriggerContext) error {
	return nil
}
