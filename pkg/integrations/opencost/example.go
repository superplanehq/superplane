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
var exampleDataOnCostExceedsThreshold map[string]any

var exampleOutputGetCostAllocationOnce sync.Once
var exampleOutputGetCostAllocation map[string]any

func (t *OnCostExceedsThreshold) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnCostExceedsThresholdOnce, exampleDataOnCostExceedsThresholdBytes, &exampleDataOnCostExceedsThreshold)
}

func (c *GetCostAllocation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetCostAllocationOnce, exampleOutputGetCostAllocationBytes, &exampleOutputGetCostAllocation)
}
