package cursor

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_launch_agent.json
var exampleOutputLaunchAgentBytes []byte

//go:embed example_output_get_daily_usage.json
var exampleOutputGetDailyUsageBytes []byte

var exampleOutputLaunchAgentOnce sync.Once
var exampleOutputLaunchAgent map[string]any

var exampleOutputGetDailyUsageOnce sync.Once
var exampleOutputGetDailyUsage map[string]any

func (c *LaunchCloudAgent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputLaunchAgentOnce, exampleOutputLaunchAgentBytes, &exampleOutputLaunchAgent)
}

func (c *GetDailyUsageData) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetDailyUsageOnce, exampleOutputGetDailyUsageBytes, &exampleOutputGetDailyUsage)
}
