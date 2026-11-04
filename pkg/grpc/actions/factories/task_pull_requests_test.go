package factories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type fakeTaskPullRequestGitHub struct {
	defaultBranch string
	open          map[string][]*github.PullRequest
	created       []*github.NewPullRequest
	edited        []*github.PullRequest
	calls         int
}

func (f *fakeTaskPullRequestGitHub) GetRepository(_ context.Context, repository string) (*github.Repository, *github.Response, error) {
	f.calls++
	return &github.Repository{FullName: github.Ptr(repository), DefaultBranch: github.Ptr(f.defaultBranch)}, nil, nil
}

func (f *fakeTaskPullRequestGitHub) ListPullRequests(_ context.Context, repository string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	f.calls++
	matches := []*github.PullRequest{}
	for _, pullRequest := range f.open[repository] {
		if "acme:"+pullRequest.GetHead().GetRef() == opts.Head && pullRequest.GetBase().GetRef() == opts.Base {
			matches = append(matches, pullRequest)
		}
	}
	return matches, nil, nil
}

func (f *fakeTaskPullRequestGitHub) CreatePullRequest(_ context.Context, repository string, request *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
	f.calls++
	f.created = append(f.created, request)
	number := len(f.created)
	pullRequest := &github.PullRequest{
		ID:      github.Ptr(int64(1000 + number)),
		Number:  github.Ptr(number),
		Title:   request.Title,
		Body:    request.Body,
		Draft:   request.Draft,
		HTMLURL: github.Ptr(fmt.Sprintf("https://github.com/%s/pull/%d", repository, number)),
		Head:    &github.PullRequestBranch{Ref: request.Head},
		Base:    &github.PullRequestBranch{Ref: request.Base, Repo: &github.Repository{FullName: github.Ptr(repository)}},
	}
	f.open[repository] = append(f.open[repository], pullRequest)
	return pullRequest, nil, nil
}

func (f *fakeTaskPullRequestGitHub) EditPullRequest(_ context.Context, repository string, number int, patch *github.PullRequest) (*github.PullRequest, *github.Response, error) {
	f.calls++
	f.edited = append(f.edited, patch)
	for _, pullRequest := range f.open[repository] {
		if pullRequest.GetNumber() == number {
			pullRequest.Title = patch.Title
			pullRequest.Body = patch.Body
			return pullRequest, nil, nil
		}
	}
	return nil, nil, fmt.Errorf("pull request %d not found", number)
}

func TestOpenTaskPullRequest(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	integrationID := uuid.NewString()
	repository, branch, provider := "acme/api", "main", models.ProviderGitHub
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &integrationID, AppRepository: &repository, DefaultBranch: &branch, VCSProvider: &provider,
	}))
	_, err = factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
		Repository: "acme/web",
		Access:     models.FactoryContextRepoAccessWrite,
	})
	require.NoError(t, err)
	originURL := "https://github.com/acme/api/issues/12"
	order, err := factory.CreateWorkOrder(db, "Implement", "", &r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, db.Model(order).Updates(map[string]any{"origin_url": originURL, "origin_label": "acme/api#12"}).Error)
	order, err = factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)

	fake := &fakeTaskPullRequestGitHub{defaultBranch: "trunk", open: map[string][]*github.PullRequest{}}
	original := newTaskPullRequestGitHubAPI
	newTaskPullRequestGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (taskPullRequestGitHubAPI, error) {
		return fake, nil
	}
	t.Cleanup(func() { newTaskPullRequestGitHubAPI = original })
	open := func(input TaskPullRequestInput) (*models.FactoryPullRequest, error) {
		return OpenTaskPullRequest(context.Background(), IntakeDependencies{}, "http://app.test", factory, order, input)
	}

	t.Run("opens the pull request on the task branch with the task footer", func(t *testing.T) {
		pullRequest, err := open(TaskPullRequestInput{Repository: "acme/api", Head: "feature", Title: "Change the API", Body: "Why"})
		require.NoError(t, err)

		require.Len(t, fake.created, 1)
		assert.Equal(t, "main", fake.created[0].GetBase())
		body := fake.created[0].GetBody()
		assert.Contains(t, body, "Closes acme/api#12")
		assert.Contains(t, body, fmt.Sprintf("[%s](http://app.test%s)", factory.WorkOrderKey(order.Number), order.URLPath(factory.RouteSegment())))
		assert.Contains(t, body, "Why")
		assert.Contains(t, body, "Created via [SuperPlane](https://superplane.com).")
		assert.Equal(t, order.ID, pullRequest.WorkOrderID)
		assert.Equal(t, "acme/api", pullRequest.Repository)
		assert.Equal(t, models.FactoryPullRequestStateOpen, pullRequest.State)
	})

	t.Run("updates the same pull request when the branch is opened again", func(t *testing.T) {
		pullRequest, err := open(TaskPullRequestInput{Repository: "acme/api", Head: "feature", Title: "Change the API again"})
		require.NoError(t, err)

		assert.Len(t, fake.created, 1)
		require.Len(t, fake.edited, 1)
		assert.Equal(t, "Change the API again", pullRequest.Title)
		tracked, err := ListTaskPullRequests(db, order)
		require.NoError(t, err)
		assert.Len(t, tracked, 1)
	})

	t.Run("opens a pull request in another workspace repository on its default branch", func(t *testing.T) {
		pullRequest, err := open(TaskPullRequestInput{Repository: "acme/web", Head: "feature", Title: "Change the web app", Draft: true})
		require.NoError(t, err)

		assert.Equal(t, "trunk", fake.created[len(fake.created)-1].GetBase())
		assert.Equal(t, models.FactoryPullRequestStateDraft, pullRequest.State)
		tracked, err := ListTaskPullRequests(db, order)
		require.NoError(t, err)
		assert.Len(t, tracked, 2)
		artifacts, err := order.ListArtifacts(db)
		require.NoError(t, err)
		branches := 0
		for _, artifact := range artifacts {
			if artifact.Type == models.FactoryWorkOrderArtifactTypeBranch {
				branches++
			}
		}
		assert.Equal(t, 2, branches, "one branch artifact for each repository")
	})

	t.Run("rejects a repository outside the workspace before it calls GitHub", func(t *testing.T) {
		calls := fake.calls
		_, err := open(TaskPullRequestInput{Repository: "acme/other", Head: "feature", Title: "Change other"})

		require.ErrorIs(t, err, models.ErrFactoryPullRequestRepositoryNotInWorkspace)
		assert.Equal(t, calls, fake.calls)
	})

	t.Run("does not edit the pull request of another task", func(t *testing.T) {
		other, err := factory.CreateWorkOrder(db, "Other", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = OpenTaskPullRequest(context.Background(), IntakeDependencies{}, "http://app.test", factory, other, TaskPullRequestInput{
			Repository: "acme/api", Head: "other-feature", Title: "Other task change",
		})
		require.NoError(t, err)
		edits := len(fake.edited)

		_, err = open(TaskPullRequestInput{Repository: "acme/api", Head: "other-feature", Title: "Take over"})

		require.ErrorIs(t, err, ErrTaskPullRequestOtherTask)
		assert.Len(t, fake.edited, edits)
	})

	t.Run("stops waiting for GitHub after the deadline", func(t *testing.T) {
		timeout := taskPullRequestGitHubTimeout
		taskPullRequestGitHubTimeout = 20 * time.Millisecond
		newTaskPullRequestGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (taskPullRequestGitHubAPI, error) {
			return stalledTaskPullRequestGitHub{fake}, nil
		}
		t.Cleanup(func() { taskPullRequestGitHubTimeout = timeout })
		before, err := ListTaskPullRequests(db, order)
		require.NoError(t, err)

		_, err = open(TaskPullRequestInput{Repository: "acme/api", Head: "stalled", Title: "Stalled"})

		require.ErrorIs(t, err, ErrTaskPullRequestGitHubTimeout)
		after, err := ListTaskPullRequests(db, order)
		require.NoError(t, err)
		assert.Len(t, after, len(before))
	})
}

// stalledTaskPullRequestGitHub never answers the pull request lookup.
type stalledTaskPullRequestGitHub struct {
	*fakeTaskPullRequestGitHub
}

func (stalledTaskPullRequestGitHub) ListPullRequests(ctx context.Context, _ string, _ *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	<-ctx.Done()
	return nil, nil, ctx.Err()
}
