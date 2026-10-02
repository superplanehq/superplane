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

//go:embed example_output_add_issue_assignee.json
var exampleOutputAddIssueAssignee []byte

type AddIssueAssignee struct{}

type AddIssueAssigneeConfiguration struct {
	Project   string   `mapstructure:"project"`
	IssueIID  string   `mapstructure:"issueIid"`
	Assignees []string `mapstructure:"assignees"`
}

func (c *AddIssueAssignee) Name() string {
	return "gitlab.addIssueAssignee"
}

func (c *AddIssueAssignee) Label() string {
	return "Add Issue Assignee"
}

func (c *AddIssueAssignee) Description() string {
	return "Add assignees to a GitLab issue"
}

func (c *AddIssueAssignee) Documentation() string {
	return `The Add Issue Assignee component adds one or more assignees to an existing GitLab issue. Existing assignees are kept.

## Use Cases

- **Auto-assignment**: Automatically assign issues to team members based on workflow triggers
- **Escalation**: Add an additional assignee when an issue requires attention from a specific person
- **On-call routing**: Assign issues to the current on-call engineer

## Configuration

- **Project** (required): The GitLab project containing the issue
- **Issue IID** (required): The internal ID (IID) of the issue to add assignees to (supports expressions)
- **Assignees** (required): Users to assign the issue to. These are added to any existing assignees.

## Permissions

The connected user needs at least the **Developer** role on the project to change assignees.

## Output

Returns the updated issue object, including its full assignee list and URL.`
}

func (c *AddIssueAssignee) Icon() string {
	return "gitlab"
}

func (c *AddIssueAssignee) Color() string {
	return "orange"
}

func (c *AddIssueAssignee) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AddIssueAssignee) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputAddIssueAssignee, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *AddIssueAssignee) Configuration() []configuration.Field {
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
			Description: "The internal ID (IID) of the issue to add assignees to",
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

func (c *AddIssueAssignee) Setup(ctx core.SetupContext) error {
	var config AddIssueAssigneeConfiguration
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

func (c *AddIssueAssignee) Execute(ctx core.ExecutionContext) error {
	var config AddIssueAssigneeConfiguration
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

	assigneeIDs := mergeAssigneeIDs(assigneeIDsOf(issue), parseUserIDs(config.Assignees))

	updated, err := client.UpdateIssue(context.Background(), config.Project, config.IssueIID, &UpdateIssueRequest{
		AssigneeIDs: &assigneeIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to add issue assignees: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.updateIssue",
		[]any{updated},
	)
}

func (c *AddIssueAssignee) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AddIssueAssignee) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *AddIssueAssignee) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AddIssueAssignee) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *AddIssueAssignee) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *AddIssueAssignee) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
