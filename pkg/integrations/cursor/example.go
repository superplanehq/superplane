package cursor

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_launch_agent.json
var exampleOutputLaunchAgentBytes []byte

var exampleOutputLaunchAgentOnce sync.Once
var exampleOutputLaunchAgent map[string]any

func (l *LaunchAgent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputLaunchAgentOnce, exampleOutputLaunchAgentBytes, &exampleOutputLaunchAgent)
}
