package datadog

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_event.json
var exampleOutputCreateEventBytes []byte
var exampleOutputCreateEvent = utils.NewEmbeddedJSON(exampleOutputCreateEventBytes)

func (c *CreateEvent) ExampleOutput() map[string]any {
	return exampleOutputCreateEvent.Value()
}
