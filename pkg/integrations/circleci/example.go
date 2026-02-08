package circleci

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_trigger_pipeline.json
var exampleOutputTriggerPipelineBytes []byte

//go:embed example_data_on_pipeline_completed.json
var exampleDataOnPipelineCompletedBytes []byte

var exampleOutputTriggerPipelineOnce sync.Once
var exampleOutputTriggerPipeline map[string]any

var exampleDataOnPipelineCompletedOnce sync.Once
var exampleDataOnPipelineCompleted map[string]any

func (c *TriggerPipeline) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputTriggerPipelineOnce,
		exampleOutputTriggerPipelineBytes,
		&exampleOutputTriggerPipeline,
	)
}

func (t *OnPipelineCompleted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnPipelineCompletedOnce,
		exampleDataOnPipelineCompletedBytes,
		&exampleDataOnPipelineCompleted,
	)
}
