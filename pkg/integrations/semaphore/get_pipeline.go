package semaphore

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetPipeline struct{}

type GetPipelineSpec struct {
	PipelineID string `json:"pipelineId" mapstructure:"pipelineId"`
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

- **Pipeline status check**: After Run Workflow starts a pipeline, poll or fetch its status in a loop or downstream node to decide when to proceed or notify.
- **Pipeline lookup**: Look up the result of a specific pipeline (e.g., from On Pipeline Done event data) to get full details or confirm state.
- **Status verification**: Build a status-check step that verifies a pipeline by ID before triggering a dependent action (e.g., deploy only if build pipeline passed).

## Configuration

- **Pipeline ID**: The Semaphore pipeline ID (e.g., from Run Workflow output or On Pipeline Done event). Accepts expressions.

## Output

Returns pipeline data including:
- **name**: Pipeline name
- **ppl_id**: Pipeline ID
- **wf_id**: Workflow ID
- **state**: Current state (e.g., running, done)
- **result**: Pipeline result (e.g., passed, failed)

## Notes

- Branching on state/result is done downstream via expressions
- Errors do not emit a payload (component fails)`
}

func (g *GetPipeline) Icon() string {
	return "workflow"
}

func (g *GetPipeline) Color() string {
	return "gray"
}

func (g *GetPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "pipelineId",
			Label:       "Pipeline ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. {{$.data.pipeline.id}}",
			Description: "The Semaphore pipeline ID to fetch. Supports expressions.",
		},
	}
}

func (g *GetPipeline) Setup(ctx core.SetupContext) error {
	spec := GetPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Pipeline ID validation is done at execution time since it may contain expressions
	return nil
}

func (g *GetPipeline) Execute(ctx core.ExecutionContext) error {
	spec := GetPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.PipelineID == "" {
		return fmt.Errorf("pipeline ID is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	pipeline, err := client.GetPipeline(spec.PipelineID)
	if err != nil {
		return fmt.Errorf("failed to get pipeline: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"semaphore.pipeline",
		[]any{pipeline},
	)
}

func (g *GetPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
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

func (g *GetPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
