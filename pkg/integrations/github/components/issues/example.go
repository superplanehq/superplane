package issues

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/create_issue.json
var exampleOutputCreateIssueBytes []byte

//go:embed payloads/create_issue_comment.json
var exampleOutputCreateIssueCommentBytes []byte

//go:embed payloads/update_issue_comment.json
var exampleOutputUpdateIssueCommentBytes []byte

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
var exampleOutputCreateIssue = utils.NewEmbeddedJSON(exampleOutputCreateIssueBytes)
var exampleOutputCreateIssueComment = utils.NewEmbeddedJSON(exampleOutputCreateIssueCommentBytes)
var exampleOutputUpdateIssueComment = utils.NewEmbeddedJSON(exampleOutputUpdateIssueCommentBytes)
var exampleOutputGetIssue = utils.NewEmbeddedJSON(exampleOutputGetIssueBytes)
var exampleOutputUpdateIssue = utils.NewEmbeddedJSON(exampleOutputUpdateIssueBytes)
var exampleDataOnIssueComment = utils.NewEmbeddedJSON(exampleDataOnIssueCommentBytes)
var exampleDataOnIssue = utils.NewEmbeddedJSON(exampleDataOnIssueBytes)
var exampleOutputAddIssueLabel = utils.NewEmbeddedJSON(exampleOutputAddIssueLabelBytes)
var exampleOutputRemoveIssueLabel = utils.NewEmbeddedJSON(exampleOutputRemoveIssueLabelBytes)
var exampleOutputAddIssueAssignee = utils.NewEmbeddedJSON(exampleOutputAddIssueAssigneeBytes)
var exampleOutputRemoveIssueAssignee = utils.NewEmbeddedJSON(exampleOutputRemoveIssueAssigneeBytes)

func (c *CreateIssue) ExampleOutput() map[string]any {
	return exampleOutputCreateIssue.Value()
}

func (c *CreateIssueComment) ExampleOutput() map[string]any {
	return exampleOutputCreateIssueComment.Value()
}

func (c *UpdateIssueComment) ExampleOutput() map[string]any {
	return exampleOutputUpdateIssueComment.Value()
}

func (c *GetIssue) ExampleOutput() map[string]any {
	return exampleOutputGetIssue.Value()
}

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return exampleOutputUpdateIssue.Value()
}

func (t *OnIssueComment) ExampleData() map[string]any {
	return exampleDataOnIssueComment.Value()
}

func (t *OnIssue) ExampleData() map[string]any {
	return exampleDataOnIssue.Value()
}

func (c *AddIssueLabel) ExampleOutput() map[string]any {
	return exampleOutputAddIssueLabel.Value()
}

func (c *RemoveIssueLabel) ExampleOutput() map[string]any {
	return exampleOutputRemoveIssueLabel.Value()
}

func (c *AddIssueAssignee) ExampleOutput() map[string]any {
	return exampleOutputAddIssueAssignee.Value()
}

func (c *RemoveIssueAssignee) ExampleOutput() map[string]any {
	return exampleOutputRemoveIssueAssignee.Value()
}
