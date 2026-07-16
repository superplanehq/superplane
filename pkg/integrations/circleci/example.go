package circleci

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_pipeline.json
var exampleOutputRunPipelineBytes []byte

//go:embed example_data_on_workflow_completed.json
var exampleDataOnWorkflowCompletedBytes []byte
var exampleOutputRunPipeline = utils.NewEmbeddedJSON(exampleOutputRunPipelineBytes)
var exampleDataOnWorkflowCompleted = utils.NewEmbeddedJSON(exampleDataOnWorkflowCompletedBytes)

//go:embed example_output_get_workflow.json
var exampleOutputGetWorkflowBytes []byte

//go:embed example_output_get_last_workflow.json
var exampleOutputGetLastWorkflowBytes []byte
var exampleOutputGetWorkflow = utils.NewEmbeddedJSON(exampleOutputGetWorkflowBytes)
var exampleOutputGetLastWorkflow = utils.NewEmbeddedJSON(exampleOutputGetLastWorkflowBytes)

//go:embed example_output_get_recent_workflow_runs.json
var exampleOutputGetRecentWorkflowRunsBytes []byte

//go:embed example_output_get_test_metrics.json
var exampleOutputGetTestMetricsBytes []byte
var exampleOutputGetRecentWorkflowRuns = utils.NewEmbeddedJSON(exampleOutputGetRecentWorkflowRunsBytes)
var exampleOutputGetTestMetrics = utils.NewEmbeddedJSON(exampleOutputGetTestMetricsBytes)

//go:embed example_output_get_flaky_tests.json
var exampleOutputGetFlakyTestsBytes []byte
var exampleOutputGetFlakyTests = utils.NewEmbeddedJSON(exampleOutputGetFlakyTestsBytes)

func (c *RunPipeline) ExampleOutput() map[string]any {
	return exampleOutputRunPipeline.Value()
}

func (c *GetWorkflow) ExampleOutput() map[string]any {
	return exampleOutputGetWorkflow.Value()
}

func (c *GetLastWorkflow) ExampleOutput() map[string]any {
	return exampleOutputGetLastWorkflow.Value()
}

func (c *GetRecentWorkflowRuns) ExampleOutput() map[string]any {
	return exampleOutputGetRecentWorkflowRuns.Value()
}

func (c *GetTestMetrics) ExampleOutput() map[string]any {
	return exampleOutputGetTestMetrics.Value()
}

func (c *GetFlakyTests) ExampleOutput() map[string]any {
	return exampleOutputGetFlakyTests.Value()
}

func (t *OnWorkflowCompleted) ExampleData() map[string]any {
	return exampleDataOnWorkflowCompleted.Value()
}
