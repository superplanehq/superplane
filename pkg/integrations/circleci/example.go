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

//go:embed example_output_get_workflow.json
var exampleOutputGetWorkflowBytes []byte

//go:embed example_output_get_last_workflow.json
var exampleOutputGetLastWorkflowBytes []byte

var exampleOutputGetWorkflowOnce sync.Once
var exampleOutputGetWorkflow map[string]any

var exampleOutputGetLastWorkflowOnce sync.Once
var exampleOutputGetLastWorkflow map[string]any

//go:embed example_output_get_recent_workflow_runs.json
var exampleOutputGetRecentWorkflowRunsBytes []byte

//go:embed example_output_get_test_metrics.json
var exampleOutputGetTestMetricsBytes []byte

var exampleOutputGetRecentWorkflowRunsOnce sync.Once
var exampleOutputGetRecentWorkflowRuns map[string]any

var exampleOutputGetTestMetricsOnce sync.Once
var exampleOutputGetTestMetrics map[string]any

//go:embed example_output_get_flaky_tests.json
var exampleOutputGetFlakyTestsBytes []byte

var exampleOutputGetFlakyTestsOnce sync.Once
var exampleOutputGetFlakyTests map[string]any

func (c *RunPipeline) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRunPipelineOnce,
		exampleOutputRunPipelineBytes,
		&exampleOutputRunPipeline,
	)
}

func (c *GetWorkflow) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetWorkflowOnce,
		exampleOutputGetWorkflowBytes,
		&exampleOutputGetWorkflow,
	)
}

func (c *GetLastWorkflow) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetLastWorkflowOnce,
		exampleOutputGetLastWorkflowBytes,
		&exampleOutputGetLastWorkflow,
	)
}

func (c *GetRecentWorkflowRuns) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetRecentWorkflowRunsOnce,
		exampleOutputGetRecentWorkflowRunsBytes,
		&exampleOutputGetRecentWorkflowRuns,
	)
}

func (c *GetTestMetrics) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetTestMetricsOnce,
		exampleOutputGetTestMetricsBytes,
		&exampleOutputGetTestMetrics,
	)
}

func (c *GetFlakyTests) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetFlakyTestsOnce,
		exampleOutputGetFlakyTestsBytes,
		&exampleOutputGetFlakyTests,
	)
}

func (t *OnWorkflowCompleted) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnWorkflowCompletedOnce,
		exampleDataOnWorkflowCompletedBytes,
		&exampleDataOnWorkflowCompleted,
	)
}
