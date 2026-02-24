package circleci

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetWorkflow struct{}

type GetWorkflowConfiguration struct {
	WorkflowID string `json:"workflowId" mapstructure:"workflowId"`
}

type WorkflowPipelineInfo struct {
	ID          string `json:"id"`
	Number      int    `json:"number"`
	ProjectSlug string `json:"project_slug"`
}

type GetWorkflowResult struct {
	Workflow        *WorkflowResponse     `json:"workflow"`
	Jobs            []WorkflowJobResponse `json:"jobs"`
	Pipeline        WorkflowPipelineInfo  `json:"pipeline"`
	DurationSeconds int64                 `json:"duration_seconds,omitempty"`
}

func (c *GetWorkflow) Name() string {
	return "circleci.getWorkflow"
}

func (c *GetWorkflow) Label() string {
	return "Get Workflow"
}

func (c *GetWorkflow) Description() string {
	return "Get workflow details by ID, including status, jobs, duration, and pipeline info"
}

func (c *GetWorkflow) Documentation() string {
	return `The Get Workflow component retrieves a CircleCI workflow by ID and enriches it with job-level details.

## Use Cases

- **Workflow inspection**: Fetch status, timing, and pipeline context for a specific workflow run
- **Job-level debugging**: Inspect all jobs in a workflow with their current states
- **Automated reporting**: Collect workflow duration and status data for notifications or dashboards

## Configuration

- **Workflow ID**: The CircleCI workflow ID to fetch

## Output

Returns:
- ` + "`workflow`" + `: Workflow details from CircleCI
- ` + "`jobs`" + `: Jobs associated with the workflow
- ` + "`pipeline`" + `: Pipeline reference (ID, number, project slug)
- ` + "`duration_seconds`" + `: Computed workflow duration when timing data is available`
}

func (c *GetWorkflow) Icon() string {
	return "workflow"
}

func (c *GetWorkflow) Color() string {
	return "gray"
}

func (c *GetWorkflow) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetWorkflow) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "workflowId",
			Label:       "Workflow ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "CircleCI workflow ID",
			Placeholder: "e.g. fda08377-fe7e-46b1-8992-3a7aaecac9c3",
		},
	}
}

func (c *GetWorkflow) Setup(ctx core.SetupContext) error {
	var config GetWorkflowConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.WorkflowID) == "" {
		return fmt.Errorf("workflowId is required")
	}

	return nil
}

func (c *GetWorkflow) Execute(ctx core.ExecutionContext) error {
	var config GetWorkflowConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	workflowID := strings.TrimSpace(config.WorkflowID)
	if workflowID == "" {
		return fmt.Errorf("workflowId is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	workflow, err := client.GetWorkflow(workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	jobs, err := client.GetWorkflowJobs(workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow jobs: %w", err)
	}

	result := GetWorkflowResult{
		Workflow: workflow,
		Jobs:     jobs,
		Pipeline: WorkflowPipelineInfo{
			ID:          workflow.PipelineID,
			Number:      workflow.PipelineNumber,
			ProjectSlug: workflow.ProjectSlug,
		},
	}
	if durationSeconds, ok := computeWorkflowDurationSeconds(workflow.CreatedAt, workflow.StoppedAt); ok {
		result.DurationSeconds = durationSeconds
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"circleci.workflow",
		[]any{result},
	)
}

func (c *GetWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetWorkflow) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetWorkflow) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetWorkflow) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetWorkflow) Cleanup(ctx core.SetupContext) error {
	return nil
}

func computeWorkflowDurationSeconds(createdAt, stoppedAt string) (int64, bool) {
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return 0, false
	}

	stopped, err := time.Parse(time.RFC3339, stoppedAt)
	if err != nil {
		return 0, false
	}

	if stopped.Before(created) {
		return 0, false
	}

	return int64(stopped.Sub(created).Seconds()), true
}
