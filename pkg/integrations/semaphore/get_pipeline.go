package semaphore

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetPipelinePayloadType = "semaphore.pipeline"
const GetPipelineOutputChannel = "default"

type GetPipeline struct{}

type GetPipelineSpec struct {
	PipelineID string `json:"pipelineId"`
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
	return `The Get Pipeline component fetches a Semaphore pipeline by its ID and returns its current state, result, workflow ID, and metadata.

## Use Cases

- **Pipeline status check**: After Run Workflow starts a pipeline, poll or fetch its status in a loop or downstream node to decide when to proceed or notify.
- **Pipeline lookup**: Look up the result of a specific pipeline (e.g., from On Pipeline Done event data) to get full details or confirm state.
- **Build verification**: Build a status-check step that verifies a pipeline by ID before triggering a dependent action (e.g., deploy only if build pipeline passed).

## How It Works

1. Takes a Pipeline ID as input (can be from Run Workflow output, On Pipeline Done event data, or any expression)
2. Calls the Semaphore API to fetch the pipeline details
3. Emits the pipeline data to the default output channel

## Configuration

- **Pipeline ID** (required): The Semaphore pipeline ID. Accepts expressions (e.g., ` + "`{{ event.pipeline.ppl_id }}`" + `).

## Output

Emits pipeline data to the default channel:
- **name**: Pipeline name
- **ppl_id**: Pipeline ID
- **wf_id**: Workflow ID
- **state**: Pipeline state (e.g., "running", "done")
- **result**: Pipeline result (e.g., "passed", "failed", "stopped")

## Notes

- This is a synchronous component - it fetches and returns immediately
- Use expressions to pass pipeline IDs from upstream components
- Branching on state/result can be done downstream via expressions`
}

func (g *GetPipeline) Icon() string {
	return "workflow"
}

func (g *GetPipeline) Color() string {
	return "gray"
}

func (g *GetPipeline) ExampleOutput() map[string]any {
	return map[string]any{
		"name":   "Build and Test",
		"ppl_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"wf_id":  "f0e1d2c3-b4a5-6789-0fed-cba987654321",
		"state":  "done",
		"result": "passed",
	}
}

func (g *GetPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:        GetPipelineOutputChannel,
			Label:       "Default",
			Description: "Emits the pipeline data",
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
			Description: "The Semaphore pipeline ID to fetch",
			Placeholder: "e.g. {{ event.pipeline.ppl_id }}",
		},
	}
}

func (g *GetPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetPipeline) Setup(ctx core.SetupContext) error {
	spec := GetPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Pipeline ID is required but will typically be an expression,
	// so we can't validate it at setup time - only check if it's set
	// when the value is a literal (non-expression)
	return nil
}

func (g *GetPipeline) Execute(ctx core.ExecutionContext) error {
	spec := GetPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.PipelineID == "" {
		return ctx.ExecutionState.Fail("validation_error", "pipeline ID is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Semaphore client: %w", err)
	}

	pipeline, err := client.GetPipeline(spec.PipelineID)
	if err != nil {
		return ctx.ExecutionState.Fail("api_error", fmt.Sprintf("failed to get pipeline: %v", err))
	}

	if pipeline == nil {
		return ctx.ExecutionState.Fail("not_found", fmt.Sprintf("pipeline %s not found", spec.PipelineID))
	}

	payload := map[string]any{
		"name":   pipeline.PipelineName,
		"ppl_id": pipeline.PipelineID,
		"wf_id":  pipeline.WorkflowID,
		"state":  pipeline.State,
		"result": pipeline.Result,
	}

	return ctx.ExecutionState.Emit(GetPipelineOutputChannel, GetPipelinePayloadType, []any{payload})
}

func (g *GetPipeline) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetPipeline) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions available")
}

func (g *GetPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, fmt.Errorf("webhooks not supported")
}

func (g *GetPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
