package circleci

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_pipeline.json
var exampleOutputRunPipelineBytes []byte

//go:embed example_data_on_pipeline_completed.json
var exampleDataOnPipelineCompletedBytes []byte

var exampleOutputRunPipelineOnce sync.Once
var exampleOutputRunPipeline map[string]any

var exampleDataOnPipelineCompletedOnce sync.Once
var exampleDataOnPipelineCompleted map[string]any

func (c *RunPipeline) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRunPipelineOnce,
		exampleOutputRunPipelineBytes,
		&exampleOutputRunPipeline,
	)
}

func (t *OnPipelineCompleted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnPipelineCompletedOnce,
		exampleDataOnPipelineCompletedBytes,
		&exampleDataOnPipelineCompleted,
	)
}
