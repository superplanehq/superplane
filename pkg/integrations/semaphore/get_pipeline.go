package semaphore

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetPipelineOutputChannel = "default"
const PipelinePayloadType = "semaphore.pipeline"

type GetPipeline struct{}

type GetPipelineSpec struct {
	PipelineID string `json:"pipelineId" mapstructure:"pipelineId"`
}

type GetPipelineOutput struct {
	PipelineID   string `json:"ppl_id"`
	WorkflowID   string `json:"wf_id"`
	Name         string `json:"name"`
	State        string `json:"state"`
	Result       string `json:"result"`
}

func (g *GetPipeline) Name() string {
	return "semaphore.getPipeline"
}

func (g *GetPipeline) Label() string {
	return "Get Pipeline"
}

func (g *GetPipeline) Description() string {
	return "Fetch a Semaphore pipeline by ID"
}

func (g *GetPipeline) Documentation() string {
	return `The Get Pipeline component fetches a Semaphore pipeline by ID and returns its current state, result, workflow ID, and metadata.

## Use Cases

- **Poll pipeline status**: After triggering a workflow, poll for pipeline status to decide when to proceed
- **Status verification**: Check if a specific pipeline passed or failed before triggering dependent actions
- **Workflow branching**: Branch workflows based on pipeline state (running vs done) or result (passed/failed)

## Configuration

- **Pipeline ID**: The Semaphore pipeline ID (e.g., from Run Workflow output or On Pipeline Done event). Supports expressions.

## Output

**Default channel**: Emits pipeline data including:
- ppl_id: Pipeline ID
- wf_id: Workflow ID
- name: Pipeline name
- state: Current state (e.g., pending, running, done)
- result: Result when done (e.g., passed, failed, cancelled)

Errors do not emit a payload.`
}

func (g *GetPipeline) Icon() string {
	return "search"
}

func (g *GetPipeline) Color() string {
	return "gray"
}

func (g *GetPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  GetPipelineOutputChannel,
			Label: "Default",
		},
	}
}

func (g *GetPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "pipelineId",
			Label:       "Pipeline ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., 123e4567-e89b-12d3-a456-426614174000",
			Description: "Semaphore pipeline ID to fetch",
		},
	}
}

func (g *GetPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetPipeline) Setup(ctx core.SetupContext) error {
	config := GetPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.PipelineID == "" {
		return fmt.Errorf("pipeline ID is required")
	}

	return nil
}

func (g *GetPipeline) Execute(ctx core.ExecutionContext) error {
	spec := GetPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.GetPipeline(spec.PipelineID)
	if err != nil {
		return fmt.Errorf("error fetching pipeline: %v", err)
	}

	output := GetPipelineOutput{
		PipelineID: pipeline.PipelineID,
		WorkflowID: pipeline.WorkflowID,
		Name:       pipeline.PipelineName,
		State:      pipeline.State,
		Result:     pipeline.Result,
	}

	return ctx.ExecutionState.Emit(GetPipelineOutputChannel, PipelinePayloadType, []any{output})
}

func (g *GetPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (g *GetPipeline) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetPipeline) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
