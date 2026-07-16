package codepipeline

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_pipeline.json
var exampleDataOnPipelineBytes []byte
var exampleDataOnPipeline = utils.NewEmbeddedJSON(exampleDataOnPipelineBytes)

func (p *OnPipeline) ExampleData() map[string]any {
	return exampleDataOnPipeline.Value()
}

//go:embed example_output_run_pipeline.json
var exampleOutputRunPipelineBytes []byte
var exampleOutputRunPipeline = utils.NewEmbeddedJSON(exampleOutputRunPipelineBytes)

func (r *RunPipeline) ExampleOutput() map[string]any {
	return exampleOutputRunPipeline.Value()
}

//go:embed example_output_get_pipeline.json
var exampleOutputGetPipelineBytes []byte
var exampleOutputGetPipeline = utils.NewEmbeddedJSON(exampleOutputGetPipelineBytes)

func (c *GetPipeline) ExampleOutput() map[string]any {
	return exampleOutputGetPipeline.Value()
}

//go:embed example_output_get_pipeline_execution.json
var exampleOutputGetPipelineExecutionBytes []byte
var exampleOutputGetPipelineExecution = utils.NewEmbeddedJSON(exampleOutputGetPipelineExecutionBytes)

func (c *GetPipelineExecution) ExampleOutput() map[string]any {
	return exampleOutputGetPipelineExecution.Value()
}

//go:embed example_output_retry_stage_execution.json
var exampleOutputRetryStageExecutionBytes []byte
var exampleOutputRetryStageExecution = utils.NewEmbeddedJSON(exampleOutputRetryStageExecutionBytes)

func (c *RetryStageExecution) ExampleOutput() map[string]any {
	return exampleOutputRetryStageExecution.Value()
}
