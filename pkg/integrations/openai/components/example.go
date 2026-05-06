package components

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/text_prompt.json
var exampleOutputTextPromptBytes []byte

var exampleOutputTextPromptOnce sync.Once
var exampleOutputTextPrompt map[string]any

func (c *CreateResponse) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputTextPromptOnce, exampleOutputTextPromptBytes, &exampleOutputTextPrompt)
}
