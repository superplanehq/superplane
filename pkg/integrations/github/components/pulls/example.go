package pulls

import (
	_ "embed"

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

//go:embed payloads/create_pull_request.json
var exampleOutputCreatePullRequestBytes []byte

//go:embed payloads/merge_pull_request.json
var exampleOutputMergePullRequestBytes []byte

//go:embed payloads/add_pull_request_reviewers.json
var exampleOutputAddPullRequestReviewersBytes []byte

//go:embed payloads/mark_pull_request_ready_for_review.json
var exampleOutputMarkPullRequestReadyForReviewBytes []byte
var exampleOutputAddReaction = utils.NewEmbeddedJSON(exampleOutputAddReactionBytes)
var exampleOutputCreateReview = utils.NewEmbeddedJSON(exampleOutputCreateReviewBytes)
var exampleOutputCreatePullRequest = utils.NewEmbeddedJSON(exampleOutputCreatePullRequestBytes)
var exampleOutputMergePullRequest = utils.NewEmbeddedJSON(exampleOutputMergePullRequestBytes)
var exampleOutputAddPullRequestReviewers = utils.NewEmbeddedJSON(exampleOutputAddPullRequestReviewersBytes)
var exampleOutputMarkPullRequestReadyForReview = utils.NewEmbeddedJSON(exampleOutputMarkPullRequestReadyForReviewBytes)
var exampleDataOnPullRequest = utils.NewEmbeddedJSON(exampleDataOnPullRequestBytes)
var exampleDataOnPRComment = utils.NewEmbeddedJSON(exampleDataOnPRCommentBytes)
var exampleDataOnPRReviewComment = utils.NewEmbeddedJSON(exampleDataOnPRReviewCommentBytes)

func (c *AddReaction) ExampleOutput() map[string]any {
	return exampleOutputAddReaction.Value()
}

func (t *OnPullRequest) ExampleData() map[string]any {
	return exampleDataOnPullRequest.Value()
}

func (t *OnPRComment) ExampleData() map[string]any {
	return exampleDataOnPRComment.Value()
}

func (t *OnPRReviewComment) ExampleData() map[string]any {
	return exampleDataOnPRReviewComment.Value()
}

func (c *CreateReview) ExampleOutput() map[string]any {
	return exampleOutputCreateReview.Value()
}

func (c *CreatePullRequest) ExampleOutput() map[string]any {
	return exampleOutputCreatePullRequest.Value()
}

func (c *MergePullRequest) ExampleOutput() map[string]any {
	return exampleOutputMergePullRequest.Value()
}

func (c *AddPullRequestReviewers) ExampleOutput() map[string]any {
	return exampleOutputAddPullRequestReviewers.Value()
}

func (c *MarkPullRequestReadyForReview) ExampleOutput() map[string]any {
	return exampleOutputMarkPullRequestReadyForReview.Value()
}
