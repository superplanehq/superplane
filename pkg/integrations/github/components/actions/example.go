package actions

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/run_workflow.json
var exampleOutputRunWorkflowBytes []byte

//go:embed payloads/on_workflow_run.json
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
