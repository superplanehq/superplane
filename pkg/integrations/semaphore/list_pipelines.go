package semaphore

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListPipelines struct{}

type ListPipelinesSpec struct {
	Project       string `json:"project"`
	BranchName    string `json:"branchName,omitempty"`
	YMLFilePath   string `json:"ymlFilePath,omitempty"`
	CreatedAfter  string `json:"createdAfter,omitempty"`
	CreatedBefore string `json:"createdBefore,omitempty"`
	DoneAfter     string `json:"doneAfter,omitempty"`
	DoneBefore    string `json:"doneBefore,omitempty"`
	Limit         int    `json:"limit,omitempty"`
}

type ListPipelinesNodeMetadata struct {
	Project *Project `json:"project" mapstructure:"project"`
}

func (l *ListPipelines) Name() string {
	return "semaphore.listPipelines"
}

func (l *ListPipelines) Label() string {
	return "List Pipelines"
}

func (l *ListPipelines) Description() string {
	return "List pipelines for a Semaphore project with optional filtering"
}

func (l *ListPipelines) Documentation() string {
	return `The List Pipelines component retrieves a list of pipelines from a Semaphore project.

## Use Cases

- **Dashboard creation**: Build a dashboard or status node showing recent pipeline runs
- **Find latest pipeline**: Find the latest pipeline for a given branch or ref
- **Iterate and process**: List pipelines then filter/iterate to trigger follow-up actions or report on failures
- **Pipeline discovery**: Feed pipeline IDs into Get Pipeline or other nodes

## How It Works

1. Queries the Semaphore API for pipelines matching the specified criteria
2. Optionally filters by branch name, pipeline file, and date ranges
3. Emits a single array of pipeline objects to the default output channel
4. Supports pagination with a maximum limit of 100 pipelines

## Configuration

- **Project** (required): Semaphore project ID or name
- **Branch Name** (optional): Filter by branch name
- **YML File Path** (optional): Filter by pipeline definition file (e.g., .semaphore/semaphore.yml)
- **Created After** (optional): Only pipelines created after this time (Unix timestamp or ISO date)
- **Created Before** (optional): Only pipelines created before this time (Unix timestamp or ISO date)
- **Done After** (optional): Only pipelines finished after this time (Unix timestamp or ISO date)
- **Done Before** (optional): Only pipelines finished before this time (Unix timestamp or ISO date)
- **Limit** (optional): Maximum number of pipelines to return (max 100, default 30)

## Output

Emits a single payload with a pipelines array containing objects with fields:
- **ppl_id**: Pipeline ID
- **wf_id**: Workflow ID
- **name**: Pipeline name
- **state**: Pipeline state (done, running, etc.)
- **result**: Pipeline result (passed, failed, stopped, etc.)
- Plus any additional fields from the API

## Notes

- Filters are combined with AND logic
- The component respects the API's pagination limits
- Empty results will emit an empty array, not an error`
}

func (l *ListPipelines) Icon() string {
	return "list"
}

func (l *ListPipelines) Color() string {
	return "gray"
}

func (l *ListPipelines) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *ListPipelines) Configuration() []configuration.Field {
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
			Name:        "branchName",
			Label:       "Branch Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g. main, develop",
			Description: "Filter by branch name",
		},
		{
			Name:        "ymlFilePath",
			Label:       "YML File Path",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g. .semaphore/semaphore.yml",
			Description: "Filter by pipeline definition file",
		},
		{
			Name:        "createdAfter",
			Label:       "Created After",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Unix timestamp or ISO date",
			Description: "Only pipelines created after this time",
		},
		{
			Name:        "createdBefore",
			Label:       "Created Before",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Unix timestamp or ISO date",
			Description: "Only pipelines created before this time",
		},
		{
			Name:        "doneAfter",
			Label:       "Done After",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Unix timestamp or ISO date",
			Description: "Only pipelines finished after this time",
		},
		{
			Name:        "doneBefore",
			Label:       "Done Before",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "Unix timestamp or ISO date",
			Description: "Only pipelines finished before this time",
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Placeholder: "default 30, max 100",
			Description: "Maximum number of pipelines to return",
		},
	}
}

func (l *ListPipelines) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListPipelines) Setup(ctx core.SetupContext) error {
	spec := ListPipelinesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Project == "" {
		return fmt.Errorf("project is required")
	}

	metadata := ListPipelinesNodeMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// If this is the same project, nothing to do.
	if metadata.Project != nil && (spec.Project == metadata.Project.ID || spec.Project == metadata.Project.Name) {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	project, err := client.GetProject(spec.Project)
	if err != nil {
		return fmt.Errorf("error finding project %s: %v", spec.Project, err)
	}

	err = ctx.Metadata.Set(ListPipelinesNodeMetadata{
		Project: &Project{
			ID:   project.Metadata.ProjectID,
			Name: project.Metadata.ProjectName,
			URL:  fmt.Sprintf("%s/projects/%s", string(client.OrgURL), project.Metadata.ProjectID),
		},
	})

	if err != nil {
		return fmt.Errorf("error setting metadata: %v", err)
	}

	return nil
}

func (l *ListPipelines) Execute(ctx core.ExecutionContext) error {
	spec := ListPipelinesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	metadata := ListPipelinesNodeMetadata{}
	err = mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	// Determine limit - use default of 30 if not specified, max 100
	limit := spec.Limit
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	params := &ListPipelinesParams{
		ProjectID:     metadata.Project.ID,
		BranchName:    spec.BranchName,
		YMLFilePath:   spec.YMLFilePath,
		CreatedAfter:  spec.CreatedAfter,
		CreatedBefore: spec.CreatedBefore,
		DoneAfter:     spec.DoneAfter,
		DoneBefore:    spec.DoneBefore,
		Limit:         limit,
	}

	pipelines, err := client.ListPipelinesWithFilters(params)
	if err != nil {
		return fmt.Errorf("error listing pipelines: %v", err)
	}

	if ctx.Logger != nil {
		ctx.Logger.Infof("Listed %d pipelines for project %s", len(pipelines), metadata.Project.ID)
	}

	responseData := map[string]any{
		"pipelines": pipelines,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"semaphore.pipelines.listed",
		[]any{responseData},
	)
}

func (l *ListPipelines) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (l *ListPipelines) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListPipelines) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListPipelines) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions supported")
}

func (l *ListPipelines) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 404, fmt.Errorf("webhooks not supported")
}
