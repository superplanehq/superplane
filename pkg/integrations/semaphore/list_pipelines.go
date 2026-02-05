package semaphore

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ListPipelinesPayloadType = "semaphore.pipelines"
const ListPipelinesSuccessChannel = "success"
const ListPipelinesDefaultLimit = 30
const ListPipelinesMaxLimit = 100

type ListPipelines struct{}

type ListPipelinesSpec struct {
	Project       string `json:"project" mapstructure:"project"`
	BranchName    string `json:"branchName" mapstructure:"branchName"`
	YmlFilePath   string `json:"ymlFilePath" mapstructure:"ymlFilePath"`
	CreatedAfter  string `json:"createdAfter" mapstructure:"createdAfter"`
	CreatedBefore string `json:"createdBefore" mapstructure:"createdBefore"`
	DoneAfter     string `json:"doneAfter" mapstructure:"doneAfter"`
	DoneBefore    string `json:"doneBefore" mapstructure:"doneBefore"`
	Limit         int    `json:"limit" mapstructure:"limit"`
}

type ListPipelinesNodeMetadata struct {
	Project *Project `json:"project" mapstructure:"project"`
}

type ListPipelinesOutput struct {
	Pipelines []any `json:"pipelines"`
	Count     int   `json:"count"`
}

func (l *ListPipelines) Name() string {
	return "semaphore.listPipelines"
}

func (l *ListPipelines) Label() string {
	return "List Pipelines"
}

func (l *ListPipelines) Description() string {
	return "List pipelines for a Semaphore project"
}

func (l *ListPipelines) Documentation() string {
	return `The List Pipelines component lists pipelines for a Semaphore project with optional filtering.

## Use Cases

- **Dashboard/status**: Build a dashboard or status node that shows recent pipeline runs for a project (e.g. last 10 pipelines).
- **Latest pipeline lookup**: Find the latest pipeline for a given branch or ref when the trigger only provides project context.
- **Batch processing**: Iterate over pipelines (e.g. list then filter by result) to trigger follow-up actions or report on failures.

## How It Works

1. Queries the Semaphore API for pipelines matching the project and filters
2. Applies pagination and limit settings
3. Returns the list of pipelines with their IDs, names, states, and results
4. Emits the data on the success channel for downstream processing

## Configuration

- **Project** (required): Semaphore project ID or name
- **Branch Name** (optional): Filter by branch name
- **YML File Path** (optional): Filter by pipeline definition file (e.g. .semaphore/semaphore.yml)
- **Created After** (optional): Only pipelines created after this time (Unix timestamp or ISO date)
- **Created Before** (optional): Only pipelines created before this time (Unix timestamp or ISO date)
- **Done After** (optional): Only pipelines finished after this time (Unix timestamp or ISO date)
- **Done Before** (optional): Only pipelines finished before this time (Unix timestamp or ISO date)
- **Limit** (optional): Maximum number of pipelines to return. Default is 30, maximum is 100.

## Output

Single output channel that emits:
- ` + "`pipelines`" + `: Array of pipeline objects with ppl_id, wf_id, name, state, result, and other API fields
- ` + "`count`" + `: Number of pipelines returned

## Notes

- Downstream nodes can filter or process the pipeline list using expressions
- If the project doesn't exist, an error is returned
- Results are returned in the order provided by the Semaphore API (typically newest first)`
}

func (l *ListPipelines) Icon() string {
	return "list"
}

func (l *ListPipelines) Color() string {
	return "gray"
}

func (l *ListPipelines) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  ListPipelinesSuccessChannel,
			Label: "Success",
		},
	}
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
			Placeholder: "e.g. main, feature/new-feature",
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
			Default:     ListPipelinesDefaultLimit,
			Placeholder: fmt.Sprintf("Default: %d, Max: %d", ListPipelinesDefaultLimit, ListPipelinesMaxLimit),
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

	if spec.Limit < 0 {
		return fmt.Errorf("limit must be a positive number")
	}

	if spec.Limit > ListPipelinesMaxLimit {
		return fmt.Errorf("limit cannot exceed %d", ListPipelinesMaxLimit)
	}

	metadata := ListPipelinesNodeMetadata{}
	err = mapstructure.Decode(ctx.Metadata.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	// If this is the same project, nothing to do
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
		return fmt.Errorf("error creating client: %w", err)
	}

	// Apply default limit
	limit := spec.Limit
	if limit == 0 {
		limit = ListPipelinesDefaultLimit
	}
	if limit > ListPipelinesMaxLimit {
		limit = ListPipelinesMaxLimit
	}

	ctx.Logger.Infof("Listing pipelines for project=%s (limit=%d)", metadata.Project.Name, limit)

	// Build query parameters
	params := make(map[string]string)
	params["project_id"] = metadata.Project.ID

	if spec.BranchName != "" {
		params["branch_name"] = spec.BranchName
	}
	if spec.YmlFilePath != "" {
		params["yml_file_path"] = spec.YmlFilePath
	}
	if spec.CreatedAfter != "" {
		params["created_after"] = spec.CreatedAfter
	}
	if spec.CreatedBefore != "" {
		params["created_before"] = spec.CreatedBefore
	}
	if spec.DoneAfter != "" {
		params["done_after"] = spec.DoneAfter
	}
	if spec.DoneBefore != "" {
		params["done_before"] = spec.DoneBefore
	}

	pipelines, err := client.ListPipelinesWithParams(params, limit)
	if err != nil {
		return fmt.Errorf("error listing pipelines: %w", err)
	}

	ctx.Logger.Infof("Retrieved %d pipelines for project=%s", len(pipelines), metadata.Project.Name)

	output := ListPipelinesOutput{
		Pipelines: pipelines,
		Count:     len(pipelines),
	}

	// Store metadata for reference
	ctx.Metadata.Set(map[string]any{
		"projectId":     metadata.Project.ID,
		"projectName":   metadata.Project.Name,
		"pipelineCount": len(pipelines),
	})

	return ctx.Requests.Emit(ListPipelinesSuccessChannel, ListPipelinesPayloadType, []any{output})
}

func (l *ListPipelines) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListPipelines) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListPipelines) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions available for ListPipelines")
}

func (l *ListPipelines) Cleanup(ctx core.SetupContext) error {
	return nil
}
