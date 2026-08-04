package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_remove_issue_assignee.json
var exampleOutputRemoveIssueAssignee []byte

type RemoveIssueAssignee struct{}

type RemoveIssueAssigneeConfiguration struct {
	Project   string   `mapstructure:"project"`
	IssueIID  string   `mapstructure:"issueIid"`
	Assignees []string `mapstructure:"assignees"`
}

func (c *RemoveIssueAssignee) Name() string {
	return "gitlab.removeIssueAssignee"
}

func (c *RemoveIssueAssignee) Label() string {
	return "Remove Issue Assignee"
}

func (c *RemoveIssueAssignee) Description() string {
	return "Remove assignees from a GitLab issue"
}

func (c *RemoveIssueAssignee) Documentation() string {
	return `The Remove Issue Assignee component removes one or more assignees from an existing GitLab issue. Assignees that are not listed are kept.

## Use Cases

- **Automated cleanup**: Remove an assignee once their involvement is no longer needed
- **Reassignment**: Remove an assignee before adding a different one as part of a workflow
- **Rotation**: Drop an out-of-office assignee from an open issue

## Configuration

- **Project** (required): The GitLab project containing the issue
- **Issue IID** (required): The internal ID (IID) of the issue to remove assignees from (supports expressions)
- **Assignees** (required): Users to remove from the issue's assignees. Assignees not listed are kept.

## Permissions

The connected user needs at least the **Developer** role on the project to change assignees.

## Output

Returns the updated issue object, including its remaining assignee list and URL.`
}

func (c *RemoveIssueAssignee) Icon() string {
	return "gitlab"
}

func (c *RemoveIssueAssignee) Color() string {
	return "orange"
}

func (c *RemoveIssueAssignee) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *RemoveIssueAssignee) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputRemoveIssueAssignee, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *RemoveIssueAssignee) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "project",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
				},
			},
		},
		{
			Name:        "issueIid",
			Label:       "Issue IID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "1 or {{event.data.object_attributes.iid}}",
			Description: "The internal ID (IID) of the issue to remove assignees from",
		},
		{
			Name:     "assignees",
			Label:    "Assignees",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  ResourceTypeMember,
					Multi: true,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "project",
							ValueFrom: &configuration.ParameterValueFrom{Field: "project"},
						},
					},
				},
			},
		},
	}
}

func (c *RemoveIssueAssignee) Setup(ctx core.SetupContext) error {
	var config RemoveIssueAssigneeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	if config.IssueIID == "" {
		return fmt.Errorf("issue IID is required")
	}

	if len(config.Assignees) == 0 {
		return fmt.Errorf("at least one assignee is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *RemoveIssueAssignee) Execute(ctx core.ExecutionContext) error {
	var config RemoveIssueAssigneeConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	issue, err := client.GetIssue(context.Background(), config.Project, config.IssueIID)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	assigneeIDs := removeAssigneeIDs(assigneeIDsOf(issue), parseUserIDs(config.Assignees))

	updated, err := client.UpdateIssue(context.Background(), config.Project, config.IssueIID, &UpdateIssueRequest{
		AssigneeIDs: &assigneeIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to remove issue assignees: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.updateIssue",
		[]any{updated},
	)
}

func (c *RemoveIssueAssignee) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RemoveIssueAssignee) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *RemoveIssueAssignee) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RemoveIssueAssignee) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RemoveIssueAssignee) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RemoveIssueAssignee) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
