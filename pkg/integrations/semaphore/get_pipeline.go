package semaphore

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetPipelinePayloadType = "semaphore.pipeline"

type GetPipeline struct{}

type GetPipelineSpec struct {
	PipelineID string `json:"pipelineId"`
}

func (c *GetPipeline) Name() string {
	return "semaphore.getPipeline"
}

func (c *GetPipeline) Label() string {
	return "Get Pipeline"
}

func (c *GetPipeline) Description() string {
	return "Fetch a Semaphore pipeline by ID"
}

func (c *GetPipeline) Documentation() string {
	return `The Get Pipeline component fetches a Semaphore pipeline by its ID and returns its current state and result.

## Use Cases

- **Status check**: After Run Workflow starts a pipeline, fetch its status to decide when to proceed or notify
- **Result lookup**: Look up the result of a specific pipeline to get full details or confirm state
- **Conditional deploy**: Verify a pipeline passed before triggering a dependent action (e.g. deploy only if build passed)

## Configuration

- **Pipeline ID**: The Semaphore pipeline ID (required, supports expressions). Can come from Run Workflow output or On Pipeline Done event data.

## Output

Returns the pipeline object including:
- **name**: Pipeline name
- **ppl_id**: Pipeline ID
- **wf_id**: Workflow ID
- **state**: Current pipeline state (e.g. running, done)
- **result**: Pipeline result (e.g. passed, failed)`
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
			Description: "The Semaphore pipeline ID to fetch",
		},
	}
}

func (c *GetPipeline) Setup(ctx core.SetupContext) error {
	spec := GetPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.PipelineID == "" {
		return errors.New("pipeline ID is required")
	}

	return nil
}

func (c *GetPipeline) Execute(ctx core.ExecutionContext) error {
	spec := GetPipelineSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	pipeline, err := client.GetPipeline(spec.PipelineID)
	if err != nil {
		return fmt.Errorf("failed to get pipeline: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetPipelinePayloadType,
		[]any{pipeline},
	)
}

func (c *GetPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetPipeline) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetPipeline) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
