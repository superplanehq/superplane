package admin

import (
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_get_workflow_usage.json
var exampleOutputGetWorkflowUsageBytes []byte

var exampleOutputGetWorkflowUsageOnce sync.Once
var exampleOutputGetWorkflowUsage map[string]any

func (g *GetWorkflowUsage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetWorkflowUsageOnce, exampleOutputGetWorkflowUsageBytes, &exampleOutputGetWorkflowUsage)
}
