package pulls

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/create_review.json
var exampleOutputCreateReviewBytes []byte

//go:embed payloads/on_pull_request.json
var exampleDataOnPullRequestBytes []byte

//go:embed payloads/on_pr_comment.json
var exampleDataOnPRCommentBytes []byte

//go:embed payloads/on_pr_review_comment.json
var exampleDataOnPRReviewCommentBytes []byte

//go:embed payloads/add_reaction.json
var exampleOutputAddReactionBytes []byte

var exampleOutputAddReactionOnce sync.Once
var exampleOutputAddReaction map[string]any

var exampleOutputCreateReviewOnce sync.Once
var exampleOutputCreateReview map[string]any

var exampleDataOnPullRequestOnce sync.Once
var exampleDataOnPullRequest map[string]any

var exampleDataOnPRCommentOnce sync.Once
var exampleDataOnPRComment map[string]any

var exampleDataOnPRReviewCommentOnce sync.Once
var exampleDataOnPRReviewComment map[string]any

func (c *AddReaction) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputAddReactionOnce, exampleOutputAddReactionBytes, &exampleOutputAddReaction)
}

func (t *OnPullRequest) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnPullRequestOnce, exampleDataOnPullRequestBytes, &exampleDataOnPullRequest)
}

func (t *OnPRComment) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnPRCommentOnce,
		exampleDataOnPRCommentBytes,
		&exampleDataOnPRComment,
	)
}

func (t *OnPRReviewComment) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnPRReviewCommentOnce,
		exampleDataOnPRReviewCommentBytes,
		&exampleDataOnPRReviewComment,
	)
}

func (c *CreateReview) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateReviewOnce,
		exampleOutputCreateReviewBytes,
		&exampleOutputCreateReview,
	)
}
