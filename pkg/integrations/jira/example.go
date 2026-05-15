package jira

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_issue.json
var exampleOutputCreateIssueBytes []byte

var exampleOutputCreateIssueOnce sync.Once
var exampleOutputCreateIssue map[string]any

//go:embed example_output_create_incident.json
var exampleOutputCreateIncidentBytes []byte

var exampleOutputCreateIncidentOnce sync.Once
var exampleOutputCreateIncident map[string]any

//go:embed example_output_get_incident.json
var exampleOutputGetIncidentBytes []byte

var exampleOutputGetIncidentOnce sync.Once
var exampleOutputGetIncident map[string]any

//go:embed example_output_delete_incident.json
var exampleOutputDeleteIncidentBytes []byte

var exampleOutputDeleteIncidentOnce sync.Once
var exampleOutputDeleteIncident map[string]any

//go:embed example_output_create_alert.json
var exampleOutputCreateAlertBytes []byte

var exampleOutputCreateAlertOnce sync.Once
var exampleOutputCreateAlert map[string]any

//go:embed example_output_get_alert.json
var exampleOutputGetAlertBytes []byte

var exampleOutputGetAlertOnce sync.Once
var exampleOutputGetAlert map[string]any

//go:embed example_output_delete_alert.json
var exampleOutputDeleteAlertBytes []byte

var exampleOutputDeleteAlertOnce sync.Once
var exampleOutputDeleteAlert map[string]any

//go:embed example_output_update_alert.json
var exampleOutputUpdateAlertBytes []byte

var exampleOutputUpdateAlertOnce sync.Once
var exampleOutputUpdateAlert map[string]any

func (c *CreateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIssueOnce, exampleOutputCreateIssueBytes, &exampleOutputCreateIssue)
}

//go:embed example_output_get_issue.json
var exampleOutputGetIssueBytes []byte

var exampleOutputGetIssueOnce sync.Once
var exampleOutputGetIssue map[string]any

func (c *GetIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetIssueOnce, exampleOutputGetIssueBytes, &exampleOutputGetIssue)
}

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssueBytes []byte

var exampleOutputUpdateIssueOnce sync.Once
var exampleOutputUpdateIssue map[string]any

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIssueOnce, exampleOutputUpdateIssueBytes, &exampleOutputUpdateIssue)
}

//go:embed example_output_delete_issue.json
var exampleOutputDeleteIssueBytes []byte

var exampleOutputDeleteIssueOnce sync.Once
var exampleOutputDeleteIssue map[string]any

func (c *DeleteIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteIssueOnce, exampleOutputDeleteIssueBytes, &exampleOutputDeleteIssue)
}

func (c *CreateIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIncidentOnce, exampleOutputCreateIncidentBytes, &exampleOutputCreateIncident)
}

func (c *GetIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetIncidentOnce, exampleOutputGetIncidentBytes, &exampleOutputGetIncident)
}

func (c *DeleteIncident) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteIncidentOnce, exampleOutputDeleteIncidentBytes, &exampleOutputDeleteIncident)
}

func (c *CreateAlert) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateAlertOnce, exampleOutputCreateAlertBytes, &exampleOutputCreateAlert)
}

func (c *GetAlert) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetAlertOnce, exampleOutputGetAlertBytes, &exampleOutputGetAlert)
}

func (c *DeleteAlert) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteAlertOnce, exampleOutputDeleteAlertBytes, &exampleOutputDeleteAlert)
}

func (c *UpdateAlert) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateAlertOnce, exampleOutputUpdateAlertBytes, &exampleOutputUpdateAlert)
}
