package opencost

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_cost_threshold.json
var exampleDataOnCostThresholdBytes []byte

//go:embed example_output_get_cost_allocation.json
var exampleOutputGetCostAllocationBytes []byte

var exampleDataOnCostThresholdOnce sync.Once
var exampleDataOnCostThreshold map[string]any

var exampleOutputGetCostAllocationOnce sync.Once
var exampleOutputGetCostAllocation map[string]any

func (t *OnCostThreshold) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnCostThresholdOnce, exampleDataOnCostThresholdBytes, &exampleDataOnCostThreshold)
}

func (c *GetCostAllocation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetCostAllocationOnce, exampleOutputGetCostAllocationBytes, &exampleOutputGetCostAllocation)
}
