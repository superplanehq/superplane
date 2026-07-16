package actions

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/run_workflow.json
var exampleOutputRunWorkflowBytes []byte

//go:embed payloads/on_workflow_run.json
var exampleDataOnWorkflowRunBytes []byte
var exampleOutputRunWorkflow = utils.NewEmbeddedJSON(exampleOutputRunWorkflowBytes)
var exampleDataOnWorkflowRun = utils.NewEmbeddedJSON(exampleDataOnWorkflowRunBytes)

func (c *RunWorkflow) ExampleOutput() map[string]any {
	return exampleOutputRunWorkflow.Value()
}

func (t *OnWorkflowRun) ExampleData() map[string]any {
	return exampleDataOnWorkflowRun.Value()
}
