package opencodego

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

//go:embed example_output_get_usage.json
var exampleOutputGetUsageBytes []byte

var exampleOutputGetUsageOnce sync.Once
var exampleOutputGetUsage map[string]any

func (c *GetUsage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetUsageOnce, exampleOutputGetUsageBytes, &exampleOutputGetUsage)
}
