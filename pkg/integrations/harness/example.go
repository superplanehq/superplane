package harness

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_pipeline.json
var exampleOutputRunPipelineBytes []byte

//go:embed example_data_on_pipeline_completed.json
var exampleDataOnPipelineCompletedBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

var exampleDataOnce sync.Once
var exampleData map[string]any

func (r *RunPipeline) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputRunPipelineBytes, &exampleOutput)
}

func (t *OnPipelineCompleted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataOnPipelineCompletedBytes, &exampleData)
}
