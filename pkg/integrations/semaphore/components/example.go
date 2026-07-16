package components

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/run_workflow.json
var exampleOutputRunWorkflowBytes []byte

//go:embed payloads/on_pipeline_done.json
var exampleDataOnPipelineDoneBytes []byte

//go:embed payloads/get_pipeline.json
var exampleOutputGetPipelineBytes []byte
var exampleOutput = utils.NewEmbeddedJSON(exampleOutputRunWorkflowBytes)
var exampleData = utils.NewEmbeddedJSON(exampleDataOnPipelineDoneBytes)
var exampleOutputGetPipeline = utils.NewEmbeddedJSON(exampleOutputGetPipelineBytes)

func (c *RunWorkflow) ExampleOutput() map[string]any {
	return exampleOutput.Value()
}

func (t *OnPipelineDone) ExampleData() map[string]any {
	return exampleData.Value()
}

func (c *GetPipeline) ExampleOutput() map[string]any {
	return exampleOutputGetPipeline.Value()
}
