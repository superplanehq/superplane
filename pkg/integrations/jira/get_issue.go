package jira

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetIssuePayloadType = "jira.issue"

type GetIssue struct{}

type GetIssueSpec struct {
	Project  string `json:"project" mapstructure:"project"`
	IssueKey string `json:"issueKey" mapstructure:"issueKey"`
	Expand   string `json:"expand" mapstructure:"expand"`
}

func (c *GetIssue) Name() string {
	return "jira.getIssue"
}

func (c *GetIssue) Label() string {
	return "Get Issue"
}

func (c *GetIssue) Description() string {
	return "Retrieve a Jira issue by its key"
}

func (c *GetIssue) Documentation() string {
	return `The Get Issue component retrieves a Jira issue and its fields for downstream routing and inspection.

## Use Cases

- **Routing decisions**: inspect status, assignee, labels, or priority before branching
- **Escalation context**: include issue details in notifications or downstream tickets
- **Cross-tool sync**: enrich workflows that mirror Jira state into other systems

## Configuration

- **Project**: The Jira project the issue belongs to
- **Issue Key**: The issue key (e.g. ` + "`PROJ-123`" + `)
- **Expand**: Optional comma-separated Jira expand values, such as ` + "`renderedFields,names`" + `

## Output

Returns the Jira issue object including ` + "`id`" + `, ` + "`key`" + `, ` + "`self`" + ` and the full ` + "`fields`" + ` map.`
}

func (c *GetIssue) Icon() string {
	return "jira"
}

func (c *GetIssue) Color() string {
	return "blue"
}

func (c *GetIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "project",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Jira project the issue belongs to",
			Placeholder: "Select a project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
		},
		{
			Name:        "issueKey",
			Label:       "Issue Key",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The issue key (e.g. PROJ-123)",
			Placeholder: "PROJ-123",
		},
		{
			Name:        "expand",
			Label:       "Expand",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional comma-separated list of Jira expand values",
			Placeholder: "renderedFields,names",
		},
	}
}

func (c *GetIssue) Setup(ctx core.SetupContext) error {
	spec := GetIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.Project) == "" {
		return fmt.Errorf("project is required")
	}

	if strings.TrimSpace(spec.IssueKey) == "" {
		return fmt.Errorf("issueKey is required")
	}

	project, err := requireProject(ctx.HTTP, ctx.Integration, spec.Project)
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(NodeMetadata{Project: project})
}

func (c *GetIssue) Execute(ctx core.ExecutionContext) error {
	spec := GetIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if strings.TrimSpace(spec.IssueKey) == "" {
		return fmt.Errorf("issueKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	issue, err := client.GetIssueWithOptions(strings.TrimSpace(spec.IssueKey), GetIssueOptions{
		Expand: strings.TrimSpace(spec.Expand),
	})
	if err != nil {
		return fmt.Errorf("failed to get issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetIssuePayloadType,
		[]any{issue},
	)
}

func (c *GetIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
