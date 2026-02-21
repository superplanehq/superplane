package pagerduty

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte

var exampleOutputCreateIncidentOnce sync.Once
var exampleOutputCreateIncident map[string]any

//go:embed example_output_update_incident.json
var exampleOutputUpdateIncidentBytes []byte

var exampleOutputUpdateIncidentOnce sync.Once
var exampleOutputUpdateIncident map[string]any

//go:embed example_output_annotate_incident.json
var exampleOutputAnnotateIncidentBytes []byte

var exampleOutputAnnotateIncidentOnce sync.Once
var exampleOutputAnnotateIncident map[string]any

//go:embed example_output_snooze_incident.json
var exampleOutputSnoozeIncidentBytes []byte

var exampleOutputSnoozeIncidentOnce sync.Once
var exampleOutputSnoozeIncident map[string]any

//go:embed example_output_acknowledge_incident.json
var exampleOutputAcknowledgeIncidentBytes []byte

var exampleOutputAcknowledgeIncidentOnce sync.Once
var exampleOutputAcknowledgeIncident map[string]any

//go:embed example_output_resolve_incident.json
var exampleOutputResolveIncidentBytes []byte

var exampleOutputResolveIncidentOnce sync.Once
var exampleOutputResolveIncident map[string]any

//go:embed example_output_escalate_incident.json
var exampleOutputEscalateIncidentBytes []byte

var exampleOutputEscalateIncidentOnce sync.Once
var exampleOutputEscalateIncident map[string]any

//go:embed example_data_on_incident.json
var exampleDataOnIncidentBytes []byte

var exampleDataOnIncidentOnce sync.Once
var exampleDataOnIncident map[string]any

//go:embed example_data_on_incident_status_update.json
var exampleDataOnIncidentStatusUpdateBytes []byte

var exampleDataOnIncidentStatusUpdateOnce sync.Once
var exampleDataOnIncidentStatusUpdate map[string]any

//go:embed example_data_on_incident_annotated.json
var exampleDataOnIncidentAnnotatedBytes []byte

var exampleDataOnIncidentAnnotatedOnce sync.Once
var exampleDataOnIncidentAnnotated map[string]any

//go:embed example_output_list_incidents.json
var exampleOutputListIncidentsBytes []byte

var exampleOutputListIncidentsOnce sync.Once
var exampleOutputListIncidents map[string]any

//go:embed example_output_list_notes.json
var exampleOutputListNotesBytes []byte

var exampleOutputListNotesOnce sync.Once
var exampleOutputListNotes map[string]any

//go:embed example_output_list_log_entries.json
var exampleOutputListLogEntriesBytes []byte

var exampleOutputListLogEntriesOnce sync.Once
var exampleOutputListLogEntries map[string]any

func (c *CreateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIncidentOnce, exampleOutputCreateIncidentBytes, &exampleOutputCreateIncident)
}

func (c *UpdateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIncidentOnce, exampleOutputUpdateIncidentBytes, &exampleOutputUpdateIncident)
}

func (c *AnnotateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputAnnotateIncidentOnce, exampleOutputAnnotateIncidentBytes, &exampleOutputAnnotateIncident)
}

func (c *SnoozeIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputSnoozeIncidentOnce, exampleOutputSnoozeIncidentBytes, &exampleOutputSnoozeIncident)
}

func (c *AcknowledgeIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputAcknowledgeIncidentOnce, exampleOutputAcknowledgeIncidentBytes, &exampleOutputAcknowledgeIncident)
}

func (c *ResolveIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputResolveIncidentOnce, exampleOutputResolveIncidentBytes, &exampleOutputResolveIncident)
}

func (c *EscalateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputEscalateIncidentOnce, exampleOutputEscalateIncidentBytes, &exampleOutputEscalateIncident)
}

func (l *ListIncidents) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListIncidentsOnce, exampleOutputListIncidentsBytes, &exampleOutputListIncidents)
}

func (l *ListNotes) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListNotesOnce, exampleOutputListNotesBytes, &exampleOutputListNotes)
}

func (l *ListLogEntries) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListLogEntriesOnce, exampleOutputListLogEntriesBytes, &exampleOutputListLogEntries)
}

func (t *OnIncident) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIncidentOnce, exampleDataOnIncidentBytes, &exampleDataOnIncident)
}

func (t *OnIncidentStatusUpdate) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIncidentStatusUpdateOnce, exampleDataOnIncidentStatusUpdateBytes, &exampleDataOnIncidentStatusUpdate)
}

func (t *OnIncidentAnnotated) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIncidentAnnotatedOnce, exampleDataOnIncidentAnnotatedBytes, &exampleDataOnIncidentAnnotated)
}
