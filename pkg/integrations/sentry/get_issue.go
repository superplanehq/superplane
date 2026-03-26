package sentry

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetIssue struct{}

type GetIssueConfiguration struct {
	IssueID string `json:"issueId" mapstructure:"issueId"`
}

type GetIssueNodeMetadata struct {
	IssueTitle string `json:"issueTitle,omitempty" mapstructure:"issueTitle"`
}

func (c *GetIssue) Name() string {
	return "sentry.getIssue"
}

func (c *GetIssue) Label() string {
	return "Get Issue"
}

func (c *GetIssue) Description() string {
	return "Retrieve a Sentry issue with tags, frequency, assignee, and recent events"
}

func (c *GetIssue) Documentation() string {
	return `The Get Issue component retrieves a Sentry issue and enriches it with recent events for downstream routing and escalation.

## Use Cases

- **Routing decisions**: inspect assignee, status, project, and issue frequency before branching
- **Escalation context**: include recent events and tags in notifications or ticket creation
- **Release correlation**: check whether an issue is already tied to a release before actioning it

## Configuration

- **Issue**: Select the Sentry issue to retrieve

## Output

Returns the Sentry issue object including:
- issue metadata such as title, status, assignee, tags, and frequency stats
- recent issue events for additional context`
}

func (c *GetIssue) Icon() string {
	return "bug"
}

func (c *GetIssue) Color() string {
	return "gray"
}

func (c *GetIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "issueId",
			Label:       "Issue",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select the Sentry issue to retrieve",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIssue,
				},
			},
		},
	}
}

func (c *GetIssue) Setup(ctx core.SetupContext) error {
	config, err := decodeGetIssueConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.IssueID) == "" {
		return fmt.Errorf("issueId is required")
	}

	if isExpressionValue(config.IssueID) {
		return ctx.Metadata.Set(GetIssueNodeMetadata{})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	issue, err := client.GetIssue(config.IssueID)
	if err != nil {
		return fmt.Errorf("failed to retrieve sentry issue: %w", err)
	}

	return ctx.Metadata.Set(GetIssueNodeMetadata{
		IssueTitle: displayIssueLabel(issue.ShortID, issue.Title),
	})
}

func (c *GetIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIssue) Execute(ctx core.ExecutionContext) error {
	config, err := decodeGetIssueConfiguration(ctx.Configuration)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.IssueID) == "" {
		return fmt.Errorf("issueId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create sentry client: %w", err)
	}

	issue, err := client.GetIssue(config.IssueID)
	if err != nil {
		return fmt.Errorf("failed to retrieve sentry issue: %w", err)
	}

	events, err := client.ListIssueEvents(config.IssueID)
	if err != nil {
		return fmt.Errorf("failed to retrieve sentry issue events: %w", err)
	}

	issue.Events = events

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "sentry.issue", []any{issue})
}

func (c *GetIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func decodeGetIssueConfiguration(input any) (GetIssueConfiguration, error) {
	config := GetIssueConfiguration{}
	if err := mapstructure.Decode(input, &config); err != nil {
		return GetIssueConfiguration{}, err
	}

	config.IssueID = strings.TrimSpace(config.IssueID)
	return config, nil
}
