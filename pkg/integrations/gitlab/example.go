package gitlab

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte

var exampleDataOnIssueOnce sync.Once
var exampleDataOnIssue map[string]any

func (i *OnIssue) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueOnce, exampleDataOnIssueBytes, &exampleDataOnIssue)
}
