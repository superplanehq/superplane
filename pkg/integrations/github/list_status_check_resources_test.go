package github

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__MergeStatusCheckResources(t *testing.T) {
	t.Parallel()

	t.Run("protection names are listed first and pick up urls", func(t *testing.T) {
		t.Parallel()
		resources := mergeStatusCheckResources(
			[]string{"lint", "unit"},
			[]observedStatusCheck{
				{Name: "unit", URL: "https://github.com/acme/api/actions/runs/1"},
				{Name: "e2e", URL: "https://app.circleci.com/pipelines/github/acme/api/2"},
			},
		)

		assert.Equal(t, []core.IntegrationResource{
			{Type: "status_check", Name: "lint", ID: "lint"},
			{Type: "status_check", Name: "unit", ID: "unit", URL: "https://github.com/acme/api/actions/runs/1"},
			{Type: "status_check", Name: "e2e", ID: "e2e", URL: "https://app.circleci.com/pipelines/github/acme/api/2"},
		}, resources)
	})

	t.Run("duplicate names keep the first url", func(t *testing.T) {
		t.Parallel()
		resources := mergeStatusCheckResources(
			[]string{"CI"},
			[]observedStatusCheck{
				{Name: "ci", URL: "https://acme.semaphoreci.com/workflows/1"},
				{Name: "CI", URL: "https://app.circleci.com/pipelines/github/acme/api/2"},
			},
		)

		assert.Equal(t, []core.IntegrationResource{
			{Type: "status_check", Name: "CI", ID: "CI", URL: "https://acme.semaphoreci.com/workflows/1"},
		}, resources)
	})

	t.Run("blank names are skipped", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, mergeStatusCheckResources([]string{" "}, []observedStatusCheck{{Name: ""}}))
	})
}

type fakeGitHubReviewBotAPI struct {
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

func (f *fakeGitHubReviewBotAPI) ListPullRequests(_ context.Context, _ string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	if f.pullsErr != nil {
		return nil, &github.Response{}, f.pullsErr
	}
	if opts != nil && opts.State == "open" {
		return f.openPulls, &github.Response{}, nil
	}
	return f.closedPulls, &github.Response{}, nil
}

func (f *fakeGitHubReviewBotAPI) ListReviews(_ context.Context, _ string, pullNumber int) ([]*github.PullRequestReview, error) {
	if f.reviewsErr != nil {
		return nil, f.reviewsErr
	}
	return f.reviews[pullNumber], nil
}

func (f *fakeGitHubReviewBotAPI) ListIssueComments(_ context.Context, _ string, issueNumber int) ([]*github.IssueComment, error) {
	if f.issueErr != nil {
		return nil, f.issueErr
	}
	return f.issueComments[issueNumber], nil
}

func (f *fakeGitHubReviewBotAPI) ListPullRequestComments(_ context.Context, _ string, pullNumber int) ([]*github.PullRequestComment, error) {
	if f.reviewCommentErr != nil {
		return nil, f.reviewCommentErr
	}
	return f.reviewComments[pullNumber], nil
}

func Test__ListReviewBotResourcesFromClient(t *testing.T) {
	t.Parallel()

	t.Run("keeps unique normalized bot logins and drops humans and SuperPlane", func(t *testing.T) {
		t.Parallel()
		api := &fakeGitHubReviewBotAPI{
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

		bots, err := listReviewBotResourcesFromClient(context.Background(), api, "acme/app")
		require.NoError(t, err)
		assert.ElementsMatch(t, []core.IntegrationResource{
			{Type: "review_bot", Name: "coderabbitai[bot]", ID: "coderabbitai"},
			{Type: "review_bot", Name: "bugbot[bot]", ID: "bugbot"},
		}, bots)
	})

	t.Run("returns an error when GitHub listing fails", func(t *testing.T) {
		t.Parallel()
		api := &fakeGitHubReviewBotAPI{pullsErr: errors.New("github unavailable")}

		bots, err := listReviewBotResourcesFromClient(context.Background(), api, "acme/app")
		assert.Error(t, err)
		assert.Empty(t, bots)
	})
}
