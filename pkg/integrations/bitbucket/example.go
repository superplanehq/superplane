package bitbucket

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_push.json
var exampleDataOnPushBytes []byte

var exampleDataOnPushOnce sync.Once
var exampleDataOnPush map[string]any

func (t *OnPush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPushOnce, exampleDataOnPushBytes, &exampleDataOnPush)
}

//go:embed example_data_on_pull_request.json
var exampleDataOnPullRequestBytes []byte

var exampleDataOnPullRequestOnce sync.Once
var exampleDataOnPullRequest map[string]any

func (p *OnPullRequest) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPullRequestOnce, exampleDataOnPullRequestBytes, &exampleDataOnPullRequest)
}

//go:embed example_data_on_pr_comment.json
var exampleDataOnPRCommentBytes []byte

var exampleDataOnPRCommentOnce sync.Once
var exampleDataOnPRComment map[string]any

func (c *OnPRComment) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPRCommentOnce, exampleDataOnPRCommentBytes, &exampleDataOnPRComment)
}

//go:embed example_data_on_commit_status.json
var exampleDataOnCommitStatusBytes []byte

var exampleDataOnCommitStatusOnce sync.Once
var exampleDataOnCommitStatus map[string]any

func (s *OnCommitStatus) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnCommitStatusOnce, exampleDataOnCommitStatusBytes, &exampleDataOnCommitStatus)
}
