package semaphore

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetPipelinePayloadType = "semaphore.pipeline"
const GetPipelineSuccessChannel = "success"

type GetPipeline struct{}

type GetPipelineSpec struct {
	PipelineID string `json:"pipelineId" mapstructure:"pipelineId"`
}

type GetPipelineOutput struct {
	PipelineID   string `json:"pipelineId"`
	PipelineName string `json:"pipelineName"`
	WorkflowID   string `json:"workflowId"`
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

- **Status polling**: After Run Workflow starts a pipeline, poll or fetch its status in a loop or downstream node to decide when to proceed or notify.
- **Pipeline lookup**: Look up the result of a specific pipeline (e.g. from On Pipeline Done event data) to get full details or confirm state.
- **Dependency checks**: Build a status-check step that verifies a pipeline by ID before triggering a dependent action (e.g. deploy only if build pipeline passed).

## How It Works

1. Fetches the pipeline details from Semaphore API using the provided pipeline ID
2. Returns the pipeline state, result, and metadata
3. Emits the data on the success channel for downstream processing

## Configuration

- **Pipeline ID** (required): The Semaphore pipeline ID. Can be obtained from Run Workflow output or On Pipeline Done event data. Accepts expressions.

## Output

Single output channel that emits:
- ` + "`pipelineId`" + `: The pipeline ID
- ` + "`pipelineName`" + `: The pipeline name
- ` + "`workflowId`" + `: The workflow ID this pipeline belongs to
- ` + "`state`" + `: Pipeline state (e.g., "running", "done")
- ` + "`result`" + `: Pipeline result (e.g., "passed", "failed", "stopped")

## Notes

- Branching on state/result can be done downstream via expressions
- If the pipeline doesn't exist, an error is returned
- This is a synchronous operation that returns immediately`
}

func (g *GetPipeline) Icon() string {
	return "git-branch"
}

func (g *GetPipeline) Color() string {
	return "gray"
}

func (g *GetPipeline) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  GetPipelineSuccessChannel,
			Label: "Success",
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
			Placeholder: "e.g. a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Description: "The Semaphore pipeline ID. Can be obtained from Run Workflow output or On Pipeline Done event.",
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
		return fmt.Errorf("error creating client: %w", err)
	}

	ctx.Logger.Infof("Fetching pipeline details for pipeline=%s", spec.PipelineID)
	pipeline, err := client.GetPipeline(spec.PipelineID)
	if err != nil {
		return fmt.Errorf("error fetching pipeline %s: %w", spec.PipelineID, err)
	}

	ctx.Logger.Infof("Retrieved pipeline=%s state=%s result=%s", spec.PipelineID, pipeline.State, pipeline.Result)

	output := GetPipelineOutput{
		PipelineID:   pipeline.PipelineID,
		PipelineName: pipeline.PipelineName,
		WorkflowID:   pipeline.WorkflowID,
		State:        pipeline.State,
		Result:       pipeline.Result,
	}

	// Store metadata for reference
	ctx.Metadata.Set(map[string]any{
		"pipelineId": spec.PipelineID,
		"state":      pipeline.State,
		"result":     pipeline.Result,
	})

	return ctx.Requests.Emit(GetPipelineSuccessChannel, GetPipelinePayloadType, []any{output})
}

func (g *GetPipeline) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetPipeline) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetPipeline) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions available for GetPipeline")
}

func (g *GetPipeline) Cleanup(ctx core.SetupContext) error {
	return nil
}
