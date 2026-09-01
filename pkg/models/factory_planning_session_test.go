package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestFactory_StartPlanningSession(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-start")
	db := database.Conn()
	canvas, entrypoint := createPlanningCanvas(t, org.ID, factoryModel.ID, userID)

	session, err := factoryModel.StartPlanningSession(db, StartPlanningSessionParams{
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		Entrypoint:      entrypoint,
	})
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, PlanningSessionStateRunning, session.State)
	assert.Equal(t, "acme/payments", session.Repository)
	assert.Equal(t, canvas.ID, *session.CanvasID)
	require.NotNil(t, session.CanvasRunID)
	assert.Equal(t, userID, session.CreatedByUserID)
	assert.Empty(t, session.Messages)

	var run CanvasRun
	require.NoError(t, db.First(&run, "id = ?", session.CanvasRunID).Error)
	assert.Equal(t, canvas.ID, run.WorkflowID)
	assert.Equal(t, entrypoint, run.NodeID)
	assert.Equal(t, CanvasRunStatePending, run.State)
}

func TestFactory_StartPlanningSession_RequiresRepository(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	org, userID, factoryModel := setupFactoryWithUser(t, "plan-repo")
	db := database.Conn()
	canvas, entrypoint := createPlanningCanvas(t, org.ID, factoryModel.ID, userID)

	_, err := factoryModel.StartPlanningSession(db, StartPlanningSessionParams{
		CreatedByUserID: userID,
		CanvasID:        canvas.ID,
		Entrypoint:      entrypoint,
	})
	assert.ErrorIs(t, err, ErrFactoryPlanningSessionInvalid)
}

func TestFactoryPlanningSession_HeartbeatAndEnd(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-hb")
	db := database.Conn()

	before := session.HeartbeatAt
	require.NoError(t, session.Heartbeat(db))
	reloaded, err := FindPlanningSession(db, session.OrganizationID, session.FactoryID, session.ID)
	require.NoError(t, err)
	assert.True(t, reloaded.HeartbeatAt.After(before))

	require.NoError(t, session.End(db))
	assert.Equal(t, PlanningSessionStateEnded, session.State)
	require.NotNil(t, session.EndedAt)

	err = session.Heartbeat(db)
	assert.ErrorIs(t, err, ErrFactoryPlanningSessionEnded)
}

func TestFactoryPlanningSession_EndIfStale(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-stale")
	db := database.Conn()

	require.NoError(t, db.Model(session).Update("heartbeat_at", time.Now().Add(-2*time.Minute)).Error)
	require.NoError(t, db.First(session, "id = ?", session.ID).Error)

	ended, err := session.EndIfStale(db, time.Now())
	require.NoError(t, err)
	assert.True(t, ended)
	assert.Equal(t, PlanningSessionStateEnded, session.State)
}

func TestFactoryPlanningSession_SendMessageResolvesWait(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-wait")
	db := database.Conn()

	require.NoError(t, session.BeginWait(db))
	assert.Equal(t, PlanningWaitPending, session.WaitState)

	require.NoError(t, session.SendUserMessage(db, "Add refund retries"))
	assert.Equal(t, PlanningWaitResolved, session.WaitState)
	assert.Equal(t, PlanningWaitKindMessage, session.WaitResult.Data().Kind)
	assert.Equal(t, "Add refund retries", lastTextMessage(session, PlanningSessionMessageRoleUser))

	result, err := session.ConsumeWait(db)
	require.NoError(t, err)
	assert.Equal(t, PlanningWaitKindMessage, result.Kind)
	assert.Equal(t, "Add refund retries", result.Text)
	assert.Equal(t, PlanningWaitIdle, session.WaitState)
}

func TestFactoryPlanningSession_BeginWaitDeliversQueuedUserMessage(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-queue")
	db := database.Conn()

	require.NoError(t, session.SendUserMessage(db, "Add a puppy color field"))
	assert.Equal(t, PlanningWaitIdle, session.WaitState)

	require.NoError(t, session.BeginWait(db))
	assert.Equal(t, PlanningWaitResolved, session.WaitState)

	result, err := session.ConsumeWait(db)
	require.NoError(t, err)
	assert.Equal(t, PlanningWaitKindMessage, result.Kind)
	assert.Equal(t, "Add a puppy color field", result.Text)
	assert.Equal(t, PlanningWaitIdle, session.WaitState)
}

func TestFactoryPlanningSession_ProposeAndSkipDraft(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-draft")
	db := database.Conn()

	require.NoError(t, session.ProposeDraft(db, PlanningSessionDraft{
		Title:       "Retry refunds",
		Description: "Stop double charges.",
	}))
	assert.Equal(t, "Retry refunds", session.PendingDraft.Data().Title)

	require.NoError(t, session.UpdateDraft(db, PlanningSessionDraft{
		Title:       "Retry refunds once",
		Description: "Edited.",
	}))
	require.NoError(t, session.BeginWait(db))
	require.NoError(t, session.SkipDraft(db))
	assert.Equal(t, "", session.PendingDraft.Data().Title)
	assert.Equal(t, PlanningWaitKindSkipped, session.WaitResult.Data().Kind)
}

func TestFactoryPlanningSession_CreateDraftWorkOrder(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-create")
	db := database.Conn()
	factoryModel, err := FindFactory(db, session.OrganizationID, session.FactoryID)
	require.NoError(t, err)

	require.NoError(t, session.ProposeDraft(db, PlanningSessionDraft{
		Title:       "Retry refunds",
		Description: "Stop double charges.",
	}))
	require.NoError(t, session.BeginWait(db))

	order, err := session.CreateDraftWorkOrder(db, factoryModel, session.CreatedByUserID)
	require.NoError(t, err)
	assert.Equal(t, FactoryWorkOrderStateDraft, order.State)
	assert.Equal(t, "Retry refunds", order.Title)
	require.Len(t, session.CreatedWorkOrderIDs, 1)
	assert.Equal(t, order.ID.String(), session.CreatedWorkOrderIDs[0])
	assert.Equal(t, PlanningWaitKindCreated, session.WaitResult.Data().Kind)
	assert.Equal(t, "", session.PendingDraft.Data().Title)
}

func TestFactoryPlanningSession_ListStaleOpenPlanningSessions(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	session := startTestPlanningSession(t, "plan-list-stale")
	fresh := startTestPlanningSession(t, "plan-list-fresh")
	db := database.Conn()

	require.NoError(t, db.Model(session).Update("heartbeat_at", time.Now().Add(-2*time.Minute)).Error)

	stale, err := ListStaleOpenPlanningSessions(db, time.Now(), 10)
	require.NoError(t, err)
	require.Len(t, stale, 1)
	assert.Equal(t, session.ID, stale[0].ID)
	assert.NotEqual(t, fresh.ID, stale[0].ID)
}

func startTestPlanningSession(t *testing.T, prefix string) *FactoryPlanningSession {
	t.Helper()
	org, userID, factoryModel := setupFactoryWithUser(t, prefix)
	db := database.Conn()
	canvas, entrypoint := createPlanningCanvas(t, org.ID, factoryModel.ID, userID)
	session, err := factoryModel.StartPlanningSession(db, StartPlanningSessionParams{
		CreatedByUserID: userID,
		Repository:      "acme/payments",
		CanvasID:        canvas.ID,
		Entrypoint:      entrypoint,
	})
	require.NoError(t, err)
	return session
}

func createPlanningCanvas(t *testing.T, orgID, factoryID, userID uuid.UUID) (*Canvas, string) {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	entrypoint := "start"
	canvas := &Canvas{
		ID:             uuid.New(),
		OrganizationID: orgID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           PlanningCanvasName,
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		node := CanvasNode{
			WorkflowID: canvas.ID,
			NodeID:     entrypoint,
			Name:       "Planning",
			Type:       NodeTypeTrigger,
			State:      CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(NodeRef{
				Trigger: &TriggerRef{Name: "onRun"},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		version := CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &userID,
			Nodes: datatypes.NewJSONSlice([]Node{{
				ID:   entrypoint,
				Name: "Planning",
				Type: NodeTypeTrigger,
				Ref:  NodeRef{Trigger: &TriggerRef{Name: "onRun"}},
			}}),
			Edges:     datatypes.NewJSONSlice([]Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		return tx.Create(&version).Error
	}))
	return canvas, entrypoint
}

func lastTextMessage(session *FactoryPlanningSession, role string) string {
	for i := len(session.Messages) - 1; i >= 0; i-- {
		if session.Messages[i].Kind == PlanningSessionMessageKindText && session.Messages[i].Role == role {
			return session.Messages[i].Text
		}
	}
	return ""
}
