package codepipeline

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

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
