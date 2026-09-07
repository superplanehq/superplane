package checks

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/on_check_run.json
var exampleDataOnCheckRunBytes []byte

//go:embed payloads/list_check_runs_for_ref.json
var exampleOutputListCheckRunsForRefBytes []byte

//go:embed payloads/wait_for_pull_request_checks.json
var exampleOutputWaitForPullRequestChecksBytes []byte

var exampleDataOnCheckRunOnce sync.Once
var exampleDataOnCheckRun map[string]any

var exampleOutputListCheckRunsForRefOnce sync.Once
var exampleOutputListCheckRunsForRef map[string]any

var exampleOutputWaitForPullRequestChecksOnce sync.Once
var exampleOutputWaitForPullRequestChecks map[string]any

func (t *OnCheckRun) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnCheckRunOnce, exampleDataOnCheckRunBytes, &exampleDataOnCheckRun)
}

func (c *ListCheckRunsForRef) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputListCheckRunsForRefOnce,
		exampleOutputListCheckRunsForRefBytes,
		&exampleOutputListCheckRunsForRef,
	)
}

func (c *WaitForPullRequestChecks) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputWaitForPullRequestChecksOnce,
		exampleOutputWaitForPullRequestChecksBytes,
		&exampleOutputWaitForPullRequestChecks,
	)
}
