package jira

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_issue.json
var exampleOutputCreateIssueBytes []byte

var exampleOutputCreateIssueOnce sync.Once
var exampleOutputCreateIssue map[string]any

//go:embed example_output_transition_issue.json
var exampleOutputTransitionIssueBytes []byte

var exampleOutputTransitionIssueOnce sync.Once
var exampleOutputTransitionIssue map[string]any

//go:embed example_output_get_workflow.json
var exampleOutputGetWorkflowBytes []byte

var exampleOutputGetWorkflowOnce sync.Once
var exampleOutputGetWorkflow map[string]any

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

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte

var exampleDataOnIssueOnce sync.Once
var exampleDataOnIssue map[string]any

func onIssueExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueOnce, exampleDataOnIssueBytes, &exampleDataOnIssue)
}

//go:embed example_data_on_issue_comment.json
var exampleDataOnIssueCommentBytes []byte

var exampleDataOnIssueCommentOnce sync.Once
var exampleDataOnIssueComment map[string]any

func onIssueCommentExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueCommentOnce, exampleDataOnIssueCommentBytes, &exampleDataOnIssueComment)
}

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssueBytes []byte

var exampleOutputUpdateIssueOnce sync.Once
var exampleOutputUpdateIssue map[string]any

func (c *UpdateIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateIssueOnce, exampleOutputUpdateIssueBytes, &exampleOutputUpdateIssue)
}

//go:embed example_output_delete_issue.json
var exampleOutputDeleteIssueBytes []byte

var exampleOutputDeleteIssueOnce sync.Once
var exampleOutputDeleteIssue map[string]any

func (c *DeleteIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteIssueOnce, exampleOutputDeleteIssueBytes, &exampleOutputDeleteIssue)
}

func (c *TransitionIssue) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputTransitionIssueOnce, exampleOutputTransitionIssueBytes, &exampleOutputTransitionIssue)
}

func (c *GetWorkflow) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetWorkflowOnce, exampleOutputGetWorkflowBytes, &exampleOutputGetWorkflow)
}
