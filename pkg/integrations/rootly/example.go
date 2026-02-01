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

//go:embed example_data_on_incident_created.json
var exampleDataOnIncidentCreatedBytes []byte

var exampleDataOnIncidentCreatedOnce sync.Once
var exampleDataOnIncidentCreated map[string]any

//go:embed example_data_on_incident_resolved.json
var exampleDataOnIncidentResolvedBytes []byte

var exampleDataOnIncidentResolvedOnce sync.Once
var exampleDataOnIncidentResolved map[string]any

func (c *CreateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIncidentOnce, exampleOutputCreateIncidentBytes, &exampleOutputCreateIncident)
}

func (t *OnIncidentCreated) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIncidentCreatedOnce, exampleDataOnIncidentCreatedBytes, &exampleDataOnIncidentCreated)
}

func (t *OnIncidentResolved) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIncidentResolvedOnce, exampleDataOnIncidentResolvedBytes, &exampleDataOnIncidentResolved)
}
