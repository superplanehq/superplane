package checks

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/on_check_run.json
var exampleDataOnCheckRunBytes []byte

//go:embed payloads/list_check_runs_for_ref.json
var exampleOutputListCheckRunsForRefBytes []byte
var exampleDataOnCheckRun = utils.NewEmbeddedJSON(exampleDataOnCheckRunBytes)
var exampleOutputListCheckRunsForRef = utils.NewEmbeddedJSON(exampleOutputListCheckRunsForRefBytes)

func (t *OnCheckRun) ExampleData() map[string]any {
	return exampleDataOnCheckRun.Value()
}

func (c *ListCheckRunsForRef) ExampleOutput() map[string]any {
	return exampleOutputListCheckRunsForRef.Value()
}
