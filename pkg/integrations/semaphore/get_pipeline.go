package semaphore

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetPipeline struct{}

type GetPipelineConfiguration struct {
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
	return `The Get Pipeline component retrieves a Semaphore pipeline by ID and returns its current state, result, workflow ID, and metadata.

## Use Cases

- **Pipeline status lookup**: After Run Workflow starts a pipeline, poll or fetch its status to decide when to proceed
- **Result verification**: Look up the result of a specific pipeline to get full details or confirm state
- **Status checks**: Build a status-check step that verifies a pipeline by ID before triggering dependent actions

## Configuration

- **Pipeline ID**: The Semaphore pipeline ID (e.g., from Run Workflow output or On Pipeline Done event). Accepts expressions.

## Output

Returns the pipeline object including:
- Pipeline ID, name, and workflow ID
- State (running, done, etc.)
- Result (passed, failed, stopped, etc.)
- Timestamps (created_at, done_at, etc.)`
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
			Placeholder: "e.g. 00000000-0000-0000-0000-000000000000",
			Description: "The Semaphore pipeline ID to retrieve",
		},
	}
}

func (c *GetPipeline) Setup(ctx core.SetupContext) error {
	var config GetPipelineConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.PipelineID == "" {
		return fmt.Errorf("pipeline ID is required")
	}

	return nil
}

func (c *GetPipeline) Execute(ctx core.ExecutionContext) error {
	var config GetPipelineConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Semaphore client: %w", err)
	}

	pipeline, err := client.GetPipeline(config.PipelineID)
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
