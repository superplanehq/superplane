package pagerduty

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte
var exampleOutputCreateIncident = utils.NewEmbeddedJSON(exampleOutputCreateIncidentBytes)

//go:embed example_output_update_incident.json
var exampleOutputUpdateIncidentBytes []byte
var exampleOutputUpdateIncident = utils.NewEmbeddedJSON(exampleOutputUpdateIncidentBytes)

//go:embed example_output_annotate_incident.json
var exampleOutputAnnotateIncidentBytes []byte
var exampleOutputAnnotateIncident = utils.NewEmbeddedJSON(exampleOutputAnnotateIncidentBytes)

//go:embed example_output_snooze_incident.json
var exampleOutputSnoozeIncidentBytes []byte
var exampleOutputSnoozeIncident = utils.NewEmbeddedJSON(exampleOutputSnoozeIncidentBytes)

//go:embed example_output_acknowledge_incident.json
var exampleOutputAcknowledgeIncidentBytes []byte
var exampleOutputAcknowledgeIncident = utils.NewEmbeddedJSON(exampleOutputAcknowledgeIncidentBytes)

//go:embed example_output_resolve_incident.json
var exampleOutputResolveIncidentBytes []byte
var exampleOutputResolveIncident = utils.NewEmbeddedJSON(exampleOutputResolveIncidentBytes)

//go:embed example_output_escalate_incident.json
var exampleOutputEscalateIncidentBytes []byte
var exampleOutputEscalateIncident = utils.NewEmbeddedJSON(exampleOutputEscalateIncidentBytes)

//go:embed example_data_on_incident.json
var exampleDataOnIncidentBytes []byte
var exampleDataOnIncident = utils.NewEmbeddedJSON(exampleDataOnIncidentBytes)

//go:embed example_data_on_incident_status_update.json
var exampleDataOnIncidentStatusUpdateBytes []byte
var exampleDataOnIncidentStatusUpdate = utils.NewEmbeddedJSON(exampleDataOnIncidentStatusUpdateBytes)

//go:embed example_data_on_incident_annotated.json
var exampleDataOnIncidentAnnotatedBytes []byte
var exampleDataOnIncidentAnnotated = utils.NewEmbeddedJSON(exampleDataOnIncidentAnnotatedBytes)

//go:embed example_output_list_incidents.json
var exampleOutputListIncidentsBytes []byte
var exampleOutputListIncidents = utils.NewEmbeddedJSON(exampleOutputListIncidentsBytes)

//go:embed example_output_list_notes.json
var exampleOutputListNotesBytes []byte
var exampleOutputListNotes = utils.NewEmbeddedJSON(exampleOutputListNotesBytes)

//go:embed example_output_list_log_entries.json
var exampleOutputListLogEntriesBytes []byte
var exampleOutputListLogEntries = utils.NewEmbeddedJSON(exampleOutputListLogEntriesBytes)

func (c *CreateIncident) ExampleOutput() map[string]any {
	return exampleOutputCreateIncident.Value()
}

func (c *UpdateIncident) ExampleOutput() map[string]any {
	return exampleOutputUpdateIncident.Value()
}

func (c *AnnotateIncident) ExampleOutput() map[string]any {
	return exampleOutputAnnotateIncident.Value()
}

func (c *SnoozeIncident) ExampleOutput() map[string]any {
	return exampleOutputSnoozeIncident.Value()
}

func (c *AcknowledgeIncident) ExampleOutput() map[string]any {
	return exampleOutputAcknowledgeIncident.Value()
}

func (c *ResolveIncident) ExampleOutput() map[string]any {
	return exampleOutputResolveIncident.Value()
}

func (c *EscalateIncident) ExampleOutput() map[string]any {
	return exampleOutputEscalateIncident.Value()
}

func (l *ListIncidents) ExampleOutput() map[string]any {
	return exampleOutputListIncidents.Value()
}

func (l *ListNotes) ExampleOutput() map[string]any {
	return exampleOutputListNotes.Value()
}

func (l *ListLogEntries) ExampleOutput() map[string]any {
	return exampleOutputListLogEntries.Value()
}

func (t *OnIncident) ExampleData() map[string]any {
	return exampleDataOnIncident.Value()
}

func (t *OnIncidentStatusUpdate) ExampleData() map[string]any {
	return exampleDataOnIncidentStatusUpdate.Value()
}

func (t *OnIncidentAnnotated) ExampleData() map[string]any {
	return exampleDataOnIncidentAnnotated.Value()
}
