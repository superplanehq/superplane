package servicenow

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte
var exampleOutputCreateIncident = utils.NewEmbeddedJSON(exampleOutputCreateIncidentBytes)

//go:embed example_output_get_incident.json
var exampleOutputGetIncidentBytes []byte
var exampleOutputGetIncident = utils.NewEmbeddedJSON(exampleOutputGetIncidentBytes)

func (c *CreateIncident) ExampleOutput() map[string]any {
	return exampleOutputCreateIncident.Value()
}

func (c *GetIncident) ExampleOutput() map[string]any {
	return exampleOutputGetIncident.Value()
}
