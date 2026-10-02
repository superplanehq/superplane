package factories

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
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

	useGitHub(t, openHandOffPullRequest("acme/app", 42, "feat/refunds"))

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
