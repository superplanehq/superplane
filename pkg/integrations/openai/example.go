package openai

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_text_prompt.json
var exampleOutputTextPromptBytes []byte
var exampleOutputTextPrompt = utils.NewEmbeddedJSON(exampleOutputTextPromptBytes)

func (c *CreateResponse) ExampleOutput() map[string]any {
	return exampleOutputTextPrompt.Value()
}
