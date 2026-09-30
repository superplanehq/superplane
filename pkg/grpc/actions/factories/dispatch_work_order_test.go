package factories

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

// Test__DispatchWorkOrder__CreatesLineDispatchWithSnapshot covers acceptance
// criterion 1: dispatching a work order creates one line dispatch with the
// line's current steps snapshotted, and step 1's execution references it.
func Test__DispatchWorkOrder__CreatesLineDispatchWithSnapshot(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	require.Len(t, resp.Order.LineDispatches, 1)
	dispatch := resp.Order.LineDispatches[0]
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, dispatch.State)
	assert.Equal(t, line.Name, dispatch.Line.Name)
	assert.Empty(t, dispatch.Model)
	require.Len(t, dispatch.Steps, 1)
	assert.Equal(t, app.Name, dispatch.Steps[0].Name)
	require.Len(t, dispatch.StepExecutions, 1)
	assert.Equal(t, pb.WorkOrderExecution_STATE_PENDING, dispatch.StepExecutions[0].State)

	reloaded, err := models.FindUnscopedWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)

	active, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, dispatch.Id, active.ID.String())
}

func Test__DispatchWorkOrder__ClearsAutoStartLine(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrderWithAutoStart(db, "Ship it", "", &r.User, nil, nil, &line.ID)
	require.NoError(t, err)
	require.NotNil(t, order.AutoStartLineID)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Order.GetAutoStartLineId())

	reloaded, err := models.FindUnscopedWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
	assert.Nil(t, reloaded.AutoStartLineID)
}

func Test__DispatchWorkOrder__CompletesActiveAnalysisRun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	analysisCanvas := createOnWorkOrderCanvas(t, r, factoryModel.ID)
	analysisRun, err := models.CreateCanvasRunInTransaction(
		db,
		analysisCanvas.ID,
		"start",
		models.CanvasRunStateStarted,
		"",
	)
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    analysisCanvas.ID,
		CanvasRunID: analysisRun.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)

	event := support.EmitCanvasEventForNode(t, analysisCanvas.ID, "start", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, analysisCanvas.ID, backlogRefinementNodeID, event.ID, event.ID)
	require.NoError(t, db.Model(execution).Update("run_id", analysisRun.ID).Error)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	updatedSession, err := models.FindPlanningSession(db, r.Organization.ID, factoryModel.ID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, updatedSession.State)

	updatedRun, err := models.FindUnscopedCanvasRun(db, analysisRun.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, updatedRun.State)
	assert.Equal(t, models.CanvasRunResultPassed, updatedRun.Result)
	result, err := updatedRun.CalculateResult(db)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunResultPassed, result)

	updatedExecution, err := models.FindNodeExecutionInTransaction(db, analysisCanvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateCancelling, updatedExecution.State)
}

// Test__DispatchWorkOrder__RejectsWhenAlreadyActive covers acceptance
// criterion 5: a work order with an active line dispatch cannot be
// dispatched again.
func Test__DispatchWorkOrder__RejectsWhenAlreadyActive(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.Error(t, err)
	code, _, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, code)
}

// Test__DispatchWorkOrder__SecondDispatchAfterFirstFinishesCreatesSeparateTraversal
// covers acceptance criterion 4: dispatching a work order to the same line
// twice, after the first traversal finishes, produces two traversals shown
// separately by the API.
func Test__DispatchWorkOrder__SecondDispatchAfterFirstFinishesCreatesSeparateTraversal(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	firstResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	require.Len(t, firstResp.Order.LineDispatches, 1)
	firstDispatchID := firstResp.Order.LineDispatches[0].Id

	firstDispatch, err := models.FindWorkOrderLineDispatch(db, uuid.MustParse(firstDispatchID))
	require.NoError(t, err)
	require.NoError(t, firstDispatch.Finish(db, models.CanvasRunResultFailed))

	secondResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)

	require.Len(t, secondResp.Order.LineDispatches, 2,
		"two separate traversals of the same line, not one merged bucket")
	assert.Equal(t, firstDispatchID, secondResp.Order.LineDispatches[0].Id)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_FINISHED, secondResp.Order.LineDispatches[0].State)
	assert.Equal(t, pb.WorkOrderLineDispatch_RESULT_FAILED, secondResp.Order.LineDispatches[0].Result)
	assert.NotEqual(t, firstDispatchID, secondResp.Order.LineDispatches[1].Id)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, secondResp.Order.LineDispatches[1].State)
}

func Test__DispatchWorkOrder__ReplaceActiveFinishesCurrentAndStartsAgain(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(database.DB(t.Context()), "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	firstResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	firstID := firstResp.Order.LineDispatches[0].Id

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:     factoryModel.ID.String(),
		OrderId:       order.ID.String(),
		LineName:      line.Name,
		ReplaceActive: true,
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.LineDispatches, 2)
	assert.Equal(t, firstID, resp.Order.LineDispatches[0].Id)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_FINISHED, resp.Order.LineDispatches[0].State)
	assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, resp.Order.LineDispatches[1].State)
	assert.Equal(t, int32(0), resp.Order.LineDispatches[1].StepExecutions[0].StepIndex)
}

func Test__DispatchWorkOrder__StartsAtRequestedStep(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-two", "start-two")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:      factoryModel.ID.String(),
		OrderId:        order.ID.String(),
		LineName:       line.Name,
		StartStepIndex: 1,
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.LineDispatches, 1)
	require.Len(t, resp.Order.LineDispatches[0].StepExecutions, 1)
	assert.Equal(t, int32(1), resp.Order.LineDispatches[0].StepExecutions[0].StepIndex)
	assert.Equal(t, secondApp.Name, resp.Order.LineDispatches[0].StepExecutions[0].Step)
}

func Test__DispatchWorkOrder__RerunStepKeepsEarlierExecutions(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Improve AGENTS.md", "", &r.User, nil, nil)
	require.NoError(t, err)

	firstApp, firstEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "plan", "start-plan")
	secondApp, secondEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-implement")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: firstApp.ID, Entrypoint: firstEntry},
		{Type: models.FactoryLineStepTypeRunApp, AppID: secondApp.ID, Entrypoint: secondEntry},
	})
	require.NoError(t, err)

	firstResp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	require.Len(t, firstResp.Order.LineDispatches, 1)
	firstID := firstResp.Order.LineDispatches[0].Id

	firstDispatch, err := models.FindWorkOrderLineDispatch(db, uuid.MustParse(firstID))
	require.NoError(t, err)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, startErr := firstDispatch.EnqueueOrStartStep(tx, order, 1)
		if startErr != nil {
			return startErr
		}
		return firstDispatch.Finish(tx, models.CanvasRunResultFailed)
	}))

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId:      factoryModel.ID.String(),
		OrderId:        order.ID.String(),
		LineName:       line.Name,
		StartStepIndex: 1,
		ReplaceActive:  true,
	})
	require.NoError(t, err)

	var planCount int
	for _, dispatch := range resp.Order.LineDispatches {
		if dispatch.Id == firstID {
			for _, execution := range dispatch.StepExecutions {
				if execution.StepIndex == 0 {
					planCount++
				}
			}
		}
	}
	assert.Equal(t, 1, planCount, "rerun of Implementation must keep the Planning execution")
	require.Len(t, resp.Order.LineDispatches, 1)
	require.GreaterOrEqual(t, len(resp.Order.LineDispatches[0].StepExecutions), 2)
}

func Test__DispatchWorkOrder__ReturnsOwnerNameAfterStart(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", nil, nil, nil)
	require.NoError(t, err)

	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.Assignees, 1)
	assert.Equal(t, r.User.String(), resp.Order.Assignees[0].Id)
	assert.Equal(t, r.UserModel.Name, resp.Order.Assignees[0].Name)
	assert.NotEqual(t, r.User.String(), resp.Order.Assignees[0].Name)
}

func Test__DispatchWorkOrder__PersistsChosenModel(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6", "claude-opus-4-6")

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "hosted", "claude-sonnet-4-6")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
		Model:     "claude-opus-4-6",
	})
	require.NoError(t, err)
	require.Len(t, resp.Order.LineDispatches, 1)
	assert.Equal(t, "claude-opus-4-6", resp.Order.LineDispatches[0].Model)

	active, err := order.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-4-6", active.Model)
}

func Test__DispatchWorkOrder__RejectsModelNotOnLine(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	seedHostedModels(t, db, models.UsageProviderAnthropic, "claude-sonnet-4-6")
	seedHostedModels(t, db, models.UsageProviderOpenAI, "gpt-5")

	app := createLineAppWithRunner(t, r, factoryModel.ID, runnerClaudeCode, "hosted", "claude-sonnet-4-6")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	_, err = DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		LineName:  line.Name,
		Model:     "gpt-5",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func Test__DispatchWorkOrder__FillsMissingTitleBeforeStart(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "step-one", "start-one")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	longLine := strings.Repeat("a", 300)
	skippedLead := "\n\n  ![shot](sp-file://abc)\nnull\nUntitled task\nn/a\n"
	tests := []struct {
		name        string
		title       string
		description string
		wantTitle   string
	}{
		{
			name:        "empty title",
			title:       "",
			description: "Refunds fail on retry.\n\nMore context.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "null",
			title:       "null",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "unknown",
			title:       "unknown",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "Unknown",
			title:       "Unknown",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "quoted null",
			title:       `"null"`,
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "n/a",
			title:       "n/a",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "quoted null with trailing punctuation",
			title:       `"null."`,
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "meaningful title stays",
			title:       "Ship the retry fix",
			description: "A different first line.",
			wantTitle:   "Ship the retry fix",
		},
		{
			name:        "null pointer stays",
			title:       "null pointer",
			description: "A different first line.",
			wantTitle:   "null pointer",
		},
		{
			name:        "skips blank image and placeholder lines",
			title:       "",
			description: skippedLead + "Fix the checkout retry.",
			wantTitle:   "Fix the checkout retry.",
		},
		{
			name:        "heading line",
			title:       "",
			description: "## Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "fully wrapped emphasis",
			title:       "null",
			description: "## **Refunds fail on retry.**",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "collapses whitespace",
			title:       "",
			description: "Refunds   fail\ton retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "cuts a long derived title",
			title:       "unknown",
			description: longLine + "\nMore context.",
			wantTitle:   longLine[:workOrderTitleMaxLength],
		},
		{
			name:        "untitled task is still a placeholder",
			title:       "Untitled task",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "skips a list placeholder",
			title:       "null",
			description: "- n/a\nRefunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "skips an attachment file link",
			title:       "unknown",
			description: "[notes.pdf](sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee)\nRefunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
		{
			name:        "attachment only becomes untitled task",
			title:       "",
			description: "[notes.pdf](sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee)",
			wantTitle:   untitledWorkOrderTitle,
		},
		{
			name:        "no real description line",
			title:       "",
			description: skippedLead,
			wantTitle:   untitledWorkOrderTitle,
		},
		{
			name:        "title with no letter or digit",
			title:       "???",
			description: "Refunds fail on retry.",
			wantTitle:   "Refunds fail on retry.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			order, err := factoryModel.CreateWorkOrder(db, "Seed title", tc.description, &r.User, nil, nil)
			require.NoError(t, err)
			title := tc.title
			require.NoError(t, order.UpdateContent(db, &title, nil))

			resp, err := DispatchWorkOrder(ctx, r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
				FactoryId: factoryModel.ID.String(),
				OrderId:   order.ID.String(),
				LineName:  line.Name,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.wantTitle, resp.Order.Title)
			require.NotEmpty(t, resp.Order.LineDispatches)
			assert.Equal(t, pb.WorkOrderLineDispatch_STATE_ACTIVE, resp.Order.LineDispatches[0].State)

			reloaded, err := models.FindUnscopedWorkOrder(db, order.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.wantTitle, reloaded.Title)
			assert.Equal(t, tc.description, reloaded.Description)
			assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
		})
	}
}
