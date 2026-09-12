package factories

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__FindPlanningSessionByWorkOrder__ReturnsAnalysisSession(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)

	legacyCanvas, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "planning", "start")
	_, err = factoryModel.StartPlanningSession(db, models.StartPlanningSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        legacyCanvas.ID,
		Entrypoint:      entrypoint,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)

	_, err = FindPlanningSessionByWorkOrder(ctx, r.Organization.ID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		WorkOrderId: order.ID.String(),
	})
	require.Error(t, err)

	canvas, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "backlog", "start")
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)

	found, err := FindPlanningSessionByWorkOrder(ctx, r.Organization.ID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		WorkOrderId: order.ID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, found.Session)
	assert.Equal(t, session.ID.String(), found.Session.Id)
	assert.Equal(t, order.ID.String(), found.Session.Draft.WorkOrderId)
	assert.Empty(t, found.Session.ExecutionId)

	event := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, intakeAnalysisNodeID, event.ID, event.ID)
	require.NoError(t, db.Model(execution).Update("run_id", run.ID).Error)

	found, err = FindPlanningSessionByWorkOrder(ctx, r.Organization.ID.String(), &pb.FindPlanningSessionByWorkOrderRequest{
		FactoryId:   factoryModel.ID.String(),
		WorkOrderId: order.ID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, found.Session)
	assert.Equal(t, execution.ID.String(), found.Session.ExecutionId)
}

func Test__SendPlanningSessionMessage__RestartsEndedAnalysis(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	canvas := createOnWorkOrderCanvas(t, r, factoryModel.ID)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	require.NoError(t, session.ProposeSpec(db, "# Retry refunds\n\n## Executive summary\n\nStop double charges.\n"))
	require.NoError(t, session.SendUserMessage(db, "Use the current retry form."))
	require.NoError(t, session.RecordAgentMessage(db, "I updated the plan with the current form."))
	require.NoError(t, session.End(db))

	sent, err := SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: session.ID.String(),
		Text:      "Keep the existing retry helper.",
	})
	require.NoError(t, err)
	require.NotNil(t, sent.Session)
	assert.Equal(t, models.PlanningSessionStateRunning, sent.Session.State)
	assert.Empty(t, sent.Session.CanvasRunId)
	require.Len(t, sent.Session.Messages, 3)
	assert.Equal(t, "Use the current retry form.", sent.Session.Messages[0].Text)
	assert.Equal(t, models.PlanningSessionMessageRoleUser, sent.Session.Messages[0].Role)
	assert.Equal(t, "I updated the plan with the current form.", sent.Session.Messages[1].Text)
	assert.Equal(t, models.PlanningSessionMessageRoleAgent, sent.Session.Messages[1].Role)
	assert.Equal(t, "Keep the existing retry helper.", sent.Session.Messages[2].Text)
	assert.Equal(t, models.PlanningSessionMessageRoleUser, sent.Session.Messages[2].Role)

	events, err := models.ListCanvasEvents(db, canvas.ID, "start", 10, nil)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	payload, ok := events[0].Data.Data().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, factory.OnWorkOrderPayloadType, payload["type"])
}

func Test__SendPlanningSessionMessage__RestartsCancelledAnalysisRun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	canvas := createOnWorkOrderCanvas(t, r, factoryModel.ID)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	require.NoError(t, finishCanvasRun(db, run, models.CanvasRunResultCancelled))
	require.Equal(t, models.PlanningSessionStateRunning, session.State)

	before, err := models.ListCanvasEvents(db, canvas.ID, "start", 10, nil)
	require.NoError(t, err)

	sent, err := SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: session.ID.String(),
		Text:      "Keep the existing retry helper.",
	})
	require.NoError(t, err)
	require.NotNil(t, sent.Session)
	assert.Equal(t, models.PlanningSessionStateRunning, sent.Session.State)
	assert.Empty(t, sent.Session.CanvasRunId)
	require.GreaterOrEqual(t, len(sent.Session.Messages), 1)
	assert.Equal(t, "Keep the existing retry helper.", sent.Session.Messages[len(sent.Session.Messages)-1].Text)

	after, err := models.ListCanvasEvents(db, canvas.ID, "start", 10, nil)
	require.NoError(t, err)
	require.Greater(t, len(after), len(before))
	payload, ok := after[0].Data.Data().(map[string]any)
	require.True(t, ok)
	assert.Equal(t, factory.OnWorkOrderPayloadType, payload["type"])
}

func Test__SendPlanningSessionMessage__KeepsLiveAnalysisOnTheCurrentRun(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	canvas := createOnWorkOrderCanvas(t, r, factoryModel.ID)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	before, err := models.ListCanvasEvents(db, canvas.ID, "start", 10, nil)
	require.NoError(t, err)

	sent, err := SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: session.ID.String(),
		Text:      "Keep the existing retry helper.",
	})
	require.NoError(t, err)
	require.NotNil(t, sent.Session)
	assert.Equal(t, run.ID.String(), sent.Session.CanvasRunId)

	after, err := models.ListCanvasEvents(db, canvas.ID, "start", 10, nil)
	require.NoError(t, err)
	assert.Equal(t, len(before), len(after))
}

func Test__SendPlanningSessionMessage__DoesNotRestartAfterTaskStarts(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	canvas := createOnWorkOrderCanvas(t, r, factoryModel.ID)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		CreatedByUserID: r.User,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		CanvasRunID:     run.ID,
		WorkOrderID:     order.ID,
	})
	require.NoError(t, err)
	require.NoError(t, order.TransitionOnDispatch(db, &r.User))

	_, err = SendPlanningSessionMessage(ctx, r.Organization.ID.String(), &pb.SendPlanningSessionMessageRequest{
		FactoryId: factoryModel.ID.String(),
		SessionId: session.ID.String(),
		Text:      "Add refund retries",
	})
	require.Error(t, err)

	reloaded, err := models.FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, models.PlanningSessionStateEnded, reloaded.State)
}

func createOnWorkOrderCanvas(t *testing.T, r *support.ResourceRegistry, factoryID uuid.UUID) *models.Canvas {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           "Backlog",
		CreatedBy:      &r.User,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.DB(t.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		node := models.CanvasNode{
			WorkflowID: canvas.ID,
			NodeID:     "start",
			Name:       "On Task",
			Type:       models.NodeTypeTrigger,
			State:      models.CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: factory.OnWorkOrderTriggerName},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		version := models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &r.User,
			Nodes: datatypes.NewJSONSlice([]models.Node{{
				ID:   "start",
				Name: "On Task",
				Type: models.NodeTypeTrigger,
				Ref:  models.NodeRef{Trigger: &models.TriggerRef{Name: factory.OnWorkOrderTriggerName}},
			}}),
			Edges:     datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		return tx.Create(&version).Error
	}))
	return canvas
}

func finishCanvasRun(db *gorm.DB, run *models.CanvasRun, result string) error {
	now := time.Now()
	run.State = models.CanvasRunStateFinished
	run.Result = result
	run.FinishedAt = &now
	run.UpdatedAt = &now
	return db.Model(run).Updates(map[string]any{
		"state":       run.State,
		"result":      run.Result,
		"finished_at": run.FinishedAt,
		"updated_at":  run.UpdatedAt,
	}).Error
}
