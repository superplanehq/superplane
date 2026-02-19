package servicenow

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte

var exampleOutputCreateIncidentOnce sync.Once
var exampleOutputCreateIncident map[string]any

//go:embed example_output_get_incidents.json
var exampleOutputGetIncidentsBytes []byte

var exampleOutputGetIncidentsOnce sync.Once
var exampleOutputGetIncidents map[string]any

func (c *CreateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIncidentOnce, exampleOutputCreateIncidentBytes, &exampleOutputCreateIncident)
}

func (c *GetIncidents) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetIncidentsOnce, exampleOutputGetIncidentsBytes, &exampleOutputGetIncidents)
}
