package bitbucket

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_push.json
var exampleDataOnPushBytes []byte

//go:embed example_output_create_issue.json
var exampleOutputCreateIssueBytes []byte

//go:embed example_output_get_issue.json
var exampleOutputGetIssueBytes []byte

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssueBytes []byte

//go:embed example_output_create_issue_comment.json
var exampleOutputCreateIssueCommentBytes []byte

var exampleDataOnPushOnce sync.Once
var exampleDataOnPush map[string]any

var exampleOutputCreateIssueOnce sync.Once
var exampleOutputCreateIssue map[string]any

var exampleOutputGetIssueOnce sync.Once
var exampleOutputGetIssue map[string]any

var exampleOutputUpdateIssueOnce sync.Once
var exampleOutputUpdateIssue map[string]any

var exampleOutputCreateIssueCommentOnce sync.Once
var exampleOutputCreateIssueComment map[string]any

func (t *OnPush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPushOnce, exampleDataOnPushBytes, &exampleDataOnPush)
}

func (c *CreateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIssueOnce, exampleOutputCreateIssueBytes, &exampleOutputCreateIssue)
}

func (c *GetIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetIssueOnce, exampleOutputGetIssueBytes, &exampleOutputGetIssue)
}

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIssueOnce, exampleOutputUpdateIssueBytes, &exampleOutputUpdateIssue)
}

func (c *CreateIssueComment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateIssueCommentOnce,
		exampleOutputCreateIssueCommentBytes,
		&exampleOutputCreateIssueComment,
	)
}
