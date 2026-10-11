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

//go:embed example_data_on_pull_request.json
var exampleDataOnPullRequestBytes []byte

var exampleDataOnPullRequestOnce sync.Once
var exampleDataOnPullRequest map[string]any

//go:embed example_data_on_pull_request_comment.json
var exampleDataOnPullRequestCommentBytes []byte

var exampleDataOnPullRequestCommentOnce sync.Once
var exampleDataOnPullRequestComment map[string]any

//go:embed example_output_pull_request.json
var exampleOutputPullRequestBytes []byte

var exampleOutputPullRequestOnce sync.Once
var exampleOutputPullRequest map[string]any

//go:embed example_output_pull_request_comment.json
var exampleOutputPullRequestCommentBytes []byte

var exampleOutputPullRequestCommentOnce sync.Once
var exampleOutputPullRequestComment map[string]any

func (t *OnPush) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPushOnce, exampleDataOnPushBytes, &exampleDataOnPush)
}

func (t *OnPullRequest) ExampleData() map[string]any {
	payload := utils.UnmarshalEmbeddedJSON(&exampleDataOnPullRequestOnce, exampleDataOnPullRequestBytes, &exampleDataOnPullRequest)
	event, _ := normalizeBitbucketPullRequestEvent(pullRequestEventCreated, payload)
	return map[string]any{"data": event, "timestamp": "2026-04-22T10:01:00Z", "type": "bitbucket.pullRequest"}
}

func (t *OnPullRequestComment) ExampleData() map[string]any {
	payload := utils.UnmarshalEmbeddedJSON(&exampleDataOnPullRequestCommentOnce, exampleDataOnPullRequestCommentBytes, &exampleDataOnPullRequestComment)
	event, _ := normalizePullRequestCommentEvent(payload)
	return map[string]any{"data": event, "timestamp": "2026-04-22T10:05:00Z", "type": "bitbucket.pullRequestComment"}
}

func examplePullRequestOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPullRequestOnce, exampleOutputPullRequestBytes, &exampleOutputPullRequest)
}

func examplePullRequestCommentOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputPullRequestCommentOnce,
		exampleOutputPullRequestCommentBytes,
		&exampleOutputPullRequestComment,
	)
}
