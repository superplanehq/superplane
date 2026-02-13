package snyk

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_ignore_issue.json
var exampleOutputIgnoreIssueBytes []byte

//go:embed example_data_on_new_issue_detected.json
var exampleDataOnNewIssueDetectedBytes []byte

var exampleOutputIgnoreIssueOnce sync.Once
var exampleOutputIgnoreIssue map[string]any

var exampleDataOnNewIssueDetectedOnce sync.Once
var exampleDataOnNewIssueDetected map[string]any

func (c *IgnoreIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputIgnoreIssueOnce, exampleOutputIgnoreIssueBytes, &exampleOutputIgnoreIssue)
}

func (t *OnNewIssueDetected) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnNewIssueDetectedOnce, exampleDataOnNewIssueDetectedBytes, &exampleDataOnNewIssueDetected)
}
