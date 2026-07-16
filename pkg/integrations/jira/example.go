package jira

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_issue.json
var exampleOutputCreateIssueBytes []byte
var exampleOutputCreateIssue = utils.NewEmbeddedJSON(exampleOutputCreateIssueBytes)

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte
var exampleOutputCreateIncident = utils.NewEmbeddedJSON(exampleOutputCreateIncidentBytes)

//go:embed example_output_get_incident.json
var exampleOutputGetIncidentBytes []byte
var exampleOutputGetIncident = utils.NewEmbeddedJSON(exampleOutputGetIncidentBytes)

//go:embed example_output_delete_incident.json
var exampleOutputDeleteIncidentBytes []byte
var exampleOutputDeleteIncident = utils.NewEmbeddedJSON(exampleOutputDeleteIncidentBytes)

//go:embed example_output_transition_issue.json
var exampleOutputTransitionIssueBytes []byte
var exampleOutputTransitionIssue = utils.NewEmbeddedJSON(exampleOutputTransitionIssueBytes)

//go:embed example_output_approve_workflow.json
var exampleOutputApproveWorkflowBytes []byte
var exampleOutputApproveWorkflow = utils.NewEmbeddedJSON(exampleOutputApproveWorkflowBytes)

//go:embed example_output_get_workflow.json
var exampleOutputGetWorkflowBytes []byte
var exampleOutputGetWorkflow = utils.NewEmbeddedJSON(exampleOutputGetWorkflowBytes)

//go:embed example_output_create_alert.json
var exampleOutputCreateAlertBytes []byte
var exampleOutputCreateAlert = utils.NewEmbeddedJSON(exampleOutputCreateAlertBytes)

//go:embed example_output_get_alert.json
var exampleOutputGetAlertBytes []byte
var exampleOutputGetAlert = utils.NewEmbeddedJSON(exampleOutputGetAlertBytes)

//go:embed example_output_delete_alert.json
var exampleOutputDeleteAlertBytes []byte
var exampleOutputDeleteAlert = utils.NewEmbeddedJSON(exampleOutputDeleteAlertBytes)

//go:embed example_output_update_alert.json
var exampleOutputUpdateAlertBytes []byte
var exampleOutputUpdateAlert = utils.NewEmbeddedJSON(exampleOutputUpdateAlertBytes)

func (c *CreateIssue) ExampleOutput() map[string]any {
	return exampleOutputCreateIssue.Value()
}

//go:embed example_output_get_issue.json
var exampleOutputGetIssueBytes []byte
var exampleOutputGetIssue = utils.NewEmbeddedJSON(exampleOutputGetIssueBytes)

func (c *GetIssue) ExampleOutput() map[string]any {
	return exampleOutputGetIssue.Value()
}

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssueBytes []byte
var exampleOutputUpdateIssue = utils.NewEmbeddedJSON(exampleOutputUpdateIssueBytes)

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return exampleOutputUpdateIssue.Value()
}

//go:embed example_output_delete_issue.json
var exampleOutputDeleteIssueBytes []byte
var exampleOutputDeleteIssue = utils.NewEmbeddedJSON(exampleOutputDeleteIssueBytes)

func (c *DeleteIssue) ExampleOutput() map[string]any {
	return exampleOutputDeleteIssue.Value()
}

func (c *CreateIncident) ExampleOutput() map[string]any {
	return exampleOutputCreateIncident.Value()
}

func (c *GetIncident) ExampleOutput() map[string]any {
	return exampleOutputGetIncident.Value()
}

func (c *DeleteIncident) ExampleOutput() map[string]any {
	return exampleOutputDeleteIncident.Value()
}

func (c *TransitionIssue) ExampleOutput() map[string]any {
	return exampleOutputTransitionIssue.Value()
}

func (c *ApproveWorkflow) ExampleOutput() map[string]any {
	return exampleOutputApproveWorkflow.Value()
}

func (c *GetWorkflow) ExampleOutput() map[string]any {
	return exampleOutputGetWorkflow.Value()
}

//go:embed example_output_create_heartbeat.json
var exampleOutputCreateHeartbeatBytes []byte
var exampleOutputCreateHeartbeat = utils.NewEmbeddedJSON(exampleOutputCreateHeartbeatBytes)

//go:embed example_output_ping_heartbeat.json
var exampleOutputPingHeartbeatBytes []byte
var exampleOutputPingHeartbeat = utils.NewEmbeddedJSON(exampleOutputPingHeartbeatBytes)

//go:embed example_output_update_heartbeat.json
var exampleOutputUpdateHeartbeatBytes []byte
var exampleOutputUpdateHeartbeat = utils.NewEmbeddedJSON(exampleOutputUpdateHeartbeatBytes)

//go:embed example_output_delete_heartbeat.json
var exampleOutputDeleteHeartbeatBytes []byte
var exampleOutputDeleteHeartbeat = utils.NewEmbeddedJSON(exampleOutputDeleteHeartbeatBytes)

func (c *CreateHeartbeat) ExampleOutput() map[string]any {
	return exampleOutputCreateHeartbeat.Value()
}

func (c *PingHeartbeat) ExampleOutput() map[string]any {
	return exampleOutputPingHeartbeat.Value()
}

func (c *UpdateHeartbeat) ExampleOutput() map[string]any {
	return exampleOutputUpdateHeartbeat.Value()
}

func (c *DeleteHeartbeat) ExampleOutput() map[string]any {
	return exampleOutputDeleteHeartbeat.Value()
}

func (c *CreateAlert) ExampleOutput() map[string]any {
	return exampleOutputCreateAlert.Value()
}

func (c *GetAlert) ExampleOutput() map[string]any {
	return exampleOutputGetAlert.Value()
}

func (c *DeleteAlert) ExampleOutput() map[string]any {
	return exampleOutputDeleteAlert.Value()
}

func (c *UpdateAlert) ExampleOutput() map[string]any {
	return exampleOutputUpdateAlert.Value()
}
