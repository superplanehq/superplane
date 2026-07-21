package gitlab

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

//go:embed example_output_create_merge_request.json
var exampleOutputCreateMergeRequest []byte

type CreateMergeRequest struct{}

type CreateMergeRequestConfiguration struct {
	Project            string   `mapstructure:"project"`
	SourceBranch       string   `mapstructure:"sourceBranch"`
	TargetBranch       string   `mapstructure:"targetBranch"`
	Title              string   `mapstructure:"title"`
	Description        string   `mapstructure:"description"`
	Assignees          []string `mapstructure:"assignees"`
	Reviewers          []string `mapstructure:"reviewers"`
	Labels             []string `mapstructure:"labels"`
	Milestone          string   `mapstructure:"milestone"`
	RemoveSourceBranch bool     `mapstructure:"removeSourceBranch"`
	Squash             bool     `mapstructure:"squash"`
}

func (c *CreateMergeRequest) Name() string {
	return "gitlab.createMergeRequest"
}

func (c *CreateMergeRequest) Label() string {
	return "Create Merge Request"
}

func (c *CreateMergeRequest) Description() string {
	return "Create a new merge request in a GitLab project"
}

func (c *CreateMergeRequest) Documentation() string {
	return `The Create Merge Request component opens a new merge request in a specified GitLab project.

## Use Cases

- **Automated fixes**: An agent commits a fix on a branch and opens a merge request for review
- **Dependency updates**: Open a merge request when a dependency or changelog bump is generated
- **Release automation**: Open a merge request to promote a branch into a release branch

## Configuration

- **Project** (required): The GitLab project where the merge request will be created
- **Source Branch** (required): The branch containing the changes (supports expressions)
- **Target Branch** (required): The branch you want the changes merged into (e.g. main)
- **Title** (required): The title of the new merge request
- **Description** (optional): The description/body of the merge request
- **Assignees** (optional): Users to assign the merge request to
- **Reviewers** (optional): Users to request a review from
- **Labels** (optional): Labels to apply to the merge request
- **Milestone** (optional): Milestone to associate with the merge request
- **Remove Source Branch** (optional): Remove the source branch when the merge request is merged
- **Squash** (optional): Squash commits into a single commit when merging

## Permissions

The connected user needs at least the **Developer** role on the project to create a merge request.

## Output

The component outputs the created merge request object, including:
- **iid**: The project-relative ID of the merge request
- **web_url**: The URL to view the merge request in GitLab
- **source_branch** and **target_branch**: The branches involved
- **state**: The current state of the merge request (opened)`
}

func (c *CreateMergeRequest) Icon() string {
	return "gitlab"
}

func (c *CreateMergeRequest) Color() string {
	return "orange"
}

func (c *CreateMergeRequest) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateMergeRequest) ExampleOutput() map[string]any {
	var example map[string]any
	if err := json.Unmarshal(exampleOutputCreateMergeRequest, &example); err != nil {
		return map[string]any{}
	}
	return example
}

func (c *CreateMergeRequest) Configuration() []configuration.Field {
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
			Name:        "sourceBranch",
			Label:       "Source Branch",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "feature/login-page or {{event.data.ref}}",
			Description: "The name of the branch containing the changes",
		},
		{
			Name:        "targetBranch",
			Label:       "Target Branch",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "main",
			Description: "The name of the branch you want the changes merged into",
		},
		{
			Name:     "title",
			Label:    "Title",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "description",
			Label:    "Description",
			Type:     configuration.FieldTypeText,
			Required: false,
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
			Name:     "reviewers",
			Label:    "Reviewers",
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
		{
			Name:        "removeSourceBranch",
			Label:       "Remove Source Branch",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Remove the source branch when the merge request is merged",
		},
		{
			Name:        "squash",
			Label:       "Squash",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Squash commits into a single commit when merging",
		},
	}
}

func (c *CreateMergeRequest) Setup(ctx core.SetupContext) error {
	var config CreateMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.Project == "" {
		return fmt.Errorf("project is required")
	}

	if config.SourceBranch == "" {
		return fmt.Errorf("source branch is required")
	}

	if config.TargetBranch == "" {
		return fmt.Errorf("target branch is required")
	}

	if config.Title == "" {
		return fmt.Errorf("title is required")
	}

	return ensureProjectInMetadata(
		ctx.Metadata,
		ctx.Integration,
		config.Project,
	)
}

func (c *CreateMergeRequest) Execute(ctx core.ExecutionContext) error {
	var config CreateMergeRequestConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize GitLab client: %w", err)
	}

	var milestoneID *int
	if config.Milestone != "" {
		var id int
		if _, err := fmt.Sscanf(config.Milestone, "%d", &id); err == nil {
			milestoneID = &id
		}
	}

	req := &CreateMergeRequestRequest{
		SourceBranch: normalizePipelineRef(config.SourceBranch),
		TargetBranch: normalizePipelineRef(config.TargetBranch),
		Title:        config.Title,
		Description:  config.Description,
		AssigneeIDs:  parseUserIDs(config.Assignees),
		ReviewerIDs:  parseUserIDs(config.Reviewers),
		Labels:       strings.Join(config.Labels, ","),
		MilestoneID:  milestoneID,
	}

	if config.RemoveSourceBranch {
		req.RemoveSourceBranch = &config.RemoveSourceBranch
	}

	if config.Squash {
		req.Squash = &config.Squash
	}

	mergeRequest, err := client.CreateMergeRequest(context.Background(), config.Project, req)
	if err != nil {
		return fmt.Errorf("failed to create merge request: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"gitlab.mergeRequest",
		[]any{mergeRequest},
	)
}

func (c *CreateMergeRequest) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateMergeRequest) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *CreateMergeRequest) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateMergeRequest) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateMergeRequest) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateMergeRequest) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
