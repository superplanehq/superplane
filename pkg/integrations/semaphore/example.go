package semaphore

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_workflow.json
var exampleOutputRunWorkflowBytes []byte

//go:embed example_data_on_pipeline_done.json
var exampleDataOnPipelineDoneBytes []byte

//go:embed example_output_list_pipelines.json
var exampleOutputListPipelinesBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

var exampleDataOnce sync.Once
var exampleData map[string]any

var exampleOutputListPipelinesOnce sync.Once
var exampleOutputListPipelines map[string]any

func (c *RunWorkflow) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputRunWorkflowBytes, &exampleOutput)
}

func (t *OnPipelineDone) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataOnPipelineDoneBytes, &exampleData)
}

func (l *ListPipelines) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListPipelinesOnce, exampleOutputListPipelinesBytes, &exampleOutputListPipelines)
}
