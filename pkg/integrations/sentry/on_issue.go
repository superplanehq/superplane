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
- **data.issue**: the Sentry issue object from the webhook
- **description**: Markdown of the Sentry issue and preferred event (event link, highlights, message, stack, request, user, tags, contexts, breadcrumbs). SuperPlane fetches this from the Sentry API. Use this instead of interpolating ` + "`data.issue`" + `, which renders as a Go map
- **actor**: the user or team that triggered the event when available

## Setup

This trigger uses issue webhooks from the connected Sentry organization. SuperPlane verifies each webhook signature before it routes the event to matching triggers.`
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
	if ctx.Metadata != nil {
		if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
			return fmt.Errorf("failed to decode trigger metadata: %w", err)
		}
	}

	if ctx.Integration == nil {
		if config.Project != "" {
			return fmt.Errorf("Sentry integration is not connected")
		}
		return setOnIssueMetadata(ctx.Metadata, metadata)
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
	return setOnIssueMetadata(ctx.Metadata, metadata)
}

func (t *OnIssue) subscribe(ctx core.TriggerContext, metadata OnIssueMetadata) (*string, error) {
	if ctx.Integration == nil {
		return nil, fmt.Errorf("Sentry integration is not connected")
	}

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

func (t *OnIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (t *OnIssue) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
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

	if !slices.Contains(config.Actions, message.Action) {
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
		"description":  t.issueDescription(ctx, message.Data["issue"]),
	}

	return ctx.Events.Emit("sentry.issue", payload)
}

func (t *OnIssue) issueDescription(ctx core.IntegrationMessageContext, issue any) string {
	if ctx.HTTP == nil || ctx.Integration == nil {
		return IssueDescription(issue, nil)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to create sentry client for issue enrichment: %v", err)
		}
		return IssueDescription(issue, nil)
	}

	return FetchedIssueDescription(client, issue, ctx.Logger)
}

func (t *OnIssue) Cleanup(ctx core.TriggerContext) error {
	// Integration subscriptions are tied to the node lifecycle and are cleaned up by the platform.
	return nil
}

func setOnIssueMetadata(writer core.MetadataWriter, metadata OnIssueMetadata) error {
	if writer == nil {
		return nil
	}

	return writer.Set(metadata)
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
	if integration == nil {
		return nil
	}

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
