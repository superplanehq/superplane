package firehydrant

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte

var exampleOutputCreateIncidentOnce sync.Once
var exampleOutputCreateIncident map[string]any

//go:embed example_data_on_new_incident.json
var exampleDataOnNewIncidentBytes []byte

var exampleDataOnNewIncidentOnce sync.Once
var exampleDataOnNewIncident map[string]any

func (c *CreateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIncidentOnce, exampleOutputCreateIncidentBytes, &exampleOutputCreateIncident)
}

func (t *OnNewIncident) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnNewIncidentOnce, exampleDataOnNewIncidentBytes, &exampleDataOnNewIncident)
}
