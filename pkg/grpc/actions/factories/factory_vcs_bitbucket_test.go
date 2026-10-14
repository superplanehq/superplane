package factories

import (
	"context"
	"encoding/json"
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
	"gorm.io/datatypes"
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
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id": 42, "state": "OPEN", "draft": false, "source": {"branch": {"name": "feat/x"}, "commit": {"hash": "abc123"}}, "destination": {"branch": {"name": "main", "merge_strategies": ["squash", "merge_commit"]}}}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"values": [
				{"type": "pullrequest_state_check", "status": "PASSED", "required": true, "blocking": false},
				{"type": "current_user_permission_check", "status": "PASSED", "required": true, "blocking": false},
				{"type": "git_mergeability_check", "status": "PASSED", "required": true, "blocking": false}
			]}`))},
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
	assert.Equal(t, []pb.FactoryPullRequestMergeability_MergeMethod{
		pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH,
		pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE,
	}, response.GetMergeability().GetAllowedMethods())
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

	openPR := `{"id": 42, "state": "OPEN", "draft": false, "source": {"branch": {"name": "feat/x"}, "commit": {"hash": "abc123"}}, "destination": {"branch": {"name": "main", "merge_strategies": ["squash", "merge_commit"]}}}`
	passingChecks := `{"values": [
		{"type": "pullrequest_state_check", "status": "PASSED", "required": true, "blocking": false},
		{"type": "current_user_permission_check", "status": "PASSED", "required": true, "blocking": false},
		{"type": "git_mergeability_check", "status": "PASSED", "required": true, "blocking": false}
	]}`
	passingBuilds := `{"values": [{"key": "build-a", "state": "SUCCESSFUL"}]}`
	httpCtx := &contexts.HTTPContext{
		ResponsesByPath: map[string][]*http.Response{
			"/2.0/repositories/acme/widgets/pullrequests/42": {
				okBitbucketResponse(openPR),
				okBitbucketResponse(openPR),
			},
			"/2.0/repositories/acme/widgets/pullrequests/42/mergeability/checks": {
				okBitbucketResponse(passingChecks),
				okBitbucketResponse(passingChecks),
			},
			"/2.0/repositories/acme/widgets/commit/abc123/statuses": {
				okBitbucketResponse(passingBuilds),
				okBitbucketResponse(passingBuilds),
			},
			"/2.0/repositories/acme/widgets/pullrequests/42/merge": {
				okBitbucketResponse(`{"id": 42, "state": "MERGED"}`),
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

	resp, err := MergeFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.MergeFactoryPullRequestRequest{
		FactoryId:       factory.ID.String(),
		PrId:            pullRequest.ID.String(),
		MergeMethod:     pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH,
		ExpectedHeadSha: "abc123",
	})
	require.NoError(t, err)
	assert.Equal(t, pb.FactoryPullRequest_STATE_MERGED, resp.GetPullRequest().GetState())

	mergeBody := bitbucketMergeRequestBody(t, httpCtx)
	assert.Equal(t, "squash", mergeBody["merge_strategy"])

	stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	assert.Equal(t, models.FactoryPullRequestStateMerged, stored.State)
	require.NotNil(t, stored.MergedAt)

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

func TestUpdateBitbucketDiscussionHandlerKeepsAuthorTitle(t *testing.T) {
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

	created, err := CreateFactoryPRFeedbackHandler(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryPRFeedbackHandlerRequest{
		FactoryId: factory.ID.String(),
		Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_DISCUSSION,
	})
	require.NoError(t, err)

	_, err = UpdateFactoryPRFeedbackHandler(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryPRFeedbackHandlerRequest{
		FactoryId: factory.ID.String(),
		HandlerId: created.GetHandler().GetId(),
		Settings: &pb.FactoryPRFeedbackHandler_Settings{
			Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{
				Repository: appRepo,
			},
			Discussion: &pb.FactoryPRFeedbackHandler_DiscussionSettings{
				Mention:    "@superplaneagent",
				IgnoreBots: true,
			},
		},
	})
	require.NoError(t, err)

	canvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, uuid.MustParse(created.GetHandler().GetCanvasId()))
	require.NoError(t, err)
	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)

	var title string
	for _, node := range liveVersion.Nodes {
		if node.ID != prFeedbackActivityNodeID {
			continue
		}
		title, _ = node.Configuration["title"].(string)
	}
	assert.Equal(t, prFeedbackBitbucketCommentActivityTitleExpression(), title)
	assert.NotContains(t, title, "comment.user.login")
}

func TestUpdateBitbucketDiscussionHandlerRefreshesRunnerCheckout(t *testing.T) {
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

	created, err := CreateFactoryPRFeedbackHandler(ctx, deps, r.Organization.ID.String(), &pb.CreateFactoryPRFeedbackHandlerRequest{
		FactoryId: factory.ID.String(),
		Source:    pb.FactoryPRFeedbackHandler_SOURCE_PULL_REQUEST_DISCUSSION,
	})
	require.NoError(t, err)

	// Simulate a canvas created before the short-hash checkout fix.
	canvas, err := models.FindCanvasInTransaction(db, r.Organization.ID, uuid.MustParse(created.GetHandler().GetCanvasId()))
	require.NoError(t, err)
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	require.NoError(t, err)
	nodes := append([]models.Node(nil), version.Nodes...)
	stale := false
	for i := range nodes {
		if nodes[i].ID != prFeedbackRunnerNodeID {
			continue
		}
		steps, ok := nodes[i].Configuration["steps"].([]any)
		require.True(t, ok)
		for _, raw := range steps {
			step, ok := raw.(map[string]any)
			require.True(t, ok)
			switch step["name"] {
			case "Checkout Pull Request":
				step["command"] = "echo stale-checkout"
				stale = true
			case "Commit and Push":
				step["command"] = "echo stale-push"
			}
		}
		nodes[i].Configuration["steps"] = steps
	}
	require.True(t, stale, "runner checkout step not found")
	require.NoError(t, db.Model(version).Update("nodes", datatypes.NewJSONSlice(nodes)).Error)

	_, err = UpdateFactoryPRFeedbackHandler(ctx, deps, r.Organization.ID.String(), &pb.UpdateFactoryPRFeedbackHandlerRequest{
		FactoryId: factory.ID.String(),
		HandlerId: created.GetHandler().GetId(),
		Settings: &pb.FactoryPRFeedbackHandler_Settings{
			Subject: &pb.FactoryPRFeedbackHandler_SubjectSettings{
				Repository: appRepo,
			},
			Discussion: &pb.FactoryPRFeedbackHandler_DiscussionSettings{
				Mention:    "@superplaneagent",
				IgnoreBots: true,
			},
		},
	})
	require.NoError(t, err)

	reloaded, err := models.FindCanvasInTransaction(db, r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	updated, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, reloaded)
	require.NoError(t, err)
	for _, node := range updated.Nodes {
		if node.ID != prFeedbackRunnerNodeID {
			continue
		}
		steps, ok := node.Configuration["steps"].([]any)
		require.True(t, ok)
		for _, raw := range steps {
			step := raw.(map[string]any)
			switch step["name"] {
			case "Checkout Pull Request":
				assert.Equal(t, bitbucketCheckoutCommand(), step["command"])
			case "Commit and Push":
				assert.Equal(t, bitbucketCommitPushCommand("fix: address PR #${PR_NUMBER} feedback"), step["command"])
			}
		}
		return
	}
	require.Fail(t, "runner node not found")
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
	assert.Equal(t, "The pull request head changed. Review the pull request and try again.", message)
	require.Len(t, httpCtx.Requests, 3, "no merge attempt after the head moved")
	for _, request := range httpCtx.Requests {
		assert.NotEqual(t, "/2.0/repositories/acme/widgets/pullrequests/42/merge", request.URL.Path)
	}

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
			okBitbucketResponse(`{"values": []}`),
			okBitbucketResponse(`{"values": [{"key": "build-a", "state": "FAILED"}]}`),
		)
		assert.False(t, result.CanMerge)
		assert.Equal(t, "CHECK_FAILED", result.BlockedReason)
	})

	t.Run("unfinished builds block", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}}`),
			okBitbucketResponse(`{"values": []}`),
			okBitbucketResponse(`{"values": [{"key": "build-a", "state": "INPROGRESS"}]}`),
		)
		assert.False(t, result.CanMerge)
		assert.Equal(t, "CHECKS_UNFINISHED", result.BlockedReason)
	})

	t.Run("mergeable pull requests offer squash and merge commit", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}, "destination": {"branch": {"name": "main", "merge_strategies": ["squash", "merge_commit", "fast_forward"]}}}`),
			okBitbucketResponse(`{"values": [
				{"type": "pullrequest_state_check", "status": "PASSED", "required": true, "blocking": false},
				{"type": "current_user_permission_check", "status": "PASSED", "required": true, "blocking": false},
				{"type": "git_mergeability_check", "status": "PASSED", "required": true, "blocking": false}
			]}`),
			okBitbucketResponse(`{"values": []}`),
		)
		assert.True(t, result.CanMerge)
		assert.Equal(t, "abc123", result.HeadSHA)
		assert.Equal(t, []string{"SQUASH", "MERGE"}, result.AllowedMethods)
	})

	t.Run("conflicting pull requests are blocked", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}}`),
			okBitbucketResponse(`{"values": [
				{"type": "git_mergeability_check", "status": "FAILED", "required": true, "blocking": true}
			]}`),
		)
		assert.False(t, result.CanMerge)
		assert.Equal(t, "CONFLICTING", result.BlockedReason)
		assert.Equal(t, mergeBlockedConflicting, result.Message)
	})

	t.Run("missing merge permission is blocked", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}}`),
			okBitbucketResponse(`{"values": [
				{"type": "current_user_permission_check", "status": "FAILED", "required": true, "blocking": true}
			]}`),
		)
		assert.False(t, result.CanMerge)
		assert.Equal(t, mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_UNSPECIFIED), result.BlockedReason)
		assert.Equal(t, bitbucketMergeBlockedPermission, result.Message)
	})

	t.Run("failing required checks block", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}}`),
			okBitbucketResponse(`{"values": [
				{"type": "standard_merge_check", "status": "FAILED", "required": true, "blocking": true},
				{"type": "custom_pre_merge_check", "status": "FAILED", "required": true, "blocking": false}
			]}`),
		)
		assert.False(t, result.CanMerge)
		assert.Equal(t, "CHECK_FAILED", result.BlockedReason)
		assert.Equal(t, mergeBlockedCheckFailed, result.Message)
	})

	t.Run("non-blocking failures stay mergeable", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}}`),
			okBitbucketResponse(`{"values": [
				{"type": "custom_pre_merge_check", "status": "FAILED", "required": true, "blocking": false}
			]}`),
			okBitbucketResponse(`{"values": []}`),
		)
		assert.True(t, result.CanMerge)
	})

	t.Run("fast-forward-only branches cannot merge from SuperPlane", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}, "destination": {"branch": {"name": "main", "merge_strategies": ["fast_forward"]}}}`),
			okBitbucketResponse(`{"values": []}`),
			okBitbucketResponse(`{"values": []}`),
		)
		assert.False(t, result.CanMerge)
		assert.Equal(t, mergeabilityBlockedReasonName(pb.FactoryPullRequestMergeability_BLOCKED_REASON_UNSPECIFIED), result.BlockedReason)
		assert.Equal(t, bitbucketMergeFastForwardOnly, result.Message)
		assert.Empty(t, result.AllowedMethods)
	})

	t.Run("api failures leave merging disabled", func(t *testing.T) {
		result := read(t,
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"commit": {"hash": "abc123"}}}`),
			&http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"error": {"message": "boom"}}`))},
		)
		assert.False(t, result.CanMerge)
		assert.Equal(t, "UNAVAILABLE", result.BlockedReason)
		assert.Equal(t, mergeBlockedUnavailable, result.Message)
	})

	t.Run("missing pull requests leave merging disabled", func(t *testing.T) {
		result := read(t,
			&http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error": {"message": "not found"}}`))},
		)
		assert.False(t, result.CanMerge)
		assert.Equal(t, "UNAVAILABLE", result.BlockedReason)
	})
}

func okBitbucketResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
}

func bitbucketOpenPR(hash string) string {
	return `{"id": 42, "state": "OPEN", "draft": false, "source": {"branch": {"name": "feat/x"}, "commit": {"hash": "` + hash + `"}}, "destination": {"branch": {"name": "main", "merge_strategies": ["squash", "merge_commit"]}}}`
}

func bitbucketPassingChecks() string {
	return `{"values": [
		{"type": "pullrequest_state_check", "status": "PASSED", "required": true, "blocking": false},
		{"type": "current_user_permission_check", "status": "PASSED", "required": true, "blocking": false},
		{"type": "git_mergeability_check", "status": "PASSED", "required": true, "blocking": false}
	]}`
}

func bitbucketPassingBuilds() string {
	return `{"values": [{"key": "build-a", "state": "SUCCESSFUL"}]}`
}

func stubBitbucketMergeAPI(t *testing.T, paths map[string][]*http.Response) *contexts.HTTPContext {
	t.Helper()
	httpCtx := &contexts.HTTPContext{ResponsesByPath: paths}
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
	return httpCtx
}

func bitbucketMergeReadyResponses(openPR string, rounds int) map[string][]*http.Response {
	prs, checks, builds := make([]*http.Response, 0, rounds), make([]*http.Response, 0, rounds), make([]*http.Response, 0, rounds)
	for range rounds {
		prs = append(prs, okBitbucketResponse(openPR))
		checks = append(checks, okBitbucketResponse(bitbucketPassingChecks()))
		builds = append(builds, okBitbucketResponse(bitbucketPassingBuilds()))
	}
	return map[string][]*http.Response{
		"/2.0/repositories/acme/widgets/pullrequests/42":                     prs,
		"/2.0/repositories/acme/widgets/pullrequests/42/mergeability/checks": checks,
		"/2.0/repositories/acme/widgets/commit/abc123/statuses":              builds,
	}
}

func openBitbucketMergeOrder(t *testing.T, factory *models.Factory, r *support.ResourceRegistry) (*models.FactoryWorkOrder, *models.FactoryPullRequest) {
	t.Helper()
	db := database.DB(t.Context())
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
	return order, pullRequest
}

func mergeBitbucketPR(
	ctx context.Context,
	t *testing.T,
	orgID string,
	factory *models.Factory,
	pullRequest *models.FactoryPullRequest,
	method pb.FactoryPullRequestMergeability_MergeMethod,
	sha string,
) (*pb.MergeFactoryPullRequestResponse, error) {
	t.Helper()
	return MergeFactoryPullRequest(ctx, IntakeDependencies{}, orgID, &pb.MergeFactoryPullRequestRequest{
		FactoryId:       factory.ID.String(),
		PrId:            pullRequest.ID.String(),
		MergeMethod:     method,
		ExpectedHeadSha: sha,
	})
}

func bitbucketMergeRequestBody(t *testing.T, httpCtx *contexts.HTTPContext) map[string]any {
	t.Helper()
	for _, request := range httpCtx.Requests {
		if request.Method == http.MethodPost && request.URL.Path == "/2.0/repositories/acme/widgets/pullrequests/42/merge" {
			body, err := io.ReadAll(request.Body)
			require.NoError(t, err)
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(body, &decoded))
			return decoded
		}
	}
	t.Fatal("no bitbucket merge request recorded")
	return nil
}

func assertNoBitbucketMerge(t *testing.T, httpCtx *contexts.HTTPContext) {
	t.Helper()
	for _, request := range httpCtx.Requests {
		assert.NotEqual(t, "/2.0/repositories/acme/widgets/pullrequests/42/merge", request.URL.Path)
	}
}

func TestMergeFactoryPullRequest_BitbucketMergeCommit(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	factory := bitbucketFactory(t, r)
	order, pullRequest := openBitbucketMergeOrder(t, factory, r)
	paths := bitbucketMergeReadyResponses(bitbucketOpenPR("abc123"), 2)
	paths["/2.0/repositories/acme/widgets/pullrequests/42/merge"] = []*http.Response{
		okBitbucketResponse(`{"id": 42, "state": "MERGED"}`),
	}
	httpCtx := stubBitbucketMergeAPI(t, paths)

	resp, err := mergeBitbucketPR(ctx, t, r.Organization.ID.String(), factory, pullRequest, pb.FactoryPullRequestMergeability_MERGE_METHOD_MERGE, "abc123")
	require.NoError(t, err)
	assert.Equal(t, pb.FactoryPullRequest_STATE_MERGED, resp.GetPullRequest().GetState())
	assert.Equal(t, "merge_commit", bitbucketMergeRequestBody(t, httpCtx)["merge_strategy"])

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
}

func TestMergeFactoryPullRequest_BitbucketRejectsRebase(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	factory := bitbucketFactory(t, r)
	_, pullRequest := openBitbucketMergeOrder(t, factory, r)
	httpCtx := stubBitbucketMergeAPI(t, bitbucketMergeReadyResponses(bitbucketOpenPR("abc123"), 1))

	_, err := mergeBitbucketPR(ctx, t, r.Organization.ID.String(), factory, pullRequest, pb.FactoryPullRequestMergeability_MERGE_METHOD_REBASE, "abc123")
	code, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Equal(t, "The repository does not allow this merge method.", message)
	assertNoBitbucketMerge(t, httpCtx)
}

func TestMergeFactoryPullRequest_BitbucketBlockedChecks(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	blocked := map[string]string{
		"conflicts":   `{"values": [{"type": "git_mergeability_check", "status": "FAILED", "required": true, "blocking": true}]}`,
		"permissions": `{"values": [{"type": "current_user_permission_check", "status": "FAILED", "required": true, "blocking": true}]}`,
		"checks":      `{"values": [{"type": "standard_merge_check", "status": "FAILED", "required": true, "blocking": true}]}`,
	}
	messages := map[string]string{
		"conflicts":   mergeBlockedConflicting,
		"permissions": bitbucketMergeBlockedPermission,
		"checks":      mergeBlockedCheckFailed,
	}

	for name, checks := range blocked {
		t.Run(name, func(t *testing.T) {
			factory := bitbucketFactory(t, r)
			order, pullRequest := openBitbucketMergeOrder(t, factory, r)
			httpCtx := stubBitbucketMergeAPI(t, map[string][]*http.Response{
				"/2.0/repositories/acme/widgets/pullrequests/42": {
					okBitbucketResponse(bitbucketOpenPR("abc123")),
				},
				"/2.0/repositories/acme/widgets/pullrequests/42/mergeability/checks": {
					okBitbucketResponse(checks),
				},
			})

			_, err := mergeBitbucketPR(ctx, t, r.Organization.ID.String(), factory, pullRequest, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, "abc123")
			code, message, ok := grpcerrors.HandlerStatus(err)
			require.True(t, ok)
			assert.Equal(t, codes.FailedPrecondition, code)
			assert.Equal(t, messages[name], message)
			assertNoBitbucketMerge(t, httpCtx)

			stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
			require.NoError(t, err)
			assert.Equal(t, models.FactoryPullRequestStateOpen, stored.State)

			reloaded, err := factory.FindWorkOrder(db, order.ID)
			require.NoError(t, err)
			assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
		})
	}
}

func TestMergeFactoryPullRequest_BitbucketUnavailable(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	factory := bitbucketFactory(t, r)
	_, pullRequest := openBitbucketMergeOrder(t, factory, r)
	httpCtx := stubBitbucketMergeAPI(t, map[string][]*http.Response{
		"/2.0/repositories/acme/widgets/pullrequests/42": {
			okBitbucketResponse(bitbucketOpenPR("abc123")),
		},
		"/2.0/repositories/acme/widgets/pullrequests/42/mergeability/checks": {
			{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"error": {"message": "boom"}}`))},
		},
	})

	describe, err := DescribeFactoryPullRequestMergeability(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.DescribeFactoryPullRequestMergeabilityRequest{
		FactoryId: factory.ID.String(),
		PrId:      pullRequest.ID.String(),
	})
	require.NoError(t, err)
	assert.False(t, describe.GetMergeability().GetCanMerge())
	assert.Equal(t, pb.FactoryPullRequestMergeability_BLOCKED_REASON_UNAVAILABLE, describe.GetMergeability().GetBlockedReason())

	_, err = mergeBitbucketPR(ctx, t, r.Organization.ID.String(), factory, pullRequest, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, "abc123")
	code, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assert.Equal(t, mergeBlockedUnavailable, message)
	assertNoBitbucketMerge(t, httpCtx)
}

func TestMergeFactoryPullRequest_BitbucketMissingHead(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	factory := bitbucketFactory(t, r)
	_, pullRequest := openBitbucketMergeOrder(t, factory, r)
	httpCtx := stubBitbucketMergeAPI(t, map[string][]*http.Response{
		"/2.0/repositories/acme/widgets/pullrequests/42": {
			okBitbucketResponse(`{"id": 42, "state": "OPEN", "draft": false, "source": {"branch": {"name": "feat/x"}, "commit": {}}}`),
		},
	})

	_, err := mergeBitbucketPR(ctx, t, r.Organization.ID.String(), factory, pullRequest, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, "abc123")
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
	assertNoBitbucketMerge(t, httpCtx)
}

func TestMergeFactoryPullRequest_BitbucketMergeFailureKeepsTaskOpen(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	factory := bitbucketFactory(t, r)
	order, pullRequest := openBitbucketMergeOrder(t, factory, r)
	paths := bitbucketMergeReadyResponses(bitbucketOpenPR("abc123"), 3)
	paths["/2.0/repositories/acme/widgets/pullrequests/42/merge"] = []*http.Response{
		{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error": {"message": "rejected"}}`))},
	}
	httpCtx := stubBitbucketMergeAPI(t, paths)

	_, err := mergeBitbucketPR(ctx, t, r.Organization.ID.String(), factory, pullRequest, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, "abc123")
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)

	stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	assert.Equal(t, models.FactoryPullRequestStateOpen, stored.State)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)

	prReads := 0
	for _, request := range httpCtx.Requests {
		if request.Method == http.MethodGet && request.URL.Path == "/2.0/repositories/acme/widgets/pullrequests/42" {
			prReads++
		}
	}
	assert.Equal(t, 3, prReads, "a rejected merge refreshes mergeability")
}

func TestMergeFactoryPullRequest_BitbucketUnconfirmedMergeKeepsTaskOpen(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	factory := bitbucketFactory(t, r)
	order, pullRequest := openBitbucketMergeOrder(t, factory, r)
	paths := bitbucketMergeReadyResponses(bitbucketOpenPR("abc123"), 3)
	paths["/2.0/repositories/acme/widgets/pullrequests/42/merge"] = []*http.Response{
		okBitbucketResponse(`{"id": 42, "state": "OPEN"}`),
	}
	stubBitbucketMergeAPI(t, paths)

	_, err := mergeBitbucketPR(ctx, t, r.Organization.ID.String(), factory, pullRequest, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, "abc123")
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)

	stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	assert.Equal(t, models.FactoryPullRequestStateOpen, stored.State)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
}

func TestMergeFactoryPullRequest_BitbucketIgnoresCachedBlock(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	factory := bitbucketFactory(t, r)
	_, pullRequest := openBitbucketMergeOrder(t, factory, r)
	stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	require.NoError(t, stored.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{
		Mergeable:      false,
		BlockedReason:  "CONFLICTING",
		BlockedMessage: mergeBlockedConflicting,
		HeadSHA:        "abc123",
		AllowedMethods: "SQUASH,MERGE",
	}))

	paths := bitbucketMergeReadyResponses(bitbucketOpenPR("abc123"), 2)
	paths["/2.0/repositories/acme/widgets/pullrequests/42/merge"] = []*http.Response{
		okBitbucketResponse(`{"id": 42, "state": "MERGED"}`),
	}
	stubBitbucketMergeAPI(t, paths)

	resp, err := mergeBitbucketPR(ctx, t, r.Organization.ID.String(), factory, pullRequest, pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH, "abc123")
	require.NoError(t, err)
	assert.Equal(t, pb.FactoryPullRequest_STATE_MERGED, resp.GetPullRequest().GetState())
}

func TestBitbucketProviderReadMergeability_MissingIntegration(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider: models.FactoryPullRequestProviderBitbucket,
		URL:      "https://bitbucket.org/acme/widgets/pull-requests/42",
		State:    models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	restore := newFactoryBitbucketAPI
	newFactoryBitbucketAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (*bitbucketintegration.Client, error) {
		return nil, errFactoryBitbucketNotConnected
	}
	t.Cleanup(func() { newFactoryBitbucketAPI = restore })

	result, err := (&bitbucketProvider{db: db, factory: factory}).ReadMergeability(context.Background(), pullRequest)
	require.NoError(t, err)
	assert.False(t, result.CanMerge)
	assert.Equal(t, "MISSING_INTEGRATION", result.BlockedReason)
	assert.Equal(t, bitbucketMergeBlockedMissingIntegration, result.Message)
}
