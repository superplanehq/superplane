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
	"github.com/superplanehq/superplane/pkg/features"
	bitbucketintegration "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
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
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
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
		pullRequest, err := open(TaskPullRequestInput{Repository: "acme/api", Head: "feature", Title: "Change the API again", Draft: true})
		require.NoError(t, err)

		assert.Len(t, fake.created, 1)
		require.Len(t, fake.edited, 1)
		assert.Equal(t, "Change the API again", pullRequest.Title)
		assert.Nil(t, fake.edited[0].Draft, "editing a pull request must not change its GitHub draft status")
		assert.Equal(t, models.FactoryPullRequestStateOpen, pullRequest.State)
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
		timeout := taskPullRequestHostTimeout
		taskPullRequestHostTimeout = 20 * time.Millisecond
		newTaskPullRequestGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (taskPullRequestGitHubAPI, error) {
			return stalledTaskPullRequestGitHub{fake}, nil
		}
		t.Cleanup(func() { taskPullRequestHostTimeout = timeout })
		before, err := ListTaskPullRequests(db, order)
		require.NoError(t, err)

		_, err = open(TaskPullRequestInput{Repository: "acme/api", Head: "stalled", Title: "Stalled"})

		require.ErrorIs(t, err, ErrTaskPullRequestGitHubTimeout)
		after, err := ListTaskPullRequests(db, order)
		require.NoError(t, err)
		assert.Len(t, after, len(before))
	})
}

type fakeTaskPullRequestBitbucket struct {
	mainBranch string
	open       map[string][]map[string]any
	created    []bitbucketintegration.CreatePullRequestRequest
	updated    []bitbucketintegration.UpdatePullRequestRequest
	calls      int
}

func (f *fakeTaskPullRequestBitbucket) GetMainBranch(string) (string, error) {
	f.calls++
	return f.mainBranch, nil
}

func (f *fakeTaskPullRequestBitbucket) ListPullRequests(repository string, opts bitbucketintegration.PullRequestListOptions) ([]map[string]any, error) {
	f.calls++
	matches := []map[string]any{}
	for _, pullRequest := range f.open[repository] {
		if pullRequest["sourceBranch"] == opts.SourceBranch && pullRequest["destinationBranch"] == opts.DestinationBranch {
			matches = append(matches, pullRequest)
		}
	}
	return matches, nil
}

func (f *fakeTaskPullRequestBitbucket) CreatePullRequest(repository string, request bitbucketintegration.CreatePullRequestRequest) (map[string]any, error) {
	f.calls++
	f.created = append(f.created, request)
	number := len(f.created)
	pullRequest := map[string]any{
		"id":                number,
		"title":             request.Title,
		"draft":             request.Draft,
		"links":             map[string]any{"html": map[string]any{"href": fmt.Sprintf("https://bitbucket.org/%s/pull-requests/%d", repository, number)}},
		"destination":       map[string]any{"repository": map[string]any{"full_name": repository}},
		"sourceBranch":      request.Source,
		"destinationBranch": request.Destination,
	}
	f.open[repository] = append(f.open[repository], pullRequest)
	return pullRequest, nil
}

func (f *fakeTaskPullRequestBitbucket) UpdatePullRequest(repository string, id int64, request bitbucketintegration.UpdatePullRequestRequest) (map[string]any, error) {
	f.calls++
	f.updated = append(f.updated, request)
	for _, pullRequest := range f.open[repository] {
		if int64(pullRequest["id"].(int)) == id {
			pullRequest["title"] = request.Title
			return pullRequest, nil
		}
	}
	return nil, fmt.Errorf("pull request %d not found", id)
}

// stalledTaskPullRequestBitbucket never answers the pull request lookup.
type stalledTaskPullRequestBitbucket struct {
	*fakeTaskPullRequestBitbucket
	release chan struct{}
}

func (s stalledTaskPullRequestBitbucket) ListPullRequests(string, bitbucketintegration.PullRequestListOptions) ([]map[string]any, error) {
	<-s.release
	return nil, nil
}

func bitbucketTaskPullRequestWorkspace(t *testing.T, r *support.ResourceRegistry) (*models.Factory, *models.FactoryWorkOrder) {
	t.Helper()
	db := database.DB(t.Context())
	factory := bitbucketFactory(t, r)
	integrationID := uuid.NewString()
	repository, branch := "acme/api", "main"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &integrationID, AppRepository: &repository, DefaultBranch: &branch,
	}))
	for repository, access := range map[string]string{"acme/web": models.FactoryContextRepoAccessWrite, "acme/docs": models.FactoryContextRepoAccessRead} {
		_, err := factory.CreateAgentResource(db, models.FactoryAgentResourceKindContextRepo, "", true, models.FactoryAgentResourceConfig{
			Repository: repository, VCSProvider: models.ProviderBitbucket, Access: access,
		})
		require.NoError(t, err)
	}
	order, err := factory.CreateWorkOrder(db, "Implement", "", &r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, db.Model(order).Updates(map[string]any{"origin_url": "https://github.com/acme/api/issues/12", "origin_label": "acme/api#12"}).Error)
	order, err = factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	return factory, order
}

func installTaskPullRequestBitbucket(t *testing.T, api taskPullRequestBitbucketAPI) {
	t.Helper()
	original := newTaskPullRequestBitbucketAPI
	newTaskPullRequestBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (taskPullRequestBitbucketAPI, error) {
		return api, nil
	}
	t.Cleanup(func() { newTaskPullRequestBitbucketAPI = original })
}

func TestOpenTaskPullRequestInBitbucket(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryMultiRepoTasks))
	db := database.DB(t.Context())
	factory, order := bitbucketTaskPullRequestWorkspace(t, r)
	fake := &fakeTaskPullRequestBitbucket{mainBranch: "trunk", open: map[string][]map[string]any{}}
	installTaskPullRequestBitbucket(t, fake)
	open := func(input TaskPullRequestInput) (*models.FactoryPullRequest, error) {
		return OpenTaskPullRequest(context.Background(), IntakeDependencies{}, "http://app.test", factory, order, input)
	}

	t.Run("opens the pull request on the task branch with the task footer", func(t *testing.T) {
		pullRequest, err := open(TaskPullRequestInput{Repository: "acme/api", Head: "feature", Title: "Change the API", Body: "Why"})
		require.NoError(t, err)

		require.Len(t, fake.created, 1)
		assert.Equal(t, "main", fake.created[0].Destination)
		body := fake.created[0].Description
		assert.NotContains(t, body, "Closes", "Bitbucket does not close GitHub issues")
		assert.Contains(t, body, fmt.Sprintf("[%s](http://app.test%s)", factory.WorkOrderKey(order.Number), order.URLPath(factory.RouteSegment())))
		assert.Contains(t, body, "Why\n\n---\n\nCreated via [SuperPlane](https://superplane.com).")
		assert.Equal(t, models.FactoryPullRequestProviderBitbucket, pullRequest.Provider)
		assert.Equal(t, "acme/api", pullRequest.Repository)
		assert.Equal(t, int64(1), pullRequest.Number)
		assert.Equal(t, "https://bitbucket.org/acme/api/pull-requests/1", pullRequest.URL)
		assert.Nil(t, pullRequest.ExternalID)
		assert.Equal(t, models.FactoryPullRequestStateOpen, pullRequest.State)
	})

	t.Run("updates the same pull request when the branch is opened again", func(t *testing.T) {
		pullRequest, err := open(TaskPullRequestInput{Repository: "acme/api", Head: "feature", Title: "Change the API again"})
		require.NoError(t, err)

		assert.Len(t, fake.created, 1)
		require.Len(t, fake.updated, 1)
		assert.Equal(t, "Change the API again", pullRequest.Title)
		tracked, err := ListTaskPullRequests(db, order)
		require.NoError(t, err)
		assert.Len(t, tracked, 1)
	})

	t.Run("opens a draft pull request in a Read & Write repository on its main branch", func(t *testing.T) {
		pullRequest, err := open(TaskPullRequestInput{Repository: "acme/web", Head: "feature", Title: "Change the web app", Draft: true})
		require.NoError(t, err)

		created := fake.created[len(fake.created)-1]
		assert.Equal(t, "trunk", created.Destination)
		assert.True(t, created.Draft)
		assert.Equal(t, models.FactoryPullRequestStateDraft, pullRequest.State)
		tracked, err := ListTaskPullRequests(db, order)
		require.NoError(t, err)
		assert.Len(t, tracked, 2)
	})

	t.Run("rejects a read-only repository before it calls Bitbucket", func(t *testing.T) {
		calls := fake.calls
		_, err := open(TaskPullRequestInput{Repository: "acme/docs", Head: "feature", Title: "Change docs"})

		require.ErrorIs(t, err, models.ErrFactoryPullRequestRepositoryNotInWorkspace)
		assert.Equal(t, calls, fake.calls)
	})

	t.Run("does not update the pull request of another task", func(t *testing.T) {
		other, err := factory.CreateWorkOrder(db, "Other", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = OpenTaskPullRequest(context.Background(), IntakeDependencies{}, "http://app.test", factory, other, TaskPullRequestInput{
			Repository: "acme/api", Head: "other-feature", Title: "Other task change",
		})
		require.NoError(t, err)
		updates := len(fake.updated)

		_, err = open(TaskPullRequestInput{Repository: "acme/api", Head: "other-feature", Title: "Take over"})

		require.ErrorIs(t, err, ErrTaskPullRequestOtherTask)
		assert.Len(t, fake.updated, updates)
	})

	t.Run("stops waiting for Bitbucket after the deadline", func(t *testing.T) {
		timeout := taskPullRequestHostTimeout
		taskPullRequestHostTimeout = 20 * time.Millisecond
		t.Cleanup(func() { taskPullRequestHostTimeout = timeout })
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })
		installTaskPullRequestBitbucket(t, stalledTaskPullRequestBitbucket{fake, release})
		before, err := ListTaskPullRequests(db, order)
		require.NoError(t, err)

		_, err = open(TaskPullRequestInput{Repository: "acme/api", Head: "stalled", Title: "Stalled"})

		require.ErrorIs(t, err, ErrTaskPullRequestBitbucketTimeout)
		after, err := ListTaskPullRequests(db, order)
		require.NoError(t, err)
		assert.Len(t, after, len(before))
	})
}

func TestOpenTaskPullRequestInBitbucketWithoutMultiRepoTasks(t *testing.T) {
	r := support.Setup(t)
	factory, order := bitbucketTaskPullRequestWorkspace(t, r)
	fake := &fakeTaskPullRequestBitbucket{mainBranch: "trunk", open: map[string][]map[string]any{}}
	installTaskPullRequestBitbucket(t, fake)

	_, err := OpenTaskPullRequest(context.Background(), IntakeDependencies{}, "http://app.test", factory, order, TaskPullRequestInput{
		Repository: "acme/web", Head: "feature", Title: "Change the web app",
	})

	require.ErrorIs(t, err, models.ErrFactoryPullRequestRepositoryNotInWorkspace)
	assert.Zero(t, fake.calls)
}

// stalledTaskPullRequestGitHub never answers the pull request lookup.
type stalledTaskPullRequestGitHub struct {
	*fakeTaskPullRequestGitHub
}

func (stalledTaskPullRequestGitHub) ListPullRequests(ctx context.Context, _ string, _ *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	<-ctx.Done()
	return nil, nil, ctx.Err()
}
