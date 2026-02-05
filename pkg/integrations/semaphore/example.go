package semaphore

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_workflow.json
var exampleOutputRunWorkflowBytes []byte

//go:embed example_output_get_pipeline.json
var exampleOutputGetPipelineBytes []byte

//go:embed example_data_on_pipeline_done.json
var exampleDataOnPipelineDoneBytes []byte

var exampleOutputRunWorkflowOnce sync.Once
var exampleOutputRunWorkflow map[string]any

var exampleOutputGetPipelineOnce sync.Once
var exampleOutputGetPipeline map[string]any

var exampleDataOnPipelineDoneOnce sync.Once
var exampleDataOnPipelineDone map[string]any

func (c *RunWorkflow) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunWorkflowOnce, exampleOutputRunWorkflowBytes, &exampleOutputRunWorkflow)
}

func (g *GetPipeline) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPipelineOnce, exampleOutputGetPipelineBytes, &exampleOutputGetPipeline)
}

func (t *OnPipelineDone) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPipelineDoneOnce, exampleDataOnPipelineDoneBytes, &exampleDataOnPipelineDone)
}
