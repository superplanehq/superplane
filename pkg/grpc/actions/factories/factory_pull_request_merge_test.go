package factories

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

type fakeFactoryGitHub struct {
	pullRequest   *github.PullRequest
	combined      *github.CombinedStatus
	combinedPages []*github.CombinedStatus
	checkRuns     *github.ListCheckRunsResults
	checkRunPages []*github.ListCheckRunsResults
	repository    *github.Repository
	mergeErr      error
	mergedMethod  string
	mergedSHA     string
	mergeCalls    int
	getPullErr    error
	combinedErr   error
	checkRunsErr  error
	repositoryErr error
}

func (f *fakeFactoryGitHub) GetPullRequest(context.Context, string, int) (*github.PullRequest, *github.Response, error) {
	if f.getPullErr != nil {
		return nil, nil, f.getPullErr
	}
	return f.pullRequest, nil, nil
}

func (f *fakeFactoryGitHub) GetCombinedStatus(_ context.Context, _ string, _ string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
	if f.combinedErr != nil {
		return nil, nil, f.combinedErr
	}
	if len(f.combinedPages) > 0 {
		page, next := fakeGitHubPage(opts, len(f.combinedPages))
		return f.combinedPages[page], &github.Response{NextPage: next}, nil
	}
	return f.combined, &github.Response{}, nil
}

func (f *fakeFactoryGitHub) ListCheckRunsForRef(_ context.Context, _ string, _ string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
	if f.checkRunsErr != nil {
		return nil, nil, f.checkRunsErr
	}
	if len(f.checkRunPages) > 0 {
		listOpts := github.ListOptions{}
		if opts != nil {
			listOpts = opts.ListOptions
		}
		page, next := fakeGitHubPage(&listOpts, len(f.checkRunPages))
		return f.checkRunPages[page], &github.Response{NextPage: next}, nil
	}
	return f.checkRuns, &github.Response{}, nil
}

func fakeGitHubPage(opts *github.ListOptions, pageCount int) (index int, nextPage int) {
	index = 0
	if opts != nil && opts.Page > 1 {
		index = opts.Page - 1
	}
	if index < 0 {
		index = 0
	}
	if index >= pageCount {
		return pageCount - 1, 0
	}
	if index+1 < pageCount {
		return index, index + 2
	}
	return index, 0
}

func (f *fakeFactoryGitHub) FindRepository(string) (*github.Repository, error) {
	if f.repositoryErr != nil {
		return nil, f.repositoryErr
	}
	return f.repository, nil
}

func (f *fakeFactoryGitHub) MergePullRequest(_ context.Context, _ string, _ int, _ string, options *github.PullRequestOptions) (*github.PullRequestMergeResult, *github.Response, error) {
	f.mergeCalls++
	if options != nil {
		f.mergedMethod = options.MergeMethod
		f.mergedSHA = options.SHA
	}
	if f.mergeErr != nil {
		return nil, nil, f.mergeErr
	}
	return &github.PullRequestMergeResult{Merged: github.Ptr(true)}, nil, nil
}

func mergeableGitHubPullRequest(sha string) *github.PullRequest {
	return &github.PullRequest{
		Mergeable:      github.Ptr(true),
		MergeableState: github.Ptr("clean"),
		Draft:          github.Ptr(false),
		Head:           &github.PullRequestBranch{SHA: github.Ptr(sha)},
	}
}

func allMethodsRepository() *github.Repository {
	return &github.Repository{
		AllowSquashMerge: github.Ptr(true),
		AllowMergeCommit: github.Ptr(true),
		AllowRebaseMerge: github.Ptr(true),
	}
}

func successChecks() (*github.CombinedStatus, *github.ListCheckRunsResults) {
	return emptyCombinedStatus(), &github.ListCheckRunsResults{}
}

func emptyCombinedStatus() *github.CombinedStatus {
	return &github.CombinedStatus{
		State:      github.Ptr("pending"),
		TotalCount: github.Ptr(0),
	}
}

func Test__FactoryPullRequestMerge(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	db := database.DB(t.Context())
	deps := IntakeDependencies{}
	const headSHA = "abc123def456"

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	createOrder := func(t *testing.T, factory *models.Factory) *models.FactoryWorkOrder {
		t.Helper()
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		return order
	}

	createGitHubPR := func(t *testing.T, factory *models.Factory, order *models.FactoryWorkOrder) *pb.FactoryPullRequest {
		t.Helper()
		resp, err := CreateFactoryPullRequest(ctx, IntakeDependencies{}, orgID, &pb.CreateFactoryPullRequestRequest{
			FactoryId:   factory.ID.String(),
			WorkOrderId: order.ID.String(),
			Provider:    pb.FactoryPullRequest_PROVIDER_GITHUB,
			Repository:  "acme/app",
			Number:      42,
			Url:         "https://github.com/acme/app/pull/42",
			Title:       "Ready",
			State:       pb.FactoryPullRequest_STATE_OPEN,
		})
		require.NoError(t, err)
		return resp.GetPullRequest()
	}

	useGitHub := func(t *testing.T, api *fakeFactoryGitHub) {
		t.Helper()
		original := newFactoryGitHubAPI
		newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
			return api, nil
		}
		t.Cleanup(func() { newFactoryGitHubAPI = original })
	}

	readyAPI := func() *fakeFactoryGitHub {
		combined, checks := successChecks()
		return &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    combined,
			checkRuns:   checks,
			repository:  allMethodsRepository(),
		}
	}

	merge := func(t *testing.T, factory *models.Factory, pr *pb.FactoryPullRequest, method pb.FactoryPullRequestMergeability_MergeMethod, sha string) (*pb.MergeFactoryPullRequestResponse, error) {
		t.Helper()
		return MergeFactoryPullRequest(ctx, deps, orgID, &pb.MergeFactoryPullRequestRequest{
			FactoryId:       factory.ID.String(),
			PrId:            pr.GetId(),
			MergeMethod:     method,
			ExpectedHeadSha: sha,
		})
	}

	t.Run("merges with squash", func(t *testing.T) {
		factory := newFactory(t)
		pr := createGitHubPR(t, factory, createOrder(t, factory))
		api := readyAPI()
		useGitHub(t, api)

		resp, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, headSHA)
		require.NoError(t, err)
		assert.Equal(t, 1, api.mergeCalls)
		assert.Equal(t, "squash", api.mergedMethod)
		assert.Equal(t, headSHA, api.mergedSHA)
		assert.Equal(t, pb.FactoryPullRequest_STATE_MERGED, resp.GetPullRequest().GetState())
		assert.NotNil(t, resp.GetPullRequest().GetMergedAt())
	})

	t.Run("merges with a merge commit", func(t *testing.T) {
		factory := newFactory(t)
		pr := createGitHubPR(t, factory, createOrder(t, factory))
		api := readyAPI()
		useGitHub(t, api)

		resp, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE, headSHA)
		require.NoError(t, err)
		assert.Equal(t, "merge", api.mergedMethod)
		assert.Equal(t, pb.FactoryPullRequest_STATE_MERGED, resp.GetPullRequest().GetState())
	})

	t.Run("merges with rebase", func(t *testing.T) {
		factory := newFactory(t)
		pr := createGitHubPR(t, factory, createOrder(t, factory))
		api := readyAPI()
		useGitHub(t, api)

		resp, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_REBASE, headSHA)
		require.NoError(t, err)
		assert.Equal(t, "rebase", api.mergedMethod)
		assert.Equal(t, pb.FactoryPullRequest_STATE_MERGED, resp.GetPullRequest().GetState())
	})

	t.Run("refuses a method the repository forbids", func(t *testing.T) {
		factory := newFactory(t)
		pr := createGitHubPR(t, factory, createOrder(t, factory))
		api := readyAPI()
		api.repository = &github.Repository{
			AllowSquashMerge: github.Ptr(true),
			AllowMergeCommit: github.Ptr(false),
			AllowRebaseMerge: github.Ptr(false),
		}
		useGitHub(t, api)

		_, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE, headSHA)
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, 0, api.mergeCalls)
	})

	t.Run("refuses while exclusive mutation access is held", func(t *testing.T) {
		factory := newFactory(t)
		order := createOrder(t, factory)
		pr := createGitHubPR(t, factory, order)
		grantExclusivePullRequestAccess(t, db, r, factory, pr)
		api := readyAPI()
		useGitHub(t, api)

		_, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, headSHA)
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, mergeBlockedActiveRun, message)
		assert.Equal(t, 0, api.mergeCalls)
	})

	t.Run("refuses while a run is active", func(t *testing.T) {
		factory := newFactory(t)
		order := createOrder(t, factory)
		pr := createGitHubPR(t, factory, order)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
		line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
		})
		require.NoError(t, err)
		_, err = DispatchWorkOrder(ctx, orgID, &pb.DispatchWorkOrderRequest{
			FactoryId: factory.ID.String(),
			OrderId:   order.ID.String(),
			LineName:  line.Name,
		})
		require.NoError(t, err)
		api := readyAPI()
		useGitHub(t, api)

		_, err = merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, headSHA)
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, mergeBlockedActiveRun, message)
		assert.Equal(t, 0, api.mergeCalls)
	})

	t.Run("refuses while a check run is in progress", func(t *testing.T) {
		factory := newFactory(t)
		pr := createGitHubPR(t, factory, createOrder(t, factory))
		api := readyAPI()
		api.checkRuns = &github.ListCheckRunsResults{
			CheckRuns: []*github.CheckRun{{
				Name:   github.Ptr("ci"),
				Status: github.Ptr("in_progress"),
			}},
		}
		useGitHub(t, api)

		_, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, headSHA)
		require.Error(t, err)
		_, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, mergeBlockedChecksUnfinished, message)
		assert.Equal(t, 0, api.mergeCalls)
	})

	t.Run("refuses after a failed status", func(t *testing.T) {
		factory := newFactory(t)
		pr := createGitHubPR(t, factory, createOrder(t, factory))
		api := readyAPI()
		api.combined = &github.CombinedStatus{
			State: github.Ptr("failure"),
			Statuses: []*github.RepoStatus{{
				Context: github.Ptr("ci"),
				State:   github.Ptr("failure"),
			}},
		}
		useGitHub(t, api)

		_, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, headSHA)
		require.Error(t, err)
		_, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, mergeBlockedCheckFailed, message)
		assert.Equal(t, 0, api.mergeCalls)
	})

	t.Run("refuses when the head SHA moved", func(t *testing.T) {
		factory := newFactory(t)
		pr := createGitHubPR(t, factory, createOrder(t, factory))
		api := readyAPI()
		useGitHub(t, api)

		_, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, "oldsha")
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, 0, api.mergeCalls)
	})

	t.Run("refuses a Bitbucket pull request", func(t *testing.T) {
		factory := newFactory(t)
		order := createOrder(t, factory)
		created, err := CreateFactoryPullRequest(ctx, IntakeDependencies{}, orgID, &pb.CreateFactoryPullRequestRequest{
			FactoryId:   factory.ID.String(),
			WorkOrderId: order.ID.String(),
			Provider:    pb.FactoryPullRequest_PROVIDER_BITBUCKET,
			Repository:  "acme/app",
			Number:      7,
			Url:         "https://bitbucket.org/acme/app/pull-requests/7",
			Title:       "Bitbucket",
			State:       pb.FactoryPullRequest_STATE_OPEN,
		})
		require.NoError(t, err)
		api := readyAPI()
		useGitHub(t, api)

		_, err = merge(t, factory, created.GetPullRequest(), pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, headSHA)
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, 0, api.mergeCalls)
	})

	t.Run("refuses when no GitHub integration is ready", func(t *testing.T) {
		factory := newFactory(t)
		pr := createGitHubPR(t, factory, createOrder(t, factory))

		_, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, headSHA)
		require.Error(t, err)
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, mergeBlockedMissingIntegration, message)
	})

	t.Run("persists the merged state after success", func(t *testing.T) {
		factory := newFactory(t)
		pr := createGitHubPR(t, factory, createOrder(t, factory))
		api := readyAPI()
		useGitHub(t, api)

		_, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, headSHA)
		require.NoError(t, err)

		stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: parseUUID(t, pr.GetId())})
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestStateMerged, stored.State)
		require.NotNil(t, stored.MergedAt)
	})

	t.Run("refuses when GitHub reports the head moved", func(t *testing.T) {
		factory := newFactory(t)
		pr := createGitHubPR(t, factory, createOrder(t, factory))
		api := readyAPI()
		api.mergeErr = &github.ErrorResponse{Response: &http.Response{StatusCode: http.StatusConflict}}
		useGitHub(t, api)

		_, err := merge(t, factory, pr, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, headSHA)
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
	})
}

func Test__FactoryPullRequestMergeability(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	orgID := r.Organization.ID.String()
	db := database.DB(t.Context())
	deps := IntakeDependencies{}
	const headSHA = "abc123def456"

	newFactory := func(t *testing.T) *models.Factory {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		return factory
	}

	createPR := func(t *testing.T, factory *models.Factory) *pb.FactoryPullRequest {
		t.Helper()
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		resp, err := CreateFactoryPullRequest(ctx, IntakeDependencies{}, orgID, &pb.CreateFactoryPullRequestRequest{
			FactoryId:   factory.ID.String(),
			WorkOrderId: order.ID.String(),
			Provider:    pb.FactoryPullRequest_PROVIDER_GITHUB,
			Repository:  "acme/app",
			Number:      42,
			Url:         "https://github.com/acme/app/pull/42",
			Title:       "Ready",
			State:       pb.FactoryPullRequest_STATE_OPEN,
		})
		require.NoError(t, err)
		return resp.GetPullRequest()
	}

	useGitHub := func(t *testing.T, api *fakeFactoryGitHub) {
		t.Helper()
		original := newFactoryGitHubAPI
		newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
			return api, nil
		}
		t.Cleanup(func() { newFactoryGitHubAPI = original })
	}

	describe := func(t *testing.T, factory *models.Factory, pr *pb.FactoryPullRequest) *pb.FactoryPullRequestMergeability {
		t.Helper()
		resp, err := DescribeFactoryPullRequestMergeability(ctx, deps, orgID, &pb.DescribeFactoryPullRequestMergeabilityRequest{
			FactoryId: factory.ID.String(),
			PrId:      pr.GetId(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.GetMergeability())
		return resp.GetMergeability()
	}

	t.Run("reports exclusive mutation access", func(t *testing.T) {
		factory := newFactory(t)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		resp, err := CreateFactoryPullRequest(ctx, IntakeDependencies{}, orgID, &pb.CreateFactoryPullRequestRequest{
			FactoryId:   factory.ID.String(),
			WorkOrderId: order.ID.String(),
			Provider:    pb.FactoryPullRequest_PROVIDER_GITHUB,
			Repository:  "acme/app",
			Number:      42,
			Url:         "https://github.com/acme/app/pull/42",
			State:       pb.FactoryPullRequest_STATE_OPEN,
		})
		require.NoError(t, err)
		grantExclusivePullRequestAccess(t, db, r, factory, resp.GetPullRequest())
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			repository:  allMethodsRepository(),
		})

		got := describe(t, factory, resp.GetPullRequest())
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_ACTIVE_RUN, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedActiveRun, got.GetMessage())
	})

	t.Run("reports an active run", func(t *testing.T) {
		factory := newFactory(t)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		resp, err := CreateFactoryPullRequest(ctx, IntakeDependencies{}, orgID, &pb.CreateFactoryPullRequestRequest{
			FactoryId:   factory.ID.String(),
			WorkOrderId: order.ID.String(),
			Provider:    pb.FactoryPullRequest_PROVIDER_GITHUB,
			Repository:  "acme/app",
			Number:      42,
			Url:         "https://github.com/acme/app/pull/42",
			State:       pb.FactoryPullRequest_STATE_OPEN,
		})
		require.NoError(t, err)
		app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "step-one", "start-one")
		line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
		})
		require.NoError(t, err)
		_, err = DispatchWorkOrder(ctx, orgID, &pb.DispatchWorkOrderRequest{
			FactoryId: factory.ID.String(),
			OrderId:   order.ID.String(),
			LineName:  line.Name,
		})
		require.NoError(t, err)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			repository:  allMethodsRepository(),
		})

		got := describe(t, factory, resp.GetPullRequest())
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_ACTIVE_RUN, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedActiveRun, got.GetMessage())
	})

	t.Run("reports unfinished checks", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		combined, _ := successChecks()
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    combined,
			checkRuns: &github.ListCheckRunsResults{
				CheckRuns: []*github.CheckRun{{Status: github.Ptr("queued")}},
			},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedChecksUnfinished, got.GetMessage())
	})

	t.Run("allows merge when GitHub has no commit statuses", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    emptyCombinedStatus(),
			checkRuns:   &github.ListCheckRunsResults{},
			repository:  allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.True(t, got.GetCanMerge())
		assert.Equal(t, headSHA, got.GetHeadSha())
	})

	t.Run("allows merge when only check runs passed", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    emptyCombinedStatus(),
			checkRuns: &github.ListCheckRunsResults{
				CheckRuns: []*github.CheckRun{{
					Status:     github.Ptr("completed"),
					Conclusion: github.Ptr("success"),
				}},
			},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.True(t, got.GetCanMerge())
	})

	t.Run("reports unfinished when a commit status is pending", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined: &github.CombinedStatus{
				State:      github.Ptr("pending"),
				TotalCount: github.Ptr(1),
				Statuses: []*github.RepoStatus{{
					Context: github.Ptr("ci"),
					State:   github.Ptr("pending"),
				}},
			},
			checkRuns:  &github.ListCheckRunsResults{},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedChecksUnfinished, got.GetMessage())
	})

	t.Run("reports a failed check on a later page", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combinedPages: []*github.CombinedStatus{
				{State: github.Ptr("success"), Statuses: []*github.RepoStatus{{Context: github.Ptr("lint"), State: github.Ptr("success")}}},
				{State: github.Ptr("failure"), Statuses: []*github.RepoStatus{{Context: github.Ptr("e2e"), State: github.Ptr("failure")}}},
			},
			checkRuns:  &github.ListCheckRunsResults{},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECK_FAILED, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedCheckFailed, got.GetMessage())
	})

	t.Run("reports an unfinished check on a later page", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    &github.CombinedStatus{State: github.Ptr("success")},
			checkRunPages: []*github.ListCheckRunsResults{
				{CheckRuns: []*github.CheckRun{{Status: github.Ptr("completed"), Conclusion: github.Ptr("success")}}},
				{CheckRuns: []*github.CheckRun{{Status: github.Ptr("in_progress")}}},
			},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedChecksUnfinished, got.GetMessage())
	})

	t.Run("reports an action_required check as failed", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    &github.CombinedStatus{State: github.Ptr("success")},
			checkRuns: &github.ListCheckRunsResults{
				CheckRuns: []*github.CheckRun{{
					Status:     github.Ptr("completed"),
					Conclusion: github.Ptr("action_required"),
				}},
			},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECK_FAILED, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedCheckFailed, got.GetMessage())
	})

	t.Run("reports an error check as failed", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    &github.CombinedStatus{State: github.Ptr("success")},
			checkRuns: &github.ListCheckRunsResults{
				CheckRuns: []*github.CheckRun{{
					Status:     github.Ptr("completed"),
					Conclusion: github.Ptr("error"),
				}},
			},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECK_FAILED, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedCheckFailed, got.GetMessage())
	})

	t.Run("reports an unknown check conclusion as unfinished", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    &github.CombinedStatus{State: github.Ptr("success")},
			checkRuns: &github.ListCheckRunsResults{
				CheckRuns: []*github.CheckRun{{
					Status:     github.Ptr("completed"),
					Conclusion: github.Ptr("stale"),
				}},
			},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedChecksUnfinished, got.GetMessage())
	})

	t.Run("reports a cancelled check as failed", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    &github.CombinedStatus{State: github.Ptr("success")},
			checkRuns: &github.ListCheckRunsResults{
				CheckRuns: []*github.CheckRun{{
					Status:     github.Ptr("completed"),
					Conclusion: github.Ptr("cancelled"),
				}},
			},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECK_FAILED, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedCheckFailed, got.GetMessage())
	})

	t.Run("reports a failed check", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    &github.CombinedStatus{State: github.Ptr("success")},
			checkRuns: &github.ListCheckRunsResults{
				CheckRuns: []*github.CheckRun{{
					Status:     github.Ptr("completed"),
					Conclusion: github.Ptr("timed_out"),
				}},
			},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECK_FAILED, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedCheckFailed, got.GetMessage())
	})

	t.Run("reports a draft pull request", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: &github.PullRequest{
				Mergeable:      github.Ptr(true),
				MergeableState: github.Ptr("draft"),
				Draft:          github.Ptr(true),
				Head:           &github.PullRequestBranch{SHA: github.Ptr(headSHA)},
			},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_DRAFT, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedDraft, got.GetMessage())
	})

	t.Run("reports a conflicting pull request", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: &github.PullRequest{
				Mergeable:      github.Ptr(false),
				MergeableState: github.Ptr("dirty"),
				Draft:          github.Ptr(false),
				Head:           &github.PullRequestBranch{SHA: github.Ptr(headSHA)},
			},
			repository: allMethodsRepository(),
		})

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CONFLICTING, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedConflicting, got.GetMessage())
	})

	t.Run("reports a missing integration", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)

		got := describe(t, factory, pr)
		assert.False(t, got.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_MISSING_INTEGRATION, got.GetBlockedReason())
		assert.Equal(t, mergeBlockedMissingIntegration, got.GetMessage())
	})

	t.Run("lists allowed methods from repository flags", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		combined, checks := successChecks()
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    combined,
			checkRuns:   checks,
			repository: &github.Repository{
				AllowSquashMerge: github.Ptr(false),
				AllowMergeCommit: github.Ptr(true),
				AllowRebaseMerge: github.Ptr(true),
			},
		})

		got := describe(t, factory, pr)
		assert.True(t, got.GetCanMerge())
		assert.Equal(t, []pb.FactoryPullRequestMergeability_MergeMethod{
			pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE,
			pb.FactoryPullRequestMergeability_MERGE_METHOD_REBASE,
		}, got.GetAllowedMethods())
		assert.Equal(t, headSHA, got.GetHeadSha())
	})

	t.Run("reuses a stored mergeable snapshot", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		combined, checks := successChecks()
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    combined,
			checkRuns:   checks,
			repository:  allMethodsRepository(),
		})
		first := describe(t, factory, pr)
		assert.True(t, first.GetCanMerge())
		assert.Equal(t, []pb.FactoryPullRequestMergeability_MergeMethod{
			pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH,
			pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE,
			pb.FactoryPullRequestMergeability_MERGE_METHOD_REBASE,
		}, first.GetAllowedMethods())

		useGitHub(t, &fakeFactoryGitHub{})
		second := describe(t, factory, pr)
		assert.True(t, second.GetCanMerge())
		assert.Equal(t, headSHA, second.GetHeadSha())
		assert.Equal(t, first.GetAllowedMethods(), second.GetAllowedMethods())
	})

	t.Run("reuses stored allowed merge methods", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		combined, checks := successChecks()
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    combined,
			checkRuns:   checks,
			repository: &github.Repository{
				AllowSquashMerge: github.Ptr(true),
				AllowMergeCommit: github.Ptr(false),
				AllowRebaseMerge: github.Ptr(false),
			},
		})
		first := describe(t, factory, pr)
		assert.True(t, first.GetCanMerge())
		assert.Equal(t, []pb.FactoryPullRequestMergeability_MergeMethod{
			pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH,
		}, first.GetAllowedMethods())

		useGitHub(t, &fakeFactoryGitHub{})
		second := describe(t, factory, pr)
		assert.Equal(t, first.GetAllowedMethods(), second.GetAllowedMethods())

		_, err := MergeFactoryPullRequest(ctx, deps, orgID, &pb.MergeFactoryPullRequestRequest{
			FactoryId:       factory.ID.String(),
			PrId:            pr.GetId(),
			MergeMethod:     pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE,
			ExpectedHeadSha: headSHA,
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
	})

	t.Run("ignores a snapshot after the head revision moves", func(t *testing.T) {
		factory := newFactory(t)
		pr := createPR(t, factory)
		combined, checks := successChecks()
		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest(headSHA),
			combined:    combined,
			checkRuns:   checks,
			repository:  allMethodsRepository(),
		})
		first := describe(t, factory, pr)
		assert.True(t, first.GetCanMerge())

		stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: parseUUID(t, pr.GetId())})
		require.NoError(t, err)
		_, err = stored.ObserveRevision(db, "newheadsha")
		require.NoError(t, err)

		useGitHub(t, &fakeFactoryGitHub{
			pullRequest: mergeableGitHubPullRequest("newheadsha"),
			combined: &github.CombinedStatus{
				State:      github.Ptr("pending"),
				TotalCount: github.Ptr(1),
				Statuses: []*github.RepoStatus{{
					Context: github.Ptr("ci"),
					State:   github.Ptr("pending"),
				}},
			},
			checkRuns:  &github.ListCheckRunsResults{},
			repository: allMethodsRepository(),
		})
		second := describe(t, factory, pr)
		assert.False(t, second.GetCanMerge())
		assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_CHECKS_UNFINISHED, second.GetBlockedReason())
		assert.Equal(t, "newheadsha", second.GetHeadSha())
	})
}

func grantExclusivePullRequestAccess(
	t *testing.T,
	db *gorm.DB,
	r *support.ResourceRegistry,
	factory *models.Factory,
	pr *pb.FactoryPullRequest,
) {
	t.Helper()
	canvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, support.RandomName("feedback"), "start")
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, entrypoint, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: parseUUID(t, pr.GetId())})
	require.NoError(t, err)
	result, err := stored.CreateActivity(db, models.FactoryPullRequestActivityParams{
		RunID:       run.ID,
		Access:      models.FactoryPullRequestAccessExclusive,
		Description: "Address review comment",
	})
	require.NoError(t, err)
	require.Equal(t, models.FactoryPullRequestActivityOutcomeReady, result.Outcome)
}
