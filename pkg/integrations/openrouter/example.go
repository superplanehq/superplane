package openrouter

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_text_prompt.json
var exampleOutputTextPromptBytes []byte

var exampleOutputTextPromptOnce sync.Once
var exampleOutputTextPrompt map[string]any

func (c *TextPrompt) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputTextPromptOnce, exampleOutputTextPromptBytes, &exampleOutputTextPrompt)
}
