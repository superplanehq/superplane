package perplexity

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_agent.json
var exampleOutputRunAgentBytes []byte

var exampleOutputRunAgentOnce sync.Once
var exampleOutputRunAgent map[string]any

func (c *runAgent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunAgentOnce, exampleOutputRunAgentBytes, &exampleOutputRunAgent)
}
