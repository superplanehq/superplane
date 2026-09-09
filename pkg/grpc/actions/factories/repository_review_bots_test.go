package factories

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGitHubReviewBotsAPI struct {
	closedPulls      []*github.PullRequest
	openPulls        []*github.PullRequest
	reviews          map[int][]*github.PullRequestReview
	issueComments    map[int][]*github.IssueComment
	reviewComments   map[int][]*github.PullRequestComment
	pullsErr         error
	reviewsErr       error
	issueErr         error
	reviewCommentErr error
}

func (f *fakeGitHubReviewBotsAPI) ListPullRequests(_ context.Context, _ string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	if f.pullsErr != nil {
		return nil, &github.Response{}, f.pullsErr
	}
	if opts != nil && opts.State == "open" {
		return f.openPulls, &github.Response{}, nil
	}
	return f.closedPulls, &github.Response{}, nil
}

func (f *fakeGitHubReviewBotsAPI) ListReviews(_ context.Context, _ string, pullNumber int) ([]*github.PullRequestReview, error) {
	if f.reviewsErr != nil {
		return nil, f.reviewsErr
	}
	return f.reviews[pullNumber], nil
}

func (f *fakeGitHubReviewBotsAPI) ListIssueComments(_ context.Context, _ string, issueNumber int) ([]*github.IssueComment, error) {
	if f.issueErr != nil {
		return nil, f.issueErr
	}
	return f.issueComments[issueNumber], nil
}

func (f *fakeGitHubReviewBotsAPI) ListPullRequestComments(_ context.Context, _ string, pullNumber int) ([]*github.PullRequestComment, error) {
	if f.reviewCommentErr != nil {
		return nil, f.reviewCommentErr
	}
	return f.reviewComments[pullNumber], nil
}

func TestListRepositoryReviewBotsFromClient(t *testing.T) {
	t.Parallel()

	t.Run("keeps unique normalized bot logins and drops humans and SuperPlane", func(t *testing.T) {
		t.Parallel()
		api := &fakeGitHubReviewBotsAPI{
			closedPulls: []*github.PullRequest{{Number: github.Ptr(11)}},
			reviews: map[int][]*github.PullRequestReview{
				11: {
					{User: &github.User{Login: github.Ptr("coderabbitai[bot]"), Type: github.Ptr("Bot")}},
					{User: &github.User{Login: github.Ptr("alex"), Type: github.Ptr("User")}},
					{User: &github.User{Login: github.Ptr("superplaneagent"), Type: github.Ptr("Bot")}},
				},
			},
			issueComments: map[int][]*github.IssueComment{
				11: {{User: &github.User{Login: github.Ptr("Coderabbitai[bot]"), Type: github.Ptr("Bot")}}},
			},
			reviewComments: map[int][]*github.PullRequestComment{
				11: {{User: &github.User{Login: github.Ptr("bugbot[bot]"), Type: github.Ptr("Bot")}}},
			},
		}

		bots, err := listRepositoryReviewBotsFromClient(context.Background(), api, "acme/app")
		require.NoError(t, err)
		assert.ElementsMatch(t, []repositoryReviewBot{
			{Login: "coderabbitai", DisplayName: "coderabbitai[bot]"},
			{Login: "bugbot", DisplayName: "bugbot[bot]"},
		}, bots)
	})

	t.Run("returns an empty catalog when GitHub listing fails", func(t *testing.T) {
		t.Parallel()
		api := &fakeGitHubReviewBotsAPI{pullsErr: errors.New("github unavailable")}

		bots, err := listRepositoryReviewBotsFromClient(context.Background(), api, "acme/app")
		assert.Error(t, err)
		assert.Empty(t, bots)
	})
}
