package linear

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte

var exampleDataOnIssueOnce sync.Once
var exampleDataOnIssue map[string]any

// UnmarshalExampleDataOnIssue returns example webhook payload for On Issue.
func UnmarshalExampleDataOnIssue() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueOnce, exampleDataOnIssueBytes, &exampleDataOnIssue)
}

//go:embed example_output_create_issue.json
var exampleOutputCreateIssueBytes []byte

var exampleOutputCreateIssueOnce sync.Once
var exampleOutputCreateIssue map[string]any

func (c *CreateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIssueOnce, exampleOutputCreateIssueBytes, &exampleOutputCreateIssue)
}
