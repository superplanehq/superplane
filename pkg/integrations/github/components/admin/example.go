package admin

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/get_workflow_usage.json
var exampleOutputGetWorkflowUsageBytes []byte
var exampleOutputGetWorkflowUsage = utils.NewEmbeddedJSON(exampleOutputGetWorkflowUsageBytes)

func (g *GetWorkflowUsage) ExampleOutput() map[string]any {
	return exampleOutputGetWorkflowUsage.Value()
}
