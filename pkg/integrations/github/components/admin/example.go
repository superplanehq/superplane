package admin

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/get_workflow_usage.json
var exampleOutputGetWorkflowUsageBytes []byte

var exampleOutputGetWorkflowUsageOnce sync.Once
var exampleOutputGetWorkflowUsage map[string]any

func (g *GetWorkflowUsage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetWorkflowUsageOnce, exampleOutputGetWorkflowUsageBytes, &exampleOutputGetWorkflowUsage)
}
