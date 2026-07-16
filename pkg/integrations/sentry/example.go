package sentry

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte
var exampleDataOnIssue = utils.NewEmbeddedJSON(exampleDataOnIssueBytes)

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssueBytes []byte
var exampleOutputUpdateIssue = utils.NewEmbeddedJSON(exampleOutputUpdateIssueBytes)

//go:embed example_output_get_issue.json
var exampleOutputGetIssueBytes []byte
var exampleOutputGetIssue = utils.NewEmbeddedJSON(exampleOutputGetIssueBytes)

//go:embed example_output_create_alert.json
var exampleOutputCreateAlertBytes []byte
var exampleOutputCreateAlert = utils.NewEmbeddedJSON(exampleOutputCreateAlertBytes)

//go:embed example_output_delete_alert.json
var exampleOutputDeleteAlertBytes []byte
var exampleOutputDeleteAlert = utils.NewEmbeddedJSON(exampleOutputDeleteAlertBytes)

//go:embed example_output_update_alert.json
var exampleOutputUpdateAlertBytes []byte
var exampleOutputUpdateAlert = utils.NewEmbeddedJSON(exampleOutputUpdateAlertBytes)

//go:embed example_output_list_alerts.json
var exampleOutputListAlertsBytes []byte
var exampleOutputListAlerts = utils.NewEmbeddedJSON(exampleOutputListAlertsBytes)

//go:embed example_output_get_alert.json
var exampleOutputGetAlertBytes []byte
var exampleOutputGetAlert = utils.NewEmbeddedJSON(exampleOutputGetAlertBytes)

//go:embed example_output_create_release.json
var exampleOutputCreateReleaseBytes []byte
var exampleOutputCreateRelease = utils.NewEmbeddedJSON(exampleOutputCreateReleaseBytes)

//go:embed example_output_create_deploy.json
var exampleOutputCreateDeployBytes []byte
var exampleOutputCreateDeploy = utils.NewEmbeddedJSON(exampleOutputCreateDeployBytes)

//go:embed example_output_link_github_issue.json
var exampleOutputLinkGitHubIssueBytes []byte
var exampleOutputLinkGitHubIssue = utils.NewEmbeddedJSON(exampleOutputLinkGitHubIssueBytes)

func (t *OnIssue) ExampleData() map[string]any {
	return exampleDataOnIssue.Value()
}

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return exampleOutputUpdateIssue.Value()
}

func (c *CreateAlert) ExampleOutput() map[string]any {
	return exampleOutputCreateAlert.Value()
}

func (c *UpdateAlert) ExampleOutput() map[string]any {
	return exampleOutputUpdateAlert.Value()
}

func (c *DeleteAlert) ExampleOutput() map[string]any {
	return exampleOutputDeleteAlert.Value()
}

func (c *ListAlerts) ExampleOutput() map[string]any {
	return exampleOutputListAlerts.Value()
}

func (c *GetAlert) ExampleOutput() map[string]any {
	return exampleOutputGetAlert.Value()
}

func (c *GetIssue) ExampleOutput() map[string]any {
	return exampleOutputGetIssue.Value()
}

func (c *CreateRelease) ExampleOutput() map[string]any {
	return exampleOutputCreateRelease.Value()
}

func (c *CreateDeploy) ExampleOutput() map[string]any {
	return exampleOutputCreateDeploy.Value()
}

func (c *LinkGitHubIssue) ExampleOutput() map[string]any {
	return exampleOutputLinkGitHubIssue.Value()
}
