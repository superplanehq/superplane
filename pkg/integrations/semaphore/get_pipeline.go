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

func (c *GetPipeline) Name() string {
	return "semaphore.getPipeline"
}

func (c *GetPipeline) Label() string {
	return "Get Pipeline"
}

func (c *GetPipeline) Description() string {
	return "Get a Semaphore pipeline by ID"
}

func (c *GetPipeline) Documentation() string {
	return `The Get Pipeline component fetches a Semaphore pipeline by its ID and returns its current state, result, and metadata.

## Use Cases

- **Pipeline status checking**: After Run Workflow starts a pipeline, fetch its status to decide when to proceed
- **Pipeline lookup**: Look up the result of a specific pipeline from event data to get full details
- **Conditional deployment**: Build a status-check step that verifies a pipeline before triggering dependent actions

## Configuration

- **Pipeline ID**: The Semaphore pipeline ID (supports expressions, e.g. ` + "`{{ event.pipeline.id }}`" + `)

## Output

Returns the pipeline object including:
- Pipeline ID (ppl_id)
- Pipeline name
- Workflow ID (wf_id)
- State (e.g. running, done)
- Result (e.g. passed, failed)`
}

func (c *GetPipeline) Icon() string {
	return "workflow"
}

func (c *GetPipeline) Color() string {
	return "gray"
}

func (c *GetPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetPipeline) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "pipelineId",
			Label:       "Pipeline ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Semaphore pipeline ID",
			Placeholder: "e.g. {{ event.pipeline.id }}",
		},
	}
}

func (c *GetPipeline) Setup(ctx core.SetupContext) error {
	var spec GetPipelineSpec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.PipelineID == "" {
		return fmt.Errorf("pipeline ID is required")
	}

	return nil
}

func (c *GetPipeline) Execute(ctx core.ExecutionContext) error {
	var spec GetPipelineSpec
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
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

func (c *GetPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetPipeline) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetPipeline) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
