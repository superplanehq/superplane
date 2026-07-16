package cursor

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_launch_agent.json
var exampleOutputLaunchAgentBytes []byte
var exampleOutputLaunchAgent = utils.NewEmbeddedJSON(exampleOutputLaunchAgentBytes)

func getLaunchAgentExampleOutput() map[string]any {
	return exampleOutputLaunchAgent.Value()
}
