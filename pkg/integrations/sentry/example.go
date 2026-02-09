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
var exampleDataOnIssueEventData map[string]any

var exampleOutputUpdateIssueOnce sync.Once
var exampleOutputUpdateIssueData map[string]any

func exampleDataOnIssueEvent() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueEventOnce, exampleDataOnIssueEventBytes, &exampleDataOnIssueEventData)
}

func exampleOutputUpdateIssue() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIssueOnce, exampleOutputUpdateIssueBytes, &exampleOutputUpdateIssueData)
}
