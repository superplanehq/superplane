package firehydrant

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_incident.json
var exampleDataOnIncidentBytes []byte

var exampleDataOnIncidentOnce sync.Once
var exampleDataOnIncident map[string]any

func (t *OnIncident) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIncidentOnce, exampleDataOnIncidentBytes, &exampleDataOnIncident)
}

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte

var exampleOutputCreateIncidentOnce sync.Once
var exampleOutputCreateIncident map[string]any

func (c *CreateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIncidentOnce, exampleOutputCreateIncidentBytes, &exampleOutputCreateIncident)
}
