package linear

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_issue_created.json
var exampleDataOnIssueCreatedBytes []byte

var exampleDataOnIssueCreatedOnce sync.Once
var exampleDataOnIssueCreated map[string]any

// UnmarshalExampleDataOnIssueCreated returns example webhook payload for On Issue Created.
func UnmarshalExampleDataOnIssueCreated() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueCreatedOnce, exampleDataOnIssueCreatedBytes, &exampleDataOnIssueCreated)
}

//go:embed example_output_create_issue.json
var exampleOutputCreateIssueBytes []byte

var exampleOutputCreateIssueOnce sync.Once
var exampleOutputCreateIssue map[string]any

func (c *CreateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIssueOnce, exampleOutputCreateIssueBytes, &exampleOutputCreateIssue)
}
