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

const GetWorkflowPayloadType = "circleci.workflow"

type GetWorkflow struct{}

type GetWorkflowConfiguration struct {
	WorkflowID string `json:"workflowId" mapstructure:"workflowId"`
}

func (c *GetWorkflow) Name() string {
	return "circleci.getWorkflow"
}

func (c *GetWorkflow) Label() string {
	return "Get Workflow"
}

func (c *GetWorkflow) Description() string {
	return "Retrieve a CircleCI workflow by ID"
}

func (c *GetWorkflow) Documentation() string {
	return `The Get Workflow component fetches details for a CircleCI workflow.

## Use Cases

- **Workflow inspection**: Fetch current workflow status, jobs, and metadata
- **Workflow context**: Use workflow fields to drive branching decisions in later steps

## Configuration

- **Workflow ID**: The ID of the CircleCI workflow to retrieve (supports expressions)

## Output

Emits a ` + "`circleci.workflow`" + ` payload containing workflow fields like ` + "`id`" + `, ` + "`name`" + `, ` + "`status`" + `, ` + "`createdAt`" + `, and ` + "`stoppedAt`" + `.`
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

func decodeGetWorkflowConfiguration(config any) (GetWorkflowConfiguration, error) {
	spec := GetWorkflowConfiguration{}
	if err := mapstructure.Decode(config, &spec); err != nil {
		return GetWorkflowConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.WorkflowID = strings.TrimSpace(spec.WorkflowID)
	if spec.WorkflowID == "" {
		return GetWorkflowConfiguration{}, fmt.Errorf("workflow ID is required")
	}

	return spec, nil
}

func (c *GetWorkflow) Setup(ctx core.SetupContext) error {
	_, err := decodeGetWorkflowConfiguration(ctx.Configuration)
	return err
}

func (c *GetWorkflow) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetWorkflow) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetWorkflowConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	workflow, err := client.GetWorkflow(spec.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	jobs, err := client.GetWorkflowJobs(spec.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow jobs: %w", err)
	}

	jobList := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		jobList = append(jobList, map[string]any{
			"id":          j.ID,
			"name":        j.Name,
			"type":        j.Type,
			"status":      j.Status,
			"startedAt":   j.StartedAt,
			"stoppedAt":   j.StoppedAt,
			"jobNumber":   j.JobNumber,
			"projectSlug": j.ProjectSlug,
		})
	}

	payload := map[string]any{
		"id":        workflow.ID,
		"name":      workflow.Name,
		"status":    workflow.Status,
		"createdAt": workflow.CreatedAt,
		"stoppedAt": workflow.StoppedAt,
		"jobs":      jobList,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetWorkflowPayloadType,
		[]any{payload},
	)
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
