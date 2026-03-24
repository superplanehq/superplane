package sentry

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnIssue struct{}

type OnIssueConfiguration struct {
	Project string   `json:"project" mapstructure:"project"`
	Actions []string `json:"actions" mapstructure:"actions"`
}

type OnIssueMetadata struct {
	AppSubscriptionID *string         `json:"appSubscriptionID,omitempty" mapstructure:"appSubscriptionID,omitempty"`
	Project           *ProjectSummary `json:"project,omitempty" mapstructure:"project,omitempty"`
}

func (t *OnIssue) Name() string {
	return "sentry.onIssue"
}

func (t *OnIssue) Label() string {
	return "On Issue Event"
}

func (t *OnIssue) Description() string {
	return "Listen to issue webhooks from Sentry"
}

func (t *OnIssue) Documentation() string {
	return `The On Issue Event trigger starts a workflow execution when Sentry sends issue webhooks for the connected organization.

## Use Cases

- **Escalation workflows**: react when a new issue is created in Sentry
- **Triage automation**: assign follow-up actions when issues are assigned or resolved
- **Cross-tool sync**: mirror Sentry issue state changes into incident or ticketing systems

## Configuration

- **Project**: Optionally limit the trigger to a single Sentry project
- **Actions**: Select which issue actions should trigger the workflow

## Event Data

The trigger emits the full Sentry webhook payload, including:
- **action**: the issue event action
- **data.issue**: the Sentry issue object
- **actor**: the user or team that triggered the event when available

## Setup

This trigger uses the webhook URL configured on your Sentry internal integration. SuperPlane verifies each webhook signature using your Sentry client secret before routing the event to matching triggers.`
}

func (t *OnIssue) Icon() string {
	return "bug"
}

func (t *OnIssue) Color() string {
	return "gray"
}

func (t *OnIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Only trigger for issues in this project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "actions",
			Label:       "Actions",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    true,
			Description: "Issue actions to listen for",
			Default:     []string{"created"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Created", Value: "created"},
						{Label: "Assigned", Value: "assigned"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Archived", Value: "archived"},
						{Label: "Unresolved", Value: "unresolved"},
					},
				},
			},
		},
	}
}

func (t *OnIssue) Setup(ctx core.TriggerContext) error {
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadata := OnIssueMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode trigger metadata: %w", err)
	}

	if config.Project != "" {
		project := findProject(ctx.Integration, config.Project)
		if project == nil {
			return fmt.Errorf("project %q was not found in the connected Sentry organization", config.Project)
		}
		metadata.Project = project
	} else {
		metadata.Project = nil
	}

	subscriptionID, err := t.subscribe(ctx, metadata)
	if err != nil {
		return err
	}

	metadata.AppSubscriptionID = subscriptionID
	return ctx.Metadata.Set(metadata)
}

func (t *OnIssue) subscribe(ctx core.TriggerContext, metadata OnIssueMetadata) (*string, error) {
	if metadata.AppSubscriptionID != nil {
		// Verify the subscription still exists — it may be gone if the integration was
		// deleted and re-created. If the current integration has no subscriptions, create one.
		existing, err := ctx.Integration.ListSubscriptions()
		if err == nil && len(existing) > 0 {
			return metadata.AppSubscriptionID, nil
		}
	}

	subscriptionID, err := ctx.Integration.Subscribe(SubscriptionConfiguration{
		Resources: []string{"issue"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to Sentry issue events: %w", err)
	}

	value := subscriptionID.String()
	return &value, nil
}

func (t *OnIssue) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnIssue) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (t *OnIssue) OnIntegrationMessage(ctx core.IntegrationMessageContext) error {
	config := OnIssueConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	message, err := decodeWebhookMessage(ctx.Message)
	if err != nil {
		return err
	}

	if message.Resource != "issue" {
		return nil
	}

	if len(config.Actions) > 0 && !slices.Contains(config.Actions, message.Action) {
		return nil
	}

	projectSlug := issueProjectSlug(message.Data)
	if config.Project != "" && config.Project != projectSlug {
		return nil
	}

	payload := map[string]any{
		"resource":     message.Resource,
		"action":       message.Action,
		"installation": message.Installation,
		"data":         message.Data,
		"actor":        message.Actor,
		"timestamp":    eventTimestamp(message),
	}

	return ctx.Events.Emit("sentry.issue", payload)
}

func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
	// Integration subscriptions are tied to the node lifecycle and are cleaned up by the platform.
	return nil
}

func decodeWebhookMessage(message any) (*WebhookMessage, error) {
	switch value := message.(type) {
	case WebhookMessage:
		return &value, nil
	case *WebhookMessage:
		return value, nil
	default:
		decoded := WebhookMessage{}
		if err := mapstructure.Decode(message, &decoded); err != nil {
			return nil, fmt.Errorf("failed to decode sentry webhook message: %w", err)
		}
		return &decoded, nil
	}
}

func eventTimestamp(message *WebhookMessage) string {
	if message != nil && message.Timestamp != "" {
		return message.Timestamp
	}

	if message == nil {
		return ""
	}

	return issueTimestamp(message.Data)
}

func issueTimestamp(data map[string]any) string {
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return ""
	}
	if ts, ok := issue["lastSeen"].(string); ok && ts != "" {
		return ts
	}
	if ts, ok := issue["firstSeen"].(string); ok {
		return ts
	}
	return ""
}

func issueProjectSlug(data map[string]any) string {
	issue, ok := data["issue"].(map[string]any)
	if !ok {
		return ""
	}

	project, ok := issue["project"].(map[string]any)
	if !ok {
		return ""
	}

	slug, _ := project["slug"].(string)
	return slug
}

func findProject(integration core.IntegrationContext, slug string) *ProjectSummary {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return nil
	}

	for _, project := range metadata.Projects {
		if project.Slug == slug {
			copy := project
			return &copy
		}
	}

	return nil
}
