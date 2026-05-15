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

const DeleteIssuePayloadType = "jira.issueDeleted"

type DeleteIssue struct{}

type DeleteIssueSpec struct {
	Project        string `json:"project" mapstructure:"project"`
	IssueKey       string `json:"issueKey" mapstructure:"issueKey"`
	DeleteSubtasks bool   `json:"deleteSubtasks" mapstructure:"deleteSubtasks"`
}

type DeleteIssueOutput struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Deleted bool   `json:"deleted"`
}

func (c *DeleteIssue) Name() string {
	return "jira.deleteIssue"
}

func (c *DeleteIssue) Label() string {
	return "Delete Issue"
}

func (c *DeleteIssue) Description() string {
	return "Delete a Jira issue"
}

func (c *DeleteIssue) Documentation() string {
	return `The Delete Issue component permanently removes an issue from Jira.

## Use Cases

- **Cleanup**: remove placeholder or duplicate issues created by automated flows
- **CRUD completion**: pair with create/update flows for full lifecycle management

## Configuration

- **Project**: The Jira project the issue belongs to
- **Issue Key**: The issue key (e.g. ` + "`PROJ-123`" + `)
- **Delete Subtasks**: Also delete the issue's subtasks (Jira returns an error if subtasks exist and this is false)

## Output

Returns the deleted issue's ` + "`id`" + ` and ` + "`key`" + `, plus ` + "`deleted: true`" + `.`
}

func (c *DeleteIssue) Icon() string {
	return "jira"
}

func (c *DeleteIssue) Color() string {
	return "red"
}

func (c *DeleteIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteIssue) Configuration() []configuration.Field {
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
			Name:        "deleteSubtasks",
			Label:       "Delete Subtasks",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Also delete the issue's subtasks",
			Default:     false,
		},
	}
}

func (c *DeleteIssue) Setup(ctx core.SetupContext) error {
	spec := DeleteIssueSpec{}
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

func (c *DeleteIssue) Execute(ctx core.ExecutionContext) error {
	spec := DeleteIssueSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	issueKey := strings.TrimSpace(spec.IssueKey)
	if issueKey == "" {
		return fmt.Errorf("issueKey is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	issue, err := client.GetIssue(issueKey)
	if err != nil {
		return fmt.Errorf("failed to fetch issue before delete: %v", err)
	}

	if err := client.DeleteIssue(issueKey, DeleteIssueOptions{DeleteSubtasks: spec.DeleteSubtasks}); err != nil {
		return fmt.Errorf("failed to delete issue: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeleteIssuePayloadType,
		[]any{DeleteIssueOutput{ID: issue.ID, Key: issue.Key, Deleted: true}},
	)
}

func (c *DeleteIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
