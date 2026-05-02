package components

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/run_workflow.json
var exampleOutputRunWorkflowBytes []byte

//go:embed payloads/on_pipeline_done.json
var exampleDataOnPipelineDoneBytes []byte

//go:embed payloads/get_pipeline.json
var exampleOutputGetPipelineBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

var exampleDataOnce sync.Once
var exampleData map[string]any

var exampleOutputGetPipelineOnce sync.Once
var exampleOutputGetPipeline map[string]any

func (c *RunWorkflow) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputRunWorkflowBytes, &exampleOutput)
}

func (t *OnPipelineDone) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataOnPipelineDoneBytes, &exampleData)
}

func (c *GetPipeline) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetPipelineOnce, exampleOutputGetPipelineBytes, &exampleOutputGetPipeline)
}
