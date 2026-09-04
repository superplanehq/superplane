package posthog

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_event.json
var exampleDataOnEventBytes []byte

var exampleDataOnEventOnce sync.Once
var exampleDataOnEvent map[string]any

//go:embed example_output_run_query.json
var exampleOutputRunQueryBytes []byte

var exampleOutputRunQueryOnce sync.Once
var exampleOutputRunQuery map[string]any

func (t *OnEvent) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnEventOnce, exampleDataOnEventBytes, &exampleDataOnEvent)
}

func (c *RunQuery) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunQueryOnce, exampleOutputRunQueryBytes, &exampleOutputRunQuery)
}
