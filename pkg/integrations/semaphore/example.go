package semaphore

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_workflow.json
var exampleOutputRunWorkflowBytes []byte

//go:embed example_output_get_job_logs.json
var exampleOutputGetJobLogsBytes []byte

//go:embed example_data_on_pipeline_done.json
var exampleDataOnPipelineDoneBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

var exampleOutputGetJobLogsOnce sync.Once
var exampleOutputGetJobLogs map[string]any

var exampleDataOnce sync.Once
var exampleData map[string]any

func (c *RunWorkflow) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputRunWorkflowBytes, &exampleOutput)
}

func (c *GetJobLogs) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetJobLogsOnce, exampleOutputGetJobLogsBytes, &exampleOutputGetJobLogs)
}

func (t *OnPipelineDone) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataOnPipelineDoneBytes, &exampleData)
}
