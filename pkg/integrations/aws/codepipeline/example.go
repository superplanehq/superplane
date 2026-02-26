package codepipeline

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_pipeline.json
var exampleDataOnPipelineBytes []byte

var exampleDataOnPipelineOnce sync.Once
var exampleDataOnPipeline map[string]any

func (p *OnPipeline) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnPipelineOnce,
		exampleDataOnPipelineBytes,
		&exampleDataOnPipeline,
	)
}

//go:embed example_output_run_pipeline.json
var exampleOutputRunPipelineBytes []byte

var exampleOutputRunPipelineOnce sync.Once
var exampleOutputRunPipeline map[string]any

func (r *RunPipeline) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRunPipelineOnce,
		exampleOutputRunPipelineBytes,
		&exampleOutputRunPipeline,
	)
}

//go:embed example_output_get_pipeline.json
var exampleOutputGetPipelineBytes []byte

var exampleOutputGetPipelineOnce sync.Once
var exampleOutputGetPipeline map[string]any

func (c *GetPipeline) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetPipelineOnce,
		exampleOutputGetPipelineBytes,
		&exampleOutputGetPipeline,
	)
}

//go:embed example_output_get_pipeline_execution.json
var exampleOutputGetPipelineExecutionBytes []byte

var exampleOutputGetPipelineExecutionOnce sync.Once
var exampleOutputGetPipelineExecution map[string]any

func (c *GetPipelineExecution) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetPipelineExecutionOnce,
		exampleOutputGetPipelineExecutionBytes,
		&exampleOutputGetPipelineExecution,
	)
}

//go:embed example_output_retry_stage_execution.json
var exampleOutputRetryStageExecutionBytes []byte

var exampleOutputRetryStageExecutionOnce sync.Once
var exampleOutputRetryStageExecution map[string]any

func (c *RetryStageExecution) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRetryStageExecutionOnce,
		exampleOutputRetryStageExecutionBytes,
		&exampleOutputRetryStageExecution,
	)
}
