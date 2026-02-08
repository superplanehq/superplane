package semaphore

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ListPipelines struct{}

type ListPipelinesSpec struct {
	ProjectID     string `mapstructure:"projectId"`
	WorkflowID    string `mapstructure:"workflowId"`
	BranchName    string `mapstructure:"branchName"`
	YMLFilePath   string `mapstructure:"ymlFilePath"`
	CreatedAfter  string `mapstructure:"createdAfter"`
	CreatedBefore string `mapstructure:"createdBefore"`
	DoneAfter     string `mapstructure:"doneAfter"`
	DoneBefore    string `mapstructure:"doneBefore"`
	ResultLimit   *int   `mapstructure:"resultLimit,omitempty"`
}

func (c *ListPipelines) Name() string {
	return "semaphore.listPipelines"
}

func (c *ListPipelines) Label() string {
	return "List Pipelines"
}

func (c *ListPipelines) Description() string {
	return "List pipelines for a Semaphore project or workflow"
}

func (c *ListPipelines) Documentation() string {
	return `The List Pipelines component retrieves pipelines from a Semaphore project or workflow with comprehensive filtering capabilities.

## Use Cases

- **Pipeline monitoring**: List all pipelines and filter by branch, status, or date range
- **CI/CD dashboards**: Build dashboards showing pipeline history and results
- **Branch analysis**: Filter pipelines by branch to track deployment activity
- **Time-range queries**: Find pipelines created or completed within specific time windows

## Configuration

- **Project ID**: The Semaphore project identifier (required if Workflow ID is not provided)
- **Workflow ID**: The workflow identifier (required if Project ID is not provided)
- **Branch Name**: Filter by branch name (optional)
- **YAML File Path**: Filter by pipeline definition file (optional, e.g., .semaphore/semaphore.yml)
- **Created After**: Unix timestamp or ISO date — only return pipelines created after this time (optional)
- **Created Before**: Unix timestamp or ISO date — only return pipelines created before this time (optional)
- **Done After**: Unix timestamp or ISO date — only return pipelines completed after this time (optional)
- **Done Before**: Unix timestamp or ISO date — only return pipelines completed before this time (optional)
- **Result Limit**: Maximum number of pipelines to return, 1-200 (optional, defaults to 100)

## Output

Returns a list of pipelines, each containing:
- Pipeline ID (ppl_id), name, and YAML file path
- Workflow ID (wf_id)
- State (e.g., QUEUING, RUNNING, DONE) and result (e.g., PASSED, FAILED, STOPPED)
- Branch name and commit SHA
- Creation and completion timestamps
- Error description (if any)`
}

func (c *ListPipelines) Icon() string {
	return "semaphore"
}

func (c *ListPipelines) Color() string {
	return "gray"
}

func (c *ListPipelines) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ListPipelines) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectId",
			Label:       "Project ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., a426b4db-1919-483d-926a-1234567890ab",
			Description: "Semaphore project UUID. Required if Workflow ID is not provided.",
		},
		{
			Name:        "workflowId",
			Label:       "Workflow ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., 65c398bb-57ab-4459-90b5-1234567890ab",
			Description: "Semaphore workflow UUID. Required if Project ID is not provided.",
		},
		{
			Name:        "branchName",
			Label:       "Branch Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., main",
			Description: "Filter pipelines by branch name.",
		},
		{
			Name:        "ymlFilePath",
			Label:       "YAML File Path",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., .semaphore/semaphore.yml",
			Description: "Filter pipelines by pipeline definition file path.",
		},
		{
			Name:        "createdAfter",
			Label:       "Created After",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., 1706745600 or 2024-02-01T00:00:00Z",
			Description: "Only return pipelines created after this timestamp (Unix or ISO 8601).",
		},
		{
			Name:        "createdBefore",
			Label:       "Created Before",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., 1706832000 or 2024-02-02T00:00:00Z",
			Description: "Only return pipelines created before this timestamp (Unix or ISO 8601).",
		},
		{
			Name:        "doneAfter",
			Label:       "Done After",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., 1706745600",
			Description: "Only return pipelines completed after this timestamp (Unix or ISO 8601).",
		},
		{
			Name:        "doneBefore",
			Label:       "Done Before",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., 1706832000",
			Description: "Only return pipelines completed before this timestamp (Unix or ISO 8601).",
		},
		{
			Name:        "resultLimit",
			Label:       "Result Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     100,
			Placeholder: "e.g., 100",
			Description: "Maximum number of pipelines to return (1-200). Defaults to 100.",
		},
	}
}

func (c *ListPipelines) Setup(ctx core.SetupContext) error {
	spec := ListPipelinesSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	if spec.ProjectID == "" && spec.WorkflowID == "" {
		return fmt.Errorf("either Project ID or Workflow ID is required")
	}

	return nil
}

func (c *ListPipelines) Execute(ctx core.ExecutionContext) error {
	spec := ListPipelinesSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Build query parameters
	params := url.Values{}

	if spec.ProjectID != "" {
		params.Set("project_id", spec.ProjectID)
	}

	if spec.WorkflowID != "" {
		params.Set("wf_id", spec.WorkflowID)
	}

	if spec.BranchName != "" {
		params.Set("branch_name", spec.BranchName)
	}

	if spec.YMLFilePath != "" {
		params.Set("yml_file_path", spec.YMLFilePath)
	}

	if spec.CreatedAfter != "" {
		params.Set("created_after", spec.CreatedAfter)
	}

	if spec.CreatedBefore != "" {
		params.Set("created_before", spec.CreatedBefore)
	}

	if spec.DoneAfter != "" {
		params.Set("done_after", spec.DoneAfter)
	}

	if spec.DoneBefore != "" {
		params.Set("done_before", spec.DoneBefore)
	}

	if spec.ResultLimit != nil && *spec.ResultLimit > 0 {
		limit := *spec.ResultLimit
		if limit > 200 {
			limit = 200
		}
		params.Set("page_size", strconv.Itoa(limit))
	}

	// Build the request URL
	requestURL := fmt.Sprintf("%s/api/v1alpha/pipelines?%s", client.OrgURL, params.Encode())

	body, err := client.execRequest("GET", requestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to list pipelines: %w", err)
	}

	// Parse the response
	var response struct {
		Pipelines []json.RawMessage `json:"pipelines"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to []any for emission
	pipelines := make([]any, len(response.Pipelines))
	for i, p := range response.Pipelines {
		var pipeline any
		if err := json.Unmarshal(p, &pipeline); err != nil {
			return fmt.Errorf("failed to parse pipeline %d: %w", i, err)
		}
		pipelines[i] = pipeline
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"semaphore.pipelines",
		pipelines,
	)
}

func (c *ListPipelines) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ListPipelines) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *ListPipelines) Actions() []core.Action {
	return []core.Action{}
}

func (c *ListPipelines) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ListPipelines) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ListPipelines) Cleanup(ctx core.SetupContext) error {
	return nil
}
