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

//go:embed example_data_on_monitor_alert.json
var exampleDataOnMonitorAlertBytes []byte

var exampleDataOnMonitorAlertOnce sync.Once
var exampleDataOnMonitorAlert map[string]any

func (c *CreateEvent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateEventOnce, exampleOutputCreateEventBytes, &exampleOutputCreateEvent)
}

func (t *OnMonitorAlert) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnMonitorAlertOnce, exampleDataOnMonitorAlertBytes, &exampleDataOnMonitorAlert)
}
