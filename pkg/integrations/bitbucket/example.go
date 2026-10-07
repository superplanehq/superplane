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
