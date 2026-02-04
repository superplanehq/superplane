package semaphore

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

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
	return `The Get Pipeline component fetches detailed information about a specific Semaphore pipeline.

## Configuration
- **Pipeline ID**: The unique identifier for the pipeline.

## Output Channels
- **Done**: Emitted when the pipeline details are retrieved.`
}

func (g *GetPipeline) Icon() string {
	return "workflow"
}

func (g *GetPipeline) Color() string {
	return "blue"
}

func (g *GetPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  "done",
			Label: "Done",
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
			Description: "The ID of the pipeline to fetch",
		},
	}
}

func (g *GetPipeline) Execute(ctx core.ExecutionContext) error {
	spec := GetPipelineSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	pipeline, err := client.GetPipeline(spec.PipelineID)
	if err != nil {
		return fmt.Errorf("error fetching pipeline %s: %v", spec.PipelineID, err)
	}

	return ctx.ExecutionState.Emit("done", "semaphore.pipeline.fetched", pipeline)
}

func (g *GetPipeline) Setup(ctx core.SetupContext) error                          { return nil }
func (g *GetPipeline) Cancel(ctx core.ExecutionContext) error                     { return nil }
func (g *GetPipeline) Cleanup(ctx core.SetupContext) error                        { return nil }
func (g *GetPipeline) HandleWebhook(ctx core.WebhookRequestContext) (int, error)  { return (200, nil) }
func (g *GetPipeline) Actions() []core.Action                                     { return nil }
func (g *GetPipeline) HandleAction(ctx core.ActionContext) error                  { return nil }
func (g *GetPipeline) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
