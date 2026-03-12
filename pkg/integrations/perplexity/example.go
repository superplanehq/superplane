package perplexity

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_web_search.json
var exampleOutputWebSearchBytes []byte

//go:embed example_output_run_agent.json
var exampleOutputRunAgentBytes []byte

var exampleOutputWebSearchOnce sync.Once
var exampleOutputWebSearch map[string]any

var exampleOutputRunAgentOnce sync.Once
var exampleOutputRunAgent map[string]any

func (c *webSearch) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputWebSearchOnce, exampleOutputWebSearchBytes, &exampleOutputWebSearch)
}

func (c *runAgent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunAgentOnce, exampleOutputRunAgentBytes, &exampleOutputRunAgent)
}
