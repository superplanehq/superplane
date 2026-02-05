package circleci

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_trigger_pipeline.json
var exampleOutputTriggerPipelineBytes []byte

//go:embed example_data_on_workflow_completed.json
var exampleDataOnWorkflowCompletedBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

var exampleDataOnce sync.Once
var exampleData map[string]any

func (c *TriggerPipeline) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputTriggerPipelineBytes, &exampleOutput)
}

func (t *OnWorkflowCompleted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataOnWorkflowCompletedBytes, &exampleData)
}
