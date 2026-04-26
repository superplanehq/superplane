package claude

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_text_prompt.json
var exampleOutputTextPromptBytes []byte

//go:embed example_output_run_agent.json
var exampleOutputRunAgentBytes []byte

var exampleOutputTextPromptOnce sync.Once
var exampleOutputTextPrompt map[string]any

var exampleOutputRunAgentOnce sync.Once
var exampleOutputRunAgent map[string]any

func (c *TextPrompt) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputTextPromptOnce, exampleOutputTextPromptBytes, &exampleOutputTextPrompt)
}

func getRunAgentExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunAgentOnce, exampleOutputRunAgentBytes, &exampleOutputRunAgent)
}
