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

//go:embed payloads/create_pull_request.json
var exampleOutputCreatePullRequestBytes []byte

//go:embed payloads/merge_pull_request.json
var exampleOutputMergePullRequestBytes []byte

//go:embed payloads/add_pull_request_reviewers.json
var exampleOutputAddPullRequestReviewersBytes []byte

//go:embed payloads/mark_pull_request_ready_for_review.json
var exampleOutputMarkPullRequestReadyForReviewBytes []byte

//go:embed payloads/update_pull_request.json
var exampleOutputUpdatePullRequestBytes []byte

var exampleOutputAddReactionOnce sync.Once
var exampleOutputAddReaction map[string]any

var exampleOutputCreateReviewOnce sync.Once
var exampleOutputCreateReview map[string]any

var exampleOutputCreatePullRequestOnce sync.Once
var exampleOutputCreatePullRequest map[string]any

var exampleOutputMergePullRequestOnce sync.Once
var exampleOutputMergePullRequest map[string]any

var exampleOutputAddPullRequestReviewersOnce sync.Once
var exampleOutputAddPullRequestReviewers map[string]any

var exampleOutputMarkPullRequestReadyForReviewOnce sync.Once
var exampleOutputMarkPullRequestReadyForReview map[string]any

var exampleOutputUpdatePullRequestOnce sync.Once
var exampleOutputUpdatePullRequest map[string]any

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

func (c *CreatePullRequest) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreatePullRequestOnce,
		exampleOutputCreatePullRequestBytes,
		&exampleOutputCreatePullRequest,
	)
}

func (c *MergePullRequest) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputMergePullRequestOnce,
		exampleOutputMergePullRequestBytes,
		&exampleOutputMergePullRequest,
	)
}

func (c *AddPullRequestReviewers) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputAddPullRequestReviewersOnce,
		exampleOutputAddPullRequestReviewersBytes,
		&exampleOutputAddPullRequestReviewers,
	)
}

func (c *MarkPullRequestReadyForReview) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputMarkPullRequestReadyForReviewOnce,
		exampleOutputMarkPullRequestReadyForReviewBytes,
		&exampleOutputMarkPullRequestReadyForReview,
	)
}

func (c *UpdatePullRequest) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdatePullRequestOnce,
		exampleOutputUpdatePullRequestBytes,
		&exampleOutputUpdatePullRequest,
	)
}
