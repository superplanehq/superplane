package semaphore

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetPipelinePayloadType = "semaphore.pipeline"

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
	return "Retrieve a Semaphore pipeline by ID"
}

func (c *GetPipeline) Documentation() string {
	return `The Get Pipeline component fetches a Semaphore pipeline by its ID.

## Use Cases

- **Status checks**: Inspect pipeline state and result after triggering a workflow
- **Debugging**: Fetch pipeline metadata for inspection

## Configuration

- **Pipeline ID**: The pipeline ID to retrieve (supports expressions)

## Output

Emits a ` + "`semaphore.pipeline`" + ` payload containing pipeline fields like ` + "`name`" + `, ` + "`ppl_id`" + `, ` + "`wf_id`" + `, ` + "`state`" + `, and ` + "`result`" + `.`
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
			Placeholder: "e.g., 00000000-0000-0000-0000-000000000000 or {{$.event.data.pipeline.id}}",
			Description: "Semaphore pipeline ID to retrieve",
		},
	}
}

func decodeGetPipelineConfiguration(configuration any) (GetPipelineConfiguration, error) {
	spec := GetPipelineConfiguration{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return GetPipelineConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.PipelineID = strings.TrimSpace(spec.PipelineID)
	if spec.PipelineID == "" {
		return GetPipelineConfiguration{}, fmt.Errorf("pipelineId is required")
	}

	return spec, nil
}

func (c *GetPipeline) Setup(ctx core.SetupContext) error {
	_, err := decodeGetPipelineConfiguration(ctx.Configuration)
	return err
}

func (c *GetPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetPipeline) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeGetPipelineConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.GetPipeline(spec.PipelineID)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetPipelinePayloadType,
		[]any{pipeline},
	)
}

func (c *GetPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
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
