package honeycomb

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_alert_fired.json
var exampleDataOnAlertFiredBytes []byte

//go:embed example_output_create_event.json
var exampleOutputCreateEventBytes []byte

var (
	exampleDataOnAlertFired  = utils.NewEmbeddedJSON(exampleDataOnAlertFiredBytes)
	exampleOutputCreateEvent = utils.NewEmbeddedJSON(exampleOutputCreateEventBytes)
)

func embeddedExampleDataOnAlertFired() map[string]any {
	return exampleDataOnAlertFired.Value()
}

func embeddedExampleOutputCreateEvent() map[string]any {
	return exampleOutputCreateEvent.Value()
}

func (t *OnAlertFired) ExampleData() map[string]any {
	return embeddedExampleDataOnAlertFired()
}

func (c *CreateEvent) ExampleOutput() map[string]any {
	return embeddedExampleOutputCreateEvent()
}
