package rootly

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte

var exampleOutputCreateIncidentOnce sync.Once
var exampleOutputCreateIncident map[string]any

//go:embed example_output_create_event.json
var exampleOutputCreateEventBytes []byte

var exampleOutputCreateEventOnce sync.Once
var exampleOutputCreateEvent map[string]any

//go:embed example_output_update_incident.json
var exampleOutputUpdateIncidentBytes []byte

var exampleOutputUpdateIncidentOnce sync.Once
var exampleOutputUpdateIncident map[string]any

//go:embed example_data_on_incident.json
var exampleDataOnIncidentBytes []byte

var exampleDataOnIncidentOnce sync.Once
var exampleDataOnIncident map[string]any

func (c *CreateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIncidentOnce, exampleOutputCreateIncidentBytes, &exampleOutputCreateIncident)
}

func (c *CreateEvent) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateEventOnce, exampleOutputCreateEventBytes, &exampleOutputCreateEvent)
}

func (c *UpdateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIncidentOnce, exampleOutputUpdateIncidentBytes, &exampleOutputUpdateIncident)
}

func (t *OnIncident) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIncidentOnce, exampleDataOnIncidentBytes, &exampleDataOnIncident)
}

//go:embed example_output_get_incident.json
var exampleOutputGetIncidentBytes []byte

var exampleOutputGetIncidentOnce sync.Once
var exampleOutputGetIncident map[string]any

func (c *GetIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetIncidentOnce, exampleOutputGetIncidentBytes, &exampleOutputGetIncident)
}
