package claude

import (
	_ "embed"
	"github.com/superplanehq/superplane/pkg/utils"
	"sync"
)

//go:embed example_output_create_message.json
var exampleOutputCreateMessageBytes []byte

var exampleOutputCreateMessageOnce sync.Once
var exampleOutputCreateMessage map[string]any

func (c *CreateMessage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateMessageOnce, exampleOutputCreateMessageBytes, &exampleOutputCreateMessage)
}
