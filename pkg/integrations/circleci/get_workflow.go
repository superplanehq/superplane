package circleci

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetWorkflow struct{}

type GetWorkflowSpec struct {
	WorkflowID string `json:"workflowId" mapstructure:"workflowId"`
}

func (c *GetWorkflow) Name() string {
	return "circleci.getWorkflow"
}

func (c *GetWorkflow) Label() string {
	return "Get Workflow"
}

func (c *GetWorkflow) Description() string {
	return "Retrieve workflow details by ID, including status, jobs, duration, and pipeline info"
}

func (c *GetWorkflow) Documentation() string {
	return `The Get Workflow component retrieves detailed information about a specific CircleCI workflow, including its jobs.

## Use Cases

- **Workflow inspection**: Fetch the full status, jobs, and timing of a specific workflow
- **Post-pipeline checks**: After a Run Pipeline component, inspect a workflow by ID
- **Job-level visibility**: See individual job statuses, durations, and dependencies within a workflow
- **Debugging failures**: Identify which jobs failed within a workflow

## Configuration

- **Workflow ID**: The CircleCI workflow ID to retrieve (supports expressions)

## Output

Emits workflow details including:
- Workflow ID, name, and status
- Start and stop times
- List of jobs with their statuses, durations, and dependencies`
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
			Description: "CircleCI workflow ID to retrieve (supports expressions)",
		},
	}
}

func (c *GetWorkflow) Setup(ctx core.SetupContext) error {
	spec := GetWorkflowSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.WorkflowID) == "" {
		return fmt.Errorf("workflow ID is required")
	}

	return nil
}

func (c *GetWorkflow) Execute(ctx core.ExecutionContext) error {
	spec := GetWorkflowSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	workflowID := strings.TrimSpace(spec.WorkflowID)

	workflow, err := client.GetWorkflow(workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	jobs, err := client.GetWorkflowJobs(workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow jobs: %w", err)
	}

	output := map[string]any{
		"workflow": workflow,
		"jobs":     jobs,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"circleci.workflow",
		[]any{output},
	)
}

func (c *GetWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetWorkflow) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetWorkflow) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetWorkflow) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetWorkflow) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetWorkflow) Cleanup(ctx core.SetupContext) error {
	return nil
}
