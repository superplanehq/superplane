package datadog

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_event.json
var exampleOutputCreateEventBytes []byte

var exampleOutputCreateEventOnce sync.Once
var exampleOutputCreateEvent map[string]any

func (c *CreateEvent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateEventOnce, exampleOutputCreateEventBytes, &exampleOutputCreateEvent)
}
