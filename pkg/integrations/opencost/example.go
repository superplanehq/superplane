package opencost

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_cost_allocation.json
var exampleOutputGetCostAllocationBytes []byte

//go:embed example_data_cost_exceeds_threshold.json
var exampleDataCostExceedsThresholdBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

var exampleDataOnce sync.Once
var exampleData map[string]any

func (c *GetCostAllocation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputGetCostAllocationBytes, &exampleOutput)
}

func exampleDataCostExceedsThresholdParsed() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnce, exampleDataCostExceedsThresholdBytes, &exampleData)
}
