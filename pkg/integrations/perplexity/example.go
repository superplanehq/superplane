package perplexity

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_agent.json
var exampleOutputRunAgentBytes []byte
var exampleOutputRunAgent = utils.NewEmbeddedJSON(exampleOutputRunAgentBytes)

func (c *runAgent) ExampleOutput() map[string]any {
	return exampleOutputRunAgent.Value()
}
