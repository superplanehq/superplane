package harness

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_pipeline.json
var exampleOutputRunPipelineBytes []byte

//go:embed example_data_on_pipeline_completed.json
var exampleDataOnPipelineCompletedBytes []byte
var exampleOutput = utils.NewEmbeddedJSON(exampleOutputRunPipelineBytes)
var exampleData = utils.NewEmbeddedJSON(exampleDataOnPipelineCompletedBytes)

func (r *RunPipeline) ExampleOutput() map[string]any {
	return exampleOutput.Value()
}

func (t *OnPipelineCompleted) ExampleData() map[string]any {
	return exampleData.Value()
}
