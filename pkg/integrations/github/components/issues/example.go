package issues

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/create_issue.json
var exampleOutputCreateIssueBytes []byte

//go:embed payloads/create_issue_comment.json
var exampleOutputCreateIssueCommentBytes []byte

//go:embed payloads/get_issue.json
var exampleOutputGetIssueBytes []byte

//go:embed payloads/update_issue.json
var exampleOutputUpdateIssueBytes []byte

//go:embed payloads/on_issue_comment.json
var exampleDataOnIssueCommentBytes []byte

//go:embed payloads/on_issue.json
var exampleDataOnIssueBytes []byte

//go:embed payloads/add_issue_label.json
var exampleOutputAddIssueLabelBytes []byte

//go:embed payloads/remove_issue_label.json
var exampleOutputRemoveIssueLabelBytes []byte

//go:embed payloads/add_issue_assignee.json
var exampleOutputAddIssueAssigneeBytes []byte

//go:embed payloads/remove_issue_assignee.json
var exampleOutputRemoveIssueAssigneeBytes []byte

var exampleOutputCreateIssueOnce sync.Once
var exampleOutputCreateIssue map[string]any

var exampleOutputCreateIssueCommentOnce sync.Once
var exampleOutputCreateIssueComment map[string]any

var exampleOutputGetIssueOnce sync.Once
var exampleOutputGetIssue map[string]any

var exampleOutputUpdateIssueOnce sync.Once
var exampleOutputUpdateIssue map[string]any

var exampleDataOnIssueCommentOnce sync.Once
var exampleDataOnIssueComment map[string]any

var exampleDataOnIssueOnce sync.Once
var exampleDataOnIssue map[string]any

var exampleOutputAddIssueLabelOnce sync.Once
var exampleOutputAddIssueLabel map[string]any

var exampleOutputRemoveIssueLabelOnce sync.Once
var exampleOutputRemoveIssueLabel map[string]any

var exampleOutputAddIssueAssigneeOnce sync.Once
var exampleOutputAddIssueAssignee map[string]any

var exampleOutputRemoveIssueAssigneeOnce sync.Once
var exampleOutputRemoveIssueAssignee map[string]any

func (c *CreateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIssueOnce, exampleOutputCreateIssueBytes, &exampleOutputCreateIssue)
}

func (c *CreateIssueComment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIssueCommentOnce, exampleOutputCreateIssueCommentBytes, &exampleOutputCreateIssueComment)
}

func (c *GetIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetIssueOnce, exampleOutputGetIssueBytes, &exampleOutputGetIssue)
}

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIssueOnce, exampleOutputUpdateIssueBytes, &exampleOutputUpdateIssue)
}

func (t *OnIssueComment) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueCommentOnce, exampleDataOnIssueCommentBytes, &exampleDataOnIssueComment)
}

func (t *OnIssue) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueOnce, exampleDataOnIssueBytes, &exampleDataOnIssue)
}

func (c *AddIssueLabel) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputAddIssueLabelOnce, exampleOutputAddIssueLabelBytes, &exampleOutputAddIssueLabel)
}

func (c *RemoveIssueLabel) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRemoveIssueLabelOnce, exampleOutputRemoveIssueLabelBytes, &exampleOutputRemoveIssueLabel)
}

func (c *AddIssueAssignee) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputAddIssueAssigneeOnce, exampleOutputAddIssueAssigneeBytes, &exampleOutputAddIssueAssignee)
}

func (c *RemoveIssueAssignee) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRemoveIssueAssigneeOnce, exampleOutputRemoveIssueAssigneeBytes, &exampleOutputRemoveIssueAssignee)
}
