package sentry

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_issue_event.json
var exampleDataOnIssueEventBytes []byte

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssueBytes []byte

var exampleDataOnIssueEventOnce sync.Once
var exampleDataOnIssueEvent map[string]any

var exampleOutputUpdateIssueOnce sync.Once
var exampleOutputUpdateIssue map[string]any

func (t *OnIssueEvent) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueEventOnce, exampleDataOnIssueEventBytes, &exampleDataOnIssueEvent)
}

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIssueOnce, exampleOutputUpdateIssueBytes, &exampleOutputUpdateIssue)
}
