package rootly

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte
var exampleOutputCreateIncident = utils.NewEmbeddedJSON(exampleOutputCreateIncidentBytes)

//go:embed example_output_create_event.json
var exampleOutputCreateEventBytes []byte
var exampleOutputCreateEvent = utils.NewEmbeddedJSON(exampleOutputCreateEventBytes)

//go:embed example_output_update_incident.json
var exampleOutputUpdateIncidentBytes []byte
var exampleOutputUpdateIncident = utils.NewEmbeddedJSON(exampleOutputUpdateIncidentBytes)

//go:embed example_data_on_incident.json
var exampleDataOnIncidentBytes []byte
var exampleDataOnIncident = utils.NewEmbeddedJSON(exampleDataOnIncidentBytes)

//go:embed example_data_on_incident_timeline_event.json
var exampleDataOnIncidentTimelineEventBytes []byte
var exampleDataOnIncidentTimelineEvent = utils.NewEmbeddedJSON(exampleDataOnIncidentTimelineEventBytes)

func (c *CreateIncident) ExampleOutput() map[string]any {
	return exampleOutputCreateIncident.Value()
}

func (c *CreateEvent) ExampleOutput() map[string]any {
	return exampleOutputCreateEvent.Value()
}

func (c *UpdateIncident) ExampleOutput() map[string]any {
	return exampleOutputUpdateIncident.Value()
}

func (t *OnIncident) ExampleData() map[string]any {
	return exampleDataOnIncident.Value()
}

//go:embed example_output_get_incident.json
var exampleOutputGetIncidentBytes []byte
var exampleOutputGetIncident = utils.NewEmbeddedJSON(exampleOutputGetIncidentBytes)

func (c *GetIncident) ExampleOutput() map[string]any {
	return exampleOutputGetIncident.Value()
}

func (t *OnIncidentTimelineEvent) ExampleData() map[string]any {
	return exampleDataOnIncidentTimelineEvent.Value()
}
