package factories

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__HandOffWorkOrder__ImplementWritesSpecAndStartsImplementation(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel := seedHandOffFactory(t, r, db)
	require.NoError(t, factoryModel.UpdatePlanning(db, models.FactoryPlanning{Enabled: true, Clarity: true, Confidence: true}))
	backlog := createOnWorkOrderCanvas(t, r, factoryModel.ID)

	planApp, planEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "plan", "start-plan")
	implApp, implEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	require.NoError(t, implApp.StampFactoryAppTemplate(db, implEntry, "line-implementation", 1))
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: planApp.ID, Entrypoint: planEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: implApp.ID, Entrypoint: implEntry},
	})
	require.NoError(t, err)

	result, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
		FactoryID:     factoryModel.ID.String(),
		Title:         "Ship refund retries",
		Description:   "Stop double charges.",
		Plan:          "# Retry refunds\n\nStop double charges.",
		Column:        HandOffColumnImplement,
		MCPClientID:   "superplane-local",
		MCPClientName: "Cursor",
	})
	require.NoError(t, err)
	require.NotNil(t, result.Order)
	assert.Equal(t, HandOffColumnImplement, result.Column)
	assert.Equal(t, pb.WorkOrder_STATE_OPEN, result.Order.GetState())
	assert.Equal(t, "Cursor", result.Order.GetMcpClient().GetName())
	assert.Equal(t, "superplane-local", result.Order.GetMcpClient().GetId())

	order, err := factoryModel.FindWorkOrder(db, uuid.MustParse(result.Order.GetId()))
	require.NoError(t, err)
	artifact, err := order.FindArtifactByKey(db, models.PlanningSpecArtifactKey+":"+order.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "# Retry refunds\n\nStop double charges.", planningSpecBody(t, artifact))

	var backlogEvents []models.CanvasEvent
	require.NoError(t, db.Where("workflow_id = ?", backlog.ID).Find(&backlogEvents).Error)
	assert.Empty(t, backlogEvents)

	require.Len(t, result.Order.GetLineDispatches(), 1)
	dispatch := result.Order.GetLineDispatches()[0]
	assert.Equal(t, line.Name, dispatch.GetLine().GetName())
	require.Len(t, dispatch.GetStepExecutions(), 1)
	assert.Equal(t, int32(1), dispatch.GetStepExecutions()[0].GetStepIndex())
	assert.Equal(t, pb.WorkOrderExecution_STATE_PENDING, dispatch.GetStepExecutions()[0].GetState())
	assert.NotEmpty(t, dispatch.GetStepExecutions()[0].GetRun().GetId())

	var planRuns int64
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("workflow_id = ?", planApp.ID).Count(&planRuns).Error)
	assert.Zero(t, planRuns)
	var implRuns int64
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("workflow_id = ?", implApp.ID).Count(&implRuns).Error)
	assert.EqualValues(t, 1, implRuns)
}

func Test__HandOffWorkOrder__ImplementStartsCustomStageWhenDoneFollows(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel := seedHandOffFactory(t, r, db)
	customApp, customEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "custom", "start-custom")
	doneApp, doneEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "done", "start-done")
	require.NoError(t, doneApp.StampFactoryAppTemplate(db, doneEntry, "pr-closure", 1))
	renameCanvas(t, db, doneApp, "Done")
	_, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: customApp.ID, Entrypoint: customEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: doneApp.ID, Entrypoint: doneEntry},
	})
	require.NoError(t, err)

	result, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
		FactoryID: factoryModel.ID.String(),
		Title:     "Ship custom implement",
		Plan:      "# Retry refunds",
		Column:    HandOffColumnImplement,
	})
	require.NoError(t, err)
	require.Len(t, result.Order.GetLineDispatches(), 1)
	require.Len(t, result.Order.GetLineDispatches()[0].GetStepExecutions(), 1)
	assert.Equal(t, int32(0), result.Order.GetLineDispatches()[0].GetStepExecutions()[0].GetStepIndex())
	assert.Equal(t, pb.WorkOrderExecution_STATE_PENDING, result.Order.GetLineDispatches()[0].GetStepExecutions()[0].GetState())
}

func Test__HandOffWorkOrder__VerifyWritesRecordsWithoutARun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel := seedHandOffFactory(t, r, db)
	planApp, planEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "plan", "start-plan")
	implApp, implEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	doneApp, doneEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "done", "start-done")
	require.NoError(t, implApp.StampFactoryAppTemplate(db, implEntry, "line-implementation", 1))
	require.NoError(t, doneApp.StampFactoryAppTemplate(db, doneEntry, "pr-closure", 1))
	renameCanvas(t, db, doneApp, "Done")
	_, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: planApp.ID, Entrypoint: planEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: implApp.ID, Entrypoint: implEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: doneApp.ID, Entrypoint: doneEntry},
	})
	require.NoError(t, err)

	checksCanvas := createChecksStartupCanvas(t, r, factoryModel, "acme/app")
	olderPR := checksStartupGitHubPullRequest()
	olderPR.CreatedAt = &github.Timestamp{Time: time.Now().Add(-20 * time.Minute)}
	useGitHub(t, olderPR)

	result, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
		FactoryID:      factoryModel.ID.String(),
		Title:          "Review refund retries",
		Plan:           "# Retry refunds\n\nStop double charges.",
		Column:         HandOffColumnVerify,
		PullRequestURL: "https://github.com/acme/app/pull/42",
		MCPClientName:  "Claude Code",
	})
	require.NoError(t, err)
	assert.Equal(t, HandOffColumnVerify, result.Column)
	assert.Equal(t, "https://github.com/acme/app/pull/42", result.PullRequestURL)
	assert.Equal(t, pb.WorkOrder_STATE_OPEN, result.Order.GetState())
	assert.Equal(t, "Claude Code", result.Order.GetMcpClient().GetName())
	require.Len(t, result.Order.GetPullRequests(), 1)
	assert.Equal(t, int64(42), result.Order.GetPullRequests()[0].GetNumber())

	order, err := factoryModel.FindWorkOrder(db, uuid.MustParse(result.Order.GetId()))
	require.NoError(t, err)
	_, err = order.FindArtifactByKey(db, models.PlanningSpecArtifactKey+":"+order.ID.String())
	require.NoError(t, err)
	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.True(t, hasBranchArtifact(artifacts, "feat/refunds", "acme/app"))

	require.Len(t, result.Order.GetLineDispatches(), 1)
	dispatch := result.Order.GetLineDispatches()[0]
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_FINISHED, dispatch.GetState())
	assert.Equal(t, pb.WorkOrderLineDispatch_RESULT_PASSED, dispatch.GetResult())
	require.Len(t, dispatch.GetStepExecutions(), 1)
	execution := dispatch.GetStepExecutions()[0]
	assert.Equal(t, int32(1), execution.GetStepIndex())
	assert.Equal(t, pb.WorkOrderExecution_STATE_FINISHED, execution.GetState())
	assert.Equal(t, pb.WorkOrderExecution_RESULT_PASSED, execution.GetResult())
	assert.Nil(t, execution.GetRun())

	var runCount int64
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("workflow_id IN ?", []any{planApp.ID, implApp.ID, doneApp.ID}).Count(&runCount).Error)
	assert.Zero(t, runCount)
	var requests int64
	require.NoError(t, db.Model(&models.CanvasNodeRequest{}).Where("workflow_id = ? AND type = ?", checksCanvas.ID, "pull-request-checks-start").Count(&requests).Error)
	assert.EqualValues(t, 1, requests, "Verify handoff must durably start checks without another PR webhook")
	assert.EqualValues(t, 1, countCanvasEvents(t, checksCanvas.ID))
}

func Test__HandOffWorkOrder__VerifyUsesBoardLastStageWhenClosureIsRenamed(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel := seedHandOffFactory(t, r, db)
	planApp, planEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "plan", "start-plan")
	implApp, implEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	closureApp, closureEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "close", "start-close")
	require.NoError(t, implApp.StampFactoryAppTemplate(db, implEntry, "line-implementation", 1))
	require.NoError(t, closureApp.StampFactoryAppTemplate(db, closureEntry, "pr-closure", 1))
	renameCanvas(t, db, closureApp, "Close work")
	_, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: planApp.ID, Entrypoint: planEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: implApp.ID, Entrypoint: implEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: closureApp.ID, Entrypoint: closureEntry},
	})
	require.NoError(t, err)

	useGitHub(t, openHandOffPullRequest("acme/app", 42, "feat/refunds"))

	result, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
		FactoryID:      factoryModel.ID.String(),
		Title:          "Review renamed closure",
		Plan:           "# Retry refunds",
		Column:         HandOffColumnVerify,
		PullRequestURL: "https://github.com/acme/app/pull/42",
	})
	require.NoError(t, err)
	require.Len(t, result.Order.GetLineDispatches(), 1)
	require.Len(t, result.Order.GetLineDispatches()[0].GetStepExecutions(), 1)
	assert.Equal(t, int32(2), result.Order.GetLineDispatches()[0].GetStepExecutions()[0].GetStepIndex())
}

func Test__HandOffWorkOrder__VerifyStoresForkHeadRepository(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel := seedHandOffFactory(t, r, db)
	implApp, implEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	require.NoError(t, implApp.StampFactoryAppTemplate(db, implEntry, "line-implementation", 1))
	_, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: implApp.ID, Entrypoint: implEntry},
	})
	require.NoError(t, err)

	useGitHub(t, forkHandOffPullRequest("acme/app", "contributor/app", 42, "feat/refunds"))

	result, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
		FactoryID:      factoryModel.ID.String(),
		Title:          "Review fork pull request",
		Plan:           "# Retry refunds",
		Column:         HandOffColumnVerify,
		PullRequestURL: "https://github.com/acme/app/pull/42",
	})
	require.NoError(t, err)
	require.Len(t, result.Order.GetPullRequests(), 1)
	assert.Equal(t, "acme/app", result.Order.GetPullRequests()[0].GetRepository())

	order, err := factoryModel.FindWorkOrder(db, uuid.MustParse(result.Order.GetId()))
	require.NoError(t, err)
	artifacts, err := order.ListArtifacts(db)
	require.NoError(t, err)
	require.True(t, hasBranchArtifact(artifacts, "feat/refunds", "contributor/app"))
}

func Test__HandOffWorkOrder__VerifyAcceptsBitbucketPullRequest(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel := seedBitbucketHandOffFactory(t, r, db)
	implApp, implEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	doneApp, doneEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "done", "start-done")
	require.NoError(t, implApp.StampFactoryAppTemplate(db, implEntry, "line-implementation", 1))
	require.NoError(t, doneApp.StampFactoryAppTemplate(db, doneEntry, "pr-closure", 1))
	renameCanvas(t, db, doneApp, "Done")
	_, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: implApp.ID, Entrypoint: implEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: doneApp.ID, Entrypoint: doneEntry},
	})
	require.NoError(t, err)

	handOff := func(t *testing.T, number int, body, rawURL string) *HandOffWorkOrderResult {
		t.Helper()
		stubBitbucketMergeAPI(t, map[string][]*http.Response{
			"/2.0/repositories/acme/app/pullrequests/" + strconv.Itoa(number): {okBitbucketResponse(body)},
		})
		result, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
			FactoryID:      factoryModel.ID.String(),
			Title:          "Review refund retries",
			Plan:           "# Retry refunds",
			Column:         HandOffColumnVerify,
			PullRequestURL: rawURL,
		})
		require.NoError(t, err)
		return result
	}

	t.Run("records the pull request and places the task in verify", func(t *testing.T) {
		prior, err := factoryModel.CreateWorkOrder(db, "Old repository", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = prior.CreatePullRequest(db, models.FactoryPullRequestParams{
			Provider:   models.FactoryPullRequestProviderBitbucket,
			ExternalID: "42",
			Repository: "acme/old",
			Number:     42,
			URL:        "https://bitbucket.org/acme/old/pull-requests/42",
			State:      models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		result := handOff(t, 42, `{
			"id": 42,
			"title": "Ready",
			"state": "OPEN",
			"links": {"html": {"href": "https://bitbucket.org/acme/app/pull-requests/42"}},
			"source": {"branch": {"name": "feat/refunds"}, "repository": {"full_name": "contributor/app"}}
		}`, "https://bitbucket.org/acme/app/pull-requests/42?w=1")

		assert.Equal(t, HandOffColumnVerify, result.Column)
		assert.Equal(t, "https://bitbucket.org/acme/app/pull-requests/42", result.PullRequestURL)
		assert.Equal(t, pb.WorkOrder_STATE_OPEN, result.Order.GetState())
		require.Len(t, result.Order.GetPullRequests(), 1)
		recorded := result.Order.GetPullRequests()[0]
		assert.Equal(t, pb.FactoryPullRequest_PROVIDER_BITBUCKET, recorded.GetProvider())
		assert.Empty(t, recorded.GetExternalId())
		assert.Equal(t, int64(42), recorded.GetNumber())
		assert.Equal(t, "acme/app", recorded.GetRepository())

		order, err := factoryModel.FindWorkOrder(db, uuid.MustParse(result.Order.GetId()))
		require.NoError(t, err)
		grouped, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{order.ID})
		require.NoError(t, err)
		require.Len(t, grouped[order.ID], 1)
		assert.Equal(t, models.FactoryPullRequestProviderBitbucket, grouped[order.ID][0].Provider)

		artifacts, err := order.ListArtifacts(db)
		require.NoError(t, err)
		require.True(t, hasBranchArtifact(artifacts, "feat/refunds", "contributor/app"))

		require.Len(t, result.Order.GetLineDispatches(), 1)
		dispatch := result.Order.GetLineDispatches()[0]
		assert.Equal(t, pb.WorkOrderLineDispatch_STATE_FINISHED, dispatch.GetState())
		assert.Equal(t, pb.WorkOrderLineDispatch_RESULT_PASSED, dispatch.GetResult())
		require.Len(t, dispatch.GetStepExecutions(), 1)
		assert.Equal(t, int32(0), dispatch.GetStepExecutions()[0].GetStepIndex())
		assert.Nil(t, dispatch.GetStepExecutions()[0].GetRun())

		var runCount int64
		require.NoError(t, db.Model(&models.CanvasRun{}).Where("workflow_id IN ?", []any{implApp.ID, doneApp.ID}).Count(&runCount).Error)
		assert.Zero(t, runCount)
	})

	t.Run("uses the caller URL and task repository when Bitbucket omits them", func(t *testing.T) {
		result := handOff(t, 43, `{
			"id": 43,
			"state": "OPEN",
			"source": {"branch": {"name": "feat/refunds"}}
		}`, "https://bitbucket.org/acme/app/pull-requests/43/")

		assert.Equal(t, "https://bitbucket.org/acme/app/pull-requests/43", result.PullRequestURL)
		order, err := factoryModel.FindWorkOrder(db, uuid.MustParse(result.Order.GetId()))
		require.NoError(t, err)
		artifacts, err := order.ListArtifacts(db)
		require.NoError(t, err)
		require.True(t, hasBranchArtifact(artifacts, "feat/refunds", "acme/app"))
	})
}

func Test__HandOffWorkOrder__VerifyRejectsUnsupportedPullRequestURL(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	githubFactory := seedHandOffFactory(t, r, db)
	addHandOffLine(t, r, db, githubFactory)
	bitbucketFactory := seedBitbucketHandOffFactory(t, r, db)
	addHandOffLine(t, r, db, bitbucketFactory)

	reject := func(t *testing.T, factoryID, rawURL string) error {
		t.Helper()
		_, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
			FactoryID:      factoryID,
			Title:          "Ship it",
			Plan:           "# Plan",
			Column:         HandOffColumnVerify,
			PullRequestURL: rawURL,
		})
		require.Error(t, err)
		return err
	}

	t.Run("bitbucket url on a github workspace", func(t *testing.T) {
		err := reject(t, githubFactory.ID.String(), "https://bitbucket.org/acme/app/pull-requests/42")
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assertNamesSupportedPullRequestURL(t, err)
	})

	t.Run("github url on a bitbucket workspace", func(t *testing.T) {
		err := reject(t, bitbucketFactory.ID.String(), "https://github.com/acme/app/pull/42")
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assertNamesSupportedPullRequestURL(t, err)
	})

	t.Run("malformed url", func(t *testing.T) {
		err := reject(t, bitbucketFactory.ID.String(), "https://bitbucket.org/acme/app/pull/42")
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assertNamesSupportedPullRequestURL(t, err)
	})

	t.Run("bitbucket pull request in another repository", func(t *testing.T) {
		stubBitbucketMergeAPI(t, bitbucketHandOffPRResponse("other/repo", 9, "OPEN", "feat/other"))
		err := reject(t, bitbucketFactory.ID.String(), "https://bitbucket.org/other/repo/pull-requests/9")
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "does not match the task repository")
	})

	t.Run("declined bitbucket pull request", func(t *testing.T) {
		stubBitbucketMergeAPI(t, bitbucketHandOffPRResponse("acme/app", 7, "DECLINED", "feat/closed"))
		err := reject(t, bitbucketFactory.ID.String(), "https://bitbucket.org/acme/app/pull-requests/7")
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "not open")
	})

	t.Run("merged bitbucket pull request", func(t *testing.T) {
		stubBitbucketMergeAPI(t, bitbucketHandOffPRResponse("acme/app", 8, "MERGED", "feat/merged"))
		err := reject(t, bitbucketFactory.ID.String(), "https://bitbucket.org/acme/app/pull-requests/8")
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "not open")
	})
}

func Test__HandOffWorkOrder__ImplementFailedDispatchLeavesNoTask(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel := seedHandOffFactory(t, r, db)
	implApp, implEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	require.NoError(t, implApp.StampFactoryAppTemplate(db, implEntry, "line-implementation", 1))
	_, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: implApp.ID, Entrypoint: "missing-node"},
	})
	require.NoError(t, err)

	_, err = HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
		FactoryID: factoryModel.ID.String(),
		Title:     "Ship refund retries",
		Plan:      "# Retry refunds",
		Column:    HandOffColumnImplement,
	})
	require.Error(t, err)

	orders, err := factoryModel.ListWorkOrders(db, models.ListFactoryWorkOrdersFilters{})
	require.NoError(t, err)
	assert.Empty(t, orders)
}

func Test__HandOffWorkOrder__RejectsInvalidInput(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel := seedHandOffFactory(t, r, db)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	_, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	})
	require.NoError(t, err)

	t.Run("missing plan", func(t *testing.T) {
		_, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
			FactoryID: factoryModel.ID.String(),
			Title:     "Ship it",
			Column:    HandOffColumnImplement,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "plan is required")
	})

	t.Run("verify without pull request URL", func(t *testing.T) {
		_, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
			FactoryID: factoryModel.ID.String(),
			Title:     "Ship it",
			Plan:      "# Plan",
			Column:    HandOffColumnVerify,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "pull_request_url is required")
	})

	t.Run("pull request in another repository", func(t *testing.T) {
		useGitHub(t, openHandOffPullRequest("other/repo", 9, "feat/other"))
		_, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
			FactoryID:      factoryModel.ID.String(),
			Title:          "Ship it",
			Plan:           "# Plan",
			Column:         HandOffColumnVerify,
			PullRequestURL: "https://github.com/other/repo/pull/9",
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "does not match the task repository")
	})

	t.Run("closed pull request", func(t *testing.T) {
		pr := openHandOffPullRequest("acme/app", 7, "feat/closed")
		pr.State = github.Ptr("closed")
		useGitHub(t, pr)
		_, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
			FactoryID:      factoryModel.ID.String(),
			Title:          "Ship it",
			Plan:           "# Plan",
			Column:         HandOffColumnVerify,
			PullRequestURL: "https://github.com/acme/app/pull/7",
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "not open")
	})

	t.Run("ambiguous implementation stages", func(t *testing.T) {
		otherFactory := seedHandOffFactory(t, r, db)
		first, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, otherFactory.ID, "alpha", "start-a")
		second, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, otherFactory.ID, "beta", "start-b")
		_, err := otherFactory.CreateLine(db, "ship", []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: first.ID, Entrypoint: firstEntry},
			{Type: models.FactoryLineStepTypeRunApp, AppID: second.ID, Entrypoint: secondEntry},
		})
		require.NoError(t, err)
		_, err = HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
			FactoryID: otherFactory.ID.String(),
			Title:     "Ship it",
			Plan:      "# Plan",
			Column:    HandOffColumnImplement,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "Cannot choose an implementation stage")
		assert.Contains(t, grpcerrors.StatusMessage(err), first.Name)
		assert.Contains(t, grpcerrors.StatusMessage(err), second.Name)
	})

	t.Run("one-step done line", func(t *testing.T) {
		otherFactory := seedHandOffFactory(t, r, db)
		doneApp, doneEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, otherFactory.ID, "done", "start-done")
		require.NoError(t, doneApp.StampFactoryAppTemplate(db, doneEntry, "pr-closure", 1))
		renameCanvas(t, db, doneApp, "Done")
		_, err := otherFactory.CreateLine(db, "ship", []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: doneApp.ID, Entrypoint: doneEntry},
		})
		require.NoError(t, err)
		_, err = HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
			FactoryID: otherFactory.ID.String(),
			Title:     "Ship it",
			Plan:      "# Plan",
			Column:    HandOffColumnImplement,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "Cannot choose an implementation stage")
	})

	t.Run("planning and closure without implementation", func(t *testing.T) {
		otherFactory := seedHandOffFactory(t, r, db)
		planApp, planEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, otherFactory.ID, "plan", "start-plan")
		closureApp, closureEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, otherFactory.ID, "close", "start-close")
		require.NoError(t, closureApp.StampFactoryAppTemplate(db, closureEntry, "pr-closure", 1))
		renameCanvas(t, db, closureApp, "Close work")
		_, err := otherFactory.CreateLine(db, "ship", []models.FactoryLineStep{
			{Type: models.FactoryLineStepTypeRunApp, AppID: planApp.ID, Entrypoint: planEntry},
			{Type: models.FactoryLineStepTypeRunApp, AppID: closureApp.ID, Entrypoint: closureEntry},
		})
		require.NoError(t, err)
		_, err = HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
			FactoryID: otherFactory.ID.String(),
			Title:     "Ship it",
			Plan:      "# Plan",
			Column:    HandOffColumnImplement,
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "Cannot choose an implementation stage")
		assert.Contains(t, grpcerrors.StatusMessage(err), planApp.Name)
		assert.Contains(t, grpcerrors.StatusMessage(err), "Close work")
	})
}

func seedBitbucketHandOffFactory(t *testing.T, r *support.ResourceRegistry, db *gorm.DB) *models.Factory {
	t.Helper()
	factoryModel := seedHandOffFactory(t, r, db)
	provider := models.ProviderBitbucket
	require.NoError(t, factoryModel.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSProvider: &provider}))
	return factoryModel
}

func addHandOffLine(t *testing.T, r *support.ResourceRegistry, db *gorm.DB, factoryModel *models.Factory) {
	t.Helper()
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	_, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	})
	require.NoError(t, err)
}

func bitbucketHandOffPRResponse(repository string, number int, state, branch string) map[string][]*http.Response {
	body := `{"id": ` + strconv.Itoa(number) + `, "state": "` + state + `", "source": {"branch": {"name": "` + branch + `"}}}`
	path := "/2.0/repositories/" + repository + "/pullrequests/" + strconv.Itoa(number)
	return map[string][]*http.Response{path: {okBitbucketResponse(body)}}
}

func assertNamesSupportedPullRequestURL(t *testing.T, err error) {
	t.Helper()
	message := grpcerrors.StatusMessage(err)
	assert.Contains(t, message, "pull request URL")
	assert.Contains(t, message, "GitHub")
	assert.Contains(t, message, "Bitbucket")
}

func seedHandOffFactory(t *testing.T, r *support.ResourceRegistry, db *gorm.DB) *models.Factory {
	t.Helper()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	repo := "acme/app"
	branch := "main"
	require.NoError(t, factoryModel.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		AppRepository: &repo,
		DefaultBranch: &branch,
	}))
	return factoryModel
}

func useGitHub(t *testing.T, pullRequest *github.PullRequest) {
	t.Helper()
	original := newFactoryGitHubAPI
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return &fakeFactoryGitHub{pullRequest: pullRequest}, nil
	}
	t.Cleanup(func() { newFactoryGitHubAPI = original })
}

func openHandOffPullRequest(repository string, number int, headRef string) *github.PullRequest {
	return forkHandOffPullRequest(repository, repository, number, headRef)
}

func forkHandOffPullRequest(baseRepository, headRepository string, number int, headRef string) *github.PullRequest {
	return &github.PullRequest{
		ID:      github.Ptr(int64(1000 + number)),
		Number:  github.Ptr(number),
		Title:   github.Ptr("Ready"),
		State:   github.Ptr("open"),
		HTMLURL: github.Ptr("https://github.com/" + baseRepository + "/pull/" + strconv.Itoa(number)),
		Head: &github.PullRequestBranch{
			Ref:  github.Ptr(headRef),
			Repo: &github.Repository{FullName: github.Ptr(headRepository)},
		},
		Base: &github.PullRequestBranch{Repo: &github.Repository{FullName: github.Ptr(baseRepository)}},
	}
}

func renameCanvas(t *testing.T, db *gorm.DB, canvas *models.Canvas, name string) {
	t.Helper()
	require.NoError(t, db.Model(canvas).Update("name", name).Error)
	canvas.Name = name
}

func planningSpecBody(t *testing.T, artifact *models.FactoryWorkOrderArtifact) string {
	t.Helper()
	var data map[string]any
	require.NoError(t, json.Unmarshal(artifact.Data, &data))
	body, _ := data["body"].(string)
	return body
}

func hasBranchArtifact(artifacts []models.FactoryWorkOrderArtifact, name, repository string) bool {
	for _, artifact := range artifacts {
		if artifact.Type != models.FactoryWorkOrderArtifactTypeBranch {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(artifact.Data, &data); err != nil {
			continue
		}
		if data["name"] == name && data["repository"] == repository {
			return true
		}
	}
	return false
}

func createChecksStartupCanvas(t *testing.T, r *support.ResourceRegistry, factoryModel *models.Factory, repository string) *models.Canvas {
	t.Helper()
	db := database.DB(t.Context())
	template := buildChecksPRFeedbackCanvas(prFeedbackBuildRequest{Repository: repository, MaximumAttempts: prFeedbackDefaultMaximumAttempts, CheckNames: []string{"build"}})
	var nodes []models.CanvasNode
	for _, node := range template.Nodes() {
		if node.ID == prFeedbackPullRequestTriggerNodeID {
			node.ID = "renamed-pr-trigger"
			node.Configuration["ignoreDrafts"] = true
		}
		nodes = append(nodes, models.CanvasNode{NodeID: node.ID, Name: node.Name, Type: node.Type, Ref: datatypes.NewJSONType(node.Ref), Configuration: datatypes.NewJSONType(node.Configuration)})
	}
	edges := template.Edges()
	for i := range edges {
		if edges[i].SourceID == prFeedbackPullRequestTriggerNodeID {
			edges[i].SourceID = "renamed-pr-trigger"
		}
	}
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nodes, edges)
	require.NoError(t, db.Model(canvas).Update("factory_id", factoryModel.ID).Error)
	_, err := factoryModel.CreatePRFeedbackHandler(db, canvas.ID, models.FactoryPRFeedbackHandlerSubjectGitHubPullRequest, models.FactoryPRFeedbackHandlerSourcePullRequestChecks)
	require.NoError(t, err)
	return canvas
}

func TestSubscriptionHandOffWaitsForAssignmentAndExplicitStart(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		column string
		reopen bool
	}{
		{name: "implement draft", column: HandOffColumnImplement},
		{name: "verify draft", column: HandOffColumnVerify},
		{name: "implement reopened", column: HandOffColumnImplement, reopen: true},
		{name: "verify reopened", column: HandOffColumnVerify, reopen: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			column := scenario.column
			r := support.Setup(t)
			db := database.DB(t.Context())
			require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
			f := seedHandOffFactory(t, r, db)
			subscription := true
			require.NoError(t, f.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
			app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, f.ID, "implement", "start")
			line, err := f.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry}})
			require.NoError(t, err)
			ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
			result, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
				FactoryID: f.ID.String(), Title: "Keep my handoff", Description: "Description", Plan: "The saved plan", Column: column, PullRequestURL: "https://github.com/acme/app/pull/42", MCPClientName: "Codex",
			})
			require.NoError(t, err)
			assert.Equal(t, "backlog", result.Column)
			assert.Equal(t, pb.WorkOrder_STATE_DRAFT, result.Order.State)
			assert.Empty(t, result.Order.Assignees)
			assert.Empty(t, result.Order.LineDispatches)
			assert.Empty(t, result.Order.PullRequests)
			require.NotNil(t, result.Order.PendingHandoff)
			assert.Equal(t, column, result.Order.PendingHandoff.Column)
			assert.Equal(t, line.ID.String(), result.Order.PendingHandoff.LineId)
			order, err := f.FindWorkOrder(db, uuid.MustParse(result.Order.Id))
			require.NoError(t, err)
			artifact, err := order.FindArtifactByKey(db, models.PlanningSpecArtifactKey+":"+order.ID.String())
			require.NoError(t, err)
			assert.Equal(t, "The saved plan", planningSpecBody(t, artifact))
			connection, err := models.CreateFactoryCodexConnection(db, r.Organization.ID, f.ID, r.User, time.Now())
			require.NoError(t, err)
			require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
			require.NoError(t, order.UpdateAssignees(db, []uuid.UUID{r.User}, r.User))
			assigned, err := f.FindWorkOrder(db, order.ID)
			require.NoError(t, err)
			assert.Equal(t, models.FactoryWorkOrderStateDraft, assigned.State)
			assert.NotNil(t, assigned.PendingHandOff)
			if column == HandOffColumnVerify {
				useGitHub(t, openHandOffPullRequest("acme/app", 42, "feature"))
			}
			if scenario.reopen {
				_, err = CloseWorkOrder(ctx, r.Organization.ID.String(), &pb.CloseWorkOrderRequest{
					FactoryId: f.ID.String(), OrderId: order.ID.String(), Result: pb.WorkOrder_RESULT_REJECTED,
				})
				require.NoError(t, err)
				_, err = ResumeHandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.ResumeHandOffWorkOrderRequest{FactoryId: f.ID.String(), OrderId: order.ID.String()})
				require.Error(t, err)
				assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
				closed, err := f.FindWorkOrder(db, order.ID)
				require.NoError(t, err)
				require.NotNil(t, closed.PendingHandOff)
				assert.Equal(t, column, closed.PendingHandOff.Data().Column)
				dispatches, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, []uuid.UUID{order.ID})
				require.NoError(t, err)
				assert.Empty(t, dispatches[order.ID])
				reopened, err := UpdateWorkOrderStatus(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderStatusRequest{
					FactoryId: f.ID.String(), OrderId: order.ID.String(), State: pb.WorkOrder_STATE_OPEN,
				})
				require.NoError(t, err)
				assert.Equal(t, pb.WorkOrder_STATE_OPEN, reopened.Order.State)
				require.NotNil(t, reopened.Order.PendingHandoff)
			}
			resumed, err := ResumeHandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.ResumeHandOffWorkOrderRequest{FactoryId: f.ID.String(), OrderId: order.ID.String()})
			require.NoError(t, err)
			assert.Equal(t, pb.WorkOrder_STATE_OPEN, resumed.Order.State)
			assert.Nil(t, resumed.Order.PendingHandoff)
			require.Len(t, resumed.Order.LineDispatches, 1)
			_, err = ResumeHandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.ResumeHandOffWorkOrderRequest{FactoryId: f.ID.String(), OrderId: order.ID.String()})
			require.Error(t, err)
			dispatches, err := models.ListWorkOrderLineDispatchesByWorkOrderIDs(db, []uuid.UUID{order.ID})
			require.NoError(t, err)
			assert.Len(t, dispatches[order.ID], 1)
		})
	}
}

func seedPendingVerifyHandOff(t *testing.T) (context.Context, *support.ResourceRegistry, *models.Factory, *models.FactoryWorkOrder) {
	t.Helper()
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	f := seedHandOffFactory(t, r, db)
	subscription := true
	require.NoError(t, f.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	addHandOffLine(t, r, db, f)
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	result, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
		FactoryID: f.ID.String(), Title: "Stale handoff", Description: "Description", Plan: "The saved plan", Column: HandOffColumnVerify, PullRequestURL: "https://github.com/acme/app/pull/42", MCPClientName: "Codex",
	})
	require.NoError(t, err)
	require.NotNil(t, result.Order.PendingHandoff)
	order, err := f.FindWorkOrder(db, uuid.MustParse(result.Order.Id))
	require.NoError(t, err)
	connection, err := models.CreateFactoryCodexConnection(db, r.Organization.ID, f.ID, r.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
	require.NoError(t, order.UpdateAssignees(db, []uuid.UUID{r.User}, r.User))
	return ctx, r, f, order
}

func useGitHubError(t *testing.T, err error) {
	t.Helper()
	original := newFactoryGitHubAPI
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return &fakeFactoryGitHub{getPullErr: err}, nil
	}
	t.Cleanup(func() { newFactoryGitHubAPI = original })
}

func requireHandOffDiscarded(t *testing.T, f *models.Factory, order *models.FactoryWorkOrder, resumed *pb.ResumeHandOffWorkOrderResponse) {
	t.Helper()
	db := database.DB(t.Context())
	assert.Equal(t, "backlog", resumed.Column)
	assert.Equal(t, pb.WorkOrder_STATE_DRAFT, resumed.Order.State)
	assert.Nil(t, resumed.Order.PendingHandoff)
	assert.Empty(t, resumed.Order.PullRequests)
	stored, err := f.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Nil(t, stored.PendingHandOff)
	artifact, err := stored.FindArtifactByKey(db, models.PlanningSpecArtifactKey+":"+stored.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "The saved plan", planningSpecBody(t, artifact))
}

func TestResumeHandOffDiscardsVerifyHandOffWhenPullRequestIsNoLongerOpen(t *testing.T) {
	ctx, r, f, order := seedPendingVerifyHandOff(t)
	merged := openHandOffPullRequest("acme/app", 42, "feature")
	merged.State = github.Ptr("closed")
	useGitHub(t, merged)

	resumed, err := ResumeHandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.ResumeHandOffWorkOrderRequest{FactoryId: f.ID.String(), OrderId: order.ID.String()})
	require.NoError(t, err)
	requireHandOffDiscarded(t, f, order, resumed)
}

func TestResumeHandOffDiscardsVerifyHandOffWhenPullRequestIsGone(t *testing.T) {
	ctx, r, f, order := seedPendingVerifyHandOff(t)
	useGitHubError(t, &github.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}, Message: "Not Found"})

	resumed, err := ResumeHandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.ResumeHandOffWorkOrderRequest{FactoryId: f.ID.String(), OrderId: order.ID.String()})
	require.NoError(t, err)
	requireHandOffDiscarded(t, f, order, resumed)
}

func TestResumeHandOffKeepsVerifyHandOffWhenPullRequestLoadFailsTemporarily(t *testing.T) {
	ctx, r, f, order := seedPendingVerifyHandOff(t)
	useGitHubError(t, &github.ErrorResponse{Response: &http.Response{StatusCode: http.StatusBadGateway}, Message: "Bad Gateway"})

	_, err := ResumeHandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.ResumeHandOffWorkOrderRequest{FactoryId: f.ID.String(), OrderId: order.ID.String()})
	require.Error(t, err)
	stored, err := f.FindWorkOrder(database.DB(t.Context()), order.ID)
	require.NoError(t, err)
	assert.NotNil(t, stored.PendingHandOff)
}

func TestIsHandOffPullRequestGoneRecognizesBitbucketNotFound(t *testing.T) {
	assert.True(t, isHandOffPullRequestGone(&bitbucket.APIError{StatusCode: http.StatusNotFound}))
	assert.False(t, isHandOffPullRequestGone(&bitbucket.APIError{StatusCode: http.StatusInternalServerError}))
	assert.False(t, isHandOffPullRequestGone(errors.New("timeout")))
}

func TestSubscriptionHandOffRejectsInvalidPullRequestURLBeforeSaving(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	ctx := authentication.SetUserIdInMetadata(t.Context(), r.User.String())
	githubFactory := seedHandOffFactory(t, r, db)
	bitbucketFactory := seedBitbucketHandOffFactory(t, r, db)
	for _, factory := range []*models.Factory{githubFactory, bitbucketFactory} {
		subscription := true
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
		addHandOffLine(t, r, db, factory)
	}
	githubClientCalls := 0
	original := newFactoryGitHubAPI
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		githubClientCalls++
		return &fakeFactoryGitHub{}, nil
	}
	t.Cleanup(func() { newFactoryGitHubAPI = original })
	bitbucketHTTP := stubBitbucketMergeAPI(t, nil)
	for _, scenario := range []struct {
		name    string
		factory *models.Factory
		url     string
	}{
		{name: "malformed GitHub URL", factory: githubFactory, url: "https://github.com/acme/app/issues/42"},
		{name: "unsupported URL scheme", factory: githubFactory, url: "ftp://github.com/acme/app/pull/42"},
		{name: "URL with credentials", factory: githubFactory, url: "https://someone@github.com/acme/app/pull/42"},
		{name: "malformed Bitbucket URL", factory: bitbucketFactory, url: "https://bitbucket.org/acme/app/pull/42"},
		{name: "Bitbucket URL in GitHub workspace", factory: githubFactory, url: "https://bitbucket.org/acme/app/pull-requests/42"},
		{name: "GitHub URL in Bitbucket workspace", factory: bitbucketFactory, url: "https://github.com/acme/app/pull/42"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			_, err := HandOffWorkOrder(ctx, IntakeDependencies{}, r.Organization.ID.String(), HandOffWorkOrderRequest{
				FactoryID: scenario.factory.ID.String(), Title: "Invalid handoff", Plan: "The saved plan", Column: HandOffColumnVerify, PullRequestURL: scenario.url,
			})
			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
			assertNamesSupportedPullRequestURL(t, err)
			orders, err := scenario.factory.ListWorkOrders(db, models.ListFactoryWorkOrdersFilters{})
			require.NoError(t, err)
			assert.Empty(t, orders)
			assert.Zero(t, githubClientCalls)
			assert.Empty(t, bitbucketHTTP.Requests)
		})
	}
}
