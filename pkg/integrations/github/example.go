package github

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_issue.json
var exampleOutputCreateIssueBytes []byte

//go:embed example_output_create_issue_comment.json
var exampleOutputCreateIssueCommentBytes []byte

//go:embed example_output_get_issue.json
var exampleOutputGetIssueBytes []byte

//go:embed example_output_update_issue.json
var exampleOutputUpdateIssueBytes []byte

//go:embed example_output_publish_commit_status.json
var exampleOutputPublishCommitStatusBytes []byte

//go:embed example_output_create_release.json
var exampleOutputCreateReleaseBytes []byte

//go:embed example_output_get_release.json
var exampleOutputGetReleaseBytes []byte

//go:embed example_output_update_release.json
var exampleOutputUpdateReleaseBytes []byte

//go:embed example_output_delete_release.json
var exampleOutputDeleteReleaseBytes []byte

//go:embed example_output_run_workflow.json
var exampleOutputRunWorkflowBytes []byte

//go:embed example_output_create_review.json
var exampleOutputCreateReviewBytes []byte

//go:embed example_data_on_issue_comment.json
var exampleDataOnIssueCommentBytes []byte

//go:embed example_data_on_issue.json
var exampleDataOnIssueBytes []byte

//go:embed example_data_on_pull_request.json
var exampleDataOnPullRequestBytes []byte

//go:embed example_data_on_pull_request_review_comment.json
var exampleDataOnPullRequestReviewCommentBytes []byte

//go:embed example_data_on_push.json
var exampleDataOnPushBytes []byte

//go:embed example_data_on_release.json
var exampleDataOnReleaseBytes []byte

//go:embed example_data_on_tag_created.json
var exampleDataOnTagCreatedBytes []byte

//go:embed example_data_on_branch_created.json
var exampleDataOnBranchCreatedBytes []byte

//go:embed example_data_on_workflow_run.json
var exampleDataOnWorkflowRunBytes []byte

//go:embed example_output_get_workflow_usage.json
var exampleOutputGetWorkflowUsageBytes []byte

var exampleOutputCreateIssueOnce sync.Once
var exampleOutputCreateIssue map[string]any

var exampleOutputCreateIssueCommentOnce sync.Once
var exampleOutputCreateIssueComment map[string]any

var exampleOutputGetIssueOnce sync.Once
var exampleOutputGetIssue map[string]any

var exampleOutputUpdateIssueOnce sync.Once
var exampleOutputUpdateIssue map[string]any

var exampleOutputPublishCommitStatusOnce sync.Once
var exampleOutputPublishCommitStatus map[string]any

var exampleOutputCreateReleaseOnce sync.Once
var exampleOutputCreateRelease map[string]any

var exampleOutputGetReleaseOnce sync.Once
var exampleOutputGetRelease map[string]any

var exampleOutputUpdateReleaseOnce sync.Once
var exampleOutputUpdateRelease map[string]any

var exampleOutputDeleteReleaseOnce sync.Once
var exampleOutputDeleteRelease map[string]any

var exampleOutputRunWorkflowOnce sync.Once
var exampleOutputRunWorkflow map[string]any

var exampleOutputCreateReviewOnce sync.Once
var exampleOutputCreateReview map[string]any

var exampleDataOnIssueCommentOnce sync.Once
var exampleDataOnIssueComment map[string]any

var exampleDataOnIssueOnce sync.Once
var exampleDataOnIssue map[string]any

var exampleDataOnPullRequestOnce sync.Once
var exampleDataOnPullRequest map[string]any

var exampleDataOnPullRequestReviewCommentOnce sync.Once
var exampleDataOnPullRequestReviewComment map[string]any

var exampleDataOnPushOnce sync.Once
var exampleDataOnPush map[string]any

var exampleDataOnReleaseOnce sync.Once
var exampleDataOnRelease map[string]any

var exampleDataOnTagCreatedOnce sync.Once
var exampleDataOnTagCreated map[string]any

var exampleDataOnBranchCreatedOnce sync.Once
var exampleDataOnBranchCreated map[string]any

var exampleDataOnWorkflowRunOnce sync.Once
var exampleDataOnWorkflowRun map[string]any

var exampleOutputGetWorkflowUsageOnce sync.Once
var exampleOutputGetWorkflowUsage map[string]any

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

func (c *PublishCommitStatus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputPublishCommitStatusOnce,
		exampleOutputPublishCommitStatusBytes,
		&exampleOutputPublishCommitStatus,
	)
}

func (c *CreateRelease) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateReleaseOnce, exampleOutputCreateReleaseBytes, &exampleOutputCreateRelease)
}

func (c *GetRelease) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetReleaseOnce, exampleOutputGetReleaseBytes, &exampleOutputGetRelease)
}

func (c *UpdateRelease) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputUpdateReleaseOnce, exampleOutputUpdateReleaseBytes, &exampleOutputUpdateRelease)
}

func (c *DeleteRelease) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteReleaseOnce, exampleOutputDeleteReleaseBytes, &exampleOutputDeleteRelease)
}

func (c *RunWorkflow) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunWorkflowOnce, exampleOutputRunWorkflowBytes, &exampleOutputRunWorkflow)
}

func (c *CreateReview) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateReviewOnce,
		exampleOutputCreateReviewBytes,
		&exampleOutputCreateReview,
	)
}

func (t *OnIssueComment) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueCommentOnce, exampleDataOnIssueCommentBytes, &exampleDataOnIssueComment)
}

func (t *OnIssue) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnIssueOnce, exampleDataOnIssueBytes, &exampleDataOnIssue)
}

func (t *OnPullRequest) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPullRequestOnce, exampleDataOnPullRequestBytes, &exampleDataOnPullRequest)
}

func (t *OnPRComment) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnPullRequestReviewCommentOnce,
		exampleDataOnPullRequestReviewCommentBytes,
		&exampleDataOnPullRequestReviewComment,
	)
}

func (t *OnPush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPushOnce, exampleDataOnPushBytes, &exampleDataOnPush)
}

func (t *OnRelease) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnReleaseOnce, exampleDataOnReleaseBytes, &exampleDataOnRelease)
}

func (t *OnTagCreated) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnTagCreatedOnce, exampleDataOnTagCreatedBytes, &exampleDataOnTagCreated)
}

func (t *OnBranchCreated) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnBranchCreatedOnce, exampleDataOnBranchCreatedBytes, &exampleDataOnBranchCreated)
}

func (t *OnWorkflowRun) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnWorkflowRunOnce, exampleDataOnWorkflowRunBytes, &exampleDataOnWorkflowRun)
}

func (g *GetWorkflowUsage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetWorkflowUsageOnce, exampleOutputGetWorkflowUsageBytes, &exampleOutputGetWorkflowUsage)
}
