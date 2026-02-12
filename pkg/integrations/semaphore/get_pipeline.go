package semaphore

import (
	"fmt"

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
	return "Fetch a Semaphore pipeline by ID and return its current state"
}

func (c *GetPipeline) Documentation() string {
	return `The Get Pipeline component fetches a Semaphore pipeline by ID and returns its current state, result, workflow ID, and metadata.

## Use Cases

- **Status polling**: After Run Workflow starts a pipeline, fetch its status to decide when to proceed
- **Result verification**: Look up the result of a specific pipeline to get full details
- **Status checks**: Verify a pipeline by ID before triggering dependent actions (e.g. deploy only if build passed)
- **Workflow branching**: Branch on pipeline state (running vs done) or result (passed/failed)

## Configuration

- **Pipeline ID**: The Semaphore pipeline ID (required). Accepts expressions and can use output from Run Workflow or On Pipeline Done events.

## Output

Returns pipeline data including:
- name: Pipeline name
- ppl_id: Pipeline ID
- wf_id: Workflow ID
- state: Current state (running, done, etc.)
- result: Pipeline result (passed, failed, stopped, canceled)

Use expressions downstream to branch on state or result.`
}

func (c *GetPipeline) Icon() string {
	return "semaphore"
}

func (c *GetPipeline) Color() string {
	return "green"
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
			Description: "Semaphore pipeline ID (e.g., from Run Workflow output or event data)",
		},
	}
}

func (c *GetPipeline) Execute(ctx core.ExecutionContext) ([]core.OutputChannel, error) {
	var config GetPipelineConfiguration
	if err := mapstructure.Decode(ctx.Configuration(), &config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if config.PipelineID == "" {
		return nil, fmt.Errorf("pipelineId is required")
	}

	client, err := NewClient(ctx.SyncContext().HTTP, ctx.SyncContext().Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	pipeline, err := client.GetPipeline(config.PipelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	return []core.OutputChannel{
		{
			Name: core.DefaultOutputChannel.Name,
			Output: map[string]any{
				"name":   pipeline.PipelineName,
				"ppl_id": pipeline.PipelineID,
				"wf_id":  pipeline.WorkflowID,
				"state":  pipeline.State,
				"result": pipeline.Result,
			},
		},
	}, nil
}
