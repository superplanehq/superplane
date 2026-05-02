package actions

import (
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_workflow.json
var exampleOutputRunWorkflowBytes []byte

//go:embed example_data_on_workflow_run.json
var exampleDataOnWorkflowRunBytes []byte

var exampleOutputRunWorkflowOnce sync.Once
var exampleOutputRunWorkflow map[string]any

var exampleDataOnWorkflowRunOnce sync.Once
var exampleDataOnWorkflowRun map[string]any

func (c *RunWorkflow) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunWorkflowOnce, exampleOutputRunWorkflowBytes, &exampleOutputRunWorkflow)
}

func (t *OnWorkflowRun) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnWorkflowRunOnce, exampleDataOnWorkflowRunBytes, &exampleDataOnWorkflowRun)
}
