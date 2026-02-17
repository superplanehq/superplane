package circleci

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_pipeline.json
var exampleOutputRunPipelineBytes []byte

//go:embed example_data_on_workflow_completed.json
var exampleDataOnWorkflowCompletedBytes []byte

var exampleOutputRunPipelineOnce sync.Once
var exampleOutputRunPipeline map[string]any

var exampleDataOnWorkflowCompletedOnce sync.Once
var exampleDataOnWorkflowCompleted map[string]any

func (c *RunPipeline) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRunPipelineOnce,
		exampleOutputRunPipelineBytes,
		&exampleOutputRunPipeline,
	)
}

func (t *OnWorkflowCompleted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnWorkflowCompletedOnce,
		exampleDataOnWorkflowCompletedBytes,
		&exampleDataOnWorkflowCompleted,
	)
}
