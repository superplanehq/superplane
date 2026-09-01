package openrouter

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_chat_completion.json
var exampleOutputChatCompletionBytes []byte

var exampleOutputChatCompletionOnce sync.Once
var exampleOutputChatCompletion map[string]any

func (c *ChatCompletion) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputChatCompletionOnce, exampleOutputChatCompletionBytes, &exampleOutputChatCompletion)
}

//go:embed example_output_get_credits.json
var exampleOutputGetCreditsBytes []byte

var exampleOutputGetCreditsOnce sync.Once
var exampleOutputGetCredits map[string]any

func (c *GetCredits) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetCreditsOnce, exampleOutputGetCreditsBytes, &exampleOutputGetCredits)
}
