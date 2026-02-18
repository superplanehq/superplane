package opencost

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_cost_exceeds_threshold.json
var exampleDataOnCostExceedsThresholdBytes []byte

//go:embed example_output_get_cost_allocation.json
var exampleOutputGetCostAllocationBytes []byte

var exampleDataOnCostExceedsThresholdOnce sync.Once
var exampleDataOnCostExceedsThresholdData map[string]any

var exampleOutputGetCostAllocationOnce sync.Once
var exampleOutputGetCostAllocationData map[string]any

func exampleDataOnCostExceedsThreshold() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnCostExceedsThresholdOnce, exampleDataOnCostExceedsThresholdBytes, &exampleDataOnCostExceedsThresholdData)
}

func (c *GetCostAllocation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetCostAllocationOnce, exampleOutputGetCostAllocationBytes, &exampleOutputGetCostAllocationData)
}
