package factories

import (
	"context"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGitHubStatusAPI struct {
	repo         *github.Repository
	protection   *github.Protection
	closedPulls  []*github.PullRequest
	openPulls    []*github.PullRequest
	checkRuns    map[string]*github.ListCheckRunsResults
	statuses     map[string]*github.CombinedStatus
	listedRefs   []string
	listedStates []string
	repoErr      error
	protectErr   error
	pullsErr     error
	checkRunErr  error
	statusErr    error
}

func (f *fakeGitHubStatusAPI) FindRepository(string) (*github.Repository, error) {
	return f.repo, f.repoErr
}

func (f *fakeGitHubStatusAPI) GetBranchProtection(context.Context, string, string) (*github.Protection, error) {
	return f.protection, f.protectErr
}

func (f *fakeGitHubStatusAPI) ListPullRequests(_ context.Context, _ string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	state := ""
	if opts != nil {
		state = opts.State
	}
	f.listedStates = append(f.listedStates, state)
	if state == "open" {
		return f.openPulls, &github.Response{}, f.pullsErr
	}
	return f.closedPulls, &github.Response{}, f.pullsErr
}

func (f *fakeGitHubStatusAPI) ListCheckRunsForRef(_ context.Context, _ string, ref string, _ *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
	f.listedRefs = append(f.listedRefs, ref)
	if f.checkRuns == nil {
		return &github.ListCheckRunsResults{}, &github.Response{}, f.checkRunErr
	}
	return f.checkRuns[ref], &github.Response{}, f.checkRunErr
}

func (f *fakeGitHubStatusAPI) GetCombinedStatus(_ context.Context, _ string, ref string, _ *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
	if f.statuses == nil {
		return &github.CombinedStatus{}, &github.Response{}, f.statusErr
	}
	return f.statuses[ref], &github.Response{}, f.statusErr
}

func TestLoadGitHubRepositoryStatusCheckSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("reads required checks and recent PR checks", func(t *testing.T) {
		t.Parallel()
		api := &fakeGitHubStatusAPI{
			repo: &github.Repository{DefaultBranch: github.Ptr("main")},
			protection: &github.Protection{
				RequiredStatusChecks: &github.RequiredStatusChecks{
					Checks: &[]*github.RequiredStatusCheck{{Context: "lint"}},
				},
			},
			closedPulls: []*github.PullRequest{
				{Head: &github.PullRequestBranch{SHA: github.Ptr("abc123")}},
			},
			checkRuns: map[string]*github.ListCheckRunsResults{
				"abc123": {
					CheckRuns: []*github.CheckRun{
						{Name: github.Ptr("unit"), DetailsURL: github.Ptr("https://acme.semaphoreci.com/jobs/1")},
					},
				},
			},
			statuses: map[string]*github.CombinedStatus{
				"abc123": {
					Statuses: []*github.RepoStatus{
						{Context: github.Ptr("deploy"), TargetURL: github.Ptr("https://app.circleci.com/pipelines/1")},
					},
				},
			},
		}

		snapshot, err := loadGitHubRepositoryStatusCheckSnapshot(context.Background(), api, "acme/api")
		require.NoError(t, err)
		assert.Equal(t, []string{"lint"}, snapshot.Required)
		assert.Equal(t, []observedRepositoryStatusCheck{
			{Name: "unit", DetailsURL: "https://acme.semaphoreci.com/jobs/1"},
			{Name: "deploy", DetailsURL: "https://app.circleci.com/pipelines/1"},
		}, snapshot.Observed)
		assert.Equal(t, []string{"abc123"}, api.listedRefs)
	})

	t.Run("falls back to the default branch when no pull requests exist", func(t *testing.T) {
		t.Parallel()
		api := &fakeGitHubStatusAPI{
			repo: &github.Repository{DefaultBranch: github.Ptr("develop")},
			checkRuns: map[string]*github.ListCheckRunsResults{
				"develop": {CheckRuns: []*github.CheckRun{{Name: github.Ptr("build")}}},
			},
		}

		snapshot, err := loadGitHubRepositoryStatusCheckSnapshot(context.Background(), api, "acme/api")
		require.NoError(t, err)
		assert.Equal(t, []observedRepositoryStatusCheck{{Name: "build"}}, snapshot.Observed)
		assert.Equal(t, []string{"develop"}, api.listedRefs)
	})

	t.Run("still lists closed PR checks when branch protection is unreadable", func(t *testing.T) {
		t.Parallel()
		api := &fakeGitHubStatusAPI{
			repo:       &github.Repository{DefaultBranch: github.Ptr("main")},
			protectErr: assert.AnError,
			closedPulls: []*github.PullRequest{
				{Head: &github.PullRequestBranch{SHA: github.Ptr("f95a333")}},
			},
			checkRuns: map[string]*github.ListCheckRunsResults{
				"f95a333": {
					CheckRuns: []*github.CheckRun{
						{Name: github.Ptr("GitGuardian Security Checks"), DetailsURL: github.Ptr("https://dashboard.gitguardian.com")},
					},
				},
			},
			statuses: map[string]*github.CombinedStatus{
				"f95a333": {
					Statuses: []*github.RepoStatus{
						{Context: github.Ptr("ci/semaphoreci/push: CI"), TargetURL: github.Ptr("https://lucaspin.semaphoreci.com/workflows/1")},
					},
				},
			},
		}

		snapshot, err := loadGitHubRepositoryStatusCheckSnapshot(context.Background(), api, "lucaspin/decks-api")
		require.NoError(t, err)
		assert.Empty(t, snapshot.Required)
		assert.Equal(t, []observedRepositoryStatusCheck{
			{Name: "GitGuardian Security Checks", DetailsURL: "https://dashboard.gitguardian.com"},
			{Name: "ci/semaphoreci/push: CI", DetailsURL: "https://lucaspin.semaphoreci.com/workflows/1"},
		}, snapshot.Observed)
		assert.Equal(t, []string{"closed", "open"}, api.listedStates)
		assert.Equal(t, []string{"f95a333"}, api.listedRefs)
	})

	t.Run("inspects at most three recent pull request SHAs", func(t *testing.T) {
		t.Parallel()
		api := &fakeGitHubStatusAPI{
			repo: &github.Repository{DefaultBranch: github.Ptr("main")},
			closedPulls: []*github.PullRequest{
				{Head: &github.PullRequestBranch{SHA: github.Ptr("sha-1")}},
				{Head: &github.PullRequestBranch{SHA: github.Ptr("sha-2")}},
				{Head: &github.PullRequestBranch{SHA: github.Ptr("sha-3")}},
				{Head: &github.PullRequestBranch{SHA: github.Ptr("sha-4")}},
				{Head: &github.PullRequestBranch{SHA: github.Ptr("sha-5")}},
			},
			checkRuns: map[string]*github.ListCheckRunsResults{
				"sha-1": {CheckRuns: []*github.CheckRun{{Name: github.Ptr("lint")}}},
				"sha-2": {CheckRuns: []*github.CheckRun{{Name: github.Ptr("unit")}}},
				"sha-3": {CheckRuns: []*github.CheckRun{{Name: github.Ptr("e2e")}}},
				"sha-4": {CheckRuns: []*github.CheckRun{{Name: github.Ptr("extra")}}},
			},
		}

		snapshot, err := loadGitHubRepositoryStatusCheckSnapshot(context.Background(), api, "acme/api")
		require.NoError(t, err)
		assert.Equal(t, []string{"sha-1", "sha-2", "sha-3"}, api.listedRefs)
		assert.Equal(t, []observedRepositoryStatusCheck{
			{Name: "lint"},
			{Name: "unit"},
			{Name: "e2e"},
		}, snapshot.Observed)
	})
}
