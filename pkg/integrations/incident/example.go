package incident

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte
var exampleOutputCreateIncident = utils.NewEmbeddedJSON(exampleOutputCreateIncidentBytes)

//go:embed example_data_on_incident.json
var exampleDataOnIncidentBytes []byte
var exampleDataOnIncident = utils.NewEmbeddedJSON(exampleDataOnIncidentBytes)

func (c *CreateIncident) ExampleOutput() map[string]any {
	return exampleOutputCreateIncident.Value()
}

func (t *OnIncident) ExampleData() map[string]any {
	return exampleDataOnIncident.Value()
}
