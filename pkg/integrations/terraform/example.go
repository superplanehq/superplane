package terraform

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_plan.json
var exampleOutputRunPlanBytes []byte

//go:embed example_data_on_run_event.json
var exampleDataOnRunEventBytes []byte

var exampleOutputRunPlanOnce sync.Once
var exampleOutputRunPlan map[string]any

var exampleDataOnRunEventOnce sync.Once
var exampleDataOnRunEvent map[string]any

func (c *RunPlan) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunPlanOnce, exampleOutputRunPlanBytes, &exampleOutputRunPlan)
}

func (t *RunEvent) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnRunEventOnce, exampleDataOnRunEventBytes, &exampleDataOnRunEvent)
}
