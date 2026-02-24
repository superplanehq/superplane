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
