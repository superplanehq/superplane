package honeycomb

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_alert_fired.json
var exampleDataOnAlertFiredBytes []byte

//go:embed example_output_create_event.json
var exampleOutputCreateEventBytes []byte

var (
	exampleDataOnAlertFiredOnce sync.Once
	exampleDataOnAlertFired     map[string]any

	exampleOutputCreateEventOnce sync.Once
	exampleOutputCreateEvent     map[string]any
)

func embeddedExampleDataOnAlertFired() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnAlertFiredOnce,
		exampleDataOnAlertFiredBytes,
		&exampleDataOnAlertFired,
	)
}

func embeddedExampleOutputCreateEvent() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateEventOnce,
		exampleOutputCreateEventBytes,
		&exampleOutputCreateEvent,
	)
}

func (t *OnAlertFired) ExampleData() map[string]any {
	return embeddedExampleDataOnAlertFired()
}

func (c *CreateEvent) ExampleOutput() map[string]any {
	return embeddedExampleOutputCreateEvent()
}
