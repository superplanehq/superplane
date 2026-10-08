package factories

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	bitbucketintegration "github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func TestDescribeFactoryPullRequestMergeability_BitbucketFactory(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:   "https://github.com/acme/app/pull/42",
		State: models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	_, err = DescribeFactoryPullRequestMergeability(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.DescribeFactoryPullRequestMergeabilityRequest{
		FactoryId: factory.ID.String(),
		PrId:      pullRequest.ID.String(),
	})
	require.ErrorIs(t, err, errFactoryPullRequestNotGitHub)
	_, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, "Only pull requests on the workspace Git host can merge from SuperPlane.", message)
}

func TestDescribeFactoryPullRequestMergeability_BitbucketPR(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider: models.FactoryPullRequestProviderBitbucket,
		URL:      "https://bitbucket.org/acme/widgets/pull-requests/42",
		State:    models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"values": [{"key": "build-a", "state": "SUCCESSFUL"}]}`))},
		},
	}
	stubClient, err := bitbucketintegration.NewClient(
		bitbucketintegration.AuthTypeWorkspaceAccessToken,
		httpCtx,
		&contexts.IntegrationContext{Configuration: map[string]any{"token": "token"}},
	)
	require.NoError(t, err)
	restore := newFactoryBitbucketAPI
	newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucketintegration.Client, error) {
		return stubClient, nil
	}
	t.Cleanup(func() { newFactoryBitbucketAPI = restore })

	response, err := DescribeFactoryPullRequestMergeability(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.DescribeFactoryPullRequestMergeabilityRequest{
		FactoryId: factory.ID.String(),
		PrId:      pullRequest.ID.String(),
	})
	require.NoError(t, err)
	require.True(t, response.GetMergeability().GetCanMerge())
	assert.Equal(t, "abc123", response.GetMergeability().GetHeadSha())
}

func TestMergeFactoryPullRequest_BitbucketFactory(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider: models.FactoryPullRequestProviderBitbucket,
		URL:      "https://bitbucket.org/acme/widgets/pull-requests/42",
		State:    models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	openPR := `{"id": 42, "state": "OPEN", "draft": false, "source": {"branch": {"name": "feat/x"}, "commit": {"hash": "abc123"}}, "destination": {"branch": {"name": "main"}}}`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(openPR))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"values": [{"key": "build-a", "state": "SUCCESSFUL"}]}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(openPR))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id": 42, "state": "MERGED"}`))},
		},
	}
	stubClient, err := bitbucketintegration.NewClient(
		bitbucketintegration.AuthTypeWorkspaceAccessToken,
		httpCtx,
		&contexts.IntegrationContext{Configuration: map[string]any{"token": "token"}},
	)
	require.NoError(t, err)
	restore := newFactoryBitbucketAPI
	newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucketintegration.Client, error) {
		return stubClient, nil
	}
	t.Cleanup(func() { newFactoryBitbucketAPI = restore })

	response, err := MergeFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.MergeFactoryPullRequestRequest{
		FactoryId:       factory.ID.String(),
		PrId:            pullRequest.ID.String(),
		MergeMethod:     pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH,
		ExpectedHeadSha: "abc123",
	})
	require.NoError(t, err)
	require.NotNil(t, response.GetPullRequest())

	require.Len(t, httpCtx.Requests, 4)
	mergeRequest := httpCtx.Requests[3]
	assert.Equal(t, http.MethodPost, mergeRequest.Method)
	assert.Equal(t, "/2.0/repositories/acme/widgets/pullrequests/42/merge", mergeRequest.URL.Path)

	stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	assert.Equal(t, models.FactoryPullRequestStateMerged, stored.State)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
	assert.Equal(t, models.FactoryWorkOrderResultCompleted, reloaded.Result)
}

func TestCreateBitbucketChecksHandlerWaitsForBuilds(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}

	factory := bitbucketFactory(t, r)
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, models.ProviderBitbucket)
	appRepo := "acme/widgets"
	provider := models.ProviderBitbucket
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSProvider:      &provider,
		VCSIntegrationID: &integrationID,
		AppRepository:    &appRepo,
	}))

	response, err := CreateFactoryPRFeedbackHandler(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryPRFeedbackHandlerRequest{
		FactoryId: factory.ID.String(),
		Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_CHECKS,
		Settings: &pb.FactoryPRFeedbackHandler_Settings{
			Checks: &pb.FactoryPRFeedbackHandler_CheckSettings{
				Names: []string{"build-a"},
			},
		},
	})
	require.NoError(t, err)
	handler := response.GetHandler()
	assert.True(t, handler.GetHealthy())
	assert.Equal(t, []string{"build-a"}, handler.GetSettings().GetChecks().GetNames())

	canvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
	require.NoError(t, err)
	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)

	var foundTrigger, foundWait bool
	for _, node := range liveVersion.Nodes {
		switch node.ID {
		case prFeedbackPullRequestTriggerNodeID:
			foundTrigger = true
			assert.Equal(t, "bitbucket.onPullRequest", node.ComponentName())
			assert.NotNil(t, node.IntegrationID)
		case prFeedbackWaitChecksNodeID:
			foundWait = true
			assert.Equal(t, "bitbucket.waitForBuilds", node.ComponentName())
			assert.Equal(t, []any{"build-a"}, node.Configuration["buildKeys"])
			_, hasCheckNames := node.Configuration["checkNames"]
			assert.False(t, hasCheckNames)
			assert.NotNil(t, node.IntegrationID)
		}
		assert.NotContains(t, node.ComponentName(), "github.")
	}
	assert.True(t, foundTrigger)
	assert.True(t, foundWait)
}

func TestCreateBitbucketDiscussionHandlerListensForComments(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	deps := IntakeDependencies{
		Registry:       r.Registry,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "http://localhost:8000",
	}

	factory := bitbucketFactory(t, r)
	integrationID := createReadyOnboardingIntegration(t, r.Organization.ID, models.ProviderBitbucket)
	appRepo := "acme/widgets"
	provider := models.ProviderBitbucket
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSProvider:      &provider,
		VCSIntegrationID: &integrationID,
		AppRepository:    &appRepo,
	}))

	response, err := CreateFactoryPRFeedbackHandler(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryPRFeedbackHandlerRequest{
		FactoryId: factory.ID.String(),
		Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_DISCUSSION,
	})
	require.NoError(t, err)
	handler := response.GetHandler()
	assert.True(t, handler.GetHealthy())

	canvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, uuid.MustParse(handler.GetCanvasId()))
	require.NoError(t, err)
	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)

	graph := resolvePRFeedbackGraph(models.LiveCanvasSpec{Nodes: liveVersion.Nodes, Edges: liveVersion.Edges})
	assert.Equal(t, prFeedbackCommentTriggerNodeID, graph.CommentTriggerNodeID)
	assert.True(t, graph.Healthy(models.LiveCanvasSpec{Nodes: liveVersion.Nodes, Edges: liveVersion.Edges}))

	for _, node := range liveVersion.Nodes {
		assert.NotContains(t, node.ComponentName(), "github.")
	}
}

func TestMergeFactoryPullRequest_BitbucketHeadMoved(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider: models.FactoryPullRequestProviderBitbucket,
		URL:      "https://bitbucket.org/acme/widgets/pull-requests/42",
		State:    models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "moved999"}}}`,
			))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"values": []}`))},
		},
	}
	stubClient, err := bitbucketintegration.NewClient(
		bitbucketintegration.AuthTypeWorkspaceAccessToken,
		httpCtx,
		&contexts.IntegrationContext{Configuration: map[string]any{"token": "token"}},
	)
	require.NoError(t, err)
	restore := newFactoryBitbucketAPI
	newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucketintegration.Client, error) {
		return stubClient, nil
	}
	t.Cleanup(func() { newFactoryBitbucketAPI = restore })

	_, err = MergeFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.MergeFactoryPullRequestRequest{
		FactoryId:       factory.ID.String(),
		PrId:            pullRequest.ID.String(),
		MergeMethod:     pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH,
		ExpectedHeadSha: "abc123",
	})
	code, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Contains(t, message, "head changed")
	require.Len(t, httpCtx.Requests, 2, "no merge attempt after the head moved")

	stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	assert.Equal(t, models.FactoryPullRequestStateOpen, stored.State)
}

func TestSendWorkOrderToBacklog_BitbucketFactoryDeclines(t *testing.T) {

	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Closed task", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultFailed,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider: models.FactoryPullRequestProviderBitbucket,
		URL:      "https://bitbucket.org/acme/widgets/pull-requests/42",
		State:    models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":42,"state":"DECLINED"}`)),
			},
		},
	}
	stubClient, err := bitbucketintegration.NewClient(
		bitbucketintegration.AuthTypeWorkspaceAccessToken,
		httpCtx,
		&contexts.IntegrationContext{Configuration: map[string]any{"token": "token"}},
	)
	require.NoError(t, err)
	restore := newFactoryBitbucketAPI
	newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucketintegration.Client, error) {
		return stubClient, nil
	}
	t.Cleanup(func() { newFactoryBitbucketAPI = restore })

	resp, err := SendWorkOrderToBacklog(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.SendWorkOrderToBacklogRequest{
		FactoryId:         factory.ID.String(),
		OrderId:           order.ID.String(),
		ClosePullRequests: true,
	})
	require.NoError(t, err)
	assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resp.GetOrder().GetState())

	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
	assert.Equal(t, "/2.0/repositories/acme/widgets/pullrequests/42/decline", httpCtx.Requests[0].URL.Path)

	grouped, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{order.ID})
	require.NoError(t, err)
	require.Len(t, grouped[order.ID], 1)
	assert.Equal(t, models.FactoryPullRequestStateClosed, grouped[order.ID][0].State)
}

func TestSendWorkOrderToBacklog_BitbucketFactoryRejectsGitHubPR(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Closed task", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultFailed,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:   "https://github.com/acme/app/pull/42",
		State: models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	_, err = SendWorkOrderToBacklog(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.SendWorkOrderToBacklogRequest{
		FactoryId:         factory.ID.String(),
		OrderId:           order.ID.String(),
		ClosePullRequests: true,
	})
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
}

func bitbucketFactory(t *testing.T, r *support.ResourceRegistry) *models.Factory {
	t.Helper()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	provider := models.ProviderBitbucket
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSProvider: &provider,
	}))
	return factory
}

func TestBitbucketProviderReadMergeability(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider: models.FactoryPullRequestProviderBitbucket,
		URL:      "https://bitbucket.org/acme/widgets/pull-requests/42",
		State:    models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	read := func(t *testing.T, responses ...*http.Response) vcs.Mergeability {
		t.Helper()
		httpCtx := &contexts.HTTPContext{Responses: responses}
		stubClient, err := bitbucketintegration.NewClient(
			bitbucketintegration.AuthTypeWorkspaceAccessToken,
			httpCtx,
			&contexts.IntegrationContext{Configuration: map[string]any{"token": "token"}},
		)
		require.NoError(t, err)
		restore := newFactoryBitbucketAPI
		newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucketintegration.Client, error) {
			return stubClient, nil
		}
		t.Cleanup(func() { newFactoryBitbucketAPI = restore })

		result, err := (&bitbucketProvider{db: db, factory: factory}).ReadMergeability(context.Background(), pullRequest)
		require.NoError(t, err)
		return result
	}

	t.Run("draft pull requests are blocked", func(t *testing.T) {
		result := read(t, okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": true, "source": {"commit": {"hash": "abc123"}}}`))
		assert.False(t, result.CanMerge)
		assert.Equal(t, "DRAFT", result.BlockedReason)
		assert.Equal(t, "abc123", result.HeadSHA)
	})

	t.Run("failed builds block", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}}`),
			okBitbucketResponse(`{"values": [{"key": "build-a", "state": "FAILED"}]}`),
		)
		assert.False(t, result.CanMerge)
		assert.Equal(t, "CHECK_FAILED", result.BlockedReason)
	})

	t.Run("unfinished builds block", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}}`),
			okBitbucketResponse(`{"values": [{"key": "build-a", "state": "INPROGRESS"}]}`),
		)
		assert.False(t, result.CanMerge)
		assert.Equal(t, "CHECKS_UNFINISHED", result.BlockedReason)
	})

	t.Run("empty builds do not block", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}}`),
			okBitbucketResponse(`{"values": []}`),
		)
		assert.True(t, result.CanMerge)
		assert.Equal(t, "abc123", result.HeadSHA)
		assert.Equal(t, []string{"SQUASH", "MERGE", "REBASE"}, result.AllowedMethods)
	})
}

func okBitbucketResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
}
