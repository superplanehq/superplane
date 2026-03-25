package sentry

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte

var exampleDataOnIssueOnce sync.Once
var exampleDataOnIssue map[string]any

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssueBytes []byte

var exampleOutputUpdateIssueOnce sync.Once
var exampleOutputUpdateIssue map[string]any

//go:embed example_output_get_issue.json
var exampleOutputGetIssueBytes []byte

var exampleOutputGetIssueOnce sync.Once
var exampleOutputGetIssue map[string]any

//go:embed example_output_create_release.json
var exampleOutputCreateReleaseBytes []byte

var exampleOutputCreateReleaseOnce sync.Once
var exampleOutputCreateRelease map[string]any

//go:embed example_output_create_deploy.json
var exampleOutputCreateDeployBytes []byte

var exampleOutputCreateDeployOnce sync.Once
var exampleOutputCreateDeploy map[string]any

func (t *OnIssue) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueOnce, exampleDataOnIssueBytes, &exampleDataOnIssue)
}

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIssueOnce, exampleOutputUpdateIssueBytes, &exampleOutputUpdateIssue)
}

func (c *GetIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetIssueOnce, exampleOutputGetIssueBytes, &exampleOutputGetIssue)
}

func (c *CreateRelease) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateReleaseOnce,
		exampleOutputCreateReleaseBytes,
		&exampleOutputCreateRelease,
	)
}

func (c *CreateDeploy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateDeployOnce,
		exampleOutputCreateDeployBytes,
		&exampleOutputCreateDeploy,
	)
}
