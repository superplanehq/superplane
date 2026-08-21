package linear

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_issue.json
var exampleOutputCreateIssueBytes []byte

var exampleOutputCreateIssueOnce sync.Once
var exampleOutputCreateIssue map[string]any

func (c *CreateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateIssueOnce, exampleOutputCreateIssueBytes, &exampleOutputCreateIssue)
}

//go:embed example_output_get_issue.json
var exampleOutputGetIssueBytes []byte

var exampleOutputGetIssueOnce sync.Once
var exampleOutputGetIssue map[string]any

func (c *GetIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetIssueOnce, exampleOutputGetIssueBytes, &exampleOutputGetIssue)
}

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssueBytes []byte

var exampleOutputUpdateIssueOnce sync.Once
var exampleOutputUpdateIssue map[string]any

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIssueOnce, exampleOutputUpdateIssueBytes, &exampleOutputUpdateIssue)
}

//go:embed example_output_add_issue_label.json
var exampleOutputAddIssueLabelBytes []byte

var exampleOutputAddIssueLabelOnce sync.Once
var exampleOutputAddIssueLabel map[string]any

func (c *AddIssueLabel) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputAddIssueLabelOnce, exampleOutputAddIssueLabelBytes, &exampleOutputAddIssueLabel)
}

//go:embed example_output_add_issue_comment.json
var exampleOutputAddIssueCommentBytes []byte

var exampleOutputAddIssueCommentOnce sync.Once
var exampleOutputAddIssueComment map[string]any

func (c *AddIssueComment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputAddIssueCommentOnce, exampleOutputAddIssueCommentBytes, &exampleOutputAddIssueComment)
}

//go:embed example_output_update_issue_comment.json
var exampleOutputUpdateIssueCommentBytes []byte

var exampleOutputUpdateIssueCommentOnce sync.Once
var exampleOutputUpdateIssueComment map[string]any

func (c *UpdateIssueComment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIssueCommentOnce, exampleOutputUpdateIssueCommentBytes, &exampleOutputUpdateIssueComment)
}

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte

var exampleDataOnIssueOnce sync.Once
var exampleDataOnIssue map[string]any

func (i *OnIssue) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueOnce, exampleDataOnIssueBytes, &exampleDataOnIssue)
}

//go:embed example_data_on_issue_label.json
var exampleDataOnIssueLabelBytes []byte

var exampleDataOnIssueLabelOnce sync.Once
var exampleDataOnIssueLabel map[string]any

func (i *OnIssueLabel) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueLabelOnce, exampleDataOnIssueLabelBytes, &exampleDataOnIssueLabel)
}

//go:embed example_data_on_issue_comment.json
var exampleDataOnIssueCommentBytes []byte

var exampleDataOnIssueCommentOnce sync.Once
var exampleDataOnIssueComment map[string]any

func (i *OnIssueComment) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueCommentOnce, exampleDataOnIssueCommentBytes, &exampleDataOnIssueComment)
}

//go:embed example_output_remove_issue_label.json
var exampleOutputRemoveIssueLabelBytes []byte

var exampleOutputRemoveIssueLabelOnce sync.Once
var exampleOutputRemoveIssueLabel map[string]any

func (c *RemoveIssueLabel) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRemoveIssueLabelOnce, exampleOutputRemoveIssueLabelBytes, &exampleOutputRemoveIssueLabel)
}

//go:embed example_output_add_reaction.json
var exampleOutputAddReactionBytes []byte

var exampleOutputAddReactionOnce sync.Once
var exampleOutputAddReaction map[string]any

func (c *AddReaction) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputAddReactionOnce, exampleOutputAddReactionBytes, &exampleOutputAddReaction)
}

//go:embed example_output_create_attachment.json
var exampleOutputCreateAttachmentBytes []byte

var exampleOutputCreateAttachmentOnce sync.Once
var exampleOutputCreateAttachment map[string]any

func (c *CreateAttachment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateAttachmentOnce, exampleOutputCreateAttachmentBytes, &exampleOutputCreateAttachment)
}

//go:embed example_output_delete_attachment.json
var exampleOutputDeleteAttachmentBytes []byte

var exampleOutputDeleteAttachmentOnce sync.Once
var exampleOutputDeleteAttachment map[string]any

func (c *DeleteAttachment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteAttachmentOnce, exampleOutputDeleteAttachmentBytes, &exampleOutputDeleteAttachment)
}

//go:embed example_data_on_issue_attachment.json
var exampleDataOnIssueAttachmentBytes []byte

var exampleDataOnIssueAttachmentOnce sync.Once
var exampleDataOnIssueAttachment map[string]any

func (i *OnIssueAttachment) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueAttachmentOnce, exampleDataOnIssueAttachmentBytes, &exampleDataOnIssueAttachment)
}
