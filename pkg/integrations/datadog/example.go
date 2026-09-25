package datadog

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_event.json
var exampleOutputCreateEventBytes []byte

//go:embed example_data_on_error_tracking_alert.json
var exampleDataOnErrorTrackingAlertBytes []byte

var exampleOutputCreateEventOnce sync.Once
var exampleOutputCreateEvent map[string]any

var exampleDataOnErrorTrackingAlertOnce sync.Once
var exampleDataOnErrorTrackingAlert map[string]any

func (c *CreateEvent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateEventOnce, exampleOutputCreateEventBytes, &exampleOutputCreateEvent)
}

func onErrorTrackingAlertExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnErrorTrackingAlertOnce,
		exampleDataOnErrorTrackingAlertBytes,
		&exampleDataOnErrorTrackingAlert,
	)
}
