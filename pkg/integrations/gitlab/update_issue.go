package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssue []byte

type UpdateIssue struct{}

const (
	IssueStateEventClose  = "close"
	IssueStateEventReopen = "reopen"
)

type UpdateIssueConfiguration struct {
	Project   string   `mapstructure:"project"`
	IssueIID  string   `mapstructure:"issueIid"`
	Title     string   `mapstructure:"title"`
	Body      string   `mapstructure:"body"`
	State     string   `mapstructure:"state"`
	Labels    []string `mapstructure:"labels"`
	Assignees []string `mapstructure:"assignees"`
	Milestone string   `mapstructure:"milestone"`
}

func (c *UpdateIssue) Name() string {
	return "gitlab.updateIssue"
}

func (c *UpdateIssue) Label() string {
	return "Update Issue"
}

func (c *UpdateIssue) Description() string {
	return "Update an existing GitLab issue"
}

func (c *UpdateIssue) Documentation() string {
	return `The Update Issue component modifies an existing GitLab issue: its title, description, state, labels, assignees, or milestone.

## Use Cases

- **Status updates**: Close or reopen an issue based on workflow results
- **Label management**: Apply labels to an issue automatically
- **Assignee updates**: Assign an issue to team members automatically
- **Content updates**: Update the issue title or description with new information

## Configuration

- **Project** (required): The GitLab project containing the issue
- **Issue IID** (required): The internal ID (IID) of the issue to update (supports expressions)
- **Title** (optional): New title for the issue
- **Description** (optional): New description for the issue
- **State** (optional): Close or reopen the issue. Leave unset to keep the current state.
- **Labels** (optional): Labels to set on the issue, replacing any existing labels
- **Assignees** (optional): Users to assign the issue to, replacing any existing assignees
- **Milestone** (optional): Milestone to associate with the issue

Fields left empty are not changed.

## Output

Returns the updated issue object.`
}

func (c *UpdateIssue) Icon() string {
	return "gitlab"
}

func (c *UpdateIssue) Color() string {
	return "orange"
}

func (c *UpdateIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIssue) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputUpdateIssue, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *UpdateIssue) Configuration() []configuration.Field {
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
			Description: "The internal ID (IID) of the issue to update",
		},
		{
			Name:     "title",
			Label:    "Title",
			Type:     configuration.FieldTypeString,
			Required: false,
		},
		{
			Name:     "body",
			Label:    "Description",
			Type:     configuration.FieldTypeText,
			Required: false,
		},
		{
			Name:        "state",
			Label:       "State",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Leave unset to keep the current state",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Close", Value: IssueStateEventClose},
						{Label: "Reopen", Value: IssueStateEventReopen},
					},
				},
			},
		},
		{
			Name:     "labels",
			Label:    "Labels",
			Type:     configuration.FieldTypeList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:     "assignees",
			Label:    "Assignees",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
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
		{
			Name:     "milestone",
			Label:    "Milestone",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeMilestone,
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

func (c *UpdateIssue) Setup(ctx core.SetupContext) error {
	var config UpdateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return errors.New("project is required")
	}

	if config.IssueIID == "" {
		return errors.New("issue IID is required")
	}

	if config.State != "" && config.State != IssueStateEventClose && config.State != IssueStateEventReopen {
		return fmt.Errorf("invalid state: %s", config.State)
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *UpdateIssue) Execute(ctx core.ExecutionContext) error {
	var config UpdateIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.State != "" && config.State != IssueStateEventClose && config.State != IssueStateEventReopen {
		return fmt.Errorf("invalid state: %s", config.State)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	var assigneeIDs []int
	for _, idStr := range config.Assignees {
		var id int
		if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
			assigneeIDs = append(assigneeIDs, id)
		}
	}

	var milestoneID *int
	if config.Milestone != "" {
		id, err := strconv.Atoi(config.Milestone)
		if err == nil {
			milestoneID = &id
		}
	}

	req := &UpdateIssueRequest{
		Title:       config.Title,
		Description: config.Body,
		StateEvent:  config.State,
		Labels:      strings.Join(config.Labels, ","),
		AssigneeIDs: assigneeIDs,
		MilestoneID: milestoneID,
	}

	issue, err := client.UpdateIssue(context.Background(), config.Project, config.IssueIID, req)
	if err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.updateIssue",
		[]any{issue},
	)
}

func (c *UpdateIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *UpdateIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateIssue) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateIssue) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
