package claude

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_text_prompt.json
var exampleOutputTextPromptBytes []byte
var exampleOutputTextPrompt = utils.NewEmbeddedJSON(exampleOutputTextPromptBytes)

func (c *TextPrompt) ExampleOutput() map[string]any {
	return exampleOutputTextPrompt.Value()
}

//go:embed example_output_create_batch_message.json
var exampleOutputCreateBatchMessageBytes []byte
var exampleOutputCreateBatchMessage = utils.NewEmbeddedJSON(exampleOutputCreateBatchMessageBytes)

func (c *CreateBatchMessage) ExampleOutput() map[string]any {
	return exampleOutputCreateBatchMessage.Value()
}
